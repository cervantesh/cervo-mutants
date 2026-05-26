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
			MutantID:     "pkg/foo.go:10:conditionals:eq-to-ne",
			Status:       engine.StatusSurvived,
			Duration:     time.Second,
			TestCommand:  []string{"go", "test", "./pkg"},
			StatusReason: "tests passed with mutant applied",
			Output:       "ok",
			Mutant: engine.Mutant{
				ID:       "pkg/foo.go:10:conditionals:eq-to-ne",
				Package:  "pkg",
				File:     "pkg/foo.go",
				Line:     10,
				Function: "Check",
				Operator: "conditionals",
				Original: "==",
				Mutated:  "!=",
				Diff:     "--- pkg/foo.go\n+++ pkg/foo.go\n",
				Hint:     "Add an assertion for the opposite branch.",
			},
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
	for _, want := range []string{"baseline", "cache", "quarantine", "unified_diff", "status_reason", "selected_tests"} {
		if !strings.Contains(text, want) {
			t.Fatalf("JSON report missing %q: %s", want, text)
		}
	}
}
