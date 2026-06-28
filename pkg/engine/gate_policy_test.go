package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cervantesh/cervo-mutants/pkg/config"
)

func TestEvaluateGatePolicyAccumulatesConfiguredFailuresInStableOrder(t *testing.T) {
	cfg := config.Defaults()
	cfg.CI.FailUnder = 80
	cfg.CI.FailOnTimeout = true
	cfg.CI.FailOnCompileError = true
	cfg.Baseline.Enabled = true
	cfg.Baseline.FailOnRegression = true
	cfg.Baseline.FailOnNewSurvivors = true

	result := RunResult{
		Summary: Summary{
			Score:        70,
			TimedOut:     1,
			CompileError: 2,
		},
		Baseline: BaselineComparison{
			Enabled:       true,
			Available:     true,
			Regression:    true,
			NewSurvivors:  []string{"m1", "m2"},
			PreviousScore: 85,
			CurrentScore:  70,
		},
	}

	evaluation := EvaluateGatePolicy(cfg, result)
	if !evaluation.Evaluated || evaluation.Passed {
		t.Fatalf("evaluation = %+v", evaluation)
	}
	if got, want := strings.Join(evaluation.FailedChecks, ","), "fail_under,baseline_regression,baseline_new_survivors,timed_out,compile_error"; got != want {
		t.Fatalf("failed checks = %q, want %q", got, want)
	}
}

func TestEvaluateGatePolicySkipsBaselineChecksWhenNoBaselineExists(t *testing.T) {
	cfg := config.Defaults()
	result := RunResult{
		Summary:  Summary{Score: 100},
		Baseline: BaselineComparison{Enabled: true, CurrentScore: 100},
	}

	evaluation := EvaluateGatePolicy(cfg, result)
	if !evaluation.Evaluated || !evaluation.Passed {
		t.Fatalf("evaluation = %+v", evaluation)
	}
	skipped := gateChecksByStatusForTest(evaluation, GateCheckSkipped)
	if got, want := strings.Join(skipped, ","), "baseline_regression,baseline_new_survivors"; got != want {
		t.Fatalf("skipped checks = %q, want %q", got, want)
	}
}

func TestEvaluateGatePolicyUsesNormalizedSummaryCounts(t *testing.T) {
	cfg := config.Defaults()
	cfg.CI.FailOnCompileError = true
	cfg.CI.FailOnTimeout = true

	results := []MutantResult{
		{Status: StatusCached, PreviousStatus: StatusTimedOut},
		{Status: StatusCached, PreviousStatus: StatusCompileError},
	}
	run := RunResult{Summary: summarize(results)}
	evaluation := EvaluateGatePolicy(cfg, run)

	if got, want := strings.Join(evaluation.FailedChecks, ","), "timed_out,compile_error"; got != want {
		t.Fatalf("failed checks = %q, want %q", got, want)
	}
}

func TestRunReturnsEnvironmentErrorWhenBaselineFileIsInvalid(t *testing.T) {
	cfg := config.Defaults()
	dir := t.TempDir()
	isolateArtifacts(&cfg, dir)
	cfg.Baseline.Path = filepath.Join(dir, "baseline.json")
	if err := os.WriteFile(cfg.Baseline.Path, []byte("{not-json"), 0o600); err != nil {
		t.Fatal(err)
	}

	e := New(cfg)
	e.deps.discoverMutants = func(_ *Engine, _ []string) ([]Mutant, error) {
		return []Mutant{{ID: "m1"}}, nil
	}
	e.deps.runBaseline = func(_ *runSession, _ context.Context, _ []string) (MutantResult, error) {
		return MutantResult{}, nil
	}
	e.deps.runMutants = func(_ *runSession, _ context.Context, _ []Mutant, _ map[string]bool) ([]MutantResult, error) {
		return []MutantResult{{MutantID: "m1", Status: StatusKilled, Mutant: Mutant{ID: "m1"}}}, nil
	}
	e.deps.writeReports = func(_ *Engine, _ RunResult) error {
		t.Fatal("writeReports should not run when baseline is invalid")
		return nil
	}

	_, err := e.Run(context.Background(), RunRequest{Targets: []string{"./..."}})
	if err == nil || !strings.Contains(err.Error(), "environment_error:") {
		t.Fatalf("Run should fail with environment_error, got %v", err)
	}
}

func gateChecksByStatusForTest(evaluation GateEvaluation, status GateCheckStatus) []string {
	names := []string{}
	for _, check := range evaluation.Checks {
		if check.Status == status {
			names = append(names, check.Name)
		}
	}
	return names
}
