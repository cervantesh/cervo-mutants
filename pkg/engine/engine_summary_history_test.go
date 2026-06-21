package engine

import (
	"encoding/json"
	"github.com/cervantesh/cervo-mutants/pkg/config"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestSummarizeReportsCoverageEfficacyAndMutatorStats(t *testing.T) {
	result := summarize([]MutantResult{
		{Status: StatusKilled, Mutant: Mutant{Operator: "conditionals-negation"}},
		{Status: StatusSurvived, Mutant: Mutant{Operator: "conditionals-negation"}},
		{Status: StatusNotCovered, Mutant: Mutant{Operator: "logical"}},
		{Status: StatusSkipped, Mutant: Mutant{Operator: "boolean"}},
	})

	if result.NotCovered != 1 {
		t.Fatalf("not covered = %d, want 1", result.NotCovered)
	}
	if result.GeneratedMutants != 4 || result.CoveredMutants != 2 || result.ExecutedMutants != 2 {
		t.Fatalf("decomposed counts not populated: %+v", result)
	}
	if result.EffectiveMutants != 2 || result.ScoreDenominator != 2 {
		t.Fatalf("effective denominator not populated: %+v", result)
	}
	if result.Score != 50 {
		t.Fatalf("score = %.2f, want 50", result.Score)
	}
	if result.TestEfficacy != 50 {
		t.Fatalf("test efficacy = %.2f, want 50", result.TestEfficacy)
	}
	if result.MutationCoverage != 66.66666666666666 {
		t.Fatalf("mutation coverage = %.14f, want 66.66666666666666", result.MutationCoverage)
	}
	if result.MutatorStats["conditionals-negation"].Killed != 1 || result.MutatorStats["conditionals-negation"].Survived != 1 {
		t.Fatalf("conditionals stats not populated: %+v", result.MutatorStats["conditionals-negation"])
	}
	if result.MutatorStats["logical"].NotCovered != 1 {
		t.Fatalf("logical not-covered stats not populated: %+v", result.MutatorStats["logical"])
	}
}

func TestSummarizeSeparatesTestEfficacyFromDenominatorHealth(t *testing.T) {
	result := summarize([]MutantResult{
		{Status: StatusKilled, Mutant: Mutant{Operator: "conditionals-negation"}},
		{Status: StatusTimedOut, Mutant: Mutant{Operator: "arithmetic-basic"}},
		{Status: StatusTimedOut, Mutant: Mutant{Operator: "arithmetic-basic"}},
		{Status: StatusNotCovered, Mutant: Mutant{Operator: "logical"}},
		{Status: StatusNotCovered, Mutant: Mutant{Operator: "logical"}},
	})

	if result.Score < 33.3333 || result.Score > 33.3334 {
		t.Fatalf("score = %.4f, want %.4f", result.Score, 100.0/3.0)
	}
	if result.TestEfficacy != 100 {
		t.Fatalf("test efficacy = %.2f, want 100 over killed+survived", result.TestEfficacy)
	}
	if result.EffectiveMutants != 1 || result.ScoreDenominator != 3 {
		t.Fatalf("unexpected denominators: %+v", result)
	}
	warnings := strings.Join(result.DenominatorHealth.Warnings, ",")
	for _, want := range []string{"timed_out_exceeds_effective", "not_covered_exceeds_effective", "high_score_poor_denominator_health"} {
		if !strings.Contains(warnings, want) {
			t.Fatalf("denominator warnings missing %q: %+v", want, result.DenominatorHealth)
		}
	}
	if result.DenominatorHealth.Healthy {
		t.Fatalf("denominator health should not be healthy: %+v", result.DenominatorHealth)
	}
}

func TestHistoryTracksNewAndLongStandingSurvivors(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Defaults()
	cfg.History.Path = filepath.Join(dir, "history.json")
	e := New(cfg)
	first := []MutantResult{{
		MutantID: "m-history",
		Status:   StatusSurvived,
		Mutant:   Mutant{Operator: "conditionals-negation"},
	}}

	stats := e.applyHistory(first)
	if stats.NewSurvivors != 1 {
		t.Fatalf("new survivors = %d, want 1", stats.NewSurvivors)
	}
	if first[0].HistoryStatus != "new_survivor" || first[0].SurvivorAgeRuns != 1 {
		t.Fatalf("first run history not populated: %+v", first[0])
	}

	second := []MutantResult{{
		MutantID: "m-history",
		Status:   StatusSurvived,
		Mutant:   Mutant{Operator: "conditionals-negation"},
	}}
	stats = e.applyHistory(second)
	if stats.LongStandingSurvivors != 1 {
		t.Fatalf("long-standing survivors = %d, want 1", stats.LongStandingSurvivors)
	}
	if second[0].PreviousStatus != StatusSurvived || second[0].HistoryStatus != "long_standing_survivor" || second[0].SurvivorAgeRuns != 2 {
		t.Fatalf("second run history not populated: %+v", second[0])
	}
}

func TestHistoryDisabledAndMixedStatuses(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Defaults()
	cfg.History.Enabled = false
	stats := New(cfg).applyHistory([]MutantResult{{MutantID: "m-disabled", Status: StatusSurvived}})
	if stats.Enabled {
		t.Fatalf("history should be disabled: %+v", stats)
	}

	cfg = config.Defaults()
	cfg.History.Path = filepath.Join(dir, "history.json")
	e := New(cfg)
	results := []MutantResult{
		{MutantID: "k", Status: StatusKilled, Mutant: Mutant{Operator: "op"}},
		{MutantID: "n", Status: StatusNotCovered, Mutant: Mutant{Operator: "op"}},
		{MutantID: "c", Status: StatusCompileError, Mutant: Mutant{Operator: "op"}},
		{MutantID: "t", Status: StatusTimedOut, Mutant: Mutant{Operator: "op"}},
	}
	stats = e.applyHistory(results)
	if stats.UpdatedMutants != len(results) || stats.OperatorUsefulSurvivor["op"] != 0 {
		t.Fatalf("mixed history stats not populated: %+v", stats)
	}
	for _, result := range results {
		if result.HistoryStatus != "seen" || result.FirstSeen == "" || result.LastSeen == "" {
			t.Fatalf("history fields not populated: %+v", result)
		}
	}
}

func TestRecordHistoryRunAppendsHistoricalSnapshots(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Defaults()
	cfg.History.Path = filepath.Join(dir, "history.json")
	e := New(cfg)

	firstMutants := []MutantResult{{
		MutantID: "m1",
		Status:   StatusSurvived,
		Mutant:   Mutant{Operator: "logical"},
	}}
	firstHistory := e.applyHistory(firstMutants)
	firstRun := RunResult{
		Summary: Summary{
			Score:               50,
			Survived:            1,
			TimedOut:            1,
			NonProgressTimeouts: 1,
			Actionable: ActionableSummary{
				ActionableScore:         66.67,
				TrueActionableSurvivors: 1,
			},
		},
		History: firstHistory,
		Mutants: firstMutants,
	}

	e.recordHistoryRun(&firstRun)
	if len(firstRun.History.Runs) != 1 {
		t.Fatalf("first historical run snapshot missing: %+v", firstRun.History.Runs)
	}
	if firstRun.History.Runs[0].RawScore != 50 || firstRun.History.Runs[0].ActionableScore != 66.67 {
		t.Fatalf("first historical run scores missing: %+v", firstRun.History.Runs[0])
	}

	secondMutants := []MutantResult{{
		MutantID: "m1",
		Status:   StatusKilled,
		Mutant:   Mutant{Operator: "logical"},
	}}
	secondHistory := e.applyHistory(secondMutants)
	secondRun := RunResult{
		Summary: Summary{
			Score:    75,
			Survived: 0,
			Actionable: ActionableSummary{
				ActionableScore:         100,
				TrueActionableSurvivors: 0,
			},
		},
		History: secondHistory,
		Mutants: secondMutants,
	}

	e.recordHistoryRun(&secondRun)
	if len(secondRun.History.Runs) != 2 {
		t.Fatalf("historical snapshots were not appended: %+v", secondRun.History.Runs)
	}
	if secondRun.History.Runs[1].RawScore != 75 || secondRun.History.Runs[1].Survived != 0 {
		t.Fatalf("second historical snapshot mismatch: %+v", secondRun.History.Runs[1])
	}

	data, err := os.ReadFile(cfg.History.Path)
	if err != nil {
		t.Fatalf("history file missing: %v", err)
	}
	var stored struct {
		Runs []HistoryRun `json:"runs"`
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("history file is not valid JSON: %v", err)
	}
	if len(stored.Runs) != 2 {
		t.Fatalf("stored history runs = %d, want 2", len(stored.Runs))
	}
}

func TestBudgetSchedulingPrioritizesFastOperators(t *testing.T) {
	cfg := config.Defaults()
	cfg.Execution.Budget = 1
	e := New(cfg)
	mutants := []Mutant{
		{ID: "z", Recommendation: "aggressive"},
		{ID: "b", Recommendation: "fast-ci"},
		{ID: "a", Recommendation: "default"},
	}

	e.scheduleMutants(mutants)

	got := []string{mutants[0].ID, mutants[1].ID, mutants[2].ID}
	want := []string{"b", "a", "z"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("scheduled IDs = %v, want %v", got, want)
	}
}

func TestBudgetSchedulingUsesTimeoutRiskWithinSameRecommendation(t *testing.T) {
	cfg := config.Defaults()
	cfg.Execution.Budget = time.Minute
	e := New(cfg)
	mutants := []Mutant{
		{ID: "slow", Recommendation: "default", Operator: "loop-control"},
		{ID: "fast", Recommendation: "default", Operator: "conditionals-negation"},
		{ID: "medium", Recommendation: "default", Operator: "arithmetic-basic"},
	}

	e.scheduleMutants(mutants)

	got := []string{mutants[0].ID, mutants[1].ID, mutants[2].ID}
	want := []string{"fast", "medium", "slow"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("scheduled IDs = %v, want %v", got, want)
	}
}

func TestSummarizeIncludesTimeoutRiskStats(t *testing.T) {
	result := summarize([]MutantResult{
		{MutantID: "fast", Status: StatusKilled, Mutant: Mutant{ID: "fast", Operator: "conditionals-negation"}},
		{MutantID: "slow", Status: StatusTimedOut, Mutant: Mutant{ID: "slow", Operator: "loop-control"}},
	})
	if result.TimeoutRiskStats["low"] != 1 || result.TimeoutRiskStats["very_high"] != 1 {
		t.Fatalf("unexpected timeout risk stats: %+v", result.TimeoutRiskStats)
	}
}

func TestSummarizeIncludesSemanticTriageStats(t *testing.T) {
	results := []MutantResult{
		{MutantID: "loop", Status: StatusTimedOut, Mutant: Mutant{ID: "loop", Operator: "inc-dec", NonProgressRisk: "high"}},
		{MutantID: "perm", Status: StatusSurvived, Actionability: "medium", Mutant: Mutant{ID: "perm", PlatformSensitive: true}},
		{MutantID: "group-a", Status: StatusSurvived, Actionability: "high", Mutant: Mutant{ID: "group-a", EquivalentRisk: "high", SemanticGroup: "sort:1", GroupLabel: "sort comparator boundary"}},
		{MutantID: "group-b", Status: StatusSurvived, Actionability: "high", Mutant: Mutant{ID: "group-b", EquivalentRisk: "high", SemanticGroup: "sort:1", GroupLabel: "sort comparator boundary"}},
		{MutantID: "keep", Status: StatusSurvived, Actionability: "medium", Mutant: Mutant{ID: "keep", Operator: "logical"}},
		{MutantID: "killed", Status: StatusKilled, Mutant: Mutant{ID: "killed", Operator: "logical"}},
	}
	result := summarize(results)
	if result.NonProgressTimeouts != 1 {
		t.Fatalf("non-progress timeouts = %d, want 1", result.NonProgressTimeouts)
	}
	if result.PlatformSensitiveSurvivors != 1 {
		t.Fatalf("platform-sensitive survivors = %d, want 1", result.PlatformSensitiveSurvivors)
	}
	if result.SemanticGroupStats["sort comparator boundary"] != 2 {
		t.Fatalf("semantic group stats = %+v", result.SemanticGroupStats)
	}
	if result.Actionable.RawScore == 0 || result.Actionable.ActionableScore == 0 {
		t.Fatalf("actionable score block missing scores: %+v", result.Actionable)
	}
	expectedActionableSurvivors := 4
	expectedTrueActionable := 3
	expectedActionableScore := 25.0
	if runtime.GOOS == "windows" {
		expectedActionableSurvivors = 3
		expectedTrueActionable = 2
		expectedActionableScore = 100.0 / 3.0
	}
	if result.Actionable.Survivors != 4 || result.Actionable.ActionableSurvivors != expectedActionableSurvivors || result.Actionable.TrueActionableSurvivors != expectedTrueActionable {
		t.Fatalf("unexpected actionable survivor counts: %+v", result.Actionable)
	}
	if result.Actionable.EquivalentRiskSurvivors != 2 || result.Actionable.SemanticGroupReviewUnits != 1 || result.Actionable.CollapsedSemanticDuplicates != 1 {
		t.Fatalf("unexpected actionable grouping counters: %+v", result.Actionable)
	}
	if result.Actionable.PlatformSensitiveSurvivors != 1 || result.Actionable.NonProgressTimeouts != 1 {
		t.Fatalf("unexpected actionable platform/timeout counters: %+v", result.Actionable)
	}
	if math.Abs(result.Actionable.ActionableScore-expectedActionableScore) > 1e-9 {
		t.Fatalf("actionable score = %.14f, want %.14f", result.Actionable.ActionableScore, expectedActionableScore)
	}
}

func TestSummarizeCoversCachedAndSuppressionStatusBranches(t *testing.T) {
	result := summarize([]MutantResult{
		{Status: StatusCached, PreviousStatus: StatusKilled, Mutant: Mutant{Operator: "cached-killed"}},
		{Status: StatusCached, PreviousStatus: StatusSurvived, Mutant: Mutant{Operator: "cached-survived", EquivalentRisk: "high"}},
		{Status: StatusCached, PreviousStatus: StatusNotCovered, Mutant: Mutant{Operator: "cached-not-covered"}},
		{Status: StatusCached, PreviousStatus: StatusCompileError, Mutant: Mutant{Operator: "cached-compile"}},
		{Status: StatusCached, PreviousStatus: StatusTimedOut, Mutant: Mutant{Operator: "cached-timeout"}},
		{Status: StatusCached, PreviousStatus: StatusMemoryKilled, Mutant: Mutant{Operator: "cached-memory"}},
		{Status: StatusCompileError, Mutant: Mutant{Operator: "compile"}},
		{Status: StatusSkippedResource, Mutant: Mutant{Operator: "resource-skip"}},
		{Status: StatusPendingBudget, Mutant: Mutant{Operator: "budget-pending"}},
		{Status: StatusIgnored, Mutant: Mutant{Operator: "ignored"}},
		{Status: StatusQuarantined, Mutant: Mutant{Operator: "quarantined", SuppressionAudit: []SuppressionAudit{
			{Action: config.SuppressionReportOnly},
			{Action: config.SuppressionLowerPriority},
			{Action: "suppress"},
			{Action: "quarantine-required"},
		}}},
	})
	if result.Cached != 6 || result.Killed != 1 || result.Survived != 1 || result.NotCovered != 1 || result.CompileError != 2 || result.TimedOut != 1 || result.MemoryKilled != 1 || result.SkippedResource != 1 || result.PendingBudget != 1 {
		t.Fatalf("cached/status branches not summarized: %+v", result)
	}
	if result.SuppressionReportOnly != 1 || result.SuppressionLowerPriority != 1 || result.SuppressionSuppressed != 1 || result.SuppressionQuarantineRequired != 1 {
		t.Fatalf("suppression audit branches not summarized: %+v", result)
	}
}

func TestRankSurvivorsPrioritizesLowerEquivalentRisk(t *testing.T) {
	results := []MutantResult{
		{MutantID: "high", Status: StatusSurvived, Mutant: Mutant{EquivalentRisk: "high", Recommendation: "fast-ci"}},
		{MutantID: "low", Status: StatusSurvived, Mutant: Mutant{EquivalentRisk: "low", Recommendation: "conservative", NearbyTests: []string{"x_test.go"}}},
		{MutantID: "killed", Status: StatusKilled, Mutant: Mutant{EquivalentRisk: "low"}},
	}

	rankSurvivors(results)

	if results[1].SurvivorRank != 1 || results[0].SurvivorRank != 2 {
		t.Fatalf("unexpected survivor ranks: %+v", results)
	}
	if results[2].SurvivorRank != 0 {
		t.Fatalf("killed mutant should not be ranked: %+v", results[2])
	}
}

func TestRankSurvivorsDeprioritizesSemanticGroups(t *testing.T) {
	results := []MutantResult{
		{MutantID: "plain", Status: StatusSurvived, Mutant: Mutant{EquivalentRisk: "low", Recommendation: "fast-ci", NearbyTests: []string{"x_test.go"}}},
		{MutantID: "group-a", Status: StatusSurvived, Mutant: Mutant{EquivalentRisk: "low", Recommendation: "fast-ci", NearbyTests: []string{"x_test.go"}, SemanticGroup: "sort:1", GroupLabel: "sort comparator boundary"}},
		{MutantID: "group-b", Status: StatusSurvived, Mutant: Mutant{EquivalentRisk: "low", Recommendation: "fast-ci", NearbyTests: []string{"x_test.go"}, SemanticGroup: "sort:1", GroupLabel: "sort comparator boundary"}},
	}

	rankSurvivors(results)

	if results[0].SurvivorRank != 1 {
		t.Fatalf("plain survivor should rank first: %+v", results)
	}
	if results[1].SemanticGroupSize != 2 || results[2].SemanticGroupSize != 2 {
		t.Fatalf("semantic group size not recorded: %+v", results)
	}
	if !strings.Contains(results[1].SuggestedSkipReason, "review once") {
		t.Fatalf("grouped survivor missing semantic skip reason: %+v", results[1])
	}
}
