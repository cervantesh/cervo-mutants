package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cervantesh/CervoMutants/pkg/engine"
)

func TestJSONReportSchemaV1IncludesActionableFields(t *testing.T) {
	run := engine.RunResult{
		SchemaVersion: "1",
		Environment:   engine.Environment{OS: "linux", Arch: "amd64", GoVersion: "go1.25.6", Isolation: "overlay", Workers: 1, TestTimeout: "30s", WSL: true},
		Checkpoint:    engine.Checkpoint{Fingerprint: "abc123", Mutants: 1, IncludesFileDigests: true, Reason: "final"},
		Summary: engine.Summary{
			Total:       1,
			Survived:    1,
			Score:       0,
			Quarantined: 0,
		},
		Mutants: []engine.MutantResult{{
			MutantID:        "pkg/foo.go:10:conditionals-negation:eq-to-ne",
			Status:          engine.StatusSurvived,
			FailureKind:     "runner_error",
			Duration:        time.Second,
			TestCommand:     []string{"go", "test", "./pkg"},
			StatusReason:    "tests passed with mutant applied",
			SelectionReason: "coverage profile matched mutant file",
			CoverageSource:  "coverage-mode",
			Output:          "ok",
			Mutant: engine.Mutant{
				ID:               "pkg/foo.go:10:conditionals-negation:eq-to-ne",
				Package:          "pkg",
				File:             "pkg/foo.go",
				Line:             10,
				Function:         "Check",
				Operator:         "conditionals-negation",
				Original:         "==",
				Mutated:          "!=",
				Diff:             "--- pkg/foo.go\n+++ pkg/foo.go\n",
				Hint:             "Add an assertion for the opposite branch.",
				Description:      "Changed == to != in Check.",
				NearbyTests:      []string{"pkg/foo_test.go"},
				EquivalentRisk:   "medium",
				Recommendation:   "fast-ci",
				CompileErrorRisk: "low",
				SuppressionAudit: []engine.SuppressionAudit{{
					Name:          "audit-high-equivalent-risk",
					Action:        "report-only",
					Reason:        "visible audit",
					EvidenceLevel: "suspected",
				}},
			},
			SurvivorRank:       1,
			RankScore:          140,
			RankReason:         "risk=medium recommendation=fast-ci nearby_tests=1",
			Actionability:      "high",
			SuggestedTestScope: "./pkg",
			NearestTests:       []string{"pkg/foo_test.go"},
			PreviousStatus:     engine.StatusKilled,
			FirstSeen:          "2026-05-26T00:00:00Z",
			LastSeen:           "2026-05-26T01:00:00Z",
			SurvivorAgeRuns:    2,
			HistoryStatus:      "long_standing_survivor",
			OperatorYield:      0.5,
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
	for _, want := range []string{"environment", "go_version", "isolation", "checkpoint", "fingerprint", "includes_file_digests", "failure_kind", "baseline", "cache", "quarantine", "history", "unified_diff", "status_reason", "selection_reason", "coverage_source", "selected_tests", "description", "nearby_tests", "equivalent_risk", "recommendation", "compile_error_risk", "suppression_audit", "evidence_level", "survivor_rank", "rank_score", "rank_reason", "actionability", "suggested_test_scope", "nearest_tests", "previous_status", "first_seen", "survivor_age_runs", "operator_historical_yield"} {
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
			EffectiveMutants: 2,
			ScoreDenominator: 2,
			TestEfficacy:     50,
			MutationCoverage: 66.66666666666666,
			DenominatorHealth: engine.DenominatorHealth{
				Generated:        3,
				Covered:          2,
				Executed:         2,
				Effective:        2,
				ScoreDenominator: 2,
				Killed:           1,
				Survived:         1,
				NotCovered:       1,
				Healthy:          true,
			},
			HighRiskSurvivors:     1,
			NewSurvivors:          1,
			LongStandingSurvivors: 1,
			SuppressionReportOnly: 2,
			EquivalentRiskStats:   map[string]int{"high": 1, "medium": 2},
			MutatorStats: map[string]engine.MutatorStat{
				"conditionals-negation": {Total: 2, Killed: 1, Survived: 1, Recommendation: "fast-ci"},
				"logical":               {Total: 1, NotCovered: 1, Recommendation: "conservative"},
			},
		},
		Environment: engine.Environment{OS: "linux", Arch: "amd64", GoVersion: "go1.25.6", Isolation: "overlay", Workers: 1, TestTimeout: "30s"},
	}

	text := Summary(run)
	for _, want := range []string{
		"Not covered: 1",
		"Effective mutants: 2",
		"Score denominator: 2",
		"Test efficacy: 50.00%",
		"Mutation coverage: 66.67%",
		"Denominator health: healthy=true generated=3 covered=2 executed=2 effective=2 score_denominator=2 killed=1 survived=1 not_covered=1 timed_out=0 compile_error=0",
		"High-risk survivors: 1",
		"New survivors: 1",
		"Long-standing survivors: 1",
		"Suppression audits: report_only=2",
		"Equivalent-risk statistics:",
		"conditionals-negation: total=2 killed=1 survived=1 not_covered=0",
		"recommendation=fast-ci",
		"logical: total=1 killed=0 survived=0 not_covered=1",
		"recommendation=conservative",
		"Environment: os=linux arch=amd64 go=go1.25.6 isolation=overlay workers=1 timeout=30s",
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
	if !strings.Contains(text, "#1 0.0 first") || strings.Index(text, "#1 0.0 first") > strings.Index(text, "#2 0.0 later") {
		t.Fatalf("survivors were not ranked:\n%s", text)
	}
}

func TestWriteFormatsHonorsConfiguredFormats(t *testing.T) {
	dir := t.TempDir()
	run := engine.RunResult{SchemaVersion: "1", Summary: engine.Summary{Total: 1}}

	if err := WriteFormats(dir, run, []string{"summary", "json"}); err != nil {
		t.Fatalf("WriteFormats returned error: %v", err)
	}
	for _, want := range []string{"summary.txt", "survivors.txt", "mutation-report.json"} {
		if _, err := os.Stat(filepath.Join(dir, want)); err != nil {
			t.Fatalf("missing %s: %v", want, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "index.html")); !os.IsNotExist(err) {
		t.Fatalf("index.html should not be written for summary/json formats: %v", err)
	}
}

func TestWriteFormatsDefaultsAndErrors(t *testing.T) {
	dir := t.TempDir()
	run := engine.RunResult{Summary: engine.Summary{Total: 1}}
	if err := WriteFormats(dir, run, nil); err != nil {
		t.Fatalf("WriteFormats default formats returned error: %v", err)
	}
	for _, want := range []string{"summary.txt", "survivors.txt", "mutation-report.json"} {
		if _, err := os.Stat(filepath.Join(dir, want)); err != nil {
			t.Fatalf("default formats missing %s: %v", want, err)
		}
	}
	filePath := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(filePath, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteFormats(filePath, run, []string{"summary"}); err == nil {
		t.Fatal("WriteFormats accepted a file as output directory")
	}
}

func TestJUnitHTMLAndWriteAll(t *testing.T) {
	dir := t.TempDir()
	run := engine.RunResult{
		SchemaVersion: "1",
		Summary:       engine.Summary{Total: 2, Killed: 1, Survived: 1, Score: 50},
		Mutants: []engine.MutantResult{
			{MutantID: "killed", Status: engine.StatusKilled, Mutant: engine.Mutant{Diff: "-a\n+b\n"}},
			{MutantID: "survived", Status: engine.StatusSurvived, StatusReason: "tests passed", Mutant: engine.Mutant{Diff: "<unsafe>"}},
		},
	}

	junit, err := JUnit(run)
	if err != nil {
		t.Fatalf("JUnit returned error: %v", err)
	}
	if !strings.Contains(string(junit), `tests="2"`) || !strings.Contains(string(junit), `failures="1"`) {
		t.Fatalf("unexpected junit: %s", junit)
	}
	html := HTML(run)
	if !strings.Contains(html, "cervomut mutation report") || strings.Contains(html, "<unsafe>") {
		t.Fatalf("html should include report title and escape diffs: %s", html)
	}
	if err := WriteAll(dir, run); err != nil {
		t.Fatalf("WriteAll returned error: %v", err)
	}
	for _, want := range []string{"summary.txt", "survivors.txt", "mutation-report.json", "junit.xml", "index.html"} {
		if _, err := os.Stat(filepath.Join(dir, want)); err != nil {
			t.Fatalf("missing %s: %v", want, err)
		}
	}
}
