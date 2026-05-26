package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEvalCommandWritesEvaluationArtifacts(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module fixture\n\ngo 1.25.6\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "calc.go"), []byte(`package fixture

func IsPositiveOrZero(n int) bool {
	return n >= 0
}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "calc_test.go"), []byte(`package fixture

import "testing"

func TestIsPositiveOrZero(t *testing.T) {
	if !IsPositiveOrZero(1) {
		t.Fatal("want positive")
	}
}
`), 0o600); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(dir, "eval-out")
	if err := run([]string{"eval", dir, "--max-mutants", "1", "--out", out}); err != nil {
		t.Fatalf("eval command returned error: %v", err)
	}
	for _, name := range []string{"evaluation.json", "evaluation.md", "mutation-report.json"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Fatalf("%s was not written: %v", name, err)
		}
	}
}
