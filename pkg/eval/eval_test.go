package eval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cervantesh/CervoMutants/pkg/engine"
)

func TestBuildCreatesDecisionCompleteEvaluation(t *testing.T) {
	run := engine.RunResult{
		SchemaVersion: "1",
		Summary: engine.Summary{
			Total:            10,
			Killed:           6,
			Survived:         2,
			NotCovered:       1,
			Cached:           1,
			Quarantined:      1,
			Score:            75,
			TestEfficacy:     75,
			MutationCoverage: 90,
		},
		Cache: engine.CacheStats{Hits: 1, Misses: 9},
		Mutants: []engine.MutantResult{
			{MutantID: "m1", Status: engine.StatusSurvived},
			{MutantID: "m2", Status: engine.StatusKilled},
		},
	}

	evaluation := Build(BuildRequest{
		Tool:       "cervo-mutants",
		Target:     "fixture",
		Commit:     "abc123",
		Command:    []string{"cervomut", "eval", "./..."},
		Framework:  "generic-go",
		Run:        run,
		ManualMode: true,
	})

	if evaluation.SchemaVersion != "1" {
		t.Fatalf("schema version = %q", evaluation.SchemaVersion)
	}
	if evaluation.Decision != DecisionNeedsReview {
		t.Fatalf("decision = %q, want %q", evaluation.Decision, DecisionNeedsReview)
	}
	if evaluation.Scorecard.Total <= 0 {
		t.Fatalf("expected objective score to be populated: %+v", evaluation.Scorecard)
	}
	if evaluation.Scorecard.FaultRevealing.Evidence != EvidenceRequiresReview {
		t.Fatalf("manual evidence not marked for review: %+v", evaluation.Scorecard.FaultRevealing)
	}
	if evaluation.Metrics.NotCovered != 1 {
		t.Fatalf("not covered = %d, want 1", evaluation.Metrics.NotCovered)
	}
	if evaluation.Metrics.TestEfficacy != 75 {
		t.Fatalf("test efficacy = %.2f, want 75", evaluation.Metrics.TestEfficacy)
	}
	if evaluation.Metrics.MutationCoverage != 90 {
		t.Fatalf("mutation coverage = %.2f, want 90", evaluation.Metrics.MutationCoverage)
	}

	data, err := json.Marshal(evaluation)
	if err != nil {
		t.Fatalf("evaluation is not JSON serializable: %v", err)
	}
	if !strings.Contains(string(data), `"scorecard"`) || !strings.Contains(string(data), `"required_manual_review"`) {
		t.Fatalf("evaluation JSON missing required fields: %s", data)
	}
}

func TestWriteOutputsJSONMarkdownAndSchema(t *testing.T) {
	dir := t.TempDir()
	evaluation := Build(BuildRequest{
		Tool:      "cervo-mutants",
		Target:    "fixture",
		Commit:    "abc123",
		Command:   []string{"cervomut", "eval", "./..."},
		Framework: "generic-go",
		Run:       engine.RunResult{SchemaVersion: "1", Summary: engine.Summary{Total: 1, Killed: 1, Score: 100}},
	})

	if err := Write(dir, evaluation); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	for _, name := range []string{"evaluation.json", "evaluation.md", "evaluation.schema.json"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("%s was not written: %v", name, err)
		}
	}
	md, err := os.ReadFile(filepath.Join(dir, "evaluation.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(md), "## Required Manual Review") {
		t.Fatalf("evaluation markdown missing manual review section: %s", md)
	}
}

func TestScoreAndDecisionBranches(t *testing.T) {
	high := Metrics{TotalMutants: 10, Survived: 1, Cached: 2}
	scorecard := score(high, false)
	scorecard.Total = 85
	scorecard.FaultRevealing.Score = 18
	scorecard.Actionability.Score = 12
	if decision := decide(scorecard, false); decision != DecisionCandidateDefault {
		t.Fatalf("candidate decision = %q", decision)
	}

	scorecard.Noise.Score = 4
	scorecard.Total = 40
	if decision := decide(scorecard, false); decision != DecisionNeedsTuning {
		t.Fatalf("tuning decision = %q", decision)
	}

	if ciScore(Metrics{TotalMutants: 1}) != 15 {
		t.Fatal("ciScore should cap at 15 for clean non-empty run")
	}
	if actionabilityScore(Metrics{TotalMutants: 1, Survived: 1}) != 15 {
		t.Fatal("actionabilityScore should cap at 15")
	}
	if costScore(Metrics{Cached: 1}) != 8 {
		t.Fatal("costScore should cap at 8")
	}
	if defaultString("", "fallback") != "fallback" || defaultString("value", "fallback") != "value" {
		t.Fatal("defaultString returned unexpected values")
	}
}
