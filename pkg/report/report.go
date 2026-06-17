package report

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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
		fmt.Fprintf(&b, "#%d %.1f %s %s:%d %s %s -> %s actionability=%s scope=%s group=%s group_size=%d platform_sensitive=%t skip=%s (%s)\n", mutant.SurvivorRank, mutant.RankScore, mutant.MutantID, mutant.Mutant.File, mutant.Mutant.Line, mutant.Mutant.Operator, mutant.Mutant.Original, mutant.Mutant.Mutated, mutant.Actionability, mutant.SuggestedTestScope, mutant.Mutant.GroupLabel, mutant.SemanticGroupSize, mutant.Mutant.PlatformSensitive, mutant.SuggestedSkipReason, mutant.RankReason)
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

type htmlReportRow struct {
	MutantID          string
	Status            string
	Survivor          bool
	SurvivorRank      int
	Actionability     string
	Operator          string
	EquivalentRisk    string
	GroupFilter       string
	GroupLabel        string
	HistoryStatus     string
	AgeBand           string
	AgeLabel          string
	TimingBand        string
	TimingLabel       string
	DurationText      string
	File              string
	Line              int
	Function          string
	Original          string
	Mutated           string
	Description       string
	FailureKind       string
	StatusReason      string
	SuggestedSkip     string
	SuggestedScope    string
	NearestTests      string
	RankReason        string
	Diff              string
	Actionable        bool
	PlatformSensitive bool
	NonProgressRisk   string
	Search            string
}

type htmlFilterOption struct {
	Value string
	Label string
	Count int
}

