package baseline

import (
	"encoding/json"
	"os"

	"gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/engine"
)

func Load(path string) (engine.RunResult, bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return engine.RunResult{}, false, nil
	}
	if err != nil {
		return engine.RunResult{}, false, err
	}
	var result engine.RunResult
	if err := json.Unmarshal(data, &result); err != nil {
		return engine.RunResult{}, false, err
	}
	return result, true, nil
}

func Save(path string, result engine.RunResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func Compare(previous, current engine.RunResult) engine.BaselineComparison {
	seen := map[string]engine.Status{}
	for _, mutant := range previous.Mutants {
		seen[mutant.MutantID] = mutant.Status
	}
	comparison := engine.BaselineComparison{
		Enabled:       true,
		PreviousScore: previous.Summary.Score,
		CurrentScore:  current.Summary.Score,
		Regression:    current.Summary.Score < previous.Summary.Score,
	}
	for _, mutant := range current.Mutants {
		if mutant.Status == engine.StatusSurvived && seen[mutant.MutantID] != engine.StatusSurvived {
			comparison.NewSurvivors = append(comparison.NewSurvivors, mutant.MutantID)
		}
	}
	return comparison
}
