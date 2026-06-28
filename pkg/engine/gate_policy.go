package engine

import (
	"fmt"

	"github.com/cervantesh/cervo-mutants/pkg/config"
)

const (
	gateCheckFailUnder            = "fail_under"
	gateCheckBaselineRegression   = "baseline_regression"
	gateCheckBaselineNewSurvivors = "baseline_new_survivors"
	gateCheckTimedOut             = "timed_out"
	gateCheckCompileError         = "compile_error"
)

func configuredThresholds(cfg config.Config) map[string]any {
	return map[string]any{
		"fail_under":            cfg.CI.FailUnder,
		"fail_on_timeout":       cfg.CI.FailOnTimeout,
		"fail_on_compile_error": cfg.CI.FailOnCompileError,
		"baseline": map[string]any{
			"enabled":               cfg.Baseline.Enabled,
			"fail_on_regression":    cfg.Baseline.FailOnRegression,
			"fail_on_new_survivors": cfg.Baseline.FailOnNewSurvivors,
		},
	}
}

func EvaluateGatePolicy(cfg config.Config, result RunResult) GateEvaluation {
	checks := []GateCheck{
		evaluateFailUnder(cfg, result),
		evaluateBaselineRegression(cfg, result),
		evaluateBaselineNewSurvivors(cfg, result),
		evaluateTimedOut(cfg, result),
		evaluateCompileError(cfg, result),
	}
	evaluation := GateEvaluation{
		Evaluated: true,
		Passed:    true,
		Checks:    checks,
	}
	for _, check := range checks {
		if check.Status == GateCheckFailed {
			evaluation.Passed = false
			evaluation.FailedChecks = append(evaluation.FailedChecks, check.Name)
		}
	}
	return evaluation
}

func evaluateFailUnder(cfg config.Config, result RunResult) GateCheck {
	threshold := cfg.CI.FailUnder
	if threshold <= 0 {
		return GateCheck{Name: gateCheckFailUnder, Status: GateCheckDisabled, Summary: "raw score threshold disabled"}
	}
	if result.Summary.Score < float64(threshold) {
		return GateCheck{
			Name:    gateCheckFailUnder,
			Status:  GateCheckFailed,
			Summary: fmt.Sprintf("raw score %.2f%% below threshold %d", result.Summary.Score, threshold),
		}
	}
	return GateCheck{
		Name:    gateCheckFailUnder,
		Status:  GateCheckPassed,
		Summary: fmt.Sprintf("raw score %.2f%% meets threshold %d", result.Summary.Score, threshold),
	}
}

func evaluateBaselineRegression(cfg config.Config, result RunResult) GateCheck {
	if !cfg.Baseline.Enabled || !cfg.Baseline.FailOnRegression {
		return GateCheck{Name: gateCheckBaselineRegression, Status: GateCheckDisabled, Summary: "baseline regression gate disabled"}
	}
	if !result.Baseline.Available {
		return GateCheck{Name: gateCheckBaselineRegression, Status: GateCheckSkipped, Summary: "no baseline file was available for comparison"}
	}
	if result.Baseline.Regression {
		return GateCheck{
			Name:    gateCheckBaselineRegression,
			Status:  GateCheckFailed,
			Summary: fmt.Sprintf("current score %.2f%% regressed from baseline %.2f%%", result.Baseline.CurrentScore, result.Baseline.PreviousScore),
		}
	}
	return GateCheck{
		Name:    gateCheckBaselineRegression,
		Status:  GateCheckPassed,
		Summary: fmt.Sprintf("current score %.2f%% did not regress from baseline %.2f%%", result.Baseline.CurrentScore, result.Baseline.PreviousScore),
	}
}

func evaluateBaselineNewSurvivors(cfg config.Config, result RunResult) GateCheck {
	if !cfg.Baseline.Enabled || !cfg.Baseline.FailOnNewSurvivors {
		return GateCheck{Name: gateCheckBaselineNewSurvivors, Status: GateCheckDisabled, Summary: "baseline new-survivor gate disabled"}
	}
	if !result.Baseline.Available {
		return GateCheck{Name: gateCheckBaselineNewSurvivors, Status: GateCheckSkipped, Summary: "no baseline file was available for comparison"}
	}
	if len(result.Baseline.NewSurvivors) > 0 {
		return GateCheck{
			Name:    gateCheckBaselineNewSurvivors,
			Status:  GateCheckFailed,
			Summary: fmt.Sprintf("%d new survivors appeared relative to baseline", len(result.Baseline.NewSurvivors)),
		}
	}
	return GateCheck{
		Name:    gateCheckBaselineNewSurvivors,
		Status:  GateCheckPassed,
		Summary: "no new survivors appeared relative to baseline",
	}
}

func evaluateTimedOut(cfg config.Config, result RunResult) GateCheck {
	if !cfg.CI.FailOnTimeout {
		return GateCheck{Name: gateCheckTimedOut, Status: GateCheckDisabled, Summary: "timeout gate disabled"}
	}
	if result.Summary.TimedOut > 0 {
		return GateCheck{
			Name:    gateCheckTimedOut,
			Status:  GateCheckFailed,
			Summary: fmt.Sprintf("%d timed out mutants were observed", result.Summary.TimedOut),
		}
	}
	return GateCheck{Name: gateCheckTimedOut, Status: GateCheckPassed, Summary: "no timed out mutants were observed"}
}

func evaluateCompileError(cfg config.Config, result RunResult) GateCheck {
	if !cfg.CI.FailOnCompileError {
		return GateCheck{Name: gateCheckCompileError, Status: GateCheckDisabled, Summary: "compile-error gate disabled"}
	}
	if result.Summary.CompileError > 0 {
		return GateCheck{
			Name:    gateCheckCompileError,
			Status:  GateCheckFailed,
			Summary: fmt.Sprintf("%d compile-error mutants were observed", result.Summary.CompileError),
		}
	}
	return GateCheck{Name: gateCheckCompileError, Status: GateCheckPassed, Summary: "no compile-error mutants were observed"}
}
