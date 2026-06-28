package report

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"

	"github.com/cervantesh/cervo-mutants/pkg/engine"
	"github.com/cervantesh/cervo-mutants/pkg/triage"
)

type SurvivorsOptions struct {
	ActionableOnly bool
}

type WriteOptions struct {
	ActionableOnly bool
}

type SurvivorViewStats struct {
	Total          int
	Shown          int
	Filtered       int
	CollapsedGroup int
}

type TriageLedger struct {
	SchemaVersion string              `json:"schema_version"`
	Entries       []TriageLedgerEntry `json:"entries"`
}

type TriageLedgerEntry struct {
	MutantID        string        `json:"mutant_id"`
	MutantIDs       []string      `json:"mutant_ids,omitempty"`
	Status          engine.Status `json:"status"`
	Risk            string        `json:"risk"`
	SuggestedAction string        `json:"suggested_action"`
	SuggestedReason string        `json:"suggested_reason"`
	Evidence        []string      `json:"evidence"`
	GroupKey        string        `json:"group_key,omitempty"`
	GroupLabel      string        `json:"group_label,omitempty"`
	GroupSize       int           `json:"group_size,omitempty"`
	Actionability   string        `json:"actionability,omitempty"`
}

func JSON(result engine.RunResult) ([]byte, error) {
	if result.SchemaVersion == "" {
		result.SchemaVersion = "1"
	}
	if result.Thresholds == nil {
		result.Thresholds = map[string]any{}
	}
	return json.MarshalIndent(result, "", "  ")
}