func HTML(result engine.RunResult) string {
	rows := htmlRows(result)
	statusOptions := htmlFilterOptions(rows, func(row htmlReportRow) (string, string) {
		return row.Status, strings.ReplaceAll(row.Status, "_", " ")
	})
	actionabilityOptions := htmlFilterOptions(rows, func(row htmlReportRow) (string, string) {
		return row.Actionability, row.Actionability
	})
	operatorOptions := htmlFilterOptions(rows, func(row htmlReportRow) (string, string) {
		return row.Operator, row.Operator
	})
	riskOptions := htmlFilterOptions(rows, func(row htmlReportRow) (string, string) {
		return row.EquivalentRisk, row.EquivalentRisk
	})
	groupOptions := htmlFilterOptions(rows, func(row htmlReportRow) (string, string) {
		return row.GroupFilter, row.GroupLabel
	})
	historyOptions := htmlFilterOptions(rows, func(row htmlReportRow) (string, string) {
		return row.HistoryStatus, strings.ReplaceAll(row.HistoryStatus, "_", " ")
	})
	ageOptions := htmlFilterOptions(rows, func(row htmlReportRow) (string, string) {
		return row.AgeBand, row.AgeLabel
	})
	timingOptions := htmlFilterOptions(rows, func(row htmlReportRow) (string, string) {
		return row.TimingBand, row.TimingLabel
	})
	actionableSurvivors, survivorGroups, longStandingSurvivors, slowSignals := htmlSummaryMetrics(rows)
	if result.Summary.Actionable.TrueActionableSurvivors > 0 || result.Summary.Actionable.ActionableSurvivors > 0 {
		actionableSurvivors = result.Summary.Actionable.TrueActionableSurvivors
	}
	if result.Summary.Actionable.SemanticGroupReviewUnits > 0 {
		survivorGroups = result.Summary.Actionable.SemanticGroupReviewUnits
	}
	initialVisible := 0
	initialGroups := map[string]bool{}
	for _, row := range rows {
		if !row.Survivor {
			continue
		}
		initialVisible++
		if row.GroupFilter != "ungrouped" {
			initialGroups[row.GroupFilter] = true
		}
	}

	var b strings.Builder
	b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\">")
	b.WriteString("<title>cervomut survivor review workbench</title>")
	b.WriteString(`<style>
body{margin:0;font-family:Segoe UI,Arial,sans-serif;background:#f5f7fb;color:#162033}
.page{max-width:1600px;margin:0 auto;padding:24px}
.hero{display:grid;gap:16px;margin-bottom:20px}
.hero h1{margin:0;font-size:30px}
.hero p{margin:0;color:#44506a;max-width:960px}
.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:12px}
.card{background:#fff;border:1px solid #d9e1f2;border-radius:14px;padding:14px 16px;box-shadow:0 4px 18px rgba(22,32,51,.05)}
.card-label{display:block;font-size:12px;font-weight:700;letter-spacing:.04em;text-transform:uppercase;color:#61708e}
.card-value{display:block;margin-top:8px;font-size:28px;font-weight:700}
.toolbar,.table-shell,.summary-shell{background:#fff;border:1px solid #d9e1f2;border-radius:14px;box-shadow:0 4px 18px rgba(22,32,51,.05)}
.toolbar{padding:16px;margin-bottom:20px}
.toolbar h2,.table-shell h2{margin:0 0 12px 0;font-size:18px}
.toolbar-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(170px,1fr));gap:12px;align-items:end}
.toolbar label{display:flex;flex-direction:column;gap:6px;font-size:13px;font-weight:600;color:#34415c}
.toolbar input,.toolbar select,.toolbar button{font:inherit}
.toolbar input,.toolbar select{padding:9px 10px;border:1px solid #bec9df;border-radius:10px;background:#fff}
.toolbar button{padding:10px 12px;border:0;border-radius:10px;background:#163b73;color:#fff;font-weight:700;cursor:pointer}
.toolbar button.quick-filter{background:#eaf0fb;color:#163b73}
.toolbar button:hover{filter:brightness(.98)}
.checkbox-label{justify-content:flex-end}
.checkbox-label input{margin-right:8px}
.quick-nav{display:grid;gap:10px;margin-top:16px}
.chip-row{display:flex;flex-wrap:wrap;gap:8px}
.chip-row strong{display:inline-flex;align-items:center;font-size:13px;color:#44506a}
.results-meta{margin-top:16px;font-size:13px;color:#44506a}
.table-shell{padding:16px}
table{width:100%;border-collapse:collapse}
th,td{padding:12px 10px;border-top:1px solid #e2e8f4;vertical-align:top;text-align:left}
thead th{border-top:0;font-size:12px;text-transform:uppercase;letter-spacing:.04em;color:#61708e}
tbody tr:hover{background:#f8fafe}
.badge{display:inline-flex;align-items:center;gap:6px;padding:4px 8px;border-radius:999px;font-size:12px;font-weight:700;background:#edf2ff;color:#27457a}
.badge-survived{background:#fce8cc;color:#8a5100}
.badge-killed{background:#dff7e5;color:#176338}
.badge-timed_out,.badge-memory_killed,.badge-compile_error{background:#fde2e1;color:#8d261d}
.badge-not_covered,.badge-pending_budget,.badge-skipped_resource,.badge-cached,.badge-quarantined,.badge-ignored,.badge-skipped{background:#eceff5;color:#55637d}
.mutant-id{font-family:Consolas,monospace;font-size:12px;color:#49556e}
.mutant-main{display:grid;gap:4px}
.mutant-file{font-weight:700}
.mutant-meta{font-size:12px;color:#61708e}
.mutant-swap{font-family:Consolas,monospace;font-size:12px;color:#24314a}
.mutant-reason{font-size:12px;color:#44506a;max-width:360px}
.diff-shell details{min-width:320px}
.diff-shell summary{cursor:pointer;font-weight:600;color:#163b73}
.diff-shell pre{margin:8px 0 0 0;padding:10px;border-radius:10px;background:#0f1727;color:#e5eefc;overflow:auto;font-size:12px}
.summary-shell{margin-top:20px;padding:16px}
.summary-shell details summary{cursor:pointer;font-weight:700;color:#163b73}
.summary-shell pre{white-space:pre-wrap}
@media (max-width:960px){.page{padding:16px}.table-shell{overflow:auto}}
</style>`)
	b.WriteString("</head><body><div class=\"page\">")
	b.WriteString("<section class=\"hero\"><div>")
	b.WriteString("<h1>cervomut survivor review workbench</h1>")
	b.WriteString("<p>Survivors stay first-class, but the full raw run remains in the table below. Use the filters to narrow review by actionability, semantic grouping, operator, equivalent risk, survivor history, age, and timing signal without mutating the underlying report.</p>")
	b.WriteString("</div><div class=\"cards\">")
	writeHTMLCard(&b, "Survivors", result.Summary.Survived)
	writeHTMLCard(&b, "Actionable score", fmt.Sprintf("%.2f%%", result.Summary.Actionable.ActionableScore))
	writeHTMLCard(&b, "True actionable survivors", actionableSurvivors)
	writeHTMLCard(&b, "Semantic review units", survivorGroups)
	writeHTMLCard(&b, "Long-standing survivors", longStandingSurvivors)
	writeHTMLCard(&b, "Slow timing signals", slowSignals)
	writeHTMLCard(&b, "Raw score", fmt.Sprintf("%.2f%%", result.Summary.Score))
	b.WriteString("</div></section>")
	b.WriteString("<section class=\"toolbar\"><h2>Filters</h2><div class=\"toolbar-grid\">")
	writeHTMLInput(&b, "filter-search", "Search", "mutant id, file, operator, tests, reason")
	writeHTMLSelect(&b, "filter-status", "Status", statusOptions, "All statuses")
	writeHTMLSelect(&b, "filter-actionability", "Actionability", actionabilityOptions, "All actionability")
	writeHTMLSelect(&b, "filter-operator", "Operator", operatorOptions, "All operators")
	writeHTMLSelect(&b, "filter-risk", "Equivalent risk", riskOptions, "All risk levels")
	writeHTMLSelect(&b, "filter-group", "Semantic group", groupOptions, "All groups")
	writeHTMLSelect(&b, "filter-history", "History", historyOptions, "All history")
	writeHTMLSelect(&b, "filter-age", "Survivor age", ageOptions, "All age bands")
	writeHTMLSelect(&b, "filter-timing", "Timing signal", timingOptions, "All timing signals")
	b.WriteString(`<label class="checkbox-label"><span>Primary queue</span><span><input id="filter-survivors-only" type="checkbox" checked>Survivors only</span></label>`)
	b.WriteString(`<label><span>Reset</span><button id="filter-reset" type="button">Reset filters</button></label>`)
	b.WriteString("</div>")
	writeHTMLQuickFilters(&b, "Group shortcuts", "filter-group", topHTMLFilterOptions(groupOptions, 6))
	writeHTMLQuickFilters(&b, "Operator shortcuts", "filter-operator", topHTMLFilterOptions(operatorOptions, 6))
	fmt.Fprintf(&b, `<div class="results-meta">Showing <strong id="visible-count">%d</strong> of <strong id="total-count">%d</strong> mutants. Visible survivors: <strong id="visible-survivors">%d</strong>. Visible semantic groups: <strong id="visible-groups">%d</strong>.</div>`, initialVisible, len(rows), initialVisible, len(initialGroups))
	b.WriteString("</section>")
	b.WriteString(`<section class="table-shell"><h2>Review queue</h2><table id="mutant-table"><thead><tr><th>Rank</th><th>Mutant</th><th>Status</th><th>Review signal</th><th>History and timing</th><th>Reason and skip guidance</th><th>Diff</th></tr></thead><tbody>`)
	for _, row := range rows {
		fmt.Fprintf(&b, `<tr data-mutant-row data-status="%s" data-survivor="%t" data-actionability="%s" data-operator="%s" data-risk="%s" data-group="%s" data-history="%s" data-age="%s" data-timing="%s" data-search="%s">`,
			html.EscapeString(row.Status),
			row.Survivor,
			html.EscapeString(row.Actionability),
			html.EscapeString(row.Operator),
			html.EscapeString(row.EquivalentRisk),
			html.EscapeString(row.GroupFilter),
			html.EscapeString(row.HistoryStatus),
			html.EscapeString(row.AgeBand),
			html.EscapeString(row.TimingBand),
			html.EscapeString(row.Search),
		)
		b.WriteString("<td>")
		if row.SurvivorRank > 0 {
			fmt.Fprintf(&b, `<span class="badge">#%d</span>`, row.SurvivorRank)
		} else {
			b.WriteString(`<span class="badge">-</span>`)
		}
		b.WriteString("</td><td><div class=\"mutant-main\">")
		fmt.Fprintf(&b, `<div class="mutant-file">%s:%d</div>`, html.EscapeString(row.File), row.Line)
		if row.Function != "" {
			fmt.Fprintf(&b, `<div class="mutant-meta">function=%s</div>`, html.EscapeString(row.Function))
		}
		fmt.Fprintf(&b, `<div class="mutant-swap">%s -> %s</div>`, html.EscapeString(row.Original), html.EscapeString(row.Mutated))
		fmt.Fprintf(&b, `<div class="mutant-id">%s</div>`, html.EscapeString(row.MutantID))
		if row.Description != "" {
			fmt.Fprintf(&b, `<div class="mutant-meta">%s</div>`, html.EscapeString(row.Description))
		}
		b.WriteString("</div></td><td>")
		fmt.Fprintf(&b, `<span class="badge badge-%s">%s</span>`, html.EscapeString(row.Status), html.EscapeString(strings.ReplaceAll(row.Status, "_", " ")))
		if row.FailureKind != "" {
			fmt.Fprintf(&b, `<div class="mutant-meta">failure=%s</div>`, html.EscapeString(row.FailureKind))
		}
		b.WriteString("</td><td>")
		fmt.Fprintf(&b, `<div class="mutant-meta">actionability=%s</div>`, html.EscapeString(row.Actionability))
		fmt.Fprintf(&b, `<div class="mutant-meta">operator=%s</div>`, html.EscapeString(row.Operator))
		fmt.Fprintf(&b, `<div class="mutant-meta">equivalent_risk=%s</div>`, html.EscapeString(row.EquivalentRisk))
		fmt.Fprintf(&b, `<div class="mutant-meta">group=%s</div>`, html.EscapeString(row.GroupLabel))
		if row.PlatformSensitive {
			b.WriteString(`<div class="mutant-meta">platform-sensitive</div>`)
		}
		if row.NonProgressRisk != "" {
			fmt.Fprintf(&b, `<div class="mutant-meta">non_progress_risk=%s</div>`, html.EscapeString(row.NonProgressRisk))
		}
		if row.SuggestedScope != "" {
			fmt.Fprintf(&b, `<div class="mutant-meta">suggested_scope=%s</div>`, html.EscapeString(row.SuggestedScope))
		}
		if row.NearestTests != "" {
			fmt.Fprintf(&b, `<div class="mutant-meta">nearby_tests=%s</div>`, html.EscapeString(row.NearestTests))
		}
		b.WriteString("</td><td>")
		fmt.Fprintf(&b, `<div class="mutant-meta">history=%s</div>`, html.EscapeString(strings.ReplaceAll(row.HistoryStatus, "_", " ")))
		fmt.Fprintf(&b, `<div class="mutant-meta">age=%s</div>`, html.EscapeString(row.AgeLabel))
		fmt.Fprintf(&b, `<div class="mutant-meta">duration=%s</div>`, html.EscapeString(row.DurationText))
		fmt.Fprintf(&b, `<div class="mutant-meta">timing_signal=%s</div>`, html.EscapeString(row.TimingLabel))
		b.WriteString("</td><td>")
		if row.StatusReason != "" {
			fmt.Fprintf(&b, `<div class="mutant-reason">%s</div>`, html.EscapeString(row.StatusReason))
		}
		if row.RankReason != "" {
			fmt.Fprintf(&b, `<div class="mutant-meta">rank=%s</div>`, html.EscapeString(row.RankReason))
		}
		if row.SuggestedSkip != "" {
			fmt.Fprintf(&b, `<div class="mutant-meta">skip=%s</div>`, html.EscapeString(row.SuggestedSkip))
		}
		b.WriteString("</td><td class=\"diff-shell\"><details><summary>Show diff</summary><pre>")
		b.WriteString(html.EscapeString(row.Diff))
		b.WriteString("</pre></details></td></tr>")
	}
	b.WriteString("</tbody></table></section>")
	b.WriteString(`<section class="summary-shell"><details><summary>Raw summary</summary><pre>`)
	b.WriteString(html.EscapeString(Summary(result)))
	b.WriteString(`</pre></details></section>`)
	b.WriteString(`<script>
(function(){
  const rows = Array.from(document.querySelectorAll('[data-mutant-row]'));
  const search = document.getElementById('filter-search');
  const status = document.getElementById('filter-status');
  const actionability = document.getElementById('filter-actionability');
  const operator = document.getElementById('filter-operator');
  const risk = document.getElementById('filter-risk');
  const group = document.getElementById('filter-group');
  const history = document.getElementById('filter-history');
  const age = document.getElementById('filter-age');
  const timing = document.getElementById('filter-timing');
  const survivorsOnly = document.getElementById('filter-survivors-only');
  const reset = document.getElementById('filter-reset');
  const visibleCount = document.getElementById('visible-count');
  const totalCount = document.getElementById('total-count');
  const visibleSurvivors = document.getElementById('visible-survivors');
  const visibleGroups = document.getElementById('visible-groups');
  totalCount.textContent = String(rows.length);

  function matches(control, value) {
    return control.value === 'all' || control.value === value;
  }

  function normalize(value) {
    return (value || '').toLowerCase();
  }

  function applyFilters() {
    const term = normalize(search.value).trim();
    let visible = 0;
    let survivors = 0;
    const groups = new Set();

    rows.forEach((row) => {
      const data = row.dataset;
      const show =
        matches(status, data.status) &&
        matches(actionability, data.actionability) &&
        matches(operator, data.operator) &&
        matches(risk, data.risk) &&
        matches(group, data.group) &&
        matches(history, data.history) &&
        matches(age, data.age) &&
        matches(timing, data.timing) &&
        (!survivorsOnly.checked || data.survivor === 'true') &&
        (term === '' || normalize(data.search).includes(term));

      row.hidden = !show;
      if (!show) {
        return;
      }
      visible += 1;
      if (data.survivor === 'true') {
        survivors += 1;
      }
      if (data.group && data.group !== 'ungrouped') {
        groups.add(data.group);
      }
    });

    visibleCount.textContent = String(visible);
    visibleSurvivors.textContent = String(survivors);
    visibleGroups.textContent = String(groups.size);
  }

  [search, status, actionability, operator, risk, group, history, age, timing].forEach((control) => {
    control.addEventListener('input', applyFilters);
    control.addEventListener('change', applyFilters);
  });
  survivorsOnly.addEventListener('change', applyFilters);

  reset.addEventListener('click', function() {
    search.value = '';
    [status, actionability, operator, risk, group, history, age, timing].forEach((control) => {
      control.value = 'all';
    });
    survivorsOnly.checked = true;
    applyFilters();
  });

  Array.from(document.querySelectorAll('[data-filter-target]')).forEach((button) => {
    button.addEventListener('click', function() {
      const target = document.getElementById(button.dataset.filterTarget);
      if (!target) {
        return;
      }
      target.value = button.dataset.filterValue || 'all';
      if (target.id === 'filter-group' || target.id === 'filter-operator') {
        survivorsOnly.checked = true;
      }
      applyFilters();
    });
  });

  applyFilters();
})();
</script>`)
	b.WriteString("</div></body></html>")
	return b.String()
}

