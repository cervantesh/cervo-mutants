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
	if err := run([]string{"eval", dir, "--policy", "ci-fast", "--resume", "--max-process-memory-mb", "128", "--budget", "1m", "--test-timeout", "5s", "--max-mutants", "1", "--sample", "deterministic", "--workers", "1", "--isolation", "overlay", "--framework", "mutation-testing", "--out", out}); err != nil {
		t.Fatalf("eval command returned error: %v", err)
	}
	for _, name := range []string{"evaluation.json", "evaluation.md", "mutation-report.json"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Fatalf("%s was not written: %v", name, err)
		}
	}
}

func TestHelpCommandDoesNotError(t *testing.T) {
	if err := run(nil); err != nil {
		t.Fatalf("empty run returned error: %v", err)
	}
	if err := run([]string{"--help"}); err != nil {
		t.Fatalf("--help returned error: %v", err)
	}
	if err := run([]string{"help"}); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
}

func TestRunDispatchesSimpleCommands(t *testing.T) {
	dir := writeCLIFixture(t)
	t.Chdir(dir)
	if err := run([]string{"init"}); err != nil {
		t.Fatalf("run init returned error: %v", err)
	}
	if err := os.Remove(filepath.Join(dir, configFileName)); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"doctor"}); err != nil {
		t.Fatalf("run doctor returned error: %v", err)
	}
	if err := run([]string{"affected", dir}); err != nil {
		t.Fatalf("run affected returned error: %v", err)
	}
	if err := run([]string{"list-mutators"}); err != nil {
		t.Fatalf("run list-mutators returned error: %v", err)
	}
	if err := run([]string{"explain", "m1"}); err != nil {
		t.Fatalf("run explain returned error: %v", err)
	}
	if err := run([]string{"run", dir, "--dry-run", "--max-mutants", "1"}); err != nil {
		t.Fatalf("run dispatch returned error: %v", err)
	}
}

func TestInitListMutatorsExplainAndExitCodes(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := cmdInit(); err != nil {
		t.Fatalf("cmdInit returned error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, configFileName))
	if err != nil {
		t.Fatalf("config not written: %v", err)
	}
	if !strings.Contains(string(data), "mutators:") || !strings.Contains(defaultConfigYAML(), "reports:") {
		t.Fatalf("default config missing expected sections: %s", data)
	}
	if err := cmdInit(); err == nil {
		t.Fatal("cmdInit overwrote existing config")
	}
	if err := cmdListMutators(); err != nil {
		t.Fatalf("cmdListMutators returned error: %v", err)
	}
	if err := cmdExplain([]string{"m1", "--format", "json"}); err != nil {
		t.Fatalf("cmdExplain returned error: %v", err)
	}
	if exitCode(os.ErrPermission) != 2 || exitCode(assertErr("threshold failed")) != 1 || exitCode(assertErr("baseline tests failed")) != 3 {
		t.Fatal("exitCode returned unexpected values")
	}
}

func TestDoctorAffectedFastAndBaselineCommands(t *testing.T) {
	dir := writeCLIFixture(t)
	out := filepath.Join(dir, "out")
	if err := cmdDoctor(); err != nil {
		t.Fatalf("cmdDoctor returned error: %v", err)
	}
	if err := cmdAffected([]string{dir}); err != nil {
		t.Fatalf("cmdAffected returned error: %v", err)
	}
	if err := cmdFast([]string{dir, "--max-mutants", "1", "--out", out}); err != nil {
		t.Fatalf("cmdFast returned error: %v", err)
	}

	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, configFileName), []byte("version: 1\nreports:\n  output: "+filepath.ToSlash(out)+"\nbaseline:\n  path: "+filepath.ToSlash(filepath.Join(out, "baseline.json"))+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cmdBaseline([]string{"update"}); err != nil {
		t.Fatalf("baseline update returned error: %v", err)
	}
	if err := cmdBaseline([]string{"compare"}); err != nil {
		t.Fatalf("baseline compare returned error: %v", err)
	}
	if err := cmdBaseline(nil); err == nil {
		t.Fatal("baseline accepted missing subcommand")
	}
	if err := cmdBaseline([]string{"unknown"}); err == nil {
		t.Fatal("baseline accepted unknown subcommand")
	}
}

func TestRunCommandErrorsAndReorderFlags(t *testing.T) {
	if err := run([]string{"unknown"}); err == nil {
		t.Fatal("run accepted unknown command")
	}
	if err := run([]string{"run", "--bad-flag"}); err == nil {
		t.Fatal("run accepted bad flag")
	}
	if err := run([]string{"affected", "--bad-flag"}); err == nil {
		t.Fatal("affected accepted bad flag")
	}
	if err := run([]string{"eval", "--bad-flag"}); err == nil {
		t.Fatal("eval accepted bad flag")
	}
	got := reorderFlags([]string{"./...", "--max-mutants", "1", "--dry-run"}, map[string]bool{flagMaxMutants: true})
	if strings.Join(got, " ") != "--max-mutants 1 --dry-run ./..." {
		t.Fatalf("reorderFlags = %q", strings.Join(got, " "))
	}
}

type assertErr string

func (e assertErr) Error() string { return string(e) }

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
	if err := run([]string{"explain", id, "--format", "text"}); err != nil {
		t.Fatalf("explain text returned error: %v", err)
	}
}

func TestReportShowExplainErrorBranches(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out")
	if err := os.MkdirAll(out, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"report", "--out", out}); err == nil {
		t.Fatal("report accepted missing action")
	}
	if err := run([]string{"report", "unknown", "--out", out}); err == nil {
		t.Fatal("report accepted unknown action")
	}
	if err := run([]string{"show", "--out", out}); err == nil {
		t.Fatal("show accepted missing mutant id")
	}
	if err := run([]string{"explain"}); err == nil {
		t.Fatal("explain accepted missing mutant id")
	}
	if err := run([]string{"compare", "--bad-flag"}); err == nil {
		t.Fatal("compare accepted bad flag")
	}
}

