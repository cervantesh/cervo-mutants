package engine

import "github.com/cervantesh/cervo-mutants/pkg/discover"

func (e *Engine) runTargets(targets []string) []string {
	if len(targets) == 0 {
		return e.cfg.Scope.Include
	}
	return targets
}

func (e *Engine) discoverMutants(targets []string) ([]Mutant, error) {
	discovered, err := discover.Discover(targets)
	if err != nil {
		return nil, err
	}
	return e.generateMutants(discovered)
}

func (e *Engine) dryRunResult(result RunResult, mutants []Mutant) RunResult {
	for _, mutant := range mutants {
		result.Mutants = append(result.Mutants, MutantResult{MutantID: mutant.ID, Status: StatusSkipped, StatusReason: "dry-run", Mutant: mutant})
	}
	result.StoppedReason, result.LastCompletedMutant = runStopMetadata(result.Mutants)
	e.applySurvivorRanking(result.Mutants)
	result.Summary = summarize(result.Mutants)
	result.Gate = GateEvaluation{Evaluated: false}
	return result
}
