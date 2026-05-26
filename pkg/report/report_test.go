package report

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/engine"
)

func TestJSONReportSchemaV1IncludesActionableFields(t *testing.T) {
	run := engine.RunResult{
		SchemaVersion: "1",
		Summary: engine.Summary{
			Total:       1,
			Survived:    1,
			Score:       0,
			Quarantined: 0,
		},
		Mutants: []engine.MutantResult{{
			MutantID:     "pkg/foo.go:10:conditionals-negation:eq-to-ne",
			Status:       engine.StatusSurvived,
			Duration:     time.Second,
			TestCommand:  []string{"go", "test", "./pkg"},
			StatusReason: "tests passed with mutant applied",
			Output:       "ok",
			Mutant: engine.Mutant{
				ID:             "pkg/foo.go:10:conditionals-negation:eq-to-ne",
				Package:        "pkg",
				File:           "pkg/foo.go",
				Line:           10,
				Function:       "Check",
				Operator:       "conditionals-negation",
				Original:       "==",
				Mutated:        "!=",
				Diff:           "--- pkg/foo.go\n+++ pkg/foo.go\n",
				Hint:           "Add an assertion for the opposite branch.",
				Description:    "Changed == to != in Check.",
				NearbyTests:    []string{"pkg/foo_test.go"},
				EquivalentRisk: "medium",
				Recommendation: "fast-ci",
				SuppressionAudit: []engine.SuppressionAudit{{
					Name:   "audit-high-equivalent-risk",
					Action: "report-only",
					Reason: "visible audit",
				}},
			},
			SurvivorRank: 1,
			RankReason:   "risk=medium recommendation=fast-ci nearby_tests=1",
		}},
	}

	data, err := JSON(run)
	if err != nil {
		t.Fatalf("JSON returned error: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("report is not JSON: %v", err)
	}
	if decoded["schema_version"] != "1" {
		t.Fatalf("schema_version = %v", decoded["schema_version"])
	}
	text := string(data)
	for _, want := range []string{"baseline", "cache", "quarantine", "unified_diff", "status_reason", "selected_tests", "description", "nearby_tests", "equivalent_risk", "recommendation", "suppression_audit", "survivor_rank", "rank_reason"} {
		if !strings.Contains(text, want) {
			t.Fatalf("JSON report missing %q: %s", want, text)
		}
	}
}

func TestSummaryIncludesGremlinsStyleCoverageMetricsAndMutatorStats(t *testing.T) {
	run := engine.RunResult{
		Summary: engine.Summary{
			Total:            3,
			Killed:           1,
			Survived:         1,
			NotCovered:       1,
			Score:            50,
			TestEfficacy:     50,
			MutationCoverage: 66.66666666666666,
			MutatorStats: map[string]engine.MutatorStat{
				"conditionals-negation": {Total: 2, Killed: 1, Survived: 1, Recommendation: "fast-ci"},
				"logical":               {Total: 1, NotCovered: 1, Recommendation: "conservative"},
			},
		},
	}

	text := Summary(run)
	for _, want := range []string{
		"Not covered: 1",
		"Test efficacy: 50.00%",
		"Mutation coverage: 66.67%",
		"conditionals-negation: total=2 killed=1 survived=1 not_covered=0",
		"recommendation=fast-ci",
		"logical: total=1 killed=0 survived=0 not_covered=1",
		"recommendation=conservative",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("summary missing %q:\n%s", want, text)
		}
	}
}

func TestSurvivorsReportIsRanked(t *testing.T) {
	run := engine.RunResult{
		Mutants: []engine.MutantResult{
			{MutantID: "later", Status: engine.StatusSurvived, SurvivorRank: 2, RankReason: "risk=high", Mutant: engine.Mutant{File: "a.go", Line: 2, Operator: "returns", Original: "x", Mutated: "y"}},
			{MutantID: "first", Status: engine.StatusSurvived, SurvivorRank: 1, RankReason: "risk=low", Mutant: engine.Mutant{File: "b.go", Line: 1, Operator: "boolean", Original: "true", Mutated: "false"}},
		},
	}

	text := Survivors(run)
	if !strings.Contains(text, "#1 first") || strings.Index(text, "#1 first") > strings.Index(text, "#2 later") {
		t.Fatalf("survivors were not ranked:\n%s", text)
	}
}