func htmlRows(result engine.RunResult) []htmlReportRow {
	sorted := append([]engine.MutantResult{}, result.Mutants...)
	sort.SliceStable(sorted, func(i, j int) bool {
		leftSurvivor := sorted[i].Status == engine.StatusSurvived
		rightSurvivor := sorted[j].Status == engine.StatusSurvived
		if leftSurvivor != rightSurvivor {
			return leftSurvivor
		}
		if leftSurvivor && rightSurvivor {
			leftRank := sorted[i].SurvivorRank
			rightRank := sorted[j].SurvivorRank
			if leftRank == 0 {
				leftRank = 1 << 20
			}
			if rightRank == 0 {
				rightRank = 1 << 20
			}
			if leftRank != rightRank {
				return leftRank < rightRank
			}
		}
		if sorted[i].Mutant.File != sorted[j].Mutant.File {
			return sorted[i].Mutant.File < sorted[j].Mutant.File
		}
		if sorted[i].Mutant.Line != sorted[j].Mutant.Line {
			return sorted[i].Mutant.Line < sorted[j].Mutant.Line
		}
		return sorted[i].MutantID < sorted[j].MutantID
	})

	rows := make([]htmlReportRow, 0, len(sorted))
	for _, mutant := range sorted {
		ageBand, ageLabel := htmlAgeBand(mutant.SurvivorAgeRuns)
		timingBand, timingLabel := htmlTimingBand(mutant.Duration)
		groupFilter := mutant.Mutant.GroupLabel
		groupLabel := mutant.Mutant.GroupLabel
		if strings.TrimSpace(groupFilter) == "" {
			groupFilter = "ungrouped"
			groupLabel = "ungrouped"
		}
		actionability := mutant.Actionability
		if strings.TrimSpace(actionability) == "" {
			actionability = "unknown"
		}
		equivalentRisk := mutant.Mutant.EquivalentRisk
		if strings.TrimSpace(equivalentRisk) == "" {
			equivalentRisk = "unknown"
		}
		historyStatus := mutant.HistoryStatus
		if strings.TrimSpace(historyStatus) == "" {
			historyStatus = "unknown"
		}
		rows = append(rows, htmlReportRow{
			MutantID:          mutant.MutantID,
			Status:            string(mutant.Status),
			Survivor:          mutant.Status == engine.StatusSurvived,
			SurvivorRank:      mutant.SurvivorRank,
			Actionability:     actionability,
			Operator:          mutant.Mutant.Operator,
			EquivalentRisk:    equivalentRisk,
			GroupFilter:       groupFilter,
			GroupLabel:        groupLabel,
			HistoryStatus:     historyStatus,
			AgeBand:           ageBand,
			AgeLabel:          ageLabel,
			TimingBand:        timingBand,
			TimingLabel:       timingLabel,
			DurationText:      htmlDurationText(mutant.Duration),
			File:              mutant.Mutant.File,
			Line:              mutant.Mutant.Line,
			Function:          mutant.Mutant.Function,
			Original:          mutant.Mutant.Original,
			Mutated:           mutant.Mutant.Mutated,
			Description:       mutant.Mutant.Description,
			FailureKind:       mutant.FailureKind,
			StatusReason:      mutant.StatusReason,
			SuggestedSkip:     ledgerSuggestedReason(mutant, ""),
			SuggestedScope:    mutant.SuggestedTestScope,
			NearestTests:      strings.Join(mutant.NearestTests, ", "),
			RankReason:        mutant.RankReason,
			Diff:              mutant.Mutant.Diff,
			Actionable:        isActionableSurvivor(result.Environment.OS, mutant),
			PlatformSensitive: mutant.Mutant.PlatformSensitive,
			NonProgressRisk:   mutant.Mutant.NonProgressRisk,
			Search:            htmlSearchText(mutant, groupLabel, historyStatus, ageLabel, timingLabel),
		})
	}
	return rows
}

