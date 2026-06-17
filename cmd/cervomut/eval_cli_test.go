package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cervantesh/cervo-mutants/internal/testharness"
	"github.com/cervantesh/cervo-mutants/pkg/baseline"
	"github.com/cervantesh/cervo-mutants/pkg/config"
	"github.com/cervantesh/cervo-mutants/pkg/engine"
	evalpkg "github.com/cervantesh/cervo-mutants/pkg/eval"
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
	if !strings.Contains(string(data), "actionable_only: false") {
		t.Fatalf("default config missing actionable_only default: %s", data)
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
	baselinePath := filepath.Join(out, "baseline.json")
	previous := engine.RunResult{
		SchemaVersion: "1",
		Summary: engine.Summary{
			Score:      80,
			Actionable: engine.ActionableSummary{ActionableScore: 84},
		},
		Mutants: []engine.MutantResult{
			{MutantID: "keep", Status: engine.StatusSurvived},
			{MutantID: "flip", Status: engine.StatusKilled},
		},
	}
	writeRunReportForTest(t, out, previous)
	if err := cmdBaseline([]string{"update"}); err != nil {
		t.Fatalf("baseline update returned error: %v", err)
	}
	current := engine.RunResult{
		SchemaVersion: "1",
		Summary: engine.Summary{
			Score:      70,
			Actionable: engine.ActionableSummary{ActionableScore: 75},
		},
		Mutants: []engine.MutantResult{
			{MutantID: "keep", Status: engine.StatusSurvived},
			{MutantID: "flip", Status: engine.StatusSurvived},
			{MutantID: "new-survivor", Status: engine.StatusSurvived},
		},
	}
	writeRunReportForTest(t, out, current)
	compareOutput := captureStdout(t, func() {
		if err := cmdBaseline([]string{"compare"}); err != nil {
			t.Fatalf("baseline compare returned error: %v", err)
		}
	})
	if !strings.Contains(compareOutput, `"new_survivors":["flip","new-survivor"]`) {
		t.Fatalf("baseline compare output mismatch:\n%s", compareOutput)
	}
	diffOutput := captureStdout(t, func() {
		if err := cmdBaseline([]string{"diff"}); err != nil {
			t.Fatalf("baseline diff returned error: %v", err)
		}
	})
	for _, want := range []string{
		"Raw score: 80.00% -> 70.00% (-10.00)",
		"Actionable score: 84.00% -> 75.00% (-9.00)",
		"New survivors: 2",
		"- flip: killed -> survived",
	} {
		if !strings.Contains(diffOutput, want) {
			t.Fatalf("baseline diff output missing %q:\n%s", want, diffOutput)
		}
	}
	acceptOutput := captureStdout(t, func() {
		if err := cmdBaseline([]string{"accept"}); err != nil {
			t.Fatalf("baseline accept returned error: %v", err)
		}
	})
	candidatePath := baseline.CandidatePath(baselinePath)
	if _, err := os.Stat(candidatePath); err != nil {
		t.Fatalf("candidate baseline missing: %v", err)
	}
	if !strings.Contains(acceptOutput, filepath.ToSlash(candidatePath)) {
		t.Fatalf("baseline accept output missing candidate path:\n%s", acceptOutput)
	}
	diffCandidateJSON := captureStdout(t, func() {
		if err := cmdBaseline([]string{"diff", "--candidate", "--json"}); err != nil {
			t.Fatalf("baseline diff --candidate --json returned error: %v", err)
		}
	})
	if !strings.Contains(diffCandidateJSON, `"status_changes"`) || !strings.Contains(diffCandidateJSON, `"new_survivors":["flip","new-survivor"]`) {
		t.Fatalf("candidate diff json mismatch:\n%s", diffCandidateJSON)
	}
	promoteOutput := captureStdout(t, func() {
		if err := cmdBaseline([]string{"promote"}); err != nil {
			t.Fatalf("baseline promote returned error: %v", err)
		}
	})
	if !strings.Contains(promoteOutput, filepath.ToSlash(candidatePath)) {
		t.Fatalf("baseline promote output missing candidate path:\n%s", promoteOutput)
	}
	if _, err := os.Stat(candidatePath); !os.IsNotExist(err) {
		t.Fatalf("candidate baseline should be removed after promote: %v", err)
	}
	promoted, ok, err := baseline.Load(baselinePath)
	if err != nil || !ok || promoted.Summary.Score != 70 {
		t.Fatalf("promoted baseline mismatch: ok=%t err=%v result=%+v", ok, err, promoted)
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
	if err := cmdBaseline([]string{"promote"}); err == nil {
		t.Fatal("baseline promote accepted a missing candidate")
	}
}

func TestRunCommandErrorsAndReorderFlags(t *testing.T) {
	if err := run([]string{"unknown"}); err == nil {
		t.Fatal("run accepted unknown command")
	}
	if err := run([]string{"run", "--bad-flag"}); err == nil {
		t.Fatal("run accepted bad flag")
	}
	if err := run([]string{"run", "./...", "--shard", "bad"}); err == nil {
		t.Fatal("run accepted malformed shard")
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
	sarifOutput := captureStdout(t, func() {
		if err := run([]string{"report", "sarif", "--out", out}); err != nil {
			t.Fatalf("report sarif --out returned error: %v", err)
		}
	})
	if !strings.Contains(sarifOutput, `"version": "2.1.0"`) || !strings.Contains(sarifOutput, `"ruleId": "survived"`) {
		t.Fatalf("report sarif output unexpected:\n%s", sarifOutput)
	}
	githubSummaryOutput := captureStdout(t, func() {
		if err := run([]string{"report", "github-summary", "--out", out}); err != nil {
			t.Fatalf("report github-summary --out returned error: %v", err)
		}
	})
	if !strings.Contains(githubSummaryOutput, "## CervoMutants Mutation Summary") || !strings.Contains(githubSummaryOutput, "Top Survivor Queue") {
		t.Fatalf("report github-summary output unexpected:\n%s", githubSummaryOutput)
	}
	if err := run([]string{"show", id, "--out", out}); err != nil {
		t.Fatalf("show --out returned error: %v", err)
	}
	if err := run([]string{"explain", id, "--format", "text"}); err != nil {
		t.Fatalf("explain text returned error: %v", err)
	}
}

func TestRunAndReportActionableOnlyViews(t *testing.T) {
	dir := writeCLIFixture(t)
	out := filepath.Join(dir, "actionable-out")
	restoreCLIHooks(t)
	runEngineFn = func(cfg config.Config, req engine.RunRequest) (engine.RunResult, error) {
		if !cfg.Reports.ActionableOnly {
			t.Fatalf("run config should enable actionable-only: %+v", cfg.Reports)
		}
		_ = req
		return actionableOnlyRunResult(), nil
	}

	runOutput := captureStdout(t, func() {
		if err := run([]string{"run", dir, "--out", out, "--actionable-only"}); err != nil {
			t.Fatalf("run --actionable-only returned error: %v", err)
		}
	})
	if !strings.Contains(runOutput, "Actionable-only view: showing 2 of 4 survivors") {
		t.Fatalf("run stdout missing actionable-only header:\n%s", runOutput)
	}

	raw := readRunReportForTest(t, out)
	if len(raw.Mutants) != 4 {
		t.Fatalf("raw mutation report should preserve all mutants: %+v", raw.Mutants)
	}

	actionablePath := filepath.Join(out, "survivors-actionable.txt")
	data, err := os.ReadFile(actionablePath)
	if err != nil {
		t.Fatalf("survivors-actionable.txt missing: %v", err)
	}
	text := string(data)
	for _, want := range []string{"group-lead", "keep", "Actionable-only view: showing 2 of 4 survivors"} {
		if !strings.Contains(text, want) {
			t.Fatalf("actionable artifact missing %q:\n%s", want, text)
		}
	}
	for _, avoid := range []string{"#2 0.0 group-dup ", "#3 0.0 platform "} {
		if strings.Contains(text, avoid) {
			t.Fatalf("actionable artifact should not include %q:\n%s", avoid, text)
		}
	}

	reportOutput := captureStdout(t, func() {
		if err := run([]string{"report", "survivors", "--out", out, "--actionable-only"}); err != nil {
			t.Fatalf("report survivors --actionable-only returned error: %v", err)
		}
	})
	if !strings.Contains(reportOutput, "Actionable-only view: showing 2 of 4 survivors") || strings.Contains(reportOutput, "#3 0.0 platform ") {
		t.Fatalf("report survivors actionable-only output unexpected:\n%s", reportOutput)
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
	tempRoot := filepath.Join(dir, "temp-root")
	if err := run([]string{"run", dir, "--max-mutants", "1", "--workers", "2", "--isolation", "overlay", "--temp-root", tempRoot, "--slice-by", "package", "--shard", "1/1", "--max-files-per-run", "1", "--max-mutants-per-package", "1", "--out", out}); err != nil {
		t.Fatalf("run with workers and isolation returned error: %v", err)
	}
	if _, err := os.Stat(tempRoot); err != nil {
		t.Fatalf("temp root was not created: %v", err)
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

func TestRunCommandWritesStructuredFailureArtifactsOnConfigError(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	out := filepath.Join(dir, "config-error-out")
	if err := os.WriteFile(filepath.Join(dir, configFileName), []byte("version: [\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := run([]string{"run", dir, "--out", out})
	if err == nil || !strings.Contains(err.Error(), "config_error:") {
		t.Fatalf("run should return config_error, got %v", err)
	}
	assertCorrelationIDPresent(t, err.Error())

	report := readRunReportForTest(t, out)
	if report.Failure == nil || report.Failure.Kind != "config_error" {
		t.Fatalf("failure report = %+v", report.Failure)
	}
	if report.StoppedReason != "config_error" {
		t.Fatalf("stopped_reason = %q, want config_error", report.StoppedReason)
	}
	if len(report.Mutants) != 0 {
		t.Fatalf("failure report should not include mutants: %+v", report.Mutants)
	}
	debug := readFailureDebugForTest(t, out)
	if debug.Kind != "config_error" || debug.StackTrace != "" {
		t.Fatalf("debug artifact = %+v", debug)
	}
	if report.Failure.CorrelationID != debug.CorrelationID {
		t.Fatalf("correlation ids differ report=%q debug=%q", report.Failure.CorrelationID, debug.CorrelationID)
	}
}

func TestRunCommandWritesStructuredFailureArtifactsOnEnginePanic(t *testing.T) {
	dir := writeCLIFixture(t)
	out := filepath.Join(dir, "panic-out")
	restoreCLIHooks(t)
	runEngineFn = func(_ config.Config, _ engine.RunRequest) (engine.RunResult, error) {
		panic("engine panic")
	}

	err := run([]string{"run", dir, "--out", out})
	if err == nil || !strings.Contains(err.Error(), "internal_error:") {
		t.Fatalf("run should return internal_error, got %v", err)
	}
	assertCorrelationIDPresent(t, err.Error())

	report := readRunReportForTest(t, out)
	if report.Failure == nil || report.Failure.Kind != "internal_error" {
		t.Fatalf("failure report = %+v", report.Failure)
	}
	debug := readFailureDebugForTest(t, out)
	if debug.Kind != "internal_error" || !strings.Contains(debug.Message, "engine panic") || debug.StackTrace == "" {
		t.Fatalf("debug artifact = %+v", debug)
	}
}

func TestEvalCommandWritesStructuredFailureArtifactsOnConfigError(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	out := filepath.Join(dir, "eval-config-error-out")
	if err := os.WriteFile(filepath.Join(dir, configFileName), []byte("version: [\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := run([]string{"eval", dir, "--out", out})
	if err == nil || !strings.Contains(err.Error(), "config_error:") {
		t.Fatalf("eval should return config_error, got %v", err)
	}
	assertCorrelationIDPresent(t, err.Error())

	report := readRunReportForTest(t, out)
	if report.Failure == nil || report.Failure.Kind != "config_error" {
		t.Fatalf("failure report = %+v", report.Failure)
	}
	debug := readFailureDebugForTest(t, out)
	if debug.Kind != "config_error" || debug.StackTrace != "" {
		t.Fatalf("debug artifact = %+v", debug)
	}
}

func TestEvalCommandKeepsMutationReportWhenEvaluationWriteFails(t *testing.T) {
	dir := writeCLIFixture(t)
	out := filepath.Join(dir, "eval-write-fail-out")
	restoreCLIHooks(t)
	writeEvalFn = func(string, evalpkg.Evaluation) error {
		return errors.New("evaluation write failed")
	}

	err := run([]string{"eval", dir, "--max-mutants", "1", "--workers", "1", "--out", out})
	if err == nil || !strings.Contains(err.Error(), "internal_error:") {
		t.Fatalf("eval should return internal_error, got %v", err)
	}
	assertCorrelationIDPresent(t, err.Error())

	report := readRunReportForTest(t, out)
	if report.Failure != nil {
		t.Fatalf("existing mutation report should not be overwritten: %+v", report.Failure)
	}
	if report.Summary.Total == 0 {
		t.Fatalf("mutation report should still contain run results: %+v", report.Summary)
	}
	debug := readFailureDebugForTest(t, out)
	if debug.Kind != "internal_error" || !strings.Contains(debug.Message, "evaluation write failed") {
		t.Fatalf("debug artifact = %+v", debug)
	}
	if _, err := os.Stat(filepath.Join(out, "evaluation.json")); err == nil {
		t.Fatal("evaluation.json should not exist after write failure")
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
	return testharness.WriteGoModuleTempDir(t, "fixture", map[string]string{
		"calc.go": `package fixture

func IsPositiveOrZero(n int) bool {
	return n >= 0
}
`,
		"calc_test.go": `package fixture

import "testing"

func TestIsPositiveOrZero(t *testing.T) {
	if !IsPositiveOrZero(1) {
		t.Fatal("want positive")
	}
}
`,
	})
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

func restoreCLIHooks(t *testing.T) {
	t.Helper()
	oldRunEngine := runEngineFn
	oldWriteRunResult := writeRunResultFn
	oldBuildEval := buildEvalFn
	oldWriteEval := writeEvalFn
	t.Cleanup(func() {
		runEngineFn = oldRunEngine
		writeRunResultFn = oldWriteRunResult
		buildEvalFn = oldBuildEval
		writeEvalFn = oldWriteEval
	})
}

func readRunReportForTest(t *testing.T, out string) engine.RunResult {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(out, mutationReportFileName))
	if err != nil {
		t.Fatalf("mutation report missing: %v", err)
	}
	var result engine.RunResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("mutation report is not valid JSON: %v\n%s", err, data)
	}
	return result
}

func writeRunReportForTest(t *testing.T, out string, result engine.RunResult) {
	t.Helper()
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("marshal mutation report: %v", err)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatalf("create report dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(out, mutationReportFileName), data, 0o600); err != nil {
		t.Fatalf("write mutation report: %v", err)
	}
}

func readFailureDebugForTest(t *testing.T, out string) failureDebugArtifact {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(out, failureDebugFileName))
	if err != nil {
		t.Fatalf("failure debug artifact missing: %v", err)
	}
	var artifact failureDebugArtifact
	if err := json.Unmarshal(data, &artifact); err != nil {
		t.Fatalf("failure debug artifact is not valid JSON: %v\n%s", err, data)
	}
	return artifact
}

func assertCorrelationIDPresent(t *testing.T, message string) {
	t.Helper()
	if !strings.Contains(message, "correlation_id=") {
		t.Fatalf("error missing correlation id: %v", message)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()
	fn()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func actionableOnlyRunResult() engine.RunResult {
	return engine.RunResult{
		SchemaVersion: "1",
		Environment:   engine.Environment{OS: "windows", Arch: "amd64", GoVersion: "go1.25.6"},
		Summary:       engine.Summary{Total: 4, Survived: 4},
		Mutants: []engine.MutantResult{
			{
				MutantID:            "group-lead",
				Status:              engine.StatusSurvived,
				SurvivorRank:        1,
				Actionability:       "high",
				SuggestedTestScope:  "./fixture",
				SuggestedSkipReason: "review once",
				SemanticGroupSize:   2,
				Mutant: engine.Mutant{
					ID:            "group-lead",
					File:          "calc.go",
					Line:          4,
					Operator:      "conditionals-boundary",
					Original:      "<",
					Mutated:       "<=",
					SemanticGroup: "sort:1",
					GroupLabel:    "sort comparator boundary",
					GroupReason:   "shared review",
				},
			},
			{
				MutantID:            "group-dup",
				Status:              engine.StatusSurvived,
				SurvivorRank:        2,
				Actionability:       "medium",
				SuggestedTestScope:  "./fixture",
				SuggestedSkipReason: "review once",
				SemanticGroupSize:   2,
				Mutant: engine.Mutant{
					ID:            "group-dup",
					File:          "calc.go",
					Line:          5,
					Operator:      "conditionals-boundary",
					Original:      "<",
					Mutated:       "<=",
					SemanticGroup: "sort:1",
					GroupLabel:    "sort comparator boundary",
					GroupReason:   "shared review",
				},
			},
			{
				MutantID:           "platform",
				Status:             engine.StatusSurvived,
				SurvivorRank:       3,
				Actionability:      "high",
				SuggestedTestScope: "./fixture",
				Mutant: engine.Mutant{
					ID:                "platform",
					File:              "calc.go",
					Line:              6,
					Operator:          "numeric-literals",
					Original:          "0o755",
					Mutated:           "0",
					PlatformSensitive: true,
				},
			},
			{
				MutantID:           "keep",
				Status:             engine.StatusSurvived,
				SurvivorRank:       4,
				Actionability:      "medium",
				SuggestedTestScope: "./fixture",
				Mutant: engine.Mutant{
					ID:       "keep",
					File:     "calc.go",
					Line:     7,
					Operator: "logical",
					Original: "&&",
					Mutated:  "||",
				},
			},
		},
	}
}
