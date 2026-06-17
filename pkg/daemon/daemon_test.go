package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cervantesh/cervo-mutants/pkg/engine"
)

type stubRunner struct {
	result engine.MutantResult
	err    error
}

func (r stubRunner) Run(context.Context, engine.MutantJob) (engine.MutantResult, error) {
	return r.result, r.err
}

func TestWorkerRunnerAppliesMutantInIsolatedWorkdir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module workerfixture\n\ngo 1.25.6\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	src := []byte("package workerfixture\n\nfunc Check() bool { return true }\n")
	if err := os.WriteFile(filepath.Join(dir, "check.go"), src, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "check_test.go"), []byte(`package workerfixture

import "testing"

func TestCheck(t *testing.T) {
	if !Check() {
		t.Fatal("want true")
	}
}
`), 0o600); err != nil {
		t.Fatal(err)
	}

	start := strings.Index(string(src), "true")
	job := engine.MutantJob{
		ID: "job1",
		Mutant: engine.Mutant{
			ID:          "check.go:3:boolean-literals",
			Module:      dir,
			File:        filepath.Join(dir, "check.go"),
			Package:     ".",
			Original:    "true",
			Mutated:     "false",
			StartOffset: start,
			EndOffset:   start + len("true"),
		},
		WorkDir:     dir,
		TestCommand: []string{"go", "test", "."},
		Timeout:     "30s",
	}
	var in bytes.Buffer
	if err := json.NewEncoder(&in).Encode(Message{Type: "job", Job: job}); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := ServeJSONLines(context.Background(), &in, &out, WorkerRunner{MaxOutputBytes: 12000}); err != nil {
		t.Fatalf("ServeJSONLines returned error: %v", err)
	}
	var msg Message
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &msg); err != nil {
		t.Fatalf("worker output is not JSON: %s", out.String())
	}
	if msg.Type != "result" || msg.Result.Status != engine.StatusKilled {
		t.Fatalf("worker did not execute real mutant: %+v", msg)
	}
	after, err := os.ReadFile(filepath.Join(dir, "check.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(src) {
		t.Fatal("worker mutated original workdir instead of isolated copy")
	}
}

func TestWorkerRunnerRejectsMutantFileOutsideModule(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module workerfixture\n\ngo 1.25.6\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "check.go"), []byte("package workerfixture\n\nfunc Check() bool { return true }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.go")
	if err := os.WriteFile(outside, []byte("package outside\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := WorkerRunner{MaxOutputBytes: 12000}.Run(context.Background(), engine.MutantJob{
		ID: "outside",
		Mutant: engine.Mutant{
			ID:          "outside",
			Module:      dir,
			File:        outside,
			Original:    "true",
			Mutated:     "false",
			StartOffset: 0,
			EndOffset:   1,
		},
		WorkDir:     dir,
		TestCommand: []string{"go", "test", "."},
		Timeout:     "30s",
	})
	if err == nil {
		t.Fatal("WorkerRunner accepted mutant file outside module")
	}
}

func TestServeJSONLinesReportsMalformedUnsupportedAndRunnerErrors(t *testing.T) {
	input := strings.Join([]string{
		`{bad json}`,
		`{"type":"ping"}`,
		`{"type":"job","job":{"id":"j1"}}`,
	}, "\n")
	var out bytes.Buffer
	err := ServeJSONLines(context.Background(), strings.NewReader(input), &out, stubRunner{err: errors.New("runner failed")})
	if err != nil {
		t.Fatalf("ServeJSONLines returned error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("output lines = %d: %s", len(lines), out.String())
	}
	for _, line := range lines {
		var msg Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			t.Fatalf("invalid JSON line %q: %v", line, err)
		}
		if msg.Type != "error" || msg.Error == "" {
			t.Fatalf("expected error message, got %+v", msg)
		}
	}
}

func TestApplyPatchRejectsInvalidOffsetsAndOriginalMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.go")
	if err := os.WriteFile(path, []byte("package p\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, mutant := range []engine.Mutant{
		{StartOffset: -1, EndOffset: 1, Original: "p", Mutated: "q"},
		{StartOffset: 0, EndOffset: 7, Original: "missing", Mutated: "q"},
	} {
		if err := applyPatch(path, mutant); err == nil {
			t.Fatalf("applyPatch accepted invalid mutant: %+v", mutant)
		}
	}
}