func htmlFilterOptions(rows []htmlReportRow, picker func(htmlReportRow) (string, string)) []htmlFilterOption {
	counts := map[string]int{}
	labels := map[string]string{}
	for _, row := range rows {
		value, label := picker(row)
		value = strings.TrimSpace(value)
		label = strings.TrimSpace(label)
		if value == "" || label == "" {
			continue
		}
		counts[value]++
		labels[value] = label
	}
	options := make([]htmlFilterOption, 0, len(counts))
	for value, count := range counts {
		options = append(options, htmlFilterOption{Value: value, Label: labels[value], Count: count})
	}
	sort.Slice(options, func(i, j int) bool {
		return options[i].Label < options[j].Label
	})
	return options
}

func htmlSummaryMetrics(rows []htmlReportRow) (actionableSurvivors, survivorGroups, longStandingSurvivors, slowSignals int) {
	groups := map[string]bool{}
	for _, row := range rows {
		if row.Survivor && row.Actionable {
			actionableSurvivors++
		}
		if row.Survivor && row.GroupFilter != "ungrouped" {
			groups[row.GroupFilter] = true
		}
		if row.Survivor && row.AgeBand == "long-standing" {
			longStandingSurvivors++
		}
		if row.TimingBand == "slow" {
			slowSignals++
		}
	}
	return actionableSurvivors, len(groups), longStandingSurvivors, slowSignals
}

