package pool

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/cervantesh/cervo-mutants/internal/testharness"
)

func TestWaitFreeMemoryAndStitchLogs(t *testing.T) {
	if err := waitFreeMemory(nil, CommandSpec{MinFreeMemoryMB: 128}); err != nil {
		t.Fatalf("waitFreeMemory nil monitor returned error: %v", err)
	}
	if err := waitFreeMemory(&sequenceMonitor{statuses: []MemoryStatus{{FreeMemoryMB: 512, FreeCommitMB: 1024}}}, CommandSpec{MinFreeMemoryMB: 128, MinFreeCommitMB: 256}); err != nil {
		t.Fatalf("waitFreeMemory satisfied thresholds returned error: %v", err)
	}

	err := waitFreeMemory(&sequenceMonitor{
		statuses: []MemoryStatus{{FreeMemoryMB: 64, FreeCommitMB: 64}},
		delay:    5 * time.Millisecond,
	}, CommandSpec{
		MinFreeMemoryMB: 128,
		MinFreeCommitMB: 256,
		MemoryWait:      time.Millisecond,
	})
	if err == nil || !strings.Contains(err.Error(), "skipped after waiting") {
		t.Fatalf("waitFreeMemory timeout mismatch: %v", err)
	}

	fixture := testharness.NewDir(t)
	stdoutPath := fixture.WriteFile(t, "stdout.log", "one")
	stderrPath := fixture.WriteFile(t, "stderr.log", "two")
	logPath := fixture.Path("combined.log")
	if err := stitchLogs(stdoutPath, stderrPath, logPath); err != nil {
		t.Fatalf("stitchLogs returned error: %v", err)
	}
	if got := readFile(logPath); got != "one\ntwo" {
		t.Fatalf("stitched log = %q", got)
	}
	if _, err := os.Stat(stdoutPath); !os.IsNotExist(err) {
		t.Fatalf("stdout fragment still exists: %v", err)
	}
	if _, err := os.Stat(stderrPath); !os.IsNotExist(err) {
		t.Fatalf("stderr fragment still exists: %v", err)
	}
}

func TestRealCommandRunnerRejectsEmptyPath(t *testing.T) {
	_, err := RealCommandRunner{}.Run(context.Background(), CommandSpec{})
	if err == nil || !strings.Contains(err.Error(), "command path is empty") {
		t.Fatalf("empty path error mismatch: %v", err)
	}
}

func TestRealCommandRunnerSkipsWhenPreflightMemoryWaitExpires(t *testing.T) {
	fixture := testharness.NewDir(t)
	spec := CommandSpec{
		Path:            "unused",
		LogPath:         fixture.Path("logs", "command.log"),
		MinFreeMemoryMB: 512,
		MemoryWait:      time.Millisecond,
	}
	result, err := RealCommandRunner{Monitor: &sequenceMonitor{
		statuses: []MemoryStatus{{FreeMemoryMB: 64}},
		delay:    5 * time.Millisecond,
	}}.Run(context.Background(), spec)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.ExitCode != 125 {
		t.Fatalf("ExitCode = %d, want 125", result.ExitCode)
	}
	if text := readFile(spec.LogPath); !strings.Contains(text, "skipped after waiting") {
		t.Fatalf("preflight skip log mismatch: %q", text)
	}
}

func TestRealCommandRunnerHandlesStartFailure(t *testing.T) {
	fixture := testharness.NewDir(t)
	spec := CommandSpec{
		Path:    filepath.Join(fixture.Root, "missing-binary.exe"),
		LogPath: fixture.Path("logs", "start.log"),
	}
	result, err := RealCommandRunner{}.Run(context.Background(), spec)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.ExitCode != 125 {
		t.Fatalf("ExitCode = %d, want 125", result.ExitCode)
	}
	if text := readFile(spec.LogPath); text == "" {
		t.Fatal("start failure log should not be empty")
	}
}

func TestRealCommandRunnerStitchesProcessOutput(t *testing.T) {
	fixture := testharness.NewDir(t)
	path, args := helperProcessCommand("stdout-stderr")
	spec := CommandSpec{
		Path:       path,
		Args:       args,
		LogPath:    fixture.Path("logs", "success.log"),
		MemoryPoll: 10 * time.Millisecond,
		Env:        helperProcessEnv(),
	}
	result, err := RealCommandRunner{}.Run(context.Background(), spec)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", result.ExitCode)
	}
	text := readFile(spec.LogPath)
	for _, want := range []string{"stdout line", "stderr line"} {
		if !strings.Contains(text, want) {
			t.Fatalf("stitched log missing %q:\n%s", want, text)
		}
	}
}

