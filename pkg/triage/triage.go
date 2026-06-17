package triage

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

const (
	StatusSurvived = "survived"
	StatusTimedOut = "timed_out"

	FailureKindTimeout         = "timeout"
	FailureKindNonProgressLoop = "non_progress_loop"
)

type Mutant struct {
	ID                string
	Package           string
	Function          string
	Operator          string
	Recommendation    string
	EquivalentRisk    string
	NearbyTests       []string
	SemanticGroup     string
	GroupLabel        string
	PlatformSensitive bool
	NonProgressRisk   string
	SuggestedSkip     string
	SuppressionAudit  []SuppressionAudit
}

type SuppressionAudit struct {
	Action string
}

type Result struct {
	MutantID        string
	Status          string
	FailureKind     string
	StatusReason    string
	CoverageSource  string
	SuggestedSkip   string
	HistoryStatus   string
	SurvivorAgeRuns int
	OperatorYield   float64
	Actionability   string
	Mutant          Mutant
}

type RankedSurvivor struct {
	MutantID           string
	SurvivorRank       int
	RankScore          float64
	RankReason         string
	Actionability      string
	SuggestedTestScope string
	SuggestedSkip      string
	SemanticGroupSize  int
	NearestTests       []string
}

type ActionableSummary struct {
	RawScore                    float64
	ActionableScore             float64
	Survivors                   int
	ActionableSurvivors         int
	TrueActionableSurvivors     int
	EquivalentRiskSurvivors     int
	PlatformSensitiveSurvivors  int
	NonProgressTimeouts         int
	SemanticGroupReviewUnits    int
	CollapsedSemanticDuplicates int
}

func RecommendationPriority(recommendation string) int {
	switch recommendation {
	case "fast-ci":
		return 0
	case "conservative":
		return 1
	case "default":
		return 2
	case "aggressive":
		return 3
	default:
		return 4
	}
}

func TimeoutRiskPriority(mutant Mutant) int {
	if mutant.NonProgressRisk == "high" {
		return 4
	}
	switch mutant.Operator {
	case "conditionals-negation", "conditionals-boundary", "boolean-literals", "logical":
		return 0
	case "arithmetic-basic", "string-empty-literals", "nil-checks", "numeric-literals", "return-bool-literals", "assignment-arithmetic", "inc-dec":
		return 1
	case "error-returns":
		return 2
	case "literals", "returns", "loop-control", "slice-map-len-boundary":
		return 3
	default:
		return 2
	}
}

func TimeoutRiskBand(mutant Mutant) string {
	switch TimeoutRiskPriority(mutant) {
	case 0:
		return "low"
	case 1:
		return "medium"
	case 2:
		return "high"
	default:
		return "very_high"
	}
}

func PlatformSensitivityPriority(goos string, mutant Mutant) int {
	if strings.EqualFold(goos, "windows") && mutant.PlatformSensitive {
		return 1
	}
	return 0
}

func SemanticGroupSummaryKey(mutant Mutant) string {
	if mutant.GroupLabel != "" {
		return mutant.GroupLabel
	}
	return mutant.SemanticGroup
}

func NormalizeResult(result Result) Result {
	if result.SuggestedSkip == "" {
		result.SuggestedSkip = result.Mutant.SuggestedSkip
	}
	if result.Status != StatusTimedOut {
		return result
	}
	if result.Mutant.NonProgressRisk == "high" {
		result.FailureKind = FailureKindNonProgressLoop
		result.StatusReason = "test command timed out after a likely non-progress loop mutation"
		if result.SuggestedSkip == "" {
			result.SuggestedSkip = "reviewed-skip or quarantine if the timeout is a confirmed non-progress loop"
		}
		return result
	}
	if result.FailureKind == "" {
		result.FailureKind = FailureKindTimeout
	}
	return result
}

func RankSurvivors(goos string, results []Result) []RankedSurvivor {
	normalized := make([]Result, len(results))
	survivors := make([]int, 0, len(results))
	groupSizes := map[string]int{}
	for i := range results {
		normalized[i] = NormalizeResult(results[i])
		if normalized[i].Status != StatusSurvived {
			continue
		}
		survivors = append(survivors, i)
		if key := normalized[i].Mutant.SemanticGroup; key != "" {
			groupSizes[key]++
		}
	}

	sort.SliceStable(survivors, func(i, j int) bool {
		left := normalized[survivors[i]]
		right := normalized[survivors[j]]
		leftScore, _ := survivorRankScore(goos, left, groupSizes[left.Mutant.SemanticGroup])
		rightScore, _ := survivorRankScore(goos, right, groupSizes[right.Mutant.SemanticGroup])
		if leftScore != rightScore {
			return leftScore > rightScore
		}
		if riskPriority(left.Mutant.EquivalentRisk) != riskPriority(right.Mutant.EquivalentRisk) {
			return riskPriority(left.Mutant.EquivalentRisk) < riskPriority(right.Mutant.EquivalentRisk)
		}
		if RecommendationPriority(left.Mutant.Recommendation) != RecommendationPriority(right.Mutant.Recommendation) {
			return RecommendationPriority(left.Mutant.Recommendation) < RecommendationPriority(right.Mutant.Recommendation)
		}
		if len(left.Mutant.NearbyTests) != len(right.Mutant.NearbyTests) {
			return len(left.Mutant.NearbyTests) > len(right.Mutant.NearbyTests)
		}
		return left.MutantID < right.MutantID
	})

	ranked := make([]RankedSurvivor, 0, len(survivors))
	for rank, idx := range survivors {
		groupSize := groupSizes[normalized[idx].Mutant.SemanticGroup]
		score, reason := survivorRankScore(goos, normalized[idx], groupSize)
		skip := normalized[idx].SuggestedSkip
		if skip == "" && groupSize > 1 {
			skip = "review once for this semantic group before treating each survivor independently"
		}
		ranked = append(ranked, RankedSurvivor{
			MutantID:           normalized[idx].MutantID,
			SurvivorRank:       rank + 1,
			RankScore:          score,
			RankReason:         reason,
			Actionability:      actionability(score),
			SuggestedTestScope: SuggestedTestScope(normalized[idx].Mutant),
			SuggestedSkip:      skip,
			SemanticGroupSize:  groupSize,
			NearestTests:       append([]string{}, normalized[idx].Mutant.NearbyTests...),
		})
	}
	return ranked
}

