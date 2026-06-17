//go:build windows

package engine

import (
	"os/exec"
	"testing"

	"github.com/cervantesh/cervo-mutants/pkg/config"
)

func TestApplyProcessLimitsWindowsBranches(t *testing.T) {
	cmd := exec.Command("cmd", "/c", "exit", "0")
	cleanup, err := applyProcessLimits(cmd, config.Resources{})
	if err != nil {
		t.Fatalf("applyProcessLimits without resources returned error: %v", err)
	}
	cleanup()

	_, err = applyProcessLimits(cmd, config.Resources{MaxProcessMemoryMB: 64})
	if err == nil {
		t.Fatal("applyProcessLimits accepted an unstarted process")
	}
}

func TestApplyProcessLimitsWindowsStartedProcess(t *testing.T) {
	cmd := exec.Command("cmd", "/c", "timeout", "/T", "2", "/NOBREAK")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start command: %v", err)
	}
	cleanup, err := applyProcessLimits(cmd, config.Resources{MaxProcessMemoryMB: 128, MaxProcesses: 1})
	cleanup()
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
	if err != nil {
		t.Fatalf("applyProcessLimits returned error for started process: %v", err)
	}
}
