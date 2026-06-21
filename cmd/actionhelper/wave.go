package main

import (
	"encoding/json"
	"flag"
	"io"
	"strings"

	"github.com/cervantesh/cervo-mutants/pkg/engine"
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
	PrewarmModules     bool
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
	PrewarmModules     bool                      `json:"prewarm_modules"`
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
	prewarmModules := fs.String("prewarm-modules", "false", "whether module prewarm is enabled")
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
	meta.PrewarmModules = strings.EqualFold(strings.TrimSpace(*prewarmModules), "true")
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