func IsActionableSurvivor(goos string, result Result) bool {
	if result.Status != StatusSurvived {
		return false
	}
	if result.Actionability == "low" {
		return false
	}
	if strings.EqualFold(goos, "windows") && result.Mutant.PlatformSensitive {
		return false
	}
	if result.Mutant.NonProgressRisk == "high" {
		return false
	}
	return true
}

func BuildActionableSummary(goos string, rawScore float64, killed int, results []Result) ActionableSummary {
	summary := ActionableSummary{RawScore: rawScore}
	seenGroups := map[string]bool{}
	countedActionableGroups := map[string]bool{}
	for _, result := range results {
		normalized := NormalizeResult(result)
		switch normalized.Status {
		case StatusSurvived:
			summary.Survivors++
			group := normalized.Mutant.SemanticGroup
			if group != "" {
				if seenGroups[group] {
					summary.CollapsedSemanticDuplicates++
				} else {
					seenGroups[group] = true
					summary.SemanticGroupReviewUnits++
				}
			}
			if strings.EqualFold(normalized.Mutant.EquivalentRisk, "high") || group != "" {
				summary.EquivalentRiskSurvivors++
			}
			if normalized.Mutant.PlatformSensitive {
				summary.PlatformSensitiveSurvivors++
			}
			if IsActionableSurvivor(goos, normalized) {
				summary.ActionableSurvivors++
				if group == "" {
					summary.TrueActionableSurvivors++
				} else if !countedActionableGroups[group] {
					countedActionableGroups[group] = true
					summary.TrueActionableSurvivors++
				}
			}
		case StatusTimedOut:
			if normalized.FailureKind == FailureKindNonProgressLoop {
				summary.NonProgressTimeouts++
			}
		}
	}
	denominator := killed + summary.TrueActionableSurvivors
	if denominator > 0 {
		summary.ActionableScore = float64(killed) / float64(denominator) * 100
	}
	return summary
}

func survivorRankScore(goos string, result Result, groupSize int) (float64, string) {
	score := 100.0
	risk := result.Mutant.EquivalentRisk
	switch risk {
	case "low":
		score += 20
	case "medium":
		score += 5
	case "high":
		score -= 25
	default:
		score -= 10
	}
	switch result.Mutant.Recommendation {
	case "fast-ci":
		score += 20
	case "conservative":
		score += 12
	case "default":
		score += 4
	case "aggressive":
		score -= 12
	}
	if len(result.Mutant.NearbyTests) > 0 {
		score += 12
	}
	if result.Mutant.Function != "" && strings.HasPrefix(result.Mutant.Function, strings.ToUpper(result.Mutant.Function[:1])) {
		score += 5
	}
	if result.CoverageSource != "" && result.CoverageSource != "unknown" {
		score += 6
	}
	switch result.HistoryStatus {
	case "new_survivor":
		score += 18
	case "long_standing_survivor":
		score += 10
	case "existing_survivor":
		score += 4
	}
	if result.OperatorYield > 0 {
		score += result.OperatorYield * 10
	}
	for _, audit := range result.Mutant.SuppressionAudit {
		if audit.Action == "lower-priority" || audit.Action == "report-only" {
			score -= 8
		}
	}
	if result.Mutant.SemanticGroup != "" {
		score -= 6
	}
	if groupSize > 1 {
		score -= float64(minInt(18, (groupSize-1)*6))
	}
	if strings.EqualFold(goos, "windows") && result.Mutant.PlatformSensitive {
		score -= 20
	}
	if result.Mutant.NonProgressRisk == "high" {
		score -= 24
	}
	reason := fmt.Sprintf("score=%.1f risk=%s recommendation=%s coverage_source=%s nearby_tests=%d history=%s survivor_age_runs=%d operator_yield=%.2f semantic_group=%s group_size=%d platform_sensitive=%t non_progress_risk=%s", score, risk, result.Mutant.Recommendation, result.CoverageSource, len(result.Mutant.NearbyTests), result.HistoryStatus, result.SurvivorAgeRuns, result.OperatorYield, result.Mutant.GroupLabel, groupSize, result.Mutant.PlatformSensitive, result.Mutant.NonProgressRisk)
	return score, reason
}

func actionability(score float64) string {
	switch {
	case score >= 125:
		return "high"
	case score >= 95:
		return "medium"
	default:
		return "low"
	}
}

func SuggestedTestScope(mutant Mutant) string {
	if mutant.Package != "" && mutant.Package != "." {
		return mutant.Package
	}
	if len(mutant.NearbyTests) > 0 {
		return filepath.ToSlash(filepath.Dir(mutant.NearbyTests[0]))
	}
	return "."
}

func riskPriority(risk string) int {
	switch risk {
	case "low":
		return 0
	case "medium":
		return 1
	case "high":
		return 2
	default:
		return 3
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
