//go:build windows

package engine

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/cervantesh/cervo-mutants/pkg/config"
)

func TestRunTestClassifiesMemoryKilledOnWindowsJobObjectLimit(t *testing.T) {
	dir := t.TempDir()
	probeSource := filepath.Join(dir, "probe.go")
	probeExe := filepath.Join(dir, "probe.exe")
	source := `package main

import "time"

func main() {
	blocks := make([][]byte, 0)
	for i := 0; i < 64; i++ {
		b := make([]byte, 8*1024*1024)
		for j := 0; j < len(b); j += 4096 {
			b[j] = byte(i)
		}
		blocks = append(blocks, b)
		time.Sleep(10 * time.Millisecond)
	}
	_ = blocks
	time.Sleep(2 * time.Second)
}
`
	if err := os.WriteFile(probeSource, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	build := exec.Command("go", "build", "-o", probeExe, probeSource)
	build.Dir = dir
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build probe: %v\n%s", err, out)
	}
	cfg := config.Defaults()
	cfg.Tests.Timeout = 10 * time.Second
	cfg.Execution.Resources.MaxProcessMemoryMB = 64
	result, err := New(cfg).newRunSession().runTest(context.Background(), MutantJob{
		Mutant:      Mutant{ID: "memory"},
		WorkDir:     dir,
		TestCommand: []string{probeExe},
	})
	if err != nil {
		t.Fatalf("runTest returned error: %v", err)
	}
	if result.Status != StatusMemoryKilled || result.FailureKind != "memory_limit_exceeded" {
		t.Fatalf("unexpected memory-limited result: %+v", result)
	}
	if result.MemoryPeakBytes <= 0 {
		t.Fatalf("expected memory peak bytes to be recorded: %+v", result)
	}
}
