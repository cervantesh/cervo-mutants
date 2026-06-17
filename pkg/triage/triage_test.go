package triage

import (
	"strings"
	"testing"
)

func TestNormalizeResultClassifiesTimeouts(t *testing.T) {
	loop := NormalizeResult(Result{
		Status:       StatusTimedOut,
		StatusReason: "test command timed out",
		Mutant: Mutant{
			NonProgressRisk: "high",
		},
	})
	if loop.FailureKind != FailureKindNonProgressLoop {
		t.Fatalf("failure kind = %q, want %q", loop.FailureKind, FailureKindNonProgressLoop)
	}
	if !strings.Contains(loop.SuggestedSkip, "quarantine") {
		t.Fatalf("loop timeout missing skip guidance: %+v", loop)
	}

	plain := NormalizeResult(Result{
		Status: StatusTimedOut,
		Mutant: Mutant{},
	})
	if plain.FailureKind != FailureKindTimeout {
		t.Fatalf("failure kind = %q, want %q", plain.FailureKind, FailureKindTimeout)
	}
}

func TestRankSurvivorsDeprioritizesGroupsAndPlatformSensitiveWindowsMutants(t *testing.T) {
	ranked := RankSurvivors("windows", []Result{
		{MutantID: "plain", Status: StatusSurvived, Mutant: Mutant{Package: "./pkg", EquivalentRisk: "low", Recommendation: "fast-ci", NearbyTests: []string{"pkg/foo_test.go"}}},
		{MutantID: "group-a", Status: StatusSurvived, Mutant: Mutant{EquivalentRisk: "low", Recommendation: "fast-ci", NearbyTests: []string{"pkg/foo_test.go"}, SemanticGroup: "sort:1", GroupLabel: "sort comparator boundary"}},
		{MutantID: "group-b", Status: StatusSurvived, Mutant: Mutant{EquivalentRisk: "low", Recommendation: "fast-ci", NearbyTests: []string{"pkg/foo_test.go"}, SemanticGroup: "sort:1", GroupLabel: "sort comparator boundary"}},
		{MutantID: "platform", Status: StatusSurvived, Mutant: Mutant{EquivalentRisk: "low", Recommendation: "fast-ci", NearbyTests: []string{"pkg/foo_test.go"}, PlatformSensitive: true}},
	})
	if len(ranked) != 4 {
		t.Fatalf("ranked survivor count = %d, want 4", len(ranked))
	}
	if ranked[0].MutantID != "plain" {
		t.Fatalf("plain survivor should rank first: %+v", ranked)
	}
	if ranked[1].MutantID != "group-a" && ranked[1].MutantID != "group-b" {
		t.Fatalf("grouped survivor should rank before platform-sensitive windows mutant: %+v", ranked)
	}
	if ranked[1].SemanticGroupSize != 2 || ranked[2].SemanticGroupSize != 2 {
		t.Fatalf("group size not retained: %+v", ranked)
	}
	if !strings.Contains(ranked[1].SuggestedSkip, "review once") && !strings.Contains(ranked[2].SuggestedSkip, "review once") {
		t.Fatalf("grouped survivors missing shared skip guidance: %+v", ranked)
	}
	if ranked[3].MutantID != "platform" {
		t.Fatalf("platform-sensitive windows survivor should be deprioritized: %+v", ranked)
	}
}

func TestActionableViewAndSummaryHelpers(t *testing.T) {
	if TimeoutRiskBand(Mutant{Operator: "conditionals-negation"}) != "low" {
		t.Fatal("expected low timeout risk band")
	}
	if TimeoutRiskBand(Mutant{Operator: "inc-dec", NonProgressRisk: "high"}) != "very_high" {
		t.Fatal("expected very_high timeout risk band for non-progress loop risk")
	}
	if SemanticGroupSummaryKey(Mutant{GroupLabel: "sort comparator boundary", SemanticGroup: "sort:1"}) != "sort comparator boundary" {
		t.Fatal("expected group label to win")
	}
	if !IsActionableSurvivor("linux", Result{Status: StatusSurvived, Actionability: "medium", Mutant: Mutant{}}) {
		t.Fatal("expected medium survivor to stay actionable")
	}
	if IsActionableSurvivor("windows", Result{Status: StatusSurvived, Actionability: "high", Mutant: Mutant{PlatformSensitive: true}}) {
		t.Fatal("windows platform-sensitive survivor should not be actionable")
	}
	if IsActionableSurvivor("linux", Result{Status: StatusSurvived, Actionability: "low", Mutant: Mutant{}}) {
		t.Fatal("low actionability survivor should not stay actionable")
	}

	summary := BuildActionableSummary("windows", 50, 3, []Result{
		{MutantID: "group-lead", Status: StatusSurvived, Actionability: "high", Mutant: Mutant{EquivalentRisk: "high", SemanticGroup: "sort:1"}},
		{MutantID: "group-dup", Status: StatusSurvived, Actionability: "high", Mutant: Mutant{EquivalentRisk: "high", SemanticGroup: "sort:1"}},
		{MutantID: "platform", Status: StatusSurvived, Actionability: "medium", Mutant: Mutant{PlatformSensitive: true}},
		{MutantID: "keep", Status: StatusSurvived, Actionability: "medium", Mutant: Mutant{}},
		{MutantID: "timeout", Status: StatusTimedOut, Mutant: Mutant{NonProgressRisk: "high"}},
	})
	if summary.RawScore != 50 || summary.ActionableSurvivors != 3 || summary.TrueActionableSurvivors != 2 {
		t.Fatalf("unexpected actionable summary counts: %+v", summary)
	}
	if summary.EquivalentRiskSurvivors != 2 || summary.PlatformSensitiveSurvivors != 1 || summary.NonProgressTimeouts != 1 {
		t.Fatalf("unexpected actionable summary risk counters: %+v", summary)
	}
	if summary.SemanticGroupReviewUnits != 1 || summary.CollapsedSemanticDuplicates != 1 {
		t.Fatalf("unexpected semantic review-unit counters: %+v", summary)
	}
	if summary.ActionableScore != 60 {
		t.Fatalf("actionable score = %.2f, want 60", summary.ActionableScore)
	}
}
