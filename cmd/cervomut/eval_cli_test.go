package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvalCommandWritesEvaluationArtifacts(t *testing.T) {
	dir := writeCLIFixture(t)

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

func TestHelpCommandDoesNotError(t *testing.T) {
	if err := run([]string{"--help"}); err != nil {
		t.Fatalf("--help returned error: %v", err)
	}
	if err := run([]string{"help"}); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
}

func TestReportAndShowAcceptOutputDirectory(t *testing.T) {
	dir := writeCLIFixture(t)
	out := filepath.Join(dir, "run-out")
	if err := run([]string{"run", dir, "--max-mutants", "1", "--out", out}); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(out, "mutation-report.json"))
	if err != nil {
		t.Fatalf("report missing: %v", err)
	}
	id := extractMutantIDForTest(t, string(data))

	if err := run([]string{"report", "summary", "--out", out}); err != nil {
		t.Fatalf("report summary --out returned error: %v", err)
	}
	if err := run([]string{"report", "survivors", "--out", out}); err != nil {
		t.Fatalf("report survivors --out returned error: %v", err)
	}
	if err := run([]string{"show", id, "--out", out}); err != nil {
		t.Fatalf("show --out returned error: %v", err)
	}
}

func TestRunAcceptsWorkerAndIsolationFlags(t *testing.T) {
	dir := writeCLIFixture(t)
	out := filepath.Join(dir, "parallel-out")
	if err := run([]string{"run", dir, "--max-mutants", "1", "--workers", "2", "--isolation", "overlay", "--out", out}); err != nil {
		t.Fatalf("run with workers and isolation returned error: %v", err)
	}
}

func TestCompareCommandNormalizesExternalToolReports(t *testing.T) {
	dir := t.TempDir()
	cervo := filepath.Join(dir, "cervo.json")
	gremlins := filepath.Join(dir, "gremlins.json")
	gomu := filepath.Join(dir, "gomu.txt")
	goMutesting := filepath.Join(dir, "go-mutesting.txt")
	out := filepath.Join(dir, "compare.json")
	if err := os.WriteFile(cervo, []byte(`{"summary":{"total":1,"killed":1,"score":100}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(gremlins, []byte(`{"total_mutants":1,"killed":1,"mutation_score":100}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(gomu, []byte(`total=1 killed=1 score=100`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(goMutesting, []byte(`The mutation score is 100.00%: 1 killed, 0 survived, 1 total`), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := run([]string{"compare", "--cervomut", cervo, "--gremlins", gremlins, "--gomu", gomu, "--go-mutesting", goMutesting, "--out", out}); err != nil {
		t.Fatalf("compare returned error: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("comparison output missing: %v", err)
	}
	for _, want := range []string{"cervo-mutant", "gremlins", "gomu", "go-mutesting"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("comparison output missing %q: %s", want, data)
		}
	}
}

func writeCLIFixture(t *testing.T) string {
	t.Helper()
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
	return dir
}

func extractMutantIDForTest(t *testing.T, report string) string {
	t.Helper()
	marker := `"mutant_id": "`
	start := strings.Index(report, marker)
	if start < 0 {
		t.Fatalf("report missing mutant_id: %s", report)
	}
	start += len(marker)
	end := strings.Index(report[start:], `"`)
	if end < 0 {
		t.Fatalf("report has malformed mutant_id: %s", report)
	}
	return report[start : start+end]
}