func Summary(result engine.RunResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Effective mutation score: %.2f%%\nRaw mutation score: %.2f%%\nGenerated mutants: %d\nCovered mutants: %d\nExecuted mutants: %d\nEffective mutants: %d\nScore denominator: %d\nKilled: %d\nSurvived: %d\nNot covered: %d\nPending budget: %d\nSkipped resource: %d\nQuarantined: %d\nTimed out: %d\nMemory killed: %d\nCompile errors: %d\nTest efficacy: %.2f%%\nMutation coverage: %.2f%%\nActionable score: %.2f%%\nActionable survivors: %d\nTrue actionable survivors: %d\nEquivalent-risk survivors: %d\nSemantic review units: %d\nCollapsed semantic duplicates: %d\nHigh-risk survivors: %d\nNew survivors: %d\nLong-standing survivors: %d\nPlatform-sensitive survivors: %d\nNon-progress timeouts: %d\nSuppression audits: report_only=%d lower_priority=%d suppress=%d quarantine_required=%d\n",
		result.Summary.EffectiveScore,
		result.Summary.Score,
		result.Summary.GeneratedMutants,
		result.Summary.CoveredMutants,
		result.Summary.ExecutedMutants,
		result.Summary.EffectiveMutants,
		result.Summary.ScoreDenominator,
		result.Summary.Killed,
		result.Summary.Survived,
		result.Summary.NotCovered,
		result.Summary.PendingBudget,
		result.Summary.SkippedResource,
		result.Summary.Quarantined,
		result.Summary.TimedOut,
		result.Summary.MemoryKilled,
		result.Summary.CompileError,
		result.Summary.TestEfficacy,
		result.Summary.MutationCoverage,
		result.Summary.Actionable.ActionableScore,
		result.Summary.Actionable.ActionableSurvivors,
		result.Summary.Actionable.TrueActionableSurvivors,
		result.Summary.Actionable.EquivalentRiskSurvivors,
		result.Summary.Actionable.SemanticGroupReviewUnits,
		result.Summary.Actionable.CollapsedSemanticDuplicates,
		result.Summary.HighRiskSurvivors,
		result.Summary.NewSurvivors,
		result.Summary.LongStandingSurvivors,
		result.Summary.PlatformSensitiveSurvivors,
		result.Summary.NonProgressTimeouts,
		result.Summary.SuppressionReportOnly,
		result.Summary.SuppressionLowerPriority,
		result.Summary.SuppressionSuppressed,
		result.Summary.SuppressionQuarantineRequired,
	)
	if lane, ok := reportLaneInterpretation(result); ok {
		fmt.Fprintf(&b, "Lane shape: %s\nLane note: %s\nLane guidance: %s\n",
			lane.label,
			lane.detail,
			lane.guidance,
		)
	}
	fmt.Fprintf(&b, "Gate: %s\n", gateStatusLabel(result.Gate))
	if failed := gateChecksByStatus(result.Gate, engine.GateCheckFailed); len(failed) > 0 {
		fmt.Fprintf(&b, "Gate failures: %s\n", gateCheckSummaries(failed))
	}
	if skipped := gateChecksByStatus(result.Gate, engine.GateCheckSkipped); len(skipped) > 0 {
		fmt.Fprintf(&b, "Gate skips: %s\n", gateCheckSummaries(skipped))
	}
	if result.Summary.DenominatorHealth.Generated > 0 || len(result.Summary.DenominatorHealth.Warnings) > 0 {
		health := result.Summary.DenominatorHealth
		fmt.Fprintf(&b, "Denominator health: healthy=%t generated=%d covered=%d executed=%d effective=%d score_denominator=%d killed=%d survived=%d not_covered=%d pending_budget=%d skipped_resource=%d timed_out=%d memory_killed=%d compile_error=%d\n",
			health.Healthy,
			health.Generated,
			health.Covered,
			health.Executed,
			health.Effective,
			health.ScoreDenominator,
			health.Killed,
			health.Survived,
			health.NotCovered,
			health.PendingBudget,
			health.SkippedResource,
			health.TimedOut,
			health.MemoryKilled,
			health.CompileError,
		)
		if len(health.Warnings) > 0 {
			fmt.Fprintf(&b, "Denominator warnings: %s\n", strings.Join(health.Warnings, ", "))
			if guidance := denominatorGuidanceLines(health); len(guidance) > 0 {
				b.WriteString("Denominator guidance:\n")
				for _, line := range guidance {
					fmt.Fprintf(&b, "- %s\n", line)
				}
			}
		}
	}
	if len(result.Summary.EquivalentRiskStats) > 0 {
		b.WriteString("Equivalent-risk statistics:\n")
		keys := make([]string, 0, len(result.Summary.EquivalentRiskStats))
		for key := range result.Summary.EquivalentRiskStats {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Fprintf(&b, "- %s: %d\n", key, result.Summary.EquivalentRiskStats[key])
		}
	}
	if len(result.Summary.MutatorStats) > 0 {
		b.WriteString("Mutator statistics:\n")
		keys := make([]string, 0, len(result.Summary.MutatorStats))
		for key := range result.Summary.MutatorStats {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			stat := result.Summary.MutatorStats[key]
			fmt.Fprintf(&b, "- %s: total=%d killed=%d survived=%d not_covered=%d pending_budget=%d skipped_resource=%d timed_out=%d memory_killed=%d compile_error=%d recommendation=%s equivalent_risk=%s\n",
				key,
				stat.Total,
				stat.Killed,
				stat.Survived,
				stat.NotCovered,
				stat.PendingBudget,
				stat.SkippedResource,
				stat.TimedOut,
				stat.MemoryKilled,
				stat.CompileError,
				stat.Recommendation,
				stat.EquivalentRisk,
			)
		}
	}
	if len(result.Summary.SemanticGroupStats) > 0 {
		b.WriteString("Semantic-group statistics:\n")
		keys := make([]string, 0, len(result.Summary.SemanticGroupStats))
		for key := range result.Summary.SemanticGroupStats {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Fprintf(&b, "- %s: %d\n", key, result.Summary.SemanticGroupStats[key])
		}
	}
	if result.Slice.Enabled {
		fmt.Fprintf(&b, "Slice: by=%s shard=%d/%d groups=%d selected_groups=%d files=%d max_files=%d max_mutants_per_package=%d selected_mutants=%d\n",
			result.Slice.SliceBy,
			result.Slice.ShardIndex,
			result.Slice.ShardCount,
			result.Slice.GroupCount,
			result.Slice.SelectedGroups,
			result.Slice.SelectedFiles,
			result.Slice.MaxFilesPerRun,
			result.Slice.MaxMutantsPerPackage,
			result.Slice.SelectedMutants,
		)
	}
	if result.StoppedReason != "" {
		fmt.Fprintf(&b, "Stopped reason: %s\n", result.StoppedReason)
	}
	if result.LastCompletedMutant != "" {
		fmt.Fprintf(&b, "Last completed mutant: %s\n", result.LastCompletedMutant)
	}
	if result.Environment.OS != "" {
		fmt.Fprintf(&b, "Environment: os=%s arch=%s go=%s isolation=%s workers=%d timeout=%s wsl=%t onedrive=%t\n",
			result.Environment.OS,
			result.Environment.Arch,
			result.Environment.GoVersion,
			result.Environment.Isolation,
			result.Environment.Workers,
			result.Environment.TestTimeout,
			result.Environment.WSL,
			result.Environment.WindowsOneDrive,
		)
		if result.Environment.TempRoot != "" {
			fmt.Fprintf(&b, "Temp root: %s\n", result.Environment.TempRoot)
		}
		if len(result.Environment.Warnings) > 0 {
			fmt.Fprintf(&b, "Environment warnings: %s\n", strings.Join(result.Environment.Warnings, ", "))
		}
	}
	return b.String()
}

