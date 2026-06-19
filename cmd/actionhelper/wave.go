package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cervantesh/cervo-mutants/pkg/engine"
	reportpkg "github.com/cervantesh/cervo-mutants/pkg/report"
)

type waveMetadata struct {
	GeneratedAt        string
	Name               string
	Repository         string
	InstallPath        string
	ActionRef          string
	Ref                string
	Target             string
	Profile            string
	GoVersion          string
	GoVersionTarget    string
	GoVersionActionMin string
	GoVersionRequested string
	Policy             string
	Budget             string
	Sample             string
	CoveragePrefilter  bool
	ReportDir          string
	JobStatus          string
	MaxMutants         int
	Workers            int
}

type waveResult struct {
	GeneratedAt        string                    `json:"generated_at"`
	Name               string                    `json:"name"`
	Repository         string                    `json:"repository"`
	InstallPath        string                    `json:"install_path"`
	ActionRef          string                    `json:"action_ref"`
	Ref                string                    `json:"ref"`
	Target             string                    `json:"target"`
	Profile            string                    `json:"profile"`
	GoVersion          string                    `json:"go_version"`
	GoVersionTarget    *string                   `json:"go_version_target"`
	GoVersionActionMin string                    `json:"go_version_action_min"`
	GoVersionRequested *string                   `json:"go_version_requested"`
	Policy             string                    `json:"policy"`
	Budget             string                    `json:"budget"`
	Sample             string                    `json:"sample"`
	CoveragePrefilter  bool                      `json:"coverage_prefilter"`
	MaxMutants         int                       `json:"max_mutants"`
	Workers            int                       `json:"workers"`
	JobStatus          string                    `json:"job_status"`
	ReportKind         string                    `json:"report_kind"`
	ReportDir          string                    `json:"report_dir"`
	ReportPath         string                    `json:"report_path,omitempty"`
	Summary            *waveResultSummary        `json:"summary,omitempty"`
	Triage             waveTriage                `json:"triage"`
	DenominatorHealth  *engine.DenominatorHealth `json:"denominator_health,omitempty"`
	Environment        *engine.Environment       `json:"environment,omitempty"`
	Failure            *engine.Failure           `json:"failure"`
}

type waveResultSummary struct {
	Total            int      `json:"total"`
	Killed           int      `json:"killed"`
	Survived         int      `json:"survived"`
	NotCovered       int      `json:"not_covered"`
	TimedOut         int      `json:"timed_out"`
	CompileError     int      `json:"compile_error"`
	Score            float64  `json:"score"`
	ActionableScore  *float64 `json:"actionable_score"`
	MutationCoverage float64  `json:"mutation_coverage"`
	TestEfficacy     float64  `json:"test_efficacy"`
}

type waveTriage struct {
	ActionableReviewUnits          int            `json:"actionable_review_units"`
	ActionableSurvivors            int            `json:"actionable_survivors"`
	EquivalentRiskSurvivors        int            `json:"equivalent_risk_survivors"`
	SemanticGroupReviewUnits       int            `json:"semantic_group_review_units"`
	CollapsedSemanticDuplicates    int            `json:"collapsed_semantic_duplicates"`
	SemanticGroupCount             int            `json:"semantic_group_count"`
	RecommendationEntries          int            `json:"recommendation_entries"`
	RecommendationReviewUnits      int            `json:"recommendation_review_units"`
	CollapsedRecommendationDupes   int            `json:"collapsed_recommendation_duplicates"`
	LedgerEntries                  int            `json:"ledger_entries"`
	GovernanceQuarantineTemplates  int            `json:"governance_quarantine_templates"`
	GovernanceSuppressionTemplates int            `json:"governance_suppression_templates"`
	GovernanceSuggestionsByStatus  map[string]int `json:"governance_suggestions_by_status"`
	GovernanceTotalSuggestions     int            `json:"governance_total_suggestions"`
}

type waveSummary struct {
	SchemaVersion string            `json:"schema_version"`
	TrackingIssue string            `json:"tracking_issue,omitempty"`
	ManifestPath  string            `json:"manifest_path,omitempty"`
	ActionRef     string            `json:"action_ref,omitempty"`
	InstallPath   string            `json:"install_path,omitempty"`
	GeneratedAt   string            `json:"generated_at"`
	Repos         []waveResult      `json:"repos"`
	Aggregate     waveAggregate     `json:"aggregate"`
	Triage        waveSummaryTriage `json:"triage"`
}

