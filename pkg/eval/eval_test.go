package eval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/engine"
)

func TestBuildCreatesDecisionCompleteEvaluation(t *testing.T) {
	run := engine.RunResult{
		SchemaVersion: "1",
		Summary: engine.Summary{
			Total:       10,
			Killed:      6,
			Survived:    2,
			Cached:      1,
			Quarantined: 1,
			Score:       75,
		},
		Cache: engine.CacheStats{Hits: 1, Misses: 9},
		Mutants: []engine.MutantResult{
			{MutantID: "m1", Status: engine.StatusSurvived},
			{MutantID: "m2", Status: engine.StatusKilled},
		},
	}

	evaluation := Build(BuildRequest{
		Tool:       "cervo-mutant",
		Target:     "fixture",
		Commit:     "abc123",
		Command:    []string{"cervomut", "eval", "./..."},
		Framework:  "cervosoft",
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
		Tool:      "cervo-mutant",
		Target:    "fixture",
		Commit:    "abc123",
		Command:   []string{"cervomut", "eval", "./..."},
		Framework: "cervosoft",
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
