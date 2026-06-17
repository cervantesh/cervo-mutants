package pool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/cervantesh/cervo-mutants/internal/testharness"
)

type fakeRunner struct {
	specs []CommandSpec
	run   func(CommandSpec) (CommandResult, error)
}

func (f *fakeRunner) Run(_ context.Context, spec CommandSpec) (CommandResult, error) {
	f.specs = append(f.specs, spec)
	return f.run(spec)
}

type sequenceMonitor struct {
	statuses []MemoryStatus
	errs     []error
	calls    int
	delay    time.Duration
}

func (m *sequenceMonitor) Status() (MemoryStatus, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	call := m.calls
	m.calls++
	if call < len(m.errs) && m.errs[call] != nil {
		return MemoryStatus{}, m.errs[call]
	}
	if len(m.statuses) == 0 {
		return MemoryStatus{}, nil
	}
	if call >= len(m.statuses) {
		return m.statuses[len(m.statuses)-1], nil
	}
	return m.statuses[call], nil
}

func writeManifest(t *testing.T, path string, repos []Repo) {
	t.Helper()
	dir := testharness.Dir{Root: filepath.Dir(path)}
	dir.WriteJSON(t, filepath.Base(path), Manifest{
		SchemaVersion: "1",
		Repos:         repos,
	})
}

func flagValue(args []string, flag string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag {
			return args[i+1]
		}
	}
	return ""
}

func helperProcessCommand(mode string, extra ...string) (string, []string) {
	args := []string{"-test.run=TestHelperProcess", "--", mode}
	args = append(args, extra...)
	return os.Args[0], args
}

func helperProcessEnv() []string {
	return []string{"GO_WANT_HELPER_PROCESS=1"}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	delim := -1
	for i, arg := range os.Args {
		if arg == "--" {
			delim = i
			break
		}
	}
	if delim < 0 || delim+1 >= len(os.Args) {
		os.Exit(2)
	}

	mode := os.Args[delim+1]
	switch mode {
	case "stdout-stderr":
		fmt.Fprintln(os.Stdout, "stdout line")
		fmt.Fprintln(os.Stderr, "stderr line")
	case "sleep":
		if delim+2 >= len(os.Args) {
			os.Exit(2)
		}
		duration, err := time.ParseDuration(os.Args[delim+2])
		if err != nil {
			os.Exit(2)
		}
		time.Sleep(duration)
	case "alloc-sleep":
		if delim+3 >= len(os.Args) {
			os.Exit(2)
		}
		sizeMB, err := strconv.Atoi(os.Args[delim+2])
		if err != nil {
			os.Exit(2)
		}
		duration, err := time.ParseDuration(os.Args[delim+3])
		if err != nil {
			os.Exit(2)
		}
		buf := make([]byte, sizeMB*1024*1024)
		for i := 0; i < len(buf); i += 4096 {
			buf[i] = 1
		}
		time.Sleep(duration)
		_ = buf
	default:
		os.Exit(2)
	}
	os.Exit(0)
}
