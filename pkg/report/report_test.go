package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cervantesh/cervo-mutants/pkg/engine"
)

func TestJSONReportSchemaV1IncludesActionableFields(t *testing.T) {
	run := engine.RunResult{
		SchemaVersion: "1",
		Environment:   engine.Environment{OS: "linux", Arch: "amd64", GoVersion: "go1.25.6", TempRoot: "/tmp/cervomut", Isolation: "overlay", Workers: 1, TestTimeout: "30s", WSL: true, Warnings: []string{"workspace path contains spaces"}},
		Slice:         engine.SliceMetadata{Enabled: true, SliceBy: "package", ShardIndex: 2, ShardCount: 4, GroupCount: 12, SelectedGroups: 3, MaxFilesPerRun: 5, SelectedFiles: 5, MaxMutantsPerPackage: 10, SelectedMutants: 25},
		Checkpoint:    engine.Checkpoint{Fingerprint: "abc123", Mutants: 1, IncludesFileDigests: true, Reason: "final"},
		Summary: engine.Summary{
			Total:                      1,
			Survived:                   1,
			Score:                      0,
			Actionable:                 engine.ActionableSummary{RawScore: 0, ActionableScore: 0, Survivors: 1, ActionableSurvivors: 1, TrueActionableSurvivors: 1, EquivalentRiskSurvivors: 1, SemanticGroupReviewUnits: 1},
			Quarantined:                0,
			PlatformSensitiveSurvivors: 1,
			NonProgressTimeouts:        1,
			SemanticGroupStats:         map[string]int{"sort comparator boundary": 1},
		},
		Mutants: []engine.MutantResult{{
			MutantID:        "pkg/foo.go:10:conditionals-negation:eq-to-ne",
			Status:          engine.StatusSurvived,
			FailureKind:     "runner_error",
			MemoryPeakBytes: 4096,
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
				SemanticTags:     []string{"equivalence-risk-group", "sort-comparator-boundary"},
				SemanticGroup:    "sort-boundary:pkg/foo.go:10",
				GroupLabel:       "sort comparator boundary",
				GroupReason:      "Boundary mutations inside sort comparator closures often collapse into one review decision.",
				Ownership: &engine.OwnershipRoute{
					Owner:   "pkg-owner",
					Team:    "platform",
					Contact: "@platform",
					Rule:    "pkg-review",
				},
				SuggestedSkipReason: "review once for this semantic group before treating each survivor independently",
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
			TestRecommendation: &engine.TestRecommendation{
				Priority:            "high",
				Strategy:            "tighten-branch-assertions",
				Summary:             "Start with `pkg/foo_test.go`: Add an assertion for the opposite branch.",
				CandidateTests:      []string{"pkg/foo_test.go"},
				SuggestedAssertions: []string{"Add an assertion for the opposite branch."},
				Rationale:           []string{"coverage_source=coverage-mode -> the mutant was matched by coverage data, so the next test should usually be an assertion upgrade"},
			},
			SuggestedSkipReason: "review once for this semantic group before treating each survivor independently",
			NearestTests:        []string{"pkg/foo_test.go"},
			SemanticGroupSize:   2,
			PreviousStatus:      engine.StatusKilled,
			FirstSeen:           "2026-05-26T00:00:00Z",
			LastSeen:            "2026-05-26T01:00:00Z",
			SurvivorAgeRuns:     2,
			HistoryStatus:       "long_standing_survivor",
			OperatorYield:       0.5,
		}},
		Quarantine: engine.QuarantineStats{
			Active:        1,
			Expired:       0,
			Path:          ".cervomut/quarantine.json",
			ExpireAfter:   "720h0m0s",
			RequireReason: true,
			RequireOwner:  true,
			RequireIssue:  true,
			FailOnExpired: true,
			MaxRenewals:   1,
		},
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
	for _, want := range []string{"environment", "go_version", "temp_root", "warnings", "slice", "slice_by", "shard_index", "shard_count", "selected_files", "max_mutants_per_package", "isolation", "checkpoint", "fingerprint", "includes_file_digests", "failure_kind", "memory_peak_bytes", "baseline", "cache", "quarantine", "expire_after", "require_owner", "require_issue", "max_renewals", "history", "unified_diff", "status_reason", "selection_reason", "coverage_source", "selected_tests", "description", "nearby_tests", "equivalent_risk", "recommendation", "compile_error_risk", "semantic_tags", "semantic_group", "group_label", "group_reason", "ownership", "owner", "team", "contact", "rule", "suggested_skip_reason", "semantic_group_size", "semantic_group_statistics", "platform_sensitive_survivors", "non_progress_timeouts", "actionable", "raw_score", "actionable_score", "true_actionable_survivors", "equivalent_risk_survivors", "semantic_group_review_units", "collapsed_semantic_duplicates", "suppression_audit", "evidence_level", "survivor_rank", "rank_score", "rank_reason", "actionability", "suggested_test_scope", "test_recommendation", "candidate_tests", "suggested_assertions", "rationale", "nearest_tests", "previous_status", "first_seen", "survivor_age_runs", "operator_historical_yield"} {
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
			EffectiveScore:   50,
			EffectiveMutants: 2,
			ScoreDenominator: 2,
			TestEfficacy:     50,
			MutationCoverage: 66.66666666666666,
			Actionable: engine.ActionableSummary{
				RawScore:                    50,
				ActionableScore:             75,
				Survivors:                   1,
				ActionableSurvivors:         1,
				TrueActionableSurvivors:     1,
				EquivalentRiskSurvivors:     1,
				PlatformSensitiveSurvivors:  1,
				NonProgressTimeouts:         1,
				SemanticGroupReviewUnits:    1,
				CollapsedSemanticDuplicates: 1,
			},
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
			HighRiskSurvivors:          1,
			NewSurvivors:               1,
			LongStandingSurvivors:      1,
			PlatformSensitiveSurvivors: 1,
			NonProgressTimeouts:        1,
			SuppressionReportOnly:      2,
			EquivalentRiskStats:        map[string]int{"high": 1, "medium": 2},
			SemanticGroupStats:         map[string]int{"sort comparator boundary": 2},
			MutatorStats: map[string]engine.MutatorStat{
				"conditionals-negation": {Total: 2, Killed: 1, Survived: 1, Recommendation: "fast-ci"},
				"logical":               {Total: 1, NotCovered: 1, Recommendation: "conservative"},
			},
		},
		Environment: engine.Environment{OS: "linux", Arch: "amd64", GoVersion: "go1.25.6", TempRoot: "/tmp/cervomut", Isolation: "overlay", Workers: 1, TestTimeout: "30s", Warnings: []string{"workspace path contains spaces"}},
		Slice:       engine.SliceMetadata{Enabled: true, SliceBy: "package", ShardIndex: 2, ShardCount: 4, GroupCount: 12, SelectedGroups: 3, MaxFilesPerRun: 5, SelectedFiles: 5, MaxMutantsPerPackage: 10, SelectedMutants: 25},
	}

	text := Summary(run)
	for _, want := range []string{
		"Effective mutation score: 50.00%",
		"Raw mutation score: 50.00%",
		"Not covered: 1",
		"Effective mutants: 2",
		"Score denominator: 2",
		"Test efficacy: 50.00%",
		"Mutation coverage: 66.67%",
		"Actionable score: 75.00%",
		"Actionable survivors: 1",
		"True actionable survivors: 1",
		"Equivalent-risk survivors: 1",
		"Semantic review units: 1",
		"Collapsed semantic duplicates: 1",
		"Lane shape: grouped review lane",
		"Lane note: 1 raw survivors collapsed into 1 immediate review units across 1 semantic group",
		"Lane guidance: review the grouped equivalent-risk family once before splitting it into multiple separate test tasks",
		"Denominator health: healthy=true generated=3 covered=2 executed=2 effective=2 score_denominator=2 killed=1 survived=1 not_covered=1 pending_budget=0 skipped_resource=0 timed_out=0 memory_killed=0 compile_error=0",
		"High-risk survivors: 1",
		"New survivors: 1",
		"Long-standing survivors: 1",
		"Platform-sensitive survivors: 1",
		"Non-progress timeouts: 1",
		"Suppression audits: report_only=2",
		"Equivalent-risk statistics:",
		"Semantic-group statistics:",
		"sort comparator boundary: 2",
		"conditionals-negation: total=2 killed=1 survived=1 not_covered=0 pending_budget=0 skipped_resource=0 timed_out=0 memory_killed=0 compile_error=0",
		"recommendation=fast-ci",
		"logical: total=1 killed=0 survived=0 not_covered=1 pending_budget=0 skipped_resource=0 timed_out=0 memory_killed=0 compile_error=0",
		"recommendation=conservative",
		"Slice: by=package shard=2/4 groups=12 selected_groups=3 files=5 max_files=5 max_mutants_per_package=10 selected_mutants=25",
		"Environment: os=linux arch=amd64 go=go1.25.6 isolation=overlay workers=1 timeout=30s",
		"Temp root: /tmp/cervomut",
		"Environment warnings: workspace path contains spaces",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("summary missing %q:\n%s", want, text)
		}
	}
}