func TestRunAcceptsWorkerAndIsolationFlags(t *testing.T) {
	dir := writeCLIFixture(t)
	out := filepath.Join(dir, "parallel-out")
	if err := run([]string{"run", dir, "--max-mutants", "1", "--workers", "2", "--isolation", "overlay", "--out", out}); err != nil {
		t.Fatalf("run with workers and isolation returned error: %v", err)
	}
}

func TestRunDryRunAndThresholdFailureBranches(t *testing.T) {
	dir := writeCLIFixture(t)
	out := filepath.Join(dir, "dry-out")
	if err := run([]string{"run", dir, "--dry-run", "--policy", "ci-balanced", "--coverage-prefilter", "--resume", "--max-process-memory-mb", "128", "--budget", "1m", "--test-timeout", "5s", "--max-mutants", "1", "--sample", "deterministic", "--profile", "conservative-fast", "--report", "json,summary", "--out", out}); err != nil {
		t.Fatalf("dry-run returned error: %v", err)
	}

	cfgText := `version: 1
ci:
  fail_under: 100
reports:
  output: ` + filepath.ToSlash(filepath.Join(dir, "threshold-out")) + `
limits:
  max_mutants: 1
`
	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, configFileName), []byte(cfgText), 0o600); err != nil {
		t.Fatal(err)
	}
	err := run([]string{"run", dir})
	if err == nil || !strings.Contains(err.Error(), "threshold") {
		t.Fatalf("run should fail threshold, got %v", err)
	}
}

func TestCompareRequiresAtLeastOneReport(t *testing.T) {
	if err := cmdCompare([]string{"--out", filepath.Join(t.TempDir(), "out.json")}); err == nil {
		t.Fatal("cmdCompare accepted no reports")
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
	for _, want := range []string{"cervo-mutants", "gremlins", "gomu", "go-mutesting"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("comparison output missing %q: %s", want, data)
		}
	}
}

func TestCompareCommandRecordsGremlinsEffectiveTarget(t *testing.T) {
	dir := t.TempDir()
	gremlins := filepath.Join(dir, "gremlins.json")
	out := filepath.Join(dir, "compare.json")
	if err := os.WriteFile(gremlins, []byte(`{"mutants_total":1,"mutants_killed":1,"mutants_lived":0,"test_efficacy":100}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := run([]string{"compare", "--gremlins", gremlins, "--gremlins-target", "./...", "--gremlins-target-mode", "gremlins-package-root", "--out", out}); err != nil {
		t.Fatalf("compare returned error: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("comparison output missing: %v", err)
	}
	for _, want := range []string{`"target": "./..."`, `"effective_target": "."`, `"not_comparable": true`, `"status": "ok"`} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("comparison output missing %q: %s", want, data)
		}
	}
}

func TestCompareCommandRecordsApplesToApplesPackageRootMode(t *testing.T) {
	dir := t.TempDir()
	cervo := filepath.Join(dir, "cervo.json")
	gremlins := filepath.Join(dir, "gremlins.json")
	out := filepath.Join(dir, "compare.json")
	if err := os.WriteFile(cervo, []byte(`{"summary":{"total":1,"killed":1,"score":100}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(gremlins, []byte(`{"mutants_total":1,"mutants_killed":1,"mutants_lived":0,"test_efficacy":100}`), 0o600); err != nil {
		t.Fatal(err)
	}

	err := run([]string{
		"compare",
		"--cervomut", cervo,
		"--cervomut-target", "./...",
		"--cervomut-target-mode", "package-root",
		"--gremlins", gremlins,
		"--gremlins-target", "./...",
		"--gremlins-target-mode", "package-root",
		"--out", out,
	})
	if err != nil {
		t.Fatalf("compare returned error: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("comparison output missing: %v", err)
	}
	for _, want := range []string{`"apples_to_apples": true`, `"manifest_equivalent": false`, `"target_modes": [`, `"package-root"`} {
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
