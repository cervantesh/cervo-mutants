package report

import (
	"fmt"
	"strings"

	"github.com/cervantesh/cervo-mutants/pkg/engine"
)

func gateStatusLabel(evaluation engine.GateEvaluation) string {
	if !evaluation.Evaluated {
		return "not evaluated"
	}
	if evaluation.Passed {
		return "passed"
	}
	return "failed"
}

func gateChecksByStatus(evaluation engine.GateEvaluation, status engine.GateCheckStatus) []engine.GateCheck {
	checks := make([]engine.GateCheck, 0)
	for _, check := range evaluation.Checks {
		if check.Status == status {
			checks = append(checks, check)
		}
	}
	return checks
}

func gateCheckSummaries(checks []engine.GateCheck) string {
	parts := make([]string, 0, len(checks))
	for _, check := range checks {
		part := check.Name
		if strings.TrimSpace(check.Summary) != "" {
			part = fmt.Sprintf("%s (%s)", check.Name, check.Summary)
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "; ")
}