func TestSummaryIncludesDenominatorGuidanceForLowSignalRuns(t *testing.T) {
	run := engine.RunResult{
		Summary: engine.Summary{
			DenominatorHealth: engine.DenominatorHealth{
				Generated:        10,
				Covered:          2,
				Executed:         2,
				Effective:        0,
				ScoreDenominator: 8,
				NotCovered:       8,
				Healthy:          false,
				Warnings:         []string{"no_effective_mutants", "not_covered_exceeds_effective"},
			},
		},
	}

	text := Summary(run)
	for _, want := range []string{
		"Lane shape: retargeting signal",
		"Lane note: the run completed, but denominator pressure dominates and this bounded slice did not produce immediate review work",
		"Lane guidance: keep the artifact and retarget the next run to a hotter package, subtree, or shard before judging broader rollout fit",
		"Denominator warnings: no_effective_mutants, not_covered_exceeds_effective",
		"Denominator guidance:",
		"Preserve this report and treat the run as target-selection feedback before changing score expectations.",
		"Retarget the next run to a hotter package, subtree, or bounded shard before widening to ./....",
		"Rerun on the narrower target before judging recommendation quality or broader rollout fit.",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("summary missing %q:\n%s", want, text)
		}
	}
}

func TestSummaryClassifiesHealthyNoActionLane(t *testing.T) {
	run := engine.RunResult{
		Summary: engine.Summary{
			Actionable: engine.ActionableSummary{
				ActionableScore:         100,
				TrueActionableSurvivors: 0,
			},
			DenominatorHealth: engine.DenominatorHealth{
				Healthy: true,
			},
		},
	}

	text := Summary(run)
	for _, want := range []string{
		"Lane shape: healthy no-action lane",
		"Lane note: this bounded slice produced understandable denominator health and no immediate survivor work",
		"Lane guidance: keep the artifact; widen or retarget only if you need more review pressure from a different slice",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("summary missing %q:\n%s", want, text)
		}
	}
}

