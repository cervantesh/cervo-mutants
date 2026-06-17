package selecttest

import (
	"context"

	"github.com/cervantesh/CervoMutants/pkg/engine"
)

type Selector struct {
	Mode    string
	Command []string
}

func (s Selector) Select(ctx context.Context, mutant engine.Mutant) (engine.TestPlan, error) {
	command := append([]string{}, s.Command...)
	if len(command) == 0 {
		command = []string{"go", "test", "./..."}
	}
	switch s.Mode {
	case "all":
		return engine.TestPlan{Command: command, Reason: "all tests selected", CoversMutant: true, CoverageSource: "unknown"}, nil
	case "coverage":
		return engine.TestPlan{Command: command, Reason: "coverage timing data unavailable; package fallback selected", CoversMutant: true, CoverageSource: "coverage-mode"}, nil
	default:
		if len(command) >= 3 && command[0] == "go" && command[1] == "test" && mutant.Package != "" {
			command[2] = mutant.Package
		}
		return engine.TestPlan{Command: command, Reason: "package selected from mutant file", CoversMutant: true, CoverageSource: "unknown"}, nil
	}
}
