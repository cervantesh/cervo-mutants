package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"time"
)

func buildWaveSummary(trackingIssue, manifestPath, actionRef, installPath, artifactsDir, generatedAt string) (waveSummary, error) {
	if generatedAt == "" {
		generatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	results, err := loadWaveSummaryResults(artifactsDir)
	if err != nil {
		return waveSummary{}, err
	}

	summary := waveSummary{
		SchemaVersion: "1",
		TrackingIssue: trackingIssue,
		ManifestPath:  manifestPath,
		ActionRef:     actionRef,
		InstallPath:   installPath,
		GeneratedAt:   generatedAt,
		Repos:         results,
		Aggregate: waveAggregate{
			FailureKinds: map[string]int{},
		},
		Triage: waveSummaryTriage{
			GovernanceSuggestionsByStatus: map[string]int{},
		},
	}
	accumulateWaveSummary(&summary)
	return summary, nil
}

func loadWaveSummaryResults(artifactsDir string) ([]waveResult, error) {
	paths, err := filepath.Glob(filepath.Join(artifactsDir, "*", "wave-result.json"))
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no wave-result.json files found under %s", artifactsDir)
	}
	results := make([]waveResult, 0, len(paths))
	for _, path := range paths {
		var result waveResult
		if err := readJSONFile(path, &result); err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})
	return results, nil
}

func accumulateWaveSummary(summary *waveSummary) {
	summary.Aggregate.Selected = len(summary.Repos)
	for _, result := range summary.Repos {
		accumulateWaveAggregate(&summary.Aggregate, result)
		accumulateWaveSummaryTriage(&summary.Triage, result)
	}
	if len(summary.Aggregate.FailureKinds) == 0 {
		summary.Aggregate.FailureKinds = map[string]int{}
	}
	if len(summary.Triage.GovernanceSuggestionsByStatus) == 0 {
		summary.Triage.GovernanceSuggestionsByStatus = map[string]int{}
	}
}

func accumulateWaveAggregate(aggregate *waveAggregate, result waveResult) {
	if result.ReportKind == "missing" {
		aggregate.MissingReports++
	} else {
		aggregate.Reports++
	}
	if result.DenominatorHealth != nil {
		aggregate.Generated += result.DenominatorHealth.Generated
		aggregate.Covered += result.DenominatorHealth.Covered
		aggregate.Executed += result.DenominatorHealth.Executed
		aggregate.Effective += result.DenominatorHealth.Effective
		if len(result.DenominatorHealth.Warnings) > 0 {
			aggregate.WarningRepos++
		}
	}
	if result.Summary != nil {
		aggregate.Killed += result.Summary.Killed
		aggregate.Survived += result.Summary.Survived
		aggregate.NotCovered += result.Summary.NotCovered
		aggregate.TimedOut += result.Summary.TimedOut
		aggregate.CompileError += result.Summary.CompileError
	}
	if result.Failure != nil && result.Failure.Kind != "" {
		aggregate.FailedReports++
		aggregate.FailureKinds[result.Failure.Kind]++
	}
}

func accumulateWaveSummaryTriage(triage *waveSummaryTriage, result waveResult) {
	triage.ActionableReviewUnits += result.Triage.ActionableReviewUnits
	triage.ActionableSurvivors += result.Triage.ActionableSurvivors
	triage.EquivalentRiskSurvivors += result.Triage.EquivalentRiskSurvivors
	triage.SemanticGroupReviewUnits += result.Triage.SemanticGroupReviewUnits
	triage.CollapsedSemanticDuplicates += result.Triage.CollapsedSemanticDuplicates
	triage.SemanticGroupCount += result.Triage.SemanticGroupCount
	triage.RecommendationEntries += result.Triage.RecommendationEntries
	triage.RecommendationReviewUnits += result.Triage.RecommendationReviewUnits
	triage.CollapsedRecommendationDupes += result.Triage.CollapsedRecommendationDupes
	triage.LedgerEntries += result.Triage.LedgerEntries
	triage.GovernanceQuarantineTemplates += result.Triage.GovernanceQuarantineTemplates
	triage.GovernanceSuppressionTemplates += result.Triage.GovernanceSuppressionTemplates
	triage.GovernanceTotalSuggestions += result.Triage.GovernanceTotalSuggestions
	for key, value := range result.Triage.GovernanceSuggestionsByStatus {
		triage.GovernanceSuggestionsByStatus[key] += value
	}
	if result.Triage.ActionableReviewUnits > 0 {
		triage.ReposWithActionableReviewUnits++
	}
	if result.Triage.SemanticGroupCount > 0 {
		triage.ReposWithSemanticGroups++
	}
	if result.Triage.RecommendationEntries > 0 {
		triage.ReposWithRecommendations++
	}
	if result.Triage.RecommendationReviewUnits > 0 {
		triage.ReposWithRecommendationReviewUnits++
	}
	if result.Triage.LedgerEntries > 0 {
		triage.ReposWithLedgerEntries++
	}
	if result.Triage.GovernanceTotalSuggestions > 0 {
		triage.ReposWithGovernanceSuggestions++
	}
}
