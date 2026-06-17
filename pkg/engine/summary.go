package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/cervantesh/cervo-mutants/pkg/config"
	"github.com/cervantesh/cervo-mutants/pkg/triage"
)

func summarize(results []MutantResult) Summary {
	s := Summary{MutatorStats: map[string]MutatorStat{}, EquivalentRiskStats: map[string]int{}, TimeoutRiskStats: map[string]int{}, SemanticGroupStats: map[string]int{}}
	s.Total = len(results)
	s.GeneratedMutants = len(results)
	triageResults := make([]triage.Result, 0, len(results))
	for _, result := range results {
		applySemanticResultMetadata(&result)
		triageResults = append(triageResults, triageResult(result))
		operator := result.Mutant.Operator
		if operator == "" {
			operator = "unknown"
		}
		stat := s.MutatorStats[operator]
		stat.Total++
		risk := result.Mutant.EquivalentRisk
		if risk == "" {
			risk = "unknown"
		}
		s.EquivalentRiskStats[risk]++
		s.TimeoutRiskStats[timeoutRiskBand(result.Mutant)]++
		if stat.Recommendation == "" {
			stat.Recommendation = result.Mutant.Recommendation
		}
		if stat.EquivalentRisk == "" {
			stat.EquivalentRisk = result.Mutant.EquivalentRisk
		}
		status := result.Status
		if status == StatusCached {
			status = result.PreviousStatus
		}
		if status == StatusSurvived {
			if result.Mutant.PlatformSensitive {
				s.PlatformSensitiveSurvivors++
			}
			if key := semanticGroupSummaryKey(result.Mutant); key != "" {
				s.SemanticGroupStats[key]++
			}
		}
		if status == StatusTimedOut && result.FailureKind == "non_progress_loop" {
			s.NonProgressTimeouts++
		}
		applyStatusToSummary(&s, &stat, result)
		applySuppressionAudits(&s, result.Mutant.SuppressionAudit)
		s.MutatorStats[operator] = stat
	}
	eligible := s.Total - s.Ignored - s.Quarantined - s.Skipped - s.SkippedResource - s.PendingBudget - s.NotCovered
	s.EffectiveMutants = s.Killed + s.Survived
	s.ScoreDenominator = eligible
	if eligible > 0 {
		s.Score = float64(s.Killed) / float64(eligible) * 100
	}
	if s.EffectiveMutants > 0 {
		s.EffectiveScore = float64(s.Killed) / float64(s.EffectiveMutants) * 100
		s.TestEfficacy = s.EffectiveScore
	}
	coverable := s.Total - s.Ignored - s.Quarantined - s.Skipped - s.SkippedResource - s.PendingBudget
	if coverable > 0 {
		s.MutationCoverage = float64(coverable-s.NotCovered) / float64(coverable) * 100
	}
	actionable := triage.BuildActionableSummary(runtime.GOOS, s.Score, s.Killed, triageResults)
	s.Actionable = ActionableSummary{
		RawScore:                    actionable.RawScore,
		ActionableScore:             actionable.ActionableScore,
		Survivors:                   actionable.Survivors,
		ActionableSurvivors:         actionable.ActionableSurvivors,
		TrueActionableSurvivors:     actionable.TrueActionableSurvivors,
		EquivalentRiskSurvivors:     actionable.EquivalentRiskSurvivors,
		PlatformSensitiveSurvivors:  actionable.PlatformSensitiveSurvivors,
		NonProgressTimeouts:         actionable.NonProgressTimeouts,
		SemanticGroupReviewUnits:    actionable.SemanticGroupReviewUnits,
		CollapsedSemanticDuplicates: actionable.CollapsedSemanticDuplicates,
	}
	s.DenominatorHealth = denominatorHealth(s)
	return s
}

func applyStatusToSummary(s *Summary, stat *MutatorStat, result MutantResult) {
	status := result.Status
	if status == StatusCached {
		s.Cached++
		stat.Cached++
		status = result.PreviousStatus
	}
	switch status {
	case StatusKilled:
		s.Killed++
		stat.Killed++
		s.ExecutedMutants++
		s.CoveredMutants++
	case StatusSurvived:
		s.Survived++
		stat.Survived++
		s.ExecutedMutants++
		s.CoveredMutants++
		if result.Mutant.EquivalentRisk == "high" {
			s.HighRiskSurvivors++
		}
	case StatusNotCovered:
		s.NotCovered++
		stat.NotCovered++
	case StatusTimedOut:
		s.TimedOut++
		stat.TimedOut++
		s.ExecutedMutants++
		s.CoveredMutants++
	case StatusMemoryKilled:
		s.MemoryKilled++
		stat.MemoryKilled++
		s.ExecutedMutants++
		s.CoveredMutants++
	case StatusCompileError:
		s.CompileError++
		stat.CompileError++
		s.ExecutedMutants++
		s.CoveredMutants++
	case StatusSkipped:
		s.Skipped++
		stat.Skipped++
	case StatusSkippedResource:
		s.SkippedResource++
		stat.SkippedResource++
	case StatusPendingBudget:
		s.PendingBudget++
		stat.PendingBudget++
	case StatusIgnored:
		s.Ignored++
		stat.Ignored++
	case StatusQuarantined:
		s.Quarantined++
		stat.Quarantined++
	}
}