type waveAggregate struct {
	Selected       int            `json:"selected"`
	Reports        int            `json:"reports"`
	MissingReports int            `json:"missing_reports"`
	Generated      int            `json:"generated"`
	Covered        int            `json:"covered"`
	Executed       int            `json:"executed"`
	Effective      int            `json:"effective"`
	Killed         int            `json:"killed"`
	Survived       int            `json:"survived"`
	NotCovered     int            `json:"not_covered"`
	TimedOut       int            `json:"timed_out"`
	CompileError   int            `json:"compile_error"`
	WarningRepos   int            `json:"warning_repos"`
	FailedReports  int            `json:"failed_reports"`
	FailureKinds   map[string]int `json:"failure_kinds"`
}

type waveSummaryTriage struct {
	ActionableReviewUnits              int            `json:"actionable_review_units"`
	ActionableSurvivors                int            `json:"actionable_survivors"`
	EquivalentRiskSurvivors            int            `json:"equivalent_risk_survivors"`
	SemanticGroupReviewUnits           int            `json:"semantic_group_review_units"`
	CollapsedSemanticDuplicates        int            `json:"collapsed_semantic_duplicates"`
	SemanticGroupCount                 int            `json:"semantic_group_count"`
	RecommendationEntries              int            `json:"recommendation_entries"`
	RecommendationReviewUnits          int            `json:"recommendation_review_units"`
	CollapsedRecommendationDupes       int            `json:"collapsed_recommendation_duplicates"`
	LedgerEntries                      int            `json:"ledger_entries"`
	GovernanceQuarantineTemplates      int            `json:"governance_quarantine_templates"`
	GovernanceSuppressionTemplates     int            `json:"governance_suppression_templates"`
	GovernanceSuggestionsByStatus      map[string]int `json:"governance_suggestions_by_status"`
	GovernanceTotalSuggestions         int            `json:"governance_total_suggestions"`
	ReposWithActionableReviewUnits     int            `json:"repos_with_actionable_review_units"`
	ReposWithSemanticGroups            int            `json:"repos_with_semantic_groups"`
	ReposWithRecommendations           int            `json:"repos_with_recommendations"`
	ReposWithRecommendationReviewUnits int            `json:"repos_with_recommendation_review_units"`
	ReposWithLedgerEntries             int            `json:"repos_with_ledger_entries"`
	ReposWithGovernanceSuggestions     int            `json:"repos_with_governance_suggestions"`
}

func cmdBuildWaveResult(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("build-wave-result", flag.ContinueOnError)
	var meta waveMetadata
	coveragePrefilter := fs.String("coverage-prefilter", "false", "whether coverage prefilter is enabled")
	fs.StringVar(&meta.GeneratedAt, "generated-at", "", "wave result generation timestamp")
	fs.StringVar(&meta.Name, "name", "", "matrix repo name")
	fs.StringVar(&meta.Repository, "repository", "", "repository full name")
	fs.StringVar(&meta.InstallPath, "install-path", "", "install path used for the wave")
	fs.StringVar(&meta.ActionRef, "action-ref", "", "action ref used for the wave")
	fs.StringVar(&meta.Ref, "ref", "", "target repository ref")
	fs.StringVar(&meta.Target, "target", "", "mutation target")
	fs.StringVar(&meta.Profile, "profile", "", "profile name")
	fs.StringVar(&meta.GoVersion, "go-version", "", "resolved Go version")
	fs.StringVar(&meta.GoVersionTarget, "go-version-target", "", "target repository Go version")
	fs.StringVar(&meta.GoVersionActionMin, "go-version-action-min", "", "minimum Go version required by the action source")
	fs.StringVar(&meta.GoVersionRequested, "go-version-requested", "", "requested Go version from the manifest")
	fs.StringVar(&meta.Policy, "policy", "", "policy")
	fs.StringVar(&meta.Budget, "budget", "", "budget")
	fs.StringVar(&meta.Sample, "sample", "", "sample")
	fs.StringVar(&meta.ReportDir, "report-dir", "", "report directory")
	fs.StringVar(&meta.JobStatus, "job-status", "", "job status")
	fs.IntVar(&meta.MaxMutants, "max-mutants", 0, "max mutants")
	fs.IntVar(&meta.Workers, "workers", 0, "workers")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	meta.CoveragePrefilter = strings.EqualFold(strings.TrimSpace(*coveragePrefilter), "true")
	result, err := buildWaveResult(meta)
	if err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(result)
}