func TestSurvivorsReportIsRanked(t *testing.T) {
	run := engine.RunResult{
		Mutants: []engine.MutantResult{
			{MutantID: "later", Status: engine.StatusSurvived, SurvivorRank: 2, RankReason: "risk=high", Actionability: "medium", SuggestedSkipReason: "review once", SemanticGroupSize: 2, TestRecommendation: &engine.TestRecommendation{Strategy: "tighten-value-assertions", CandidateTests: []string{"pkg/a_test.go"}}, Mutant: engine.Mutant{File: "a.go", Line: 2, Operator: "returns", Original: "x", Mutated: "y", SemanticGroup: "sort:1", GroupLabel: "sort comparator boundary", GroupReason: "Boundary mutations inside sort comparator closures often collapse into one review decision.", Ownership: &engine.OwnershipRoute{Owner: "platform-owner", Team: "platform", Rule: "pkg-platform"}}},
			{MutantID: "first", Status: engine.StatusSurvived, SurvivorRank: 1, RankReason: "risk=low", Actionability: "high", TestRecommendation: &engine.TestRecommendation{Strategy: "tighten-branch-assertions", CandidateTests: []string{"pkg/b_test.go"}}, Mutant: engine.Mutant{File: "b.go", Line: 1, Operator: "boolean", Original: "true", Mutated: "false"}},
			{MutantID: "again", Status: engine.StatusSurvived, SurvivorRank: 3, RankReason: "risk=high", Actionability: "medium", SuggestedSkipReason: "review once", SemanticGroupSize: 2, TestRecommendation: &engine.TestRecommendation{Strategy: "tighten-value-assertions", CandidateTests: []string{"pkg/c_test.go"}}, Mutant: engine.Mutant{File: "c.go", Line: 3, Operator: "returns", Original: "x", Mutated: "z", SemanticGroup: "sort:1", GroupLabel: "sort comparator boundary", GroupReason: "Boundary mutations inside sort comparator closures often collapse into one review decision."}},
		},
	}

	text := Survivors(run)
	if !strings.Contains(text, "#1 0.0 first") || strings.Index(text, "#1 0.0 first") > strings.Index(text, "#2 0.0 later") {
		t.Fatalf("survivors were not ranked:\n%s", text)
	}
	if !strings.Contains(text, "Group sort comparator boundary (2 mutants)") {
		t.Fatalf("survivors report missing semantic group header:\n%s", text)
	}
	if !strings.Contains(text, "next_test=pkg/b_test.go") || !strings.Contains(text, "strategy=tighten-branch-assertions") {
		t.Fatalf("survivors report missing test recommendation context:\n%s", text)
	}
	if !strings.Contains(text, "ownership=owner=platform-owner team=platform rule=pkg-platform") {
		t.Fatalf("survivors report missing ownership route:\n%s", text)
	}
}

func TestSurvivorsActionableOnlyFiltersAndCollapses(t *testing.T) {
	run := engine.RunResult{
		Environment: engine.Environment{OS: "windows"},
		Mutants: []engine.MutantResult{
			{MutantID: "group-lead", Status: engine.StatusSurvived, SurvivorRank: 1, Actionability: "high", SemanticGroupSize: 2, SuggestedSkipReason: "review once", Mutant: engine.Mutant{File: "a.go", Line: 1, Operator: "conditionals-boundary", Original: "<", Mutated: "<=", SemanticGroup: "sort:1", GroupLabel: "sort comparator boundary", GroupReason: "shared review"}},
			{MutantID: "group-dup", Status: engine.StatusSurvived, SurvivorRank: 2, Actionability: "medium", SemanticGroupSize: 2, SuggestedSkipReason: "review once", Mutant: engine.Mutant{File: "a.go", Line: 2, Operator: "conditionals-boundary", Original: "<", Mutated: "<=", SemanticGroup: "sort:1", GroupLabel: "sort comparator boundary", GroupReason: "shared review"}},
			{MutantID: "platform", Status: engine.StatusSurvived, SurvivorRank: 3, Actionability: "high", Mutant: engine.Mutant{File: "b.go", Line: 3, Operator: "numeric-literals", Original: "0o755", Mutated: "0", PlatformSensitive: true}},
			{MutantID: "low", Status: engine.StatusSurvived, SurvivorRank: 4, Actionability: "low", Mutant: engine.Mutant{File: "c.go", Line: 4, Operator: "literals", Original: "1", Mutated: "0"}},
			{MutantID: "keep", Status: engine.StatusSurvived, SurvivorRank: 5, Actionability: "medium", Mutant: engine.Mutant{File: "d.go", Line: 5, Operator: "logical", Original: "&&", Mutated: "||"}},
		},
	}

	text := SurvivorsWithOptions(run, SurvivorsOptions{ActionableOnly: true})
	for _, want := range []string{
		"Actionable-only view: showing 2 of 5 survivors (filtered=2 collapsed=1)",
		"Group sort comparator boundary (2 mutants): shared review",
		"group-lead",
		"keep",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("actionable-only survivors missing %q:\n%s", want, text)
		}
	}
	for _, avoid := range []string{"#2 0.0 group-dup ", "#3 0.0 platform ", "#4 0.0 low "} {
		if strings.Contains(text, avoid) {
			t.Fatalf("actionable-only survivors should not include %q:\n%s", avoid, text)
		}
	}
}

func TestRecommendationsIncludeOwnershipRoute(t *testing.T) {
	run := engine.RunResult{
		Mutants: []engine.MutantResult{{
			MutantID:           "owned",
			Status:             engine.StatusSurvived,
			SurvivorRank:       1,
			Actionability:      "high",
			SuggestedTestScope: "./pkg",
			TestRecommendation: &engine.TestRecommendation{Priority: "high", Strategy: "tighten-branch-assertions"},
			Mutant: engine.Mutant{
				File:      "pkg/foo.go",
				Line:      9,
				Operator:  "logical",
				Ownership: &engine.OwnershipRoute{Owner: "qa-owner", Team: "qa", Contact: "@qa", Rule: "pkg-qa"},
			},
		}},
	}

	text := TestRecommendations(run)
	if !strings.Contains(text, "- Ownership: `owner=qa-owner team=qa contact=@qa rule=pkg-qa`") {
		t.Fatalf("recommendations missing ownership route:\n%s", text)
	}
}