func htmlAgeBand(runs int) (string, string) {
	switch {
	case runs <= 0:
		return "unknown", "unknown"
	case runs == 1:
		return "new", "new (1 run)"
	case runs < 5:
		return "aging", "aging (2-4 runs)"
	default:
		return "long-standing", "long-standing (5+ runs)"
	}
}

func htmlTimingBand(duration time.Duration) (string, string) {
	switch {
	case duration <= 0:
		return "unknown", "not recorded"
	case duration < 500*time.Millisecond:
		return "fast", "fast (<500ms)"
	case duration < 2*time.Second:
		return "medium", "medium (0.5-2s)"
	default:
		return "slow", "slow (>2s)"
	}
}

func htmlDurationText(duration time.Duration) string {
	if duration <= 0 {
		return "not recorded"
	}
	return duration.Round(time.Millisecond).String()
}

func htmlSearchText(mutant engine.MutantResult, groupLabel, historyStatus, ageLabel, timingLabel string) string {
	parts := []string{
		mutant.MutantID,
		mutant.Mutant.File,
		mutant.Mutant.Function,
		mutant.Mutant.Operator,
		mutant.Mutant.Description,
		mutant.StatusReason,
		mutant.RankReason,
		groupLabel,
		historyStatus,
		ageLabel,
		timingLabel,
		mutant.SuggestedTestScope,
		mutant.SuggestedSkipReason,
		strings.Join(mutant.NearestTests, " "),
	}
	return strings.Join(parts, " ")
}

