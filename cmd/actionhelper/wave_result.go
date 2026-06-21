package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/cervantesh/cervo-mutants/pkg/engine"
	reportpkg "github.com/cervantesh/cervo-mutants/pkg/report"
)

type waveReportArtifacts struct {
	runResult  engine.RunResult
	ledger     reportpkg.TriageLedger
	governance reportpkg.GovernanceReview
}

func buildWaveResult(meta waveMetadata) (waveResult, error) {
	if meta.GeneratedAt == "" {
		meta.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	}
	reportPath, reportKind := selectWaveReport(meta.ReportDir)
	failure, err := failureFromDebugFile(filepath.Join(meta.ReportDir, "failure-debug.json"))
	if err != nil {
		return waveResult{}, err
	}

	result := newWaveResult(meta, reportKind, failure)
	if reportPath == "" {
		return result, nil
	}
	result.ReportPath = reportPath

	artifacts, err := loadWaveReportArtifacts(meta.ReportDir, reportPath)
	if err != nil {
		return waveResult{}, err
	}
	applyWaveReportArtifacts(&result, artifacts)
	return result, nil
}

func newWaveResult(meta waveMetadata, reportKind string, failure *engine.Failure) waveResult {
	return waveResult{
		GeneratedAt:        meta.GeneratedAt,
		Name:               meta.Name,
		Repository:         meta.Repository,
		InstallPath:        meta.InstallPath,
		ActionRef:          meta.ActionRef,
		Ref:                meta.Ref,
		Target:             meta.Target,
		Profile:            meta.Profile,
		GoVersion:          meta.GoVersion,
		GoVersionTarget:    nullableString(meta.GoVersionTarget),
		GoVersionActionMin: meta.GoVersionActionMin,
		GoVersionRequested: nullableString(meta.GoVersionRequested),
		Policy:             meta.Policy,
		Budget:             meta.Budget,
		Sample:             meta.Sample,
		CoveragePrefilter:  meta.CoveragePrefilter,
		PrewarmModules:     meta.PrewarmModules,
		MaxMutants:         meta.MaxMutants,
		Workers:            meta.Workers,
		JobStatus:          meta.JobStatus,
		ReportKind:         reportKind,
		ReportDir:          meta.ReportDir,
		Triage: waveTriage{
			GovernanceSuggestionsByStatus: map[string]int{},
		},
		Failure: failure,
	}
}

func loadWaveReportArtifacts(reportDir, reportPath string) (waveReportArtifacts, error) {
	runResult, err := readRunResult(reportPath)
	if err != nil {
		return waveReportArtifacts{}, err
	}
	ledger, err := readTriageLedger(filepath.Join(reportDir, "semantic-triage-ledger.json"))
	if err != nil {
		return waveReportArtifacts{}, err
	}
	governance, err := readGovernanceReview(filepath.Join(reportDir, "governance-review.json"))
	if err != nil {
		return waveReportArtifacts{}, err
	}
	return waveReportArtifacts{
		runResult:  runResult,
		ledger:     ledger,
		governance: governance,
	}, nil
}

func applyWaveReportArtifacts(result *waveResult, artifacts waveReportArtifacts) {
	result.Summary = buildWaveResultSummary(artifacts.runResult)
	result.Triage = buildWaveTriage(artifacts.runResult, artifacts.ledger, artifacts.governance)
	result.DenominatorHealth = cloneDenominatorHealth(artifacts.runResult.Summary.DenominatorHealth)
	result.Environment = cloneEnvironment(artifacts.runResult.Environment)
	if artifacts.runResult.Failure != nil {
		result.Failure = cloneFailure(*artifacts.runResult.Failure)
	}
}

func buildWaveResultSummary(runResult engine.RunResult) *waveResultSummary {
	return &waveResultSummary{
		Total:            runResult.Summary.Total,
		Killed:           runResult.Summary.Killed,
		Survived:         runResult.Summary.Survived,
		NotCovered:       runResult.Summary.NotCovered,
		TimedOut:         runResult.Summary.TimedOut,
		CompileError:     runResult.Summary.CompileError,
		Score:            runResult.Summary.Score,
		ActionableScore:  nullableFloat64(runResult.Summary.Actionable.ActionableScore),
		MutationCoverage: runResult.Summary.MutationCoverage,
		TestEfficacy:     runResult.Summary.TestEfficacy,
	}
}

func buildWaveTriage(runResult engine.RunResult, ledger reportpkg.TriageLedger, governance reportpkg.GovernanceReview) waveTriage {
	recommendationEntries, recommendationReviewUnits := recommendationTriageCounts(runResult)
	governanceSuggestionsByStatus := governanceSuggestionsByStatus(governance)
	triage := waveTriage{
		ActionableReviewUnits:          runResult.Summary.Actionable.TrueActionableSurvivors,
		ActionableSurvivors:            runResult.Summary.Actionable.ActionableSurvivors,
		EquivalentRiskSurvivors:        runResult.Summary.Actionable.EquivalentRiskSurvivors,
		SemanticGroupReviewUnits:       runResult.Summary.Actionable.SemanticGroupReviewUnits,
		CollapsedSemanticDuplicates:    runResult.Summary.Actionable.CollapsedSemanticDuplicates,
		SemanticGroupCount:             len(runResult.Summary.SemanticGroupStats),
		RecommendationEntries:          recommendationEntries,
		RecommendationReviewUnits:      recommendationReviewUnits,
		CollapsedRecommendationDupes:   recommendationEntries - recommendationReviewUnits,
		LedgerEntries:                  len(ledger.Entries),
		GovernanceQuarantineTemplates:  len(governance.QuarantineTemplates),
		GovernanceSuppressionTemplates: len(governance.SuppressionTemplates),
		GovernanceSuggestionsByStatus:  governanceSuggestionsByStatus,
		GovernanceTotalSuggestions:     len(governance.QuarantineTemplates) + len(governance.SuppressionTemplates),
	}
	if len(governanceSuggestionsByStatus) == 0 {
		triage.GovernanceSuggestionsByStatus = map[string]int{}
	}
	return triage
}

func recommendationTriageCounts(runResult engine.RunResult) (entries int, reviewUnits int) {
	groups := map[string]struct{}{}
	for _, mutant := range runResult.Mutants {
		if mutant.TestRecommendation == nil {
			continue
		}
		entries++
		if mutant.Mutant.SemanticGroup == "" {
			groups[fmt.Sprintf("mutant:%s", mutant.MutantID)] = struct{}{}
			continue
		}
		groups[mutant.Mutant.SemanticGroup] = struct{}{}
	}
	return entries, len(groups)
}

func governanceSuggestionsByStatus(review reportpkg.GovernanceReview) map[string]int {
	counts := map[string]int{}
	for _, item := range review.QuarantineTemplates {
		counts[string(item.Status)]++
	}
	for _, item := range review.SuppressionTemplates {
		counts[string(item.Status)]++
	}
	return counts
}