func TestSemanticTriageLedgerGroupsAndSuggestsActions(t *testing.T) {
	run := engine.RunResult{
		Environment: engine.Environment{OS: "windows"},
		Mutants: []engine.MutantResult{
			{MutantID: "group-lead", Status: engine.StatusSurvived, SurvivorRank: 1, Actionability: "high", SemanticGroupSize: 2, SuggestedSkipReason: "review once", Mutant: engine.Mutant{
				File:                "a.go",
				Line:                1,
				Operator:            "conditionals-boundary",
				Original:            "<",
				Mutated:             "<=",
				EquivalentRisk:      "high",
				SemanticTags:        []string{"equivalence-risk-group", "sort-comparator-boundary"},
				SemanticGroup:       "sort:1",
				GroupLabel:          "sort comparator boundary",
				GroupReason:         "shared review",
				SuggestedSkipReason: "review once",
			}},
			{MutantID: "group-dup", Status: engine.StatusSurvived, SurvivorRank: 2, Actionability: "medium", SemanticGroupSize: 2, Mutant: engine.Mutant{
				File:           "a.go",
				Line:           2,
				Operator:       "conditionals-boundary",
				Original:       "<",
				Mutated:        "<=",
				EquivalentRisk: "high",
				SemanticGroup:  "sort:1",
				GroupLabel:     "sort comparator boundary",
				GroupReason:    "shared review",
			}},
			{MutantID: "platform", Status: engine.StatusSurvived, Actionability: "medium", Mutant: engine.Mutant{
				File:                "b.go",
				Line:                3,
				Operator:            "numeric-literals",
				Original:            "0o755",
				Mutated:             "0",
				PlatformSensitive:   true,
				SuggestedSkipReason: "review on windows first",
			}},
			{MutantID: "timeout", Status: engine.StatusTimedOut, FailureKind: "non_progress_loop", StatusReason: "loop variable stopped making progress", Mutant: engine.Mutant{
				File:                "c.go",
				Line:                4,
				Operator:            "inc-dec",
				Original:            "i++",
				Mutated:             "i--",
				NonProgressRisk:     "high",
				SuggestedSkipReason: "reviewed-skip or quarantine if timeout confirms the loop is non-progress",
			}},
			{MutantID: "fallback", Status: engine.StatusSurvived, Actionability: "low", SuggestedSkipReason: "reviewed-skip after confirming fallback equivalence", Mutant: engine.Mutant{
				File:           "d.go",
				Line:           5,
				Operator:       "literals",
				Original:       "\"fallback\"",
				Mutated:        "\"noop\"",
				EquivalentRisk: "high",
				SemanticTags:   []string{"fallback-literal"},
			}},
		},
	}

	data, err := SemanticTriageLedger(run)
	if err != nil {
		t.Fatalf("SemanticTriageLedger returned error: %v", err)
	}
	var ledger TriageLedger
	if err := json.Unmarshal(data, &ledger); err != nil {
		t.Fatalf("ledger is not JSON: %v", err)
	}
	if ledger.SchemaVersion != "1" {
		t.Fatalf("ledger schema version = %q, want 1", ledger.SchemaVersion)
	}
	if len(ledger.Entries) != 4 {
		t.Fatalf("ledger entry count = %d, want 4: %+v", len(ledger.Entries), ledger.Entries)
	}

	var (
		groupEntry    *TriageLedgerEntry
		platformEntry *TriageLedgerEntry
		timeoutEntry  *TriageLedgerEntry
		fallbackEntry *TriageLedgerEntry
	)
	for i := range ledger.Entries {
		entry := &ledger.Entries[i]
		switch entry.MutantID {
		case "group-lead":
			groupEntry = entry
		case "platform":
			platformEntry = entry
		case "timeout":
			timeoutEntry = entry
		case "fallback":
			fallbackEntry = entry
		}
	}
	if groupEntry == nil || groupEntry.SuggestedAction != "reviewed-skip" || groupEntry.GroupKey != "sort:1" || groupEntry.GroupSize != 2 || len(groupEntry.MutantIDs) != 2 {
		t.Fatalf("group entry unexpected: %+v", groupEntry)
	}
	if platformEntry == nil || platformEntry.Risk != "platform-sensitive" || platformEntry.SuggestedAction != "reviewed-skip" || !strings.Contains(strings.Join(platformEntry.Evidence, " "), "goos=windows") {
		t.Fatalf("platform entry unexpected: %+v", platformEntry)
	}
	if timeoutEntry == nil || timeoutEntry.Risk != "non-progress-timeout" || timeoutEntry.SuggestedAction != "quarantine" || !strings.Contains(strings.Join(timeoutEntry.Evidence, " "), "failure_kind=non_progress_loop") {
		t.Fatalf("timeout entry unexpected: %+v", timeoutEntry)
	}
	if fallbackEntry == nil || fallbackEntry.Risk != "equivalence-risk" || fallbackEntry.SuggestedAction != "reviewed-skip" || !strings.Contains(strings.Join(fallbackEntry.Evidence, " "), "equivalent_risk=high") {
		t.Fatalf("fallback entry unexpected: %+v", fallbackEntry)
	}
}