func TestRealCommandRunnerTimeoutAndCancellation(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		fixture := testharness.NewDir(t)
		path, args := helperProcessCommand("sleep", "500ms")
		spec := CommandSpec{
			Path:       path,
			Args:       args,
			LogPath:    fixture.Path("logs", "timeout.log"),
			Timeout:    50 * time.Millisecond,
			MemoryPoll: 10 * time.Millisecond,
			Env:        helperProcessEnv(),
		}
		result, err := RealCommandRunner{}.Run(context.Background(), spec)
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
		if result.ExitCode != 124 {
			t.Fatalf("ExitCode = %d, want 124", result.ExitCode)
		}
		if text := readFile(spec.LogPath); !strings.Contains(text, "timed out after") {
			t.Fatalf("timeout log mismatch: %q", text)
		}
	})

	t.Run("canceled", func(t *testing.T) {
		fixture := testharness.NewDir(t)
		path, args := helperProcessCommand("sleep", "500ms")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		spec := CommandSpec{
			Path:       path,
			Args:       args,
			LogPath:    fixture.Path("logs", "cancel.log"),
			Timeout:    time.Second,
			MemoryPoll: 10 * time.Millisecond,
			Env:        helperProcessEnv(),
		}
		result, err := RealCommandRunner{}.Run(ctx, spec)
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
		if result.ExitCode != 124 {
			t.Fatalf("ExitCode = %d, want 124", result.ExitCode)
		}
		if text := readFile(spec.LogPath); !strings.Contains(text, "command canceled") {
			t.Fatalf("cancel log mismatch: %q", text)
		}
	})
}

func TestRealCommandRunnerMemoryWatchdogKill(t *testing.T) {
	fixture := testharness.NewDir(t)
	path, args := helperProcessCommand("sleep", "500ms")
	spec := CommandSpec{
		Path:                  path,
		Args:                  args,
		LogPath:               fixture.Path("logs", "watchdog.log"),
		Timeout:               time.Second,
		MemoryPoll:            10 * time.Millisecond,
		KillBelowFreeMemoryMB: 128,
		Env:                   helperProcessEnv(),
	}
	monitor := &sequenceMonitor{statuses: []MemoryStatus{
		{FreeMemoryMB: 512},
		{FreeMemoryMB: 64},
	}}
	result, err := RealCommandRunner{Monitor: monitor}.Run(context.Background(), spec)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.ExitCode != 126 {
		t.Fatalf("ExitCode = %d, want 126", result.ExitCode)
	}
	if text := readFile(spec.LogPath); !strings.Contains(text, "killed by memory watchdog") {
		t.Fatalf("watchdog log mismatch: %q", text)
	}
}

func TestRealCommandRunnerProcessTreeMemoryWatchdogKill(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("process-tree memory watchdog is only active on Windows")
	}
	fixture := testharness.NewDir(t)
	path, args := helperProcessCommand("alloc-sleep", "16", "500ms")
	spec := CommandSpec{
		Path:                   path,
		Args:                   args,
		LogPath:                fixture.Path("logs", "job-memory.log"),
		Timeout:                time.Second,
		MemoryPoll:             10 * time.Millisecond,
		MaxProcessTreeMemoryMB: 1,
		Env:                    helperProcessEnv(),
	}
	result, err := RealCommandRunner{}.Run(context.Background(), spec)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.ExitCode != 126 {
		t.Fatalf("ExitCode = %d, want 126", result.ExitCode)
	}
	if text := readFile(spec.LogPath); !strings.Contains(text, "process-tree memory watchdog") {
		t.Fatalf("process-tree watchdog log mismatch: %q", text)
	}
}

func TestWindowsProcessAndMemoryMonitors(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific monitor coverage")
	}
	if _, err := attachProcessMonitor(exec.Command(os.Args[0])); err == nil {
		t.Fatal("attachProcessMonitor should reject a command that has not started")
	}

	path, args := helperProcessCommand("sleep", "250ms")
	cmd := exec.Command(path, args...)
	cmd.Env = append(os.Environ(), helperProcessEnv()...)
	if err := cmd.Start(); err != nil {
		t.Fatalf("helper process start failed: %v", err)
	}
	monitor, err := attachProcessMonitor(cmd)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("attachProcessMonitor returned error: %v", err)
	}
	if peak := monitor.peakJobMemoryBytes(); peak < 0 {
		t.Fatalf("peakJobMemoryBytes = %d, want non-negative", peak)
	}
	monitor.close()
	_ = cmd.Process.Kill()
	_ = cmd.Wait()

	status, err := systemMemoryMonitor{}.Status()
	if err != nil {
		t.Fatalf("systemMemoryMonitor returned error: %v", err)
	}
	if status.TotalMemoryMB <= 0 || status.TotalCommitMB <= 0 {
		t.Fatalf("unexpected system memory status: %+v", status)
	}
}