func cmdRenderWaveResultMarkdown(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("render-wave-result-markdown", flag.ContinueOnError)
	path := fs.String("path", "", "path to wave-result.json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	var result waveResult
	if err := readJSONFile(*path, &result); err != nil {
		return err
	}
	_, err := io.WriteString(stdout, renderWaveResultMarkdown(result))
	return err
}

func cmdBuildWaveSummary(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("build-wave-summary", flag.ContinueOnError)
	trackingIssue := fs.String("tracking-issue", "", "tracking issue")
	manifestPath := fs.String("manifest-path", "", "manifest path")
	actionRef := fs.String("action-ref", "", "action ref")
	installPath := fs.String("install-path", "", "install path")
	artifactsDir := fs.String("artifacts-dir", "", "downloaded artifact directory")
	generatedAt := fs.String("generated-at", "", "summary generation timestamp")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	summary, err := buildWaveSummary(*trackingIssue, *manifestPath, *actionRef, *installPath, *artifactsDir, *generatedAt)
	if err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(summary)
}

func cmdRenderWaveSummaryMarkdown(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("render-wave-summary-markdown", flag.ContinueOnError)
	path := fs.String("path", "", "path to wave-summary.json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	var summary waveSummary
	if err := readJSONFile(*path, &summary); err != nil {
		return err
	}
	_, err := io.WriteString(stdout, renderWaveSummaryMarkdown(summary))
	return err
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

	result := waveResult{
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
	if reportPath == "" {
		return result, nil
	}
	result.ReportPath = reportPath

	runResult, err := readRunResult(reportPath)
	if err != nil {
		return waveResult{}, err
	}
	ledger, err := readTriageLedger(filepath.Join(meta.ReportDir, "semantic-triage-ledger.json"))
	if err != nil {
		return waveResult{}, err
	}
	governance, err := readGovernanceReview(filepath.Join(meta.ReportDir, "governance-review.json"))
	if err != nil {
		return waveResult{}, err
	}

	recommendationEntries := 0
	recommendationGroups := map[string]struct{}{}
	for _, mutant := range runResult.Mutants {
		if mutant.TestRecommendation == nil {
			continue
		}
		recommendationEntries++
		if mutant.Mutant.SemanticGroup == "" {
			recommendationGroups[fmt.Sprintf("mutant:%s", mutant.MutantID)] = struct{}{}
			continue
		}
		recommendationGroups[mutant.Mutant.SemanticGroup] = struct{}{}
	}

	governanceSuggestionsByStatus := map[string]int{}
	for _, item := range governance.QuarantineTemplates {
		governanceSuggestionsByStatus[string(item.Status)]++
	}
	for _, item := range governance.SuppressionTemplates {
		governanceSuggestionsByStatus[string(item.Status)]++
	}

	result.Summary = &waveResultSummary{
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
	result.Triage = waveTriage{
		ActionableReviewUnits:          runResult.Summary.Actionable.TrueActionableSurvivors,
		ActionableSurvivors:            runResult.Summary.Actionable.ActionableSurvivors,
		EquivalentRiskSurvivors:        runResult.Summary.Actionable.EquivalentRiskSurvivors,
		SemanticGroupReviewUnits:       runResult.Summary.Actionable.SemanticGroupReviewUnits,
		CollapsedSemanticDuplicates:    runResult.Summary.Actionable.CollapsedSemanticDuplicates,
		SemanticGroupCount:             len(runResult.Summary.SemanticGroupStats),
		RecommendationEntries:          recommendationEntries,
		RecommendationReviewUnits:      len(recommendationGroups),
		CollapsedRecommendationDupes:   recommendationEntries - len(recommendationGroups),
		LedgerEntries:                  len(ledger.Entries),
		GovernanceQuarantineTemplates:  len(governance.QuarantineTemplates),
		GovernanceSuppressionTemplates: len(governance.SuppressionTemplates),
		GovernanceSuggestionsByStatus:  governanceSuggestionsByStatus,
		GovernanceTotalSuggestions:     len(governance.QuarantineTemplates) + len(governance.SuppressionTemplates),
	}
	if len(governanceSuggestionsByStatus) == 0 {
		result.Triage.GovernanceSuggestionsByStatus = map[string]int{}
	}
	result.DenominatorHealth = cloneDenominatorHealth(runResult.Summary.DenominatorHealth)
	result.Environment = cloneEnvironment(runResult.Environment)
	if runResult.Failure != nil {
		result.Failure = cloneFailure(*runResult.Failure)
	}
	return result, nil
}

func buildWaveSummary(trackingIssue, manifestPath, actionRef, installPath, artifactsDir, generatedAt string) (waveSummary, error) {
	if generatedAt == "" {
		generatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	paths, err := filepath.Glob(filepath.Join(artifactsDir, "*", "wave-result.json"))
	if err != nil {
		return waveSummary{}, err
	}
	if len(paths) == 0 {
		return waveSummary{}, fmt.Errorf("no wave-result.json files found under %s", artifactsDir)
	}
	results := make([]waveResult, 0, len(paths))
	for _, path := range paths {
		var result waveResult
		if err := readJSONFile(path, &result); err != nil {
			return waveSummary{}, err
		}
		results = append(results, result)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

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

	summary.Aggregate.Selected = len(results)
	for _, result := range results {
		if result.ReportKind == "missing" {
			summary.Aggregate.MissingReports++
		} else {
			summary.Aggregate.Reports++
		}
		if result.DenominatorHealth != nil {
			summary.Aggregate.Generated += result.DenominatorHealth.Generated
			summary.Aggregate.Covered += result.DenominatorHealth.Covered
			summary.Aggregate.Executed += result.DenominatorHealth.Executed
			summary.Aggregate.Effective += result.DenominatorHealth.Effective
			if len(result.DenominatorHealth.Warnings) > 0 {
				summary.Aggregate.WarningRepos++
			}
		}
		if result.Summary != nil {
			summary.Aggregate.Killed += result.Summary.Killed
			summary.Aggregate.Survived += result.Summary.Survived
			summary.Aggregate.NotCovered += result.Summary.NotCovered
			summary.Aggregate.TimedOut += result.Summary.TimedOut
			summary.Aggregate.CompileError += result.Summary.CompileError
		}
		if result.Failure != nil && result.Failure.Kind != "" {
			summary.Aggregate.FailedReports++
			summary.Aggregate.FailureKinds[result.Failure.Kind]++
		}

		summary.Triage.ActionableReviewUnits += result.Triage.ActionableReviewUnits
		summary.Triage.ActionableSurvivors += result.Triage.ActionableSurvivors
		summary.Triage.EquivalentRiskSurvivors += result.Triage.EquivalentRiskSurvivors
		summary.Triage.SemanticGroupReviewUnits += result.Triage.SemanticGroupReviewUnits
		summary.Triage.CollapsedSemanticDuplicates += result.Triage.CollapsedSemanticDuplicates
		summary.Triage.SemanticGroupCount += result.Triage.SemanticGroupCount
		summary.Triage.RecommendationEntries += result.Triage.RecommendationEntries
		summary.Triage.RecommendationReviewUnits += result.Triage.RecommendationReviewUnits
		summary.Triage.CollapsedRecommendationDupes += result.Triage.CollapsedRecommendationDupes
		summary.Triage.LedgerEntries += result.Triage.LedgerEntries
		summary.Triage.GovernanceQuarantineTemplates += result.Triage.GovernanceQuarantineTemplates
		summary.Triage.GovernanceSuppressionTemplates += result.Triage.GovernanceSuppressionTemplates
		summary.Triage.GovernanceTotalSuggestions += result.Triage.GovernanceTotalSuggestions
		for key, value := range result.Triage.GovernanceSuggestionsByStatus {
			summary.Triage.GovernanceSuggestionsByStatus[key] += value
		}
		if result.Triage.ActionableReviewUnits > 0 {
			summary.Triage.ReposWithActionableReviewUnits++
		}
		if result.Triage.SemanticGroupCount > 0 {
			summary.Triage.ReposWithSemanticGroups++
		}
		if result.Triage.RecommendationEntries > 0 {
			summary.Triage.ReposWithRecommendations++
		}
		if result.Triage.RecommendationReviewUnits > 0 {
			summary.Triage.ReposWithRecommendationReviewUnits++
		}
		if result.Triage.LedgerEntries > 0 {
			summary.Triage.ReposWithLedgerEntries++
		}
		if result.Triage.GovernanceTotalSuggestions > 0 {
			summary.Triage.ReposWithGovernanceSuggestions++
		}
	}
	if len(summary.Aggregate.FailureKinds) == 0 {
		summary.Aggregate.FailureKinds = map[string]int{}
	}
	if len(summary.Triage.GovernanceSuggestionsByStatus) == 0 {
		summary.Triage.GovernanceSuggestionsByStatus = map[string]int{}
	}
	return summary, nil
}

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
		fmt.Sprintf("- Policy: `%s` sample=`%s` coverage_prefilter=`%t` max_mutants=`%d` workers=`%d`", result.Policy, result.Sample, result.CoveragePrefilter, result.MaxMutants, result.Workers),
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

func selectWaveReport(reportDir string) (string, string) {
	full := filepath.Join(reportDir, "mutation-report.json")
	if fileExists(full) {
		return full, "full"
	}
	partial := filepath.Join(reportDir, "partial-mutation-report.json")
	if fileExists(partial) {
		return partial, "partial"
	}
	return "", "missing"
}

func readRunResult(path string) (engine.RunResult, error) {
	var result engine.RunResult
	err := readJSONFile(path, &result)
	return result, err
}

func readTriageLedger(path string) (reportpkg.TriageLedger, error) {
	var ledger reportpkg.TriageLedger
	if !fileExists(path) {
		return reportpkg.TriageLedger{Entries: []reportpkg.TriageLedgerEntry{}}, nil
	}
	if err := readJSONFile(path, &ledger); err != nil {
		return reportpkg.TriageLedger{}, err
	}
	if ledger.Entries == nil {
		ledger.Entries = []reportpkg.TriageLedgerEntry{}
	}
	return ledger, nil
}

func readGovernanceReview(path string) (reportpkg.GovernanceReview, error) {
	var review reportpkg.GovernanceReview
	if !fileExists(path) {
		return reportpkg.GovernanceReview{
			QuarantineTemplates:  []reportpkg.GovernanceQuarantineTemplate{},
			SuppressionTemplates: []reportpkg.GovernanceSuppressionTemplate{},
		}, nil
	}
	if err := readJSONFile(path, &review); err != nil {
		return reportpkg.GovernanceReview{}, err
	}
	if review.QuarantineTemplates == nil {
		review.QuarantineTemplates = []reportpkg.GovernanceQuarantineTemplate{}
	}
	if review.SuppressionTemplates == nil {
		review.SuppressionTemplates = []reportpkg.GovernanceSuppressionTemplate{}
	}
	return review, nil
}

func readJSONFile(path string, target any) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("path must not be empty")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func nullableString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func nullableFloat64(value float64) *float64 {
	return &value
}

func cloneDenominatorHealth(src engine.DenominatorHealth) *engine.DenominatorHealth {
	dst := src
	dst.Warnings = append([]string{}, src.Warnings...)
	return &dst
}

func cloneEnvironment(src engine.Environment) *engine.Environment {
	dst := src
	dst.Warnings = append([]string{}, src.Warnings...)
	if src.Extra != nil {
		dst.Extra = map[string]string{}
		for key, value := range src.Extra {
			dst.Extra[key] = value
		}
	}
	return &dst
}

func cloneFailure(src engine.Failure) *engine.Failure {
	dst := src
	dst.Command = append([]string{}, src.Command...)
	dst.Targets = append([]string{}, src.Targets...)
	if src.RunnerResult != nil {
		runner := *src.RunnerResult
		runner.Command = append([]string{}, src.RunnerResult.Command...)
		dst.RunnerResult = &runner
	}
	return &dst
}

func denominatorField(health *engine.DenominatorHealth, selector func(engine.DenominatorHealth) int) int {
	if health == nil {
		return 0
	}
	return selector(*health)
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