func TestWriteFormatsHonorsConfiguredFormats(t *testing.T) {
	dir := t.TempDir()
	run := engine.RunResult{SchemaVersion: "1", Summary: engine.Summary{Total: 1}}

	if err := WriteFormats(dir, run, []string{"summary", "json"}); err != nil {
		t.Fatalf("WriteFormats returned error: %v", err)
	}
	for _, want := range []string{"summary.txt", "survivors.txt", "mutation-report.json", "semantic-triage-ledger.json", "test-recommendations.md", "governance-review.md", "governance-review.json", "history-dashboard.json", "history-dashboard.html"} {
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
	for _, want := range []string{"summary.txt", "survivors.txt", "mutation-report.json", "semantic-triage-ledger.json", "test-recommendations.md", "governance-review.md", "governance-review.json", "history-dashboard.json", "history-dashboard.html"} {
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

func TestWriteFormatsWithActionableViewWritesExtraArtifact(t *testing.T) {
	dir := t.TempDir()
	run := engine.RunResult{
		Environment: engine.Environment{OS: "windows"},
		Summary:     engine.Summary{Total: 2, Survived: 2},
		Mutants: []engine.MutantResult{
			{MutantID: "keep", Status: engine.StatusSurvived, SurvivorRank: 1, Actionability: "high", TestRecommendation: &engine.TestRecommendation{Summary: "Start with `pkg/a_test.go`", CandidateTests: []string{"pkg/a_test.go"}}, Mutant: engine.Mutant{File: "a.go", Line: 1, Operator: "logical", Original: "&&", Mutated: "||"}},
			{MutantID: "hide", Status: engine.StatusSurvived, SurvivorRank: 2, Actionability: "high", Mutant: engine.Mutant{File: "b.go", Line: 2, Operator: "numeric-literals", Original: "0o755", Mutated: "0", PlatformSensitive: true}},
		},
	}

	if err := WriteFormatsWithOptions(dir, run, []string{"summary"}, WriteOptions{ActionableOnly: true}); err != nil {
		t.Fatalf("WriteFormatsWithOptions returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "semantic-triage-ledger.json")); err != nil {
		t.Fatalf("semantic-triage-ledger.json missing: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "survivors-actionable.txt"))
	if err != nil {
		t.Fatalf("survivors-actionable.txt missing: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "keep") || strings.Contains(text, "hide") {
		t.Fatalf("unexpected actionable-only artifact:\n%s", text)
	}
	recommendations, err := os.ReadFile(filepath.Join(dir, "test-recommendations.md"))
	if err != nil {
		t.Fatalf("test-recommendations.md missing: %v", err)
	}
	if !strings.Contains(string(recommendations), "# CervoMutants Test Recommendations") || !strings.Contains(string(recommendations), "pkg/a_test.go") {
		t.Fatalf("unexpected recommendation artifact:\n%s", recommendations)
	}
	governance, err := os.ReadFile(filepath.Join(dir, "governance-review.md"))
	if err != nil {
		t.Fatalf("governance-review.md missing: %v", err)
	}
	if !strings.Contains(string(governance), "# CervoMutants Governance Review") {
		t.Fatalf("unexpected governance artifact:\n%s", governance)
	}
}

func TestGovernanceReviewExportsTemplatesAndPolicy(t *testing.T) {
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	run := engine.RunResult{
		Quarantine: engine.QuarantineStats{
			Active:        1,
			Expired:       0,
			Path:          ".cervomut/quarantine.json",
			ExpireAfter:   "720h0m0s",
			RequireReason: true,
			RequireOwner:  true,
			RequireIssue:  true,
			FailOnExpired: true,
			MaxRenewals:   1,
		},
		Mutants: []engine.MutantResult{
			{
				MutantID:            "timeout",
				Status:              engine.StatusTimedOut,
				FailureKind:         "non_progress_loop",
				StatusReason:        "loop variable stopped making progress",
				SuggestedSkipReason: "reviewed-skip or quarantine if timeout confirms the loop is non-progress",
				Mutant: engine.Mutant{
					File:            "pkg/loop.go",
					Line:            12,
					Operator:        "inc-dec",
					Original:        "i++",
					Mutated:         "i--",
					NonProgressRisk: "high",
					Ownership:       &engine.OwnershipRoute{Owner: "runtime-owner", Team: "runtime", Rule: "pkg-runtime"},
				},
			},
			{
				MutantID: "suppression",
				Status:   engine.StatusSurvived,
				Mutant: engine.Mutant{
					File:           "pkg/review.go",
					Line:           19,
					Operator:       "conditionals-boundary",
					Original:       "<",
					Mutated:        "<=",
					EquivalentRisk: "high",
					SuppressionAudit: []engine.SuppressionAudit{{
						Name:          "audit-high-equivalent-risk",
						Action:        "report-only",
						Reason:        "High equivalent-mutant risk must be visible before suppression is allowed.",
						EvidenceLevel: "heuristic",
						ReviewerCount: 1,
					}},
				},
			},
		},
	}

	review := buildGovernanceReview(run, now)
	if review.QuarantinePolicy.Path != ".cervomut/quarantine.json" || review.QuarantinePolicy.MaxRenewals != 1 {
		t.Fatalf("quarantine policy missing from governance review: %+v", review.QuarantinePolicy)
	}
	if len(review.QuarantineTemplates) != 1 || review.QuarantineTemplates[0].MutantID != "timeout" {
		t.Fatalf("unexpected quarantine templates: %+v", review.QuarantineTemplates)
	}
	if review.QuarantineTemplates[0].Template.Owner != "runtime-owner" {
		t.Fatalf("ownership route should prefill quarantine owner: %+v", review.QuarantineTemplates[0].Template)
	}
	if review.QuarantineTemplates[0].Template.ExpiresAt != now.Add(30*24*time.Hour).Format(time.RFC3339) {
		t.Fatalf("unexpected suggested quarantine expiry: %+v", review.QuarantineTemplates[0])
	}
	if len(review.SuppressionTemplates) != 1 || review.SuppressionTemplates[0].Rule.Name != "audit-high-equivalent-risk" {
		t.Fatalf("unexpected suppression templates: %+v", review.SuppressionTemplates)
	}

	jsonData, err := GovernanceReviewJSON(run)
	if err != nil {
		t.Fatalf("GovernanceReviewJSON returned error: %v", err)
	}
	for _, want := range []string{`"quarantine_policy"`, `"suppression_templates"`, `"quarantine_templates"`} {
		if !strings.Contains(string(jsonData), want) {
			t.Fatalf("governance json missing %q:\n%s", want, jsonData)
		}
	}
	markdown := GovernanceReviewMarkdown(run)
	for _, want := range []string{"# CervoMutants Governance Review", "## Quarantine Templates", "## Suppression Templates", "timeout", "audit-high-equivalent-risk", "runtime-owner", "ownership_route=owner=runtime-owner team=runtime rule=pkg-runtime"} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("governance markdown missing %q:\n%s", want, markdown)
		}
	}
}

func TestHistoryDashboardOutputs(t *testing.T) {
	run := engine.RunResult{
		History: engine.HistoryStats{
			Enabled: true,
			Path:    ".cervomut/history.json",
			Runs: []engine.HistoryRun{
				{
					RunAt:                   "2026-06-16T10:00:00Z",
					RawScore:                72.5,
					ActionableScore:         80,
					Survived:                8,
					TrueActionableSurvivors: 5,
					NewSurvivors:            2,
					LongStandingSurvivors:   1,
					SurvivorAgeNew:          2,
					SurvivorAgeAging:        3,
					SurvivorAgeLongStanding: 1,
					TimedOut:                2,
					NonProgressTimeouts:     1,
					OperatorUsefulSurvivor: map[string]float64{
						"logical":               0.20,
						"conditionals-boundary": 0.50,
					},
				},
				{
					RunAt:                   "2026-06-17T10:00:00Z",
					RawScore:                78,
					ActionableScore:         84.5,
					Survived:                6,
					TrueActionableSurvivors: 4,
					NewSurvivors:            1,
					LongStandingSurvivors:   2,
					SurvivorAgeNew:          1,
					SurvivorAgeAging:        2,
					SurvivorAgeLongStanding: 2,
					TimedOut:                1,
					NonProgressTimeouts:     0,
					OperatorUsefulSurvivor: map[string]float64{
						"logical":               0.10,
						"conditionals-boundary": 0.60,
					},
				},
			},
		},
	}

	jsonData, err := HistoryDashboardJSON(run)
	if err != nil {
		t.Fatalf("HistoryDashboardJSON returned error: %v", err)
	}
	for _, want := range []string{`"run_count": 2`, `"raw_score": 78`, `"actionable_score": 84.5`, `"survivor_age_long_standing": 2`} {
		if !strings.Contains(string(jsonData), want) {
			t.Fatalf("history dashboard json missing %q:\n%s", want, jsonData)
		}
	}

	text := HistorySummary(run)
	for _, want := range []string{"Historical runs: 2", "Latest raw score: 78.00%", "Delta vs previous: raw=+5.50 actionable=+4.50 survived=-2"} {
		if !strings.Contains(text, want) {
			t.Fatalf("history summary missing %q:\n%s", want, text)
		}
	}

	html := HistoryDashboardHTML(run)
	for _, want := range []string{"cervomut history dashboard", "Historical runs", "Latest actionable score", "Latest operator yield", "conditionals-boundary", "2026-06-17T10:00:00Z"} {
		if !strings.Contains(html, want) {
			t.Fatalf("history dashboard html missing %q:\n%s", want, html)
		}
	}
}

func TestSARIFAndGitHubSummaryOutputs(t *testing.T) {
	dir := t.TempDir()
	workingDir := filepath.Join(dir, "repo")
	stepSummaryPath := filepath.Join(dir, "step-summary.md")
	t.Setenv("GITHUB_STEP_SUMMARY", stepSummaryPath)
	run := engine.RunResult{
		SchemaVersion: "1",
		Summary: engine.Summary{
			Total:                      4,
			Killed:                     1,
			Survived:                   1,
			NotCovered:                 1,
			TimedOut:                   1,
			Score:                      50,
			Actionable:                 engine.ActionableSummary{ActionableScore: 66.67, TrueActionableSurvivors: 1},
			PlatformSensitiveSurvivors: 1,
			NonProgressTimeouts:        1,
			DenominatorHealth: engine.DenominatorHealth{
				Warnings: []string{"timed_out_exceeds_effective"},
			},
		},
		Environment: engine.Environment{
			WorkingDir:  workingDir,
			ToolVersion: "v0.3.0",
		},
		Baseline: engine.BaselineComparison{
			Enabled:      true,
			Regression:   true,
			NewSurvivors: []string{"m-survived"},
		},
		Mutants: []engine.MutantResult{
			{
				MutantID:            "m-survived",
				Status:              engine.StatusSurvived,
				StatusReason:        "tests passed",
				Actionability:       "high",
				SurvivorRank:        1,
				SuggestedTestScope:  "./pkg",
				TestRecommendation:  &engine.TestRecommendation{Summary: "Start with `pkg/foo_test.go`", Strategy: "tighten-branch-assertions", CandidateTests: []string{"pkg/foo_test.go"}},
				SuggestedSkipReason: "review once",
				Mutant: engine.Mutant{
					File:           filepath.Join(workingDir, "pkg", "foo.go"),
					Line:           12,
					Operator:       "conditionals-boundary",
					Original:       "<",
					Mutated:        "<=",
					EquivalentRisk: "high",
					Ownership:      &engine.OwnershipRoute{Owner: "pkg-owner", Team: "platform", Contact: "@platform", Rule: "pkg-platform"},
				},
			},
			{
				MutantID:     "m-timeout",
				Status:       engine.StatusTimedOut,
				FailureKind:  "non_progress_loop",
				StatusReason: "loop variable stopped making progress",
				Mutant: engine.Mutant{
					File:            filepath.Join(workingDir, "pkg", "loop.go"),
					Line:            21,
					Operator:        "inc-dec",
					Original:        "i++",
					Mutated:         "i--",
					NonProgressRisk: "high",
				},
			},
			{
				MutantID: "m-uncovered",
				Status:   engine.StatusNotCovered,
				Mutant: engine.Mutant{
					File:     filepath.Join(workingDir, "pkg", "miss.go"),
					Line:     7,
					Operator: "logical",
					Original: "&&",
					Mutated:  "||",
				},
			},
		},
	}

	sarifData, err := SARIF(run)
	if err != nil {
		t.Fatalf("SARIF returned error: %v", err)
	}
	text := string(sarifData)
	for _, want := range []string{
		`"version": "2.1.0"`,
		`"ruleId": "survived"`,
		`"ruleId": "timed_out.non_progress_loop"`,
		`"ruleId": "not_covered"`,
		`"uri": "pkg/foo.go"`,
		`"mutant_id": "m-survived"`,
		`"recommended_test": "pkg/foo_test.go"`,
		`"owner": "pkg-owner"`,
		`"team": "platform"`,
		`"contact": "@platform"`,
		`"ownership_rule": "pkg-platform"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("sarif output missing %q:\n%s", want, text)
		}
	}

	summary := GitHubSummary(run)
	for _, want := range []string{
		"## CervoMutants Mutation Summary",
		"Raw score: **50.00%**",
		"Actionable score: **66.67%**",
		"Lane shape: **direct review lane**",
		"Lane guidance: start with the top survivor queue and nearby-test hints before widening the target or changing policy depth",
		"Baseline regression: **true**",
		"| Rank | Mutant | Actionability | Owner route | Operator | Location | Next test | Skip guidance |",
		"`m-survived`",
		"`owner=pkg-owner team=platform contact=@platform rule=pkg-platform`",
		"pkg/foo_test.go: Start with `pkg/foo_test.go`",
		"| timed out | 1 |",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("github summary missing %q:\n%s", want, summary)
		}
	}

	if err := WriteFormats(dir, run, []string{"sarif", "github-summary"}); err != nil {
		t.Fatalf("WriteFormats with GitHub-native outputs returned error: %v", err)
	}
	for _, want := range []string{"mutation-report.sarif", "github-summary.md", "semantic-triage-ledger.json"} {
		if _, err := os.Stat(filepath.Join(dir, want)); err != nil {
			t.Fatalf("missing %s: %v", want, err)
		}
	}
	stepSummary, err := os.ReadFile(stepSummaryPath)
	if err != nil {
		t.Fatalf("step summary missing: %v", err)
	}
	if string(stepSummary) != summary {
		t.Fatalf("step summary mismatch:\nwant:\n%s\n\ngot:\n%s", summary, stepSummary)
	}
}

func TestGitHubSummaryIncludesDenominatorGuidanceForLowSignalRuns(t *testing.T) {
	run := engine.RunResult{
		Summary: engine.Summary{
			Actionable: engine.ActionableSummary{},
			DenominatorHealth: engine.DenominatorHealth{
				Healthy:  false,
				Warnings: []string{"no_effective_mutants", "score_denominator_dwarfs_effective"},
			},
		},
	}

	summary := GitHubSummary(run)
	for _, want := range []string{
		"- Lane shape: **retargeting signal**",
		"- Lane note: the run completed, but denominator pressure dominates and this bounded slice did not produce immediate review work",
		"- Lane guidance: keep the artifact and retarget the next run to a hotter package, subtree, or shard before judging broader rollout fit",
		"- Denominator warnings: `no_effective_mutants`, `score_denominator_dwarfs_effective`",
		"- Guidance: Preserve this report and treat the run as target-selection feedback before changing score expectations.",
		"- Guidance: Retarget the next run to a hotter package, subtree, or bounded shard before widening to ./....",
		"- Guidance: Rerun on the narrower target before judging recommendation quality or broader rollout fit.",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("github summary missing %q:\n%s", want, summary)
		}
	}
}

func TestRenderRequestedFormatsBuildsGitHubSummaryWithoutSinkEffects(t *testing.T) {
	stepSummaryPath := filepath.Join(t.TempDir(), "step-summary.md")
	t.Setenv("GITHUB_STEP_SUMMARY", stepSummaryPath)

	run := goldenReportFixture()
	plan, err := renderRequestedFormats(run, []string{"github-summary"}, WriteOptions{})
	if err != nil {
		t.Fatalf("renderRequestedFormats returned error: %v", err)
	}
	if got := string(plan.artifacts["github-summary.md"]); got != GitHubSummary(run) {
		t.Fatalf("github summary artifact mismatch:\nwant:\n%s\n\ngot:\n%s", GitHubSummary(run), got)
	}
	if len(plan.sinks) != 1 {
		t.Fatalf("post-write sinks = %d, want 1", len(plan.sinks))
	}
	if _, err := os.Stat(stepSummaryPath); !os.IsNotExist(err) {
		t.Fatalf("renderRequestedFormats should not write step summary, stat err=%v", err)
	}
}

func TestRunPostWriteSinksPublishesGitHubSummary(t *testing.T) {
	stepSummaryPath := filepath.Join(t.TempDir(), "step-summary.md")
	t.Setenv("GITHUB_STEP_SUMMARY", stepSummaryPath)

	run := goldenReportFixture()
	plan, err := renderRequestedFormats(run, []string{"github-summary"}, WriteOptions{})
	if err != nil {
		t.Fatalf("renderRequestedFormats returned error: %v", err)
	}
	if err := runPostWriteSinks(plan.sinks); err != nil {
		t.Fatalf("runPostWriteSinks returned error: %v", err)
	}
	data, err := os.ReadFile(stepSummaryPath)
	if err != nil {
		t.Fatalf("step summary missing: %v", err)
	}
	if got, want := string(data), string(plan.artifacts["github-summary.md"]); got != want {
		t.Fatalf("step summary mismatch:\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}

func TestGitHubSummaryClassifiesGroupedAndHealthyNoActionLanes(t *testing.T) {
	t.Run("grouped review lane", func(t *testing.T) {
		run := engine.RunResult{
			Summary: engine.Summary{
				Survived: 3,
				Actionable: engine.ActionableSummary{
					ActionableScore:             77.78,
					TrueActionableSurvivors:     2,
					SemanticGroupReviewUnits:    1,
					CollapsedSemanticDuplicates: 1,
				},
				DenominatorHealth: engine.DenominatorHealth{Healthy: true},
			},
		}

		summary := GitHubSummary(run)
		for _, want := range []string{
			"- Semantic review units: **1** (1 collapsed duplicates)",
			"- Lane shape: **grouped review lane**",
			"- Lane note: 3 raw survivors collapsed into 2 immediate review units across 1 semantic group",
			"- Lane guidance: review the grouped equivalent-risk family once before splitting it into multiple separate test tasks",
		} {
			if !strings.Contains(summary, want) {
				t.Fatalf("grouped-review summary missing %q:\n%s", want, summary)
			}
		}
	})

	t.Run("healthy no-action lane", func(t *testing.T) {
		run := engine.RunResult{
			Summary: engine.Summary{
				Survived: 0,
				Actionable: engine.ActionableSummary{
					ActionableScore:         100,
					TrueActionableSurvivors: 0,
				},
				DenominatorHealth: engine.DenominatorHealth{
					Healthy: true,
				},
			},
		}

		summary := GitHubSummary(run)
		for _, want := range []string{
			"- Lane shape: **healthy no-action lane**",
			"- Lane note: this bounded slice produced understandable denominator health and no immediate survivor work",
			"- Lane guidance: keep the artifact; widen or retarget only if you need more review pressure from a different slice",
		} {
			if !strings.Contains(summary, want) {
				t.Fatalf("healthy-no-action summary missing %q:\n%s", want, summary)
			}
		}
	})
}

func TestJUnitHTMLAndWriteAll(t *testing.T) {
	dir := t.TempDir()
	run := engine.RunResult{
		SchemaVersion: "1",
		Summary: engine.Summary{
			Total:                 3,
			Killed:                1,
			Survived:              2,
			Score:                 50,
			LongStandingSurvivors: 1,
			Actionable: engine.ActionableSummary{
				RawScore:                50,
				ActionableScore:         33.33,
				Survivors:               2,
				ActionableSurvivors:     2,
				TrueActionableSurvivors: 2,
			},
		},
		Mutants: []engine.MutantResult{
			{MutantID: "killed", Status: engine.StatusKilled, Mutant: engine.Mutant{Diff: "-a\n+b\n"}},
			{
				MutantID:           "survived",
				Status:             engine.StatusSurvived,
				StatusReason:       "tests passed",
				Duration:           3 * time.Second,
				SurvivorRank:       1,
				Actionability:      "high",
				SurvivorAgeRuns:    6,
				HistoryStatus:      "long_standing_survivor",
				SuggestedTestScope: "./pkg",
				TestRecommendation: &engine.TestRecommendation{Summary: "Promote `pkg/foo_test.go` into a named regression", Strategy: "tighten-branch-assertions", CandidateTests: []string{"pkg/foo_test.go"}, SuggestedAssertions: []string{"Add a strict ordering assertion."}},
				NearestTests:       []string{"pkg/foo_test.go"},
				RankReason:         "risk=medium recommendation=fast-ci",
				Mutant: engine.Mutant{
					File:           "pkg/foo.go",
					Line:           10,
					Function:       "Check",
					Operator:       "conditionals-boundary",
					Original:       "<",
					Mutated:        "<=",
					EquivalentRisk: "high",
					GroupLabel:     "sort comparator boundary",
					Diff:           "<unsafe>",
					Description:    "Changed < to <= in Check.",
					Ownership:      &engine.OwnershipRoute{Owner: "pkg-owner", Team: "platform", Rule: "pkg-platform"},
				},
			},
			{
				MutantID:      "survived-2",
				Status:        engine.StatusSurvived,
				Duration:      250 * time.Millisecond,
				SurvivorRank:  2,
				Actionability: "medium",
				Mutant: engine.Mutant{
					File:           "pkg/bar.go",
					Line:           11,
					Operator:       "logical",
					Original:       "&&",
					Mutated:        "||",
					EquivalentRisk: "medium",
					Diff:           "-old\n+new\n",
				},
			},
		},
	}

	junit, err := JUnit(run)
	if err != nil {
		t.Fatalf("JUnit returned error: %v", err)
	}
	if !strings.Contains(string(junit), `tests="3"`) || !strings.Contains(string(junit), `failures="2"`) {
		t.Fatalf("unexpected junit: %s", junit)
	}
	html := HTML(run)
	for _, want := range []string{
		"cervomut survivor review workbench",
		`id="filter-search"`,
		`id="filter-actionability"`,
		`id="filter-group"`,
		`id="filter-owner"`,
		`id="filter-team"`,
		`id="filter-history"`,
		`id="filter-age"`,
		`id="filter-timing"`,
		"Group shortcuts",
		"Operator shortcuts",
		`data-mutant-row`,
		`data-survivor="true"`,
		`data-actionability="high"`,
		`data-group="sort comparator boundary"`,
		`data-owner="pkg-owner"`,
		`data-team="platform"`,
		"Actionable score",
		"True actionable survivors",
		"long-standing (5+ runs)",
		"slow (&gt;2s)",
		"next_test=pkg/foo_test.go",
		"test_strategy=tighten-branch-assertions",
		"ownership=owner=pkg-owner team=platform rule=pkg-platform",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("html workbench missing %q:\n%s", want, html)
		}
	}
	if strings.Contains(html, "<unsafe>") {
		t.Fatalf("html should escape diffs: %s", html)
	}
	if err := WriteAll(dir, run); err != nil {
		t.Fatalf("WriteAll returned error: %v", err)
	}
	for _, want := range []string{"summary.txt", "survivors.txt", "mutation-report.json", "semantic-triage-ledger.json", "test-recommendations.md", "junit.xml", "index.html", "mutation-report.sarif", "github-summary.md", "history-dashboard.json", "history-dashboard.html"} {
		if _, err := os.Stat(filepath.Join(dir, want)); err != nil {
			t.Fatalf("missing %s: %v", want, err)
		}
	}
}