func applySuppressionAudits(s *Summary, audits []SuppressionAudit) {
	for _, audit := range audits {
		switch audit.Action {
		case config.SuppressionReportOnly:
			s.SuppressionReportOnly++
		case config.SuppressionLowerPriority:
			s.SuppressionLowerPriority++
		case "suppress":
			s.SuppressionSuppressed++
		case "quarantine-required":
			s.SuppressionQuarantineRequired++
		}
	}
}

func denominatorHealth(s Summary) DenominatorHealth {
	health := DenominatorHealth{
		Generated:        s.GeneratedMutants,
		Covered:          s.CoveredMutants,
		Executed:         s.ExecutedMutants,
		Effective:        s.EffectiveMutants,
		ScoreDenominator: s.ScoreDenominator,
		Killed:           s.Killed,
		Survived:         s.Survived,
		NotCovered:       s.NotCovered,
		TimedOut:         s.TimedOut,
		MemoryKilled:     s.MemoryKilled,
		CompileError:     s.CompileError,
		Skipped:          s.Skipped,
		SkippedResource:  s.SkippedResource,
		PendingBudget:    s.PendingBudget,
		Ignored:          s.Ignored,
		Quarantined:      s.Quarantined,
		Healthy:          true,
	}
	if health.Generated > 0 && health.Effective == 0 {
		health.Warnings = append(health.Warnings, "no_effective_mutants")
	}
	if health.Effective > 0 && health.TimedOut > health.Effective {
		health.Warnings = append(health.Warnings, "timed_out_exceeds_effective")
	}
	if health.Effective > 0 && health.MemoryKilled > health.Effective {
		health.Warnings = append(health.Warnings, "memory_killed_exceeds_effective")
	}
	if health.Effective > 0 && health.NotCovered > health.Effective {
		health.Warnings = append(health.Warnings, "not_covered_exceeds_effective")
	}
	if health.Effective > 0 && health.ScoreDenominator > health.Effective*2 {
		health.Warnings = append(health.Warnings, "score_denominator_dwarfs_effective")
	}
	if health.Effective > 0 && (health.TimedOut+health.MemoryKilled+health.SkippedResource+health.PendingBudget) > health.Effective {
		health.Warnings = append(health.Warnings, "resource_limited_exceeds_effective")
	}
	if health.Effective > 0 && s.TestEfficacy >= 90 && (health.TimedOut > health.Effective || health.NotCovered > health.Effective) {
		health.Warnings = append(health.Warnings, "high_score_poor_denominator_health")
	}
	health.Healthy = len(health.Warnings) == 0
	return health
}

func memoryLimitExceeded(err error, state *os.ProcessState, resources config.Resources, output string) bool {
	if err == nil || resources.MaxProcessMemoryMB <= 0 {
		return false
	}
	text := strings.ToLower(output)
	if strings.Contains(text, "out of memory") {
		return true
	}
	if state == nil {
		return false
	}
	if strings.Contains(text, "panic:") || strings.Contains(text, "build failed") || strings.Contains(text, "setup failed") || strings.Contains(text, "undefined:") || strings.Contains(text, "syntax error") || strings.Contains(text, "fail") {
		return false
	}
	return state.ExitCode() == 2
}

func runStopMetadata(results []MutantResult) (string, string) {
	if len(results) == 0 {
		return "", ""
	}
	if hasStatus(results, StatusPendingBudget) {
		return "budget_exhausted", lastCompletedMutant(results)
	}
	if allStatus(results, StatusSkippedResource) {
		return "resource_limits_unavailable", ""
	}
	return "", ""
}

func hasStatus(results []MutantResult, status Status) bool {
	for _, result := range results {
		if result.Status == status {
			return true
		}
	}
	return false
}

func allStatus(results []MutantResult, status Status) bool {
	if len(results) == 0 {
		return false
	}
	for _, result := range results {
		if result.Status != status {
			return false
		}
	}
	return true
}

func lastCompletedMutant(results []MutantResult) string {
	for i := len(results) - 1; i >= 0; i-- {
		switch results[i].Status {
		case StatusPendingBudget, StatusSkippedResource:
			continue
		default:
			if results[i].MutantID != "" {
				return results[i].MutantID
			}
		}
	}
	return ""
}

func (e *Engine) writeReports(result RunResult) error {
	if e.cfg.Reports.Output == "" {
		return nil
	}
	if err := os.MkdirAll(e.cfg.Reports.Output, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(e.cfg.Reports.Output, "mutation-report.json"), data, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(e.cfg.Reports.Output, "summary.txt"), []byte("Mutation score generated by cervomut\n"), 0o644)
}
