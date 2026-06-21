package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cervantesh/cervo-mutants/pkg/engine"
)

func renderWaveResultMarkdown(result waveResult) string {
	var lines []string
	lines = append(lines,
		"## "+result.Name,
		fmt.Sprintf("- Repository: `%s`", result.Repository),
		fmt.Sprintf("- Install path: `%s` action_ref=`%s`", result.InstallPath, result.ActionRef),
		fmt.Sprintf("- Target: `%s`", result.Target),
		fmt.Sprintf("- Profile: `%s`", result.Profile),
		fmt.Sprintf("- Go version: resolved=`%s` action_min=`%s`%s%s",
			result.GoVersion,
			result.GoVersionActionMin,
			formatOptionalInline(" target=", result.GoVersionTarget),
			formatOptionalInline(" requested=", result.GoVersionRequested),
		),
		fmt.Sprintf("- Policy: `%s` sample=`%s` coverage_prefilter=`%t` prewarm_modules=`%t` max_mutants=`%d` workers=`%d`", result.Policy, result.Sample, result.CoveragePrefilter, result.PrewarmModules, result.MaxMutants, result.Workers),
		fmt.Sprintf("- Job status: `%s`", result.JobStatus),
	)
	if result.ReportKind == "missing" {
		lines = append(lines, "- Report: missing")
	} else {
		lines = append(lines, fmt.Sprintf("- Report: %s (`%s`)", result.ReportKind, result.ReportPath))
	}
	if result.Summary != nil {
		actionableScore := 0.0
		if result.Summary.ActionableScore != nil {
			actionableScore = *result.Summary.ActionableScore
		}
		lines = append(lines, fmt.Sprintf("- Generated: **%d**, effective: **%d**, killed: **%d**, survived: **%d**, not covered: **%d**, score: **%.2f%%**, actionable: **%.2f%%**",
			denominatorField(result.DenominatorHealth, func(d engine.DenominatorHealth) int { return d.Generated }),
			denominatorField(result.DenominatorHealth, func(d engine.DenominatorHealth) int { return d.Effective }),
			result.Summary.Killed,
			result.Summary.Survived,
			result.Summary.NotCovered,
			result.Summary.Score,
			actionableScore,
		))
	} else {
		lines = append(lines, "- No report metrics captured")
	}
	if result.Failure != nil {
		lines = append(lines, fmt.Sprintf("- Failure: `%s` %s", result.Failure.Kind, result.Failure.Message))
		if result.Failure.RunnerResult != nil {
			lines = append(lines, fmt.Sprintf("- Runner detail: status=`%s` reason=`%s` command=`%s`%s",
				result.Failure.RunnerResult.Status,
				result.Failure.RunnerResult.StatusReason,
				strings.Join(result.Failure.RunnerResult.Command, " "),
				formatRunnerOutput(result.Failure.RunnerResult.Output),
			))
		}
	}
	lines = append(lines, fmt.Sprintf("- Triage: actionable_review_units=**%d**, semantic_groups=**%d**, recommendations=**%d**, ledger_entries=**%d**, governance_suggestions=**%d**",
		result.Triage.ActionableReviewUnits,
		result.Triage.SemanticGroupCount,
		result.Triage.RecommendationEntries,
		result.Triage.LedgerEntries,
		result.Triage.GovernanceTotalSuggestions,
	))
	return strings.Join(append(lines, ""), "\n")
}