func Survivors(result engine.RunResult) string {
	return SurvivorsWithOptions(result, SurvivorsOptions{})
}

func SurvivorsWithOptions(result engine.RunResult, opts SurvivorsOptions) string {
	var b strings.Builder
	survivors := rankedSurvivors(result.Mutants)
	visible, stats := filterSurvivors(result, survivors, opts)
	if opts.ActionableOnly {
		fmt.Fprintf(&b, "Actionable-only view: showing %d of %d survivors (filtered=%d collapsed=%d)\n", stats.Shown, stats.Total, stats.Filtered, stats.CollapsedGroup)
	}
	seenGroups := map[string]bool{}
	for _, mutant := range visible {
		groupSize := mutant.SemanticGroupSize
		if groupSize <= 0 {
			groupSize = 1
		}
		if group := mutant.Mutant.SemanticGroup; group != "" && groupSize > 1 && !seenGroups[group] {
			label := mutant.Mutant.GroupLabel
			if label == "" {
				label = group
			}
			fmt.Fprintf(&b, "Group %s (%d mutants): %s\n", label, groupSize, mutant.Mutant.GroupReason)
			seenGroups[group] = true
		}
		line := fmt.Sprintf("#%d %.1f %s %s:%d %s %s -> %s actionability=%s scope=%s next_test=%s strategy=%s group=%s group_size=%d platform_sensitive=%t skip=%s (%s)",
			mutant.SurvivorRank,
			mutant.RankScore,
			mutant.MutantID,
			mutant.Mutant.File,
			mutant.Mutant.Line,
			mutant.Mutant.Operator,
			mutant.Mutant.Original,
			mutant.Mutant.Mutated,
			mutant.Actionability,
			mutant.SuggestedTestScope,
			recommendationPrimaryTest(mutant.TestRecommendation),
			recommendationStrategy(mutant.TestRecommendation),
			mutant.Mutant.GroupLabel,
			mutant.SemanticGroupSize,
			mutant.Mutant.PlatformSensitive,
			mutant.SuggestedSkipReason,
			mutant.RankReason,
		)
		if ownership := ownershipRouteSummary(mutant.Mutant.Ownership); ownership != "" {
			line += " ownership=" + ownership
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func rankedSurvivors(mutants []engine.MutantResult) []engine.MutantResult {
	survivors := make([]engine.MutantResult, 0)
	for _, item := range mutants {
		if item.Status != engine.StatusSurvived {
			continue
		}
		survivors = append(survivors, item)
	}
	sort.SliceStable(survivors, func(i, j int) bool {
		if survivors[i].SurvivorRank == 0 {
			return false
		}
		if survivors[j].SurvivorRank == 0 {
			return true
		}
		return survivors[i].SurvivorRank < survivors[j].SurvivorRank
	})
	return survivors
}

func filterSurvivors(result engine.RunResult, survivors []engine.MutantResult, opts SurvivorsOptions) ([]engine.MutantResult, SurvivorViewStats) {
	stats := SurvivorViewStats{Total: len(survivors)}
	if !opts.ActionableOnly {
		stats.Shown = len(survivors)
		return survivors, stats
	}
	filtered := make([]engine.MutantResult, 0, len(survivors))
	seenGroups := map[string]bool{}
	for _, survivor := range survivors {
		if group := survivor.Mutant.SemanticGroup; group != "" {
			if seenGroups[group] {
				stats.CollapsedGroup++
				continue
			}
			seenGroups[group] = true
		}
		if !isActionableSurvivor(result.Environment.OS, survivor) {
			stats.Filtered++
			continue
		}
		filtered = append(filtered, survivor)
	}
	stats.Shown = len(filtered)
	return filtered, stats
}

func isActionableSurvivor(goos string, survivor engine.MutantResult) bool {
	return triage.IsActionableSurvivor(goos, triage.Result{
		MutantID:      survivor.MutantID,
		Status:        string(survivor.Status),
		Actionability: survivor.Actionability,
		Mutant: triage.Mutant{
			PlatformSensitive: survivor.Mutant.PlatformSensitive,
			NonProgressRisk:   survivor.Mutant.NonProgressRisk,
		},
	})
}

func SemanticTriageLedger(result engine.RunResult) ([]byte, error) {
	ledger := buildTriageLedger(result)
	return json.MarshalIndent(ledger, "", "  ")
}

func buildTriageLedger(result engine.RunResult) TriageLedger {
	entries := make([]TriageLedgerEntry, 0)
	groupedMutants := map[string]bool{}
	survivors := rankedSurvivors(result.Mutants)
	groupOrder := make([]string, 0)
	grouped := map[string][]engine.MutantResult{}

	for _, survivor := range survivors {
		group := survivor.Mutant.SemanticGroup
		if group == "" {
			continue
		}
		if len(grouped[group]) == 0 {
			groupOrder = append(groupOrder, group)
		}
		grouped[group] = append(grouped[group], survivor)
		groupedMutants[survivor.MutantID] = true
	}

	for _, group := range groupOrder {
		mutants := grouped[group]
		representative := mutants[0]
		entries = append(entries, TriageLedgerEntry{
			MutantID:        representative.MutantID,
			MutantIDs:       ledgerMutantIDs(mutants),
			Status:          representative.Status,
			Risk:            "equivalence-risk",
			SuggestedAction: "reviewed-skip",
			SuggestedReason: ledgerSuggestedReason(representative, "review once for this semantic group before treating each survivor independently"),
			Evidence:        semanticGroupEvidence(mutants),
			GroupKey:        group,
			GroupLabel:      representative.Mutant.GroupLabel,
			GroupSize:       len(mutants),
			Actionability:   representative.Actionability,
		})
	}

	for _, mutant := range result.Mutants {
		switch {
		case groupedMutants[mutant.MutantID]:
			continue
		case strings.EqualFold(result.Environment.OS, "windows") && mutant.Status == engine.StatusSurvived && mutant.Mutant.PlatformSensitive:
			entries = append(entries, TriageLedgerEntry{
				MutantID:        mutant.MutantID,
				MutantIDs:       []string{mutant.MutantID},
				Status:          mutant.Status,
				Risk:            "platform-sensitive",
				SuggestedAction: "reviewed-skip",
				SuggestedReason: ledgerSuggestedReason(mutant, "review on the target platform before treating permission-mode survivors as actionable"),
				Evidence:        platformSensitiveEvidence(result.Environment.OS, mutant),
				GroupKey:        "platform-sensitive:" + mutant.MutantID,
				GroupLabel:      "platform-sensitive survivor",
				GroupSize:       1,
				Actionability:   mutant.Actionability,
			})
		case mutant.Status == engine.StatusTimedOut && mutant.FailureKind == "non_progress_loop":
			entries = append(entries, TriageLedgerEntry{
				MutantID:        mutant.MutantID,
				MutantIDs:       []string{mutant.MutantID},
				Status:          mutant.Status,
				Risk:            "non-progress-timeout",
				SuggestedAction: "quarantine",
				SuggestedReason: ledgerSuggestedReason(mutant, "quarantine after review if the timeout confirms a non-progress loop"),
				Evidence:        nonProgressTimeoutEvidence(mutant),
				GroupKey:        "non-progress-timeout:" + mutant.MutantID,
				GroupLabel:      "non-progress loop timeout",
				GroupSize:       1,
				Actionability:   mutant.Actionability,
			})
		case mutant.Status == engine.StatusSurvived && strings.EqualFold(mutant.Mutant.EquivalentRisk, "high"):
			entries = append(entries, TriageLedgerEntry{
				MutantID:        mutant.MutantID,
				MutantIDs:       []string{mutant.MutantID},
				Status:          mutant.Status,
				Risk:            "equivalence-risk",
				SuggestedAction: "reviewed-skip",
				SuggestedReason: ledgerSuggestedReason(mutant, "reviewed-skip after confirming the high equivalent-risk pattern"),
				Evidence:        highEquivalentRiskEvidence(mutant),
				GroupKey:        "equivalence-risk:" + mutant.MutantID,
				GroupLabel:      "high equivalent-risk survivor",
				GroupSize:       1,
				Actionability:   mutant.Actionability,
			})
		}
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Risk != entries[j].Risk {
			return entries[i].Risk < entries[j].Risk
		}
		if entries[i].GroupSize != entries[j].GroupSize {
			return entries[i].GroupSize > entries[j].GroupSize
		}
		return entries[i].MutantID < entries[j].MutantID
	})

	return TriageLedger{
		SchemaVersion: "1",
		Entries:       entries,
	}
}

func ledgerMutantIDs(mutants []engine.MutantResult) []string {
	ids := make([]string, 0, len(mutants))
	for _, mutant := range mutants {
		ids = append(ids, mutant.MutantID)
	}
	return ids
}

func ledgerSuggestedReason(mutant engine.MutantResult, fallback string) string {
	if mutant.SuggestedSkipReason != "" {
		return mutant.SuggestedSkipReason
	}
	if mutant.Mutant.SuggestedSkipReason != "" {
		return mutant.Mutant.SuggestedSkipReason
	}
	return fallback
}

func semanticGroupEvidence(mutants []engine.MutantResult) []string {
	representative := mutants[0]
	evidence := []string{
		fmt.Sprintf("semantic_group=%s", representative.Mutant.SemanticGroup),
		fmt.Sprintf("group_size=%d", len(mutants)),
	}
	if representative.Mutant.GroupLabel != "" {
		evidence = append(evidence, "group_label="+representative.Mutant.GroupLabel)
	}
	if representative.Mutant.GroupReason != "" {
		evidence = append(evidence, "group_reason="+representative.Mutant.GroupReason)
	}
	if representative.Mutant.EquivalentRisk != "" {
		evidence = append(evidence, "equivalent_risk="+representative.Mutant.EquivalentRisk)
	}
	if len(representative.Mutant.SemanticTags) > 0 {
		evidence = append(evidence, "semantic_tags="+strings.Join(representative.Mutant.SemanticTags, ","))
	}
	return evidence
}

func platformSensitiveEvidence(goos string, mutant engine.MutantResult) []string {
	evidence := []string{
		"goos=" + goos,
		"platform_sensitive=true",
		"operator=" + mutant.Mutant.Operator,
	}
	if mutant.Mutant.EquivalentRisk != "" {
		evidence = append(evidence, "equivalent_risk="+mutant.Mutant.EquivalentRisk)
	}
	return evidence
}

func nonProgressTimeoutEvidence(mutant engine.MutantResult) []string {
	evidence := []string{
		"failure_kind=non_progress_loop",
		"status=timed_out",
	}
	if mutant.Mutant.NonProgressRisk != "" {
		evidence = append(evidence, "non_progress_risk="+mutant.Mutant.NonProgressRisk)
	}
	if mutant.StatusReason != "" {
		evidence = append(evidence, "status_reason="+mutant.StatusReason)
	}
	return evidence
}

func highEquivalentRiskEvidence(mutant engine.MutantResult) []string {
	evidence := []string{
		"equivalent_risk=high",
		"operator=" + mutant.Mutant.Operator,
	}
	if len(mutant.Mutant.SemanticTags) > 0 {
		evidence = append(evidence, "semantic_tags="+strings.Join(mutant.Mutant.SemanticTags, ","))
	}
	if mutant.Mutant.GroupReason != "" {
		evidence = append(evidence, "group_reason="+mutant.Mutant.GroupReason)
	}
	return evidence
}

type junitTestsuite struct {
	XMLName  xml.Name        `xml:"testsuite"`
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Cases    []junitTestcase `xml:"testcase"`
}

type junitTestcase struct {
	Name    string        `xml:"name,attr"`
	Failure *junitFailure `xml:"failure,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Text    string `xml:",chardata"`
}

func JUnit(result engine.RunResult) ([]byte, error) {
	suite := junitTestsuite{Name: "cervomut", Tests: len(result.Mutants)}
	for _, mutant := range result.Mutants {
		tc := junitTestcase{Name: mutant.MutantID}
		if mutant.Status == engine.StatusSurvived || mutant.Status == engine.StatusTimedOut || mutant.Status == engine.StatusMemoryKilled || mutant.Status == engine.StatusPendingBudget || mutant.Status == engine.StatusSkippedResource || mutant.Status == engine.StatusNotCovered {
			suite.Failures++
			tc.Failure = &junitFailure{Message: string(mutant.Status), Text: mutant.StatusReason}
		}
		suite.Cases = append(suite.Cases, tc)
	}
	return xml.MarshalIndent(suite, "", "  ")
}
