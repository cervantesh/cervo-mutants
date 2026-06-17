package engine

import (
	"runtime"

	"github.com/cervantesh/cervo-mutants/pkg/triage"
)

func recommendationPriority(recommendation string) int {
	return triage.RecommendationPriority(recommendation)
}

func timeoutRiskPriority(mutant Mutant) int {
	return triage.TimeoutRiskPriority(triageMutant(mutant))
}

func platformSensitivityPriority(mutant Mutant) int {
	return triage.PlatformSensitivityPriority(runtime.GOOS, triageMutant(mutant))
}

func timeoutRiskBand(mutant Mutant) string {
	return triage.TimeoutRiskBand(triageMutant(mutant))
}

func semanticGroupSummaryKey(mutant Mutant) string {
	return triage.SemanticGroupSummaryKey(triageMutant(mutant))
}

func suggestedTestScope(mutant Mutant) string {
	return triage.SuggestedTestScope(triageMutant(mutant))
}

func applySemanticResultMetadata(result *MutantResult) {
	normalized := triage.NormalizeResult(triageResult(*result))
	result.FailureKind = normalized.FailureKind
	result.StatusReason = normalized.StatusReason
	result.SuggestedSkipReason = normalized.SuggestedSkip
}

func rankSurvivors(results []MutantResult) {
	applySurvivorRankings(results, defaultSurvivorRankings(runtime.GOOS, results))
}

func triageResults(results []MutantResult) []triage.Result {
	out := make([]triage.Result, 0, len(results))
	for _, result := range results {
		out = append(out, triageResult(result))
	}
	return out
}

func triageResult(result MutantResult) triage.Result {
	return triage.Result{
		MutantID:        result.MutantID,
		Status:          string(result.Status),
		FailureKind:     result.FailureKind,
		StatusReason:    result.StatusReason,
		CoverageSource:  result.CoverageSource,
		SuggestedSkip:   result.SuggestedSkipReason,
		HistoryStatus:   result.HistoryStatus,
		SurvivorAgeRuns: result.SurvivorAgeRuns,
		OperatorYield:   result.OperatorYield,
		Actionability:   result.Actionability,
		Mutant:          triageMutant(result.Mutant),
	}
}

func triageMutant(mutant Mutant) triage.Mutant {
	audits := make([]triage.SuppressionAudit, 0, len(mutant.SuppressionAudit))
	for _, audit := range mutant.SuppressionAudit {
		audits = append(audits, triage.SuppressionAudit{Action: audit.Action})
	}
	return triage.Mutant{
		ID:                mutant.ID,
		File:              mutant.File,
		Package:           mutant.Package,
		Function:          mutant.Function,
		Operator:          mutant.Operator,
		Hint:              mutant.Hint,
		Description:       mutant.Description,
		Recommendation:    mutant.Recommendation,
		EquivalentRisk:    mutant.EquivalentRisk,
		NearbyTests:       append([]string{}, mutant.NearbyTests...),
		SemanticGroup:     mutant.SemanticGroup,
		GroupLabel:        mutant.GroupLabel,
		PlatformSensitive: mutant.PlatformSensitive,
		NonProgressRisk:   mutant.NonProgressRisk,
		SuggestedSkip:     mutant.SuggestedSkipReason,
		SuppressionAudit:  audits,
	}
}