func renderWaveSummaryMarkdown(summary waveSummary) string {
	var b strings.Builder
	b.WriteString("# External GitHub Action Wave Summary\n\n")
	if summary.TrackingIssue != "" {
		fmt.Fprintf(&b, "- Tracking issue: **%s**\n", summary.TrackingIssue)
	}
	if summary.ManifestPath != "" {
		fmt.Fprintf(&b, "- Manifest: `%s`\n", summary.ManifestPath)
	}
	if summary.InstallPath != "" {
		fmt.Fprintf(&b, "- Install path: `%s` action_ref=`%s`\n", summary.InstallPath, summary.ActionRef)
	}
	fmt.Fprintf(&b, "- Selected repos: **%d**\n", summary.Aggregate.Selected)
	fmt.Fprintf(&b, "- Reports captured: **%d**\n", summary.Aggregate.Reports)
	fmt.Fprintf(&b, "- Missing reports: **%d**\n", summary.Aggregate.MissingReports)
	fmt.Fprintf(&b, "- Generated mutants: **%d**\n", summary.Aggregate.Generated)
	fmt.Fprintf(&b, "- Covered mutants: **%d**\n", summary.Aggregate.Covered)
	fmt.Fprintf(&b, "- Executed mutants: **%d**\n", summary.Aggregate.Executed)
	fmt.Fprintf(&b, "- Effective mutants: **%d**\n", summary.Aggregate.Effective)
	fmt.Fprintf(&b, "- Killed: **%d**\n", summary.Aggregate.Killed)
	fmt.Fprintf(&b, "- Survived: **%d**\n", summary.Aggregate.Survived)
	fmt.Fprintf(&b, "- Not covered: **%d**\n", summary.Aggregate.NotCovered)
	fmt.Fprintf(&b, "- Timed out: **%d**\n", summary.Aggregate.TimedOut)
	fmt.Fprintf(&b, "- Compile errors: **%d**\n", summary.Aggregate.CompileError)
	fmt.Fprintf(&b, "- Repos with denominator warnings: **%d**\n", summary.Aggregate.WarningRepos)
	fmt.Fprintf(&b, "- Repos with reported failures: **%d**\n", summary.Aggregate.FailedReports)
	if len(summary.Aggregate.FailureKinds) > 0 {
		fmt.Fprintf(&b, "- Failure kinds: `%s`\n", formatStatusCounts(summary.Aggregate.FailureKinds))
	}
	fmt.Fprintf(&b, "- Triage actionable review units: **%d**\n", summary.Triage.ActionableReviewUnits)
	fmt.Fprintf(&b, "- Semantic group review units: **%d**\n", summary.Triage.SemanticGroupReviewUnits)
	fmt.Fprintf(&b, "- Semantic groups formed: **%d**\n", summary.Triage.SemanticGroupCount)
	fmt.Fprintf(&b, "- Recommendation entries: **%d**\n", summary.Triage.RecommendationEntries)
	fmt.Fprintf(&b, "- Recommendation review units: **%d**\n", summary.Triage.RecommendationReviewUnits)
	fmt.Fprintf(&b, "- Collapsed recommendation duplicates: **%d**\n", summary.Triage.CollapsedRecommendationDupes)
	fmt.Fprintf(&b, "- Ledger entries: **%d**\n", summary.Triage.LedgerEntries)
	fmt.Fprintf(&b, "- Governance suggestions: **%d**\n", summary.Triage.GovernanceTotalSuggestions)
	if len(summary.Triage.GovernanceSuggestionsByStatus) > 0 {
		fmt.Fprintf(&b, "- Governance suggestions by status: `%s`\n", formatStatusCounts(summary.Triage.GovernanceSuggestionsByStatus))
	}
	b.WriteString("\n")
	for _, repo := range summary.Repos {
		b.WriteString(renderWaveResultMarkdown(repo))
		b.WriteString("\n")
	}
	return b.String()
}

func formatStatusCounts(counts map[string]int) string {
	if len(counts) == 0 {
		return ""
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, counts[key]))
	}
	return strings.Join(parts, ", ")
}

func compactInline(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\t", " ")
	value = strings.ReplaceAll(value, "`", "")
	value = strings.Join(strings.Fields(value), " ")
	if len(value) > 240 {
		return value[:240] + "..."
	}
	return value
}

func formatOptionalInline(prefix string, value *string) string {
	if value == nil || *value == "" {
		return ""
	}
	return prefix + "`" + *value + "`"
}

func formatRunnerOutput(output string) string {
	if strings.TrimSpace(output) == "" {
		return ""
	}
	return " output=`" + compactInline(output) + "`"
}