func writeHTMLCard(b *strings.Builder, label string, value any) {
	fmt.Fprintf(b, `<div class="card"><span class="card-label">%s</span><span class="card-value">%v</span></div>`, html.EscapeString(label), value)
}

func writeHTMLInput(b *strings.Builder, id, label, placeholder string) {
	fmt.Fprintf(b, `<label for="%s"><span>%s</span><input id="%s" type="search" placeholder="%s"></label>`,
		html.EscapeString(id),
		html.EscapeString(label),
		html.EscapeString(id),
		html.EscapeString(placeholder),
	)
}

func writeHTMLSelect(b *strings.Builder, id, label string, options []htmlFilterOption, allLabel string) {
	fmt.Fprintf(b, `<label for="%s"><span>%s</span><select id="%s"><option value="all">%s</option>`,
		html.EscapeString(id),
		html.EscapeString(label),
		html.EscapeString(id),
		html.EscapeString(allLabel),
	)
	for _, option := range options {
		fmt.Fprintf(b, `<option value="%s">%s (%d)</option>`,
			html.EscapeString(option.Value),
			html.EscapeString(option.Label),
			option.Count,
		)
	}
	b.WriteString(`</select></label>`)
}

func writeHTMLQuickFilters(b *strings.Builder, label, target string, options []htmlFilterOption) {
	if len(options) == 0 {
		return
	}
	fmt.Fprintf(b, `<div class="quick-nav"><div class="chip-row"><strong>%s</strong>`, html.EscapeString(label))
	for _, option := range options {
		fmt.Fprintf(b, `<button class="quick-filter" type="button" data-filter-target="%s" data-filter-value="%s">%s (%d)</button>`,
			html.EscapeString(target),
			html.EscapeString(option.Value),
			html.EscapeString(option.Label),
			option.Count,
		)
	}
	b.WriteString(`</div></div>`)
}

