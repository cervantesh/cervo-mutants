package baseline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cervantesh/cervo-mutants/pkg/engine"
)

func TestLoadMissingBaseline(t *testing.T) {
	result, ok, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if ok {
		t.Fatalf("Load ok = true for missing file: %+v", result)
	}
}

func TestLoadRejectsMalformedBaseline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "baseline.json")
	if err := os.WriteFile(path, []byte("{bad json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Load(path); err == nil {
		t.Fatal("Load accepted malformed baseline JSON")
	}
}

func TestSaveLoadAndCompareBaseline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "baseline.json")
	previous := engine.RunResult{
		Summary: engine.Summary{
			Score:      80,
			Actionable: engine.ActionableSummary{ActionableScore: 85},
		},
		Mutants: []engine.MutantResult{
			{MutantID: "old-survivor", Status: engine.StatusSurvived},
			{MutantID: "killed", Status: engine.StatusKilled},
		},
	}
	if err := Save(path, previous); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	loaded, ok, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if !ok || loaded.Summary.Score != previous.Summary.Score || len(loaded.Mutants) != 2 {
		t.Fatalf("loaded baseline mismatch: ok=%t result=%+v", ok, loaded)
	}

	current := engine.RunResult{
		Summary: engine.Summary{
			Score:      70,
			Actionable: engine.ActionableSummary{ActionableScore: 75},
		},
		Mutants: []engine.MutantResult{
			{MutantID: "old-survivor", Status: engine.StatusSurvived},
			{MutantID: "killed", Status: engine.StatusSurvived},
			{MutantID: "new-survivor", Status: engine.StatusSurvived},
		},
	}
	comparison := Compare(previous, current)
	if !comparison.Enabled || !comparison.Regression {
		t.Fatalf("comparison flags missing: %+v", comparison)
	}
	if len(comparison.NewSurvivors) != 2 || comparison.NewSurvivors[0] != "killed" || comparison.NewSurvivors[1] != "new-survivor" {
		t.Fatalf("new survivors mismatch: %+v", comparison)
	}

	diff := BuildDiff(previous, current)
	if !diff.BaselineFound || diff.ScoreDelta != -10 || diff.ActionableScoreDelta != -10 {
		t.Fatalf("unexpected diff score summary: %+v", diff)
	}
	if len(diff.NewSurvivors) != 2 || diff.NewSurvivors[0] != "killed" || diff.NewSurvivors[1] != "new-survivor" {
		t.Fatalf("new survivors diff mismatch: %+v", diff.NewSurvivors)
	}
	if len(diff.ResolvedSurvivors) != 0 {
		t.Fatalf("resolved survivors mismatch: %+v", diff.ResolvedSurvivors)
	}
	if len(diff.StatusChanges) != 2 {
		t.Fatalf("status changes mismatch: %+v", diff.StatusChanges)
	}
	formatted := FormatDiff(diff)
	for _, want := range []string{
		"Raw score: 80.00% -> 70.00% (-10.00)",
		"Actionable score: 85.00% -> 75.00% (-10.00)",
		"New survivors: 2",
		"Status changes: 2",
		"- killed: killed -> survived",
	} {
		if !strings.Contains(formatted, want) {
			t.Fatalf("formatted diff missing %q:\n%s", want, formatted)
		}
	}
}

func TestCandidateAcceptAndPromoteBaseline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "baseline.json")
	candidate := CandidatePath(path)
	if candidate == path || filepath.Base(candidate) != "baseline.candidate.json" {
		t.Fatalf("CandidatePath(%q) = %q", path, candidate)
	}

	result := engine.RunResult{SchemaVersion: "1", Summary: engine.Summary{Score: 91}}
	written, err := Accept(path, result)
	if err != nil {
		t.Fatalf("Accept returned error: %v", err)
	}
	if written != candidate {
		t.Fatalf("Accept wrote %q, want %q", written, candidate)
	}
	if _, err := os.Stat(candidate); err != nil {
		t.Fatalf("candidate baseline missing: %v", err)
	}

	promotedCandidate, err := Promote(path)
	if err != nil {
		t.Fatalf("Promote returned error: %v", err)
	}
	if promotedCandidate != candidate {
		t.Fatalf("Promote candidate = %q, want %q", promotedCandidate, candidate)
	}
	if _, err := os.Stat(candidate); !os.IsNotExist(err) {
		t.Fatalf("candidate baseline should be removed after promote: %v", err)
	}
	loaded, ok, err := Load(path)
	if err != nil || !ok || loaded.Summary.Score != 91 {
		t.Fatalf("promoted baseline mismatch: ok=%t err=%v result=%+v", ok, err, loaded)
	}
	if _, err := Promote(path); err == nil {
		t.Fatal("Promote accepted a missing candidate baseline")
	}
}
