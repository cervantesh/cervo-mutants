package runner

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/cervantesh/cervo-mutants/pkg/engine"
)

func TestRunRejectsEmptyCommand(t *testing.T) {
	_, err := (GoTestRunner{}).Run(context.Background(), engine.MutantJob{})
	if err == nil || !strings.Contains(err.Error(), "test command is empty") {
		t.Fatalf("expected empty command error, got %v", err)
	}
}

func TestRunClassifiesSurvivedKilledCompileErrorAndTimeout(t *testing.T) {
	r := GoTestRunner{MaxOutputBytes: 256}

	survived, err := r.Run(context.Background(), engine.MutantJob{
		ID:          "survived",
		Mutant:      engine.Mutant{ID: "survived"},
		WorkDir:     writeModule(t, "pass"),
		TestCommand: []string{"go", "test", "./..."},
		Timeout:     "10s",
	})
	if err != nil {
		t.Fatalf("survived run returned error: %v", err)
	}
	if survived.Status != engine.StatusSurvived {
		t.Fatalf("survived status mismatch: %+v", survived)
	}

	killed, err := r.Run(context.Background(), engine.MutantJob{
		ID:          "killed",
		Mutant:      engine.Mutant{ID: "killed"},
		WorkDir:     writeModule(t, "fail"),
		TestCommand: []string{"go", "test", "./..."},
		Timeout:     "10s",
	})
	if err != nil {
		t.Fatalf("killed run returned error: %v", err)
	}
	if killed.Status != engine.StatusKilled {
		t.Fatalf("killed status mismatch: %+v", killed)
	}

	compileError, err := r.Run(context.Background(), engine.MutantJob{
		ID:          "compile",
		Mutant:      engine.Mutant{ID: "compile"},
		WorkDir:     writeModule(t, "compile"),
		TestCommand: []string{"go", "test", "./..."},
		Timeout:     "10s",
	})
	if err != nil {
		t.Fatalf("compile run returned error: %v", err)
	}
	if compileError.Status != engine.StatusCompileError {
		t.Fatalf("compile status mismatch: %+v", compileError)
	}

	timedOut, err := r.Run(context.Background(), engine.MutantJob{
		ID:          "timeout",
		Mutant:      engine.Mutant{ID: "timeout"},
		WorkDir:     writeModule(t, "sleep"),
		TestCommand: []string{"go", "test", "./..."},
		Timeout:     "1ms",
	})
	if err != nil {
		t.Fatalf("timeout run returned error: %v", err)
	}
	if timedOut.Status != engine.StatusTimedOut {
		t.Fatalf("timeout status mismatch: %+v", timedOut)
	}
}

func TestTrim(t *testing.T) {
	if trim("abcdef", 3) != "abc" {
		t.Fatal("trim did not cap output")
	}
	if trim("abc", 0) != "abc" {
		t.Fatal("trim should not cap when max is zero")
	}
}

func writeModule(t *testing.T, mode string) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module runnerfixture\n\ngo 1.25.6\n")
	body := "package runnerfixture\n\nfunc Value() int { return 1 }\n"
	if mode == "compile" {
		body = "package runnerfixture\n\nfunc Value() int { return }\n"
	}
	writeFile(t, filepath.Join(dir, "value.go"), body)
	imports := `"testing"`
	if mode == "sleep" {
		imports = `"testing"
	"time"`
	}
	testBody := `package runnerfixture

import (
	` + imports + `
)

func TestValue(t *testing.T) {
	if Value() != 1 {
		t.Fatal("unexpected value")
	}
}
`
	switch mode {
	case "fail":
		testBody = strings.Replace(testBody, "Value() != 1", "Value() != 2", 1)
	case "sleep":
		testBody = strings.Replace(testBody, `if Value() != 1 {`, `time.Sleep(100 * time.Millisecond)
	if Value() != 1 {`, 1)
	}
	if runtime.GOOS == "windows" {
		testBody = strings.ReplaceAll(testBody, "\n", "\r\n")
	}
	writeFile(t, filepath.Join(dir, "value_test.go"), testBody)
	return dir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
