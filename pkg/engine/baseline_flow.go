package engine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/cervantesh/cervo-mutants/pkg/internal/gotestenv"
)

func (s *runSession) runBaseline(ctx context.Context, targets []string) (MutantResult, error) {
	moduleDir, err := moduleForTargets(targets)
	if err != nil {
		return MutantResult{}, err
	}
	s.coverageBaseDir = moduleDir
	command := append([]string{}, s.engine.cfg.Tests.Command...)
	if s.engine.cfg.Selection.Mode == "coverage" || s.engine.cfg.Selection.Prefilter {
		profile := s.engine.cfg.Selection.CoverageProfile
		if !filepath.IsAbs(profile) {
			profile = filepath.Join(moduleDir, profile)
		}
		_ = os.MkdirAll(filepath.Dir(profile), 0o755)
		command = gotestenv.WithCoverProfile(command, profile)
	}
	job := MutantJob{ID: "baseline", WorkDir: moduleDir, TestCommand: command, Timeout: s.engine.cfg.Tests.Timeout.String()}
	result, err := s.runTest(ctx, job)
	if err != nil {
		return MutantResult{}, err
	}
	if result.Status != StatusSurvived {
		return result, &BaselineFailureError{Result: result}
	}
	return result, nil
}

func moduleForTargets(targets []string) (string, error) {
	if len(targets) == 0 {
		return os.Getwd()
	}
	root := strings.TrimSuffix(targets[0], "/...")
	if root == "./..." {
		root = "."
	}
	return filepath.Abs(root)
}

func (s *runSession) loadBaseline() (RunResult, bool, error) {
	data, err := os.ReadFile(s.engine.cfg.Baseline.Path)
	if os.IsNotExist(err) {
		return RunResult{}, false, nil
	}
	if err != nil {
		return RunResult{}, false, err
	}
	var result RunResult
	if err := json.Unmarshal(data, &result); err != nil {
		return RunResult{}, false, err
	}
	return result, true, nil
}

func compareBaseline(previous, current RunResult) BaselineComparison {
	seen := map[string]Status{}
	for _, mutant := range previous.Mutants {
		seen[mutant.MutantID] = mutant.Status
	}
	comparison := BaselineComparison{Enabled: true, Available: true, PreviousScore: previous.Summary.Score, CurrentScore: current.Summary.Score, Regression: current.Summary.Score < previous.Summary.Score}
	for _, mutant := range current.Mutants {
		if mutant.Status == StatusSurvived && seen[mutant.MutantID] != StatusSurvived {
			comparison.NewSurvivors = append(comparison.NewSurvivors, mutant.MutantID)
		}
	}
	return comparison
}
