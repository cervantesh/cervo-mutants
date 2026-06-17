//go:build windows

package engine

import (
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"

	"github.com/cervantesh/CervoMutants/pkg/config"
)

const (
	jobObjectExtendedLimitInformationClass = 9
	jobObjectLimitProcessMemory            = 0x100
	jobObjectLimitKillOnJobClose           = 0x2000
	jobObjectLimitActiveProcess            = 0x8
	processSetQuota                        = 0x0100
	processTerminate                       = 0x0001
	processQueryLimitedInformation         = 0x1000
)

var (
	kernel32                    = syscall.NewLazyDLL("kernel32.dll")
	procCreateJobObjectW        = kernel32.NewProc("CreateJobObjectW")
	procSetInformationJobObject = kernel32.NewProc("SetInformationJobObject")
	procAssignProcessToJob      = kernel32.NewProc("AssignProcessToJobObject")
	procCloseHandle             = kernel32.NewProc("CloseHandle")
)

type ioCounters struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

type jobObjectBasicLimitInformation struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

type jobObjectExtendedLimitInformation struct {
	BasicLimitInformation jobObjectBasicLimitInformation
	IoInfo                ioCounters
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

func applyProcessLimits(cmd *exec.Cmd, resources config.Resources) (func(), error) {
	if !hasProcessLimits(resources) {
		return noopProcessLimitCleanup, nil
	}
	if cmd.Process == nil {
		return noopProcessLimitCleanup, fmt.Errorf("process resource limits require a started process")
	}
	job, _, err := procCreateJobObjectW.Call(0, 0)
	if job == 0 {
		return noopProcessLimitCleanup, fmt.Errorf("create Windows job object: %w", err)
	}
	cleanup := func() {
		procCloseHandle.Call(job)
	}
	info := jobObjectExtendedLimitInformation{}
	info.BasicLimitInformation.LimitFlags = jobObjectLimitKillOnJobClose
	if resources.MaxProcessMemoryMB > 0 {
		info.BasicLimitInformation.LimitFlags |= jobObjectLimitProcessMemory
		info.ProcessMemoryLimit = uintptr(resources.MaxProcessMemoryMB) * 1024 * 1024
	}
	if resources.MaxProcesses > 0 {
		info.BasicLimitInformation.LimitFlags |= jobObjectLimitActiveProcess
		info.BasicLimitInformation.ActiveProcessLimit = uint32(resources.MaxProcesses)
	}
	ok, _, err := procSetInformationJobObject.Call(
		job,
		uintptr(jobObjectExtendedLimitInformationClass),
		uintptr(unsafe.Pointer(&info)),
		unsafe.Sizeof(info),
	)
	if ok == 0 {
		cleanup()
		return noopProcessLimitCleanup, fmt.Errorf("set Windows job object limits: %w", err)
	}
	process, err := syscall.OpenProcess(processSetQuota|processTerminate|processQueryLimitedInformation, false, uint32(cmd.Process.Pid))
	if err != nil {
		cleanup()
		return noopProcessLimitCleanup, fmt.Errorf("open process for Windows job object: %w", err)
	}
	defer syscall.CloseHandle(process)
	ok, _, err = procAssignProcessToJob.Call(job, uintptr(process))
	if ok == 0 {
		cleanup()
		return noopProcessLimitCleanup, fmt.Errorf("assign process to Windows job object: %w", err)
	}
	return cleanup, nil
}