func topHTMLFilterOptions(options []htmlFilterOption, limit int) []htmlFilterOption {
	if len(options) == 0 || limit <= 0 {
		return nil
	}
	ordered := append([]htmlFilterOption{}, options...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Count != ordered[j].Count {
			return ordered[i].Count > ordered[j].Count
		}
		return ordered[i].Label < ordered[j].Label
	})
	if len(ordered) > limit {
		ordered = ordered[:limit]
	}
	return ordered
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

func WriteAll(dir string, result engine.RunResult) error {
	return WriteFormats(dir, result, []string{"summary", "json", "junit", "html"})
}

func WriteFormats(dir string, result engine.RunResult, formats []string) error {
	return WriteFormatsWithOptions(dir, result, formats, WriteOptions{})
}

func WriteFormatsWithOptions(dir string, result engine.RunResult, formats []string, opts WriteOptions) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if len(formats) == 0 {
		formats = []string{"summary", "json"}
	}
	files := map[string][]byte{}
	for _, format := range formats {
		switch strings.TrimSpace(format) {
		case "summary":
			files["summary.txt"] = []byte(Summary(result))
			files["survivors.txt"] = []byte(Survivors(result))
		case "json":
			jsonData, err := JSON(result)
			if err != nil {
				return err
			}
			files["mutation-report.json"] = jsonData
		case "junit":
			junitData, err := JUnit(result)
			if err != nil {
				return err
			}
			files["junit.xml"] = junitData
		case "html":
			files["index.html"] = []byte(HTML(result))
		}
	}
	if opts.ActionableOnly {
		files["survivors-actionable.txt"] = []byte(SurvivorsWithOptions(result, SurvivorsOptions{ActionableOnly: true}))
	}
	ledgerData, err := SemanticTriageLedger(result)
	if err != nil {
		return err
	}
	files["semantic-triage-ledger.json"] = ledgerData
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}
