package engine

import (
	"context"
	"time"
)

type Status string

const (
	StatusKilled          Status = "killed"
	StatusSurvived        Status = "survived"
	StatusTimedOut        Status = "timed_out"
	StatusMemoryKilled    Status = "memory_killed"
	StatusCompileError    Status = "compile_error"
	StatusSkipped         Status = "skipped"
	StatusSkippedResource Status = "skipped_resource"
	StatusPendingBudget   Status = "pending_budget"
	StatusNotCovered      Status = "not_covered"
	StatusIgnored         Status = "ignored"
	StatusQuarantined     Status = "quarantined"
	StatusCached          Status = "cached"
)

type Mutant struct {
	ID                  string             `json:"id"`
	Module              string             `json:"module"`
	Package             string             `json:"package"`
	File                string             `json:"file"`
	Line                int                `json:"line"`
	Function            string             `json:"function"`
	Operator            string             `json:"operator"`
	Original            string             `json:"original"`
	Mutated             string             `json:"mutated"`
	StartOffset         int                `json:"start_offset"`
	EndOffset           int                `json:"end_offset"`
	Diff                string             `json:"unified_diff"`
	Fingerprint         string             `json:"fingerprint"`
	Hint                string             `json:"hint"`
	Description         string             `json:"description"`
	NearbyTests         []string           `json:"nearby_tests,omitempty"`
	EquivalentRisk      string             `json:"equivalent_risk"`
	Recommendation      string             `json:"recommendation"`
	CompileErrorRisk    string             `json:"compile_error_risk"`
	SemanticTags        []string           `json:"semantic_tags,omitempty"`
	SemanticGroup       string             `json:"semantic_group,omitempty"`
	GroupLabel          string             `json:"group_label,omitempty"`
	GroupReason         string             `json:"group_reason,omitempty"`
	PlatformSensitive   bool               `json:"platform_sensitive,omitempty"`
	NonProgressRisk     string             `json:"non_progress_risk,omitempty"`
	Ownership           *OwnershipRoute    `json:"ownership,omitempty"`
	SuggestedSkipReason string             `json:"suggested_skip_reason,omitempty"`
	SuppressionAudit    []SuppressionAudit `json:"suppression_audit,omitempty"`
}

type OwnershipRoute struct {
	Owner   string `json:"owner,omitempty"`
	Team    string `json:"team,omitempty"`
	Contact string `json:"contact,omitempty"`
	Rule    string `json:"rule,omitempty"`
}

type SuppressionAudit struct {
	Name          string `json:"name"`
	Action        string `json:"action"`
	Reason        string `json:"reason"`
	EvidenceLevel string `json:"evidence_level"`
	ReviewerCount int    `json:"reviewer_count,omitempty"`
}

type MutantJob struct {
	ID          string   `json:"id"`
	Mutant      Mutant   `json:"mutant"`
	WorkDir     string   `json:"work_dir"`
	TestCommand []string `json:"test_command"`
	Timeout     string   `json:"timeout"`
}

type MutantResult struct {
	MutantID            string              `json:"mutant_id"`
	Status              Status              `json:"status"`
	FailureKind         string              `json:"failure_kind,omitempty"`
	MemoryPeakBytes     int64               `json:"memory_peak_bytes,omitempty"`
	Duration            time.Duration       `json:"duration"`
	TestCommand         []string            `json:"selected_tests"`
	StatusReason        string              `json:"status_reason"`
	SelectionReason     string              `json:"selection_reason,omitempty"`
	CoverageSource      string              `json:"coverage_source,omitempty"`
	Output              string              `json:"output"`
	Mutant              Mutant              `json:"mutant"`
	SurvivorRank        int                 `json:"survivor_rank,omitempty"`
	RankScore           float64             `json:"rank_score,omitempty"`
	RankReason          string              `json:"rank_reason,omitempty"`
	Actionability       string              `json:"actionability,omitempty"`
	SuggestedTestScope  string              `json:"suggested_test_scope,omitempty"`
	TestRecommendation  *TestRecommendation `json:"test_recommendation,omitempty"`
	SuggestedSkipReason string              `json:"suggested_skip_reason,omitempty"`
	NearestTests        []string            `json:"nearest_tests,omitempty"`
	SemanticGroupSize   int                 `json:"semantic_group_size,omitempty"`
	PreviousStatus      Status              `json:"previous_status,omitempty"`
	FirstSeen           string              `json:"first_seen,omitempty"`
	LastSeen            string              `json:"last_seen,omitempty"`
	SurvivorAgeRuns     int                 `json:"survivor_age_runs,omitempty"`
	HistoryStatus       string              `json:"history_status,omitempty"`
	OperatorYield       float64             `json:"operator_historical_yield,omitempty"`
}

type TestRecommendation struct {
	Priority            string   `json:"priority,omitempty"`
	Strategy            string   `json:"strategy,omitempty"`
	Summary             string   `json:"summary,omitempty"`
	CandidateTests      []string `json:"candidate_tests,omitempty"`
	SuggestedAssertions []string `json:"suggested_assertions,omitempty"`
	Rationale           []string `json:"rationale,omitempty"`
}

type Summary struct {
	Total                         int                    `json:"total"`
	Killed                        int                    `json:"killed"`
	Survived                      int                    `json:"survived"`
	NotCovered                    int                    `json:"not_covered"`
	TimedOut                      int                    `json:"timed_out"`
	MemoryKilled                  int                    `json:"memory_killed"`
	CompileError                  int                    `json:"compile_error"`
	Skipped                       int                    `json:"skipped"`
	SkippedResource               int                    `json:"skipped_resource"`
	PendingBudget                 int                    `json:"pending_budget"`
	Ignored                       int                    `json:"ignored"`
	Quarantined                   int                    `json:"quarantined"`
	Cached                        int                    `json:"cached"`
	ExpiredQuarantine             int                    `json:"expired_quarantine"`
	GeneratedMutants              int                    `json:"generated_mutants"`
	CoveredMutants                int                    `json:"covered_mutants"`
	ExecutedMutants               int                    `json:"executed_mutants"`
	EffectiveMutants              int                    `json:"effective_mutants"`
	ScoreDenominator              int                    `json:"score_denominator"`
	Score                         float64                `json:"score"`
	EffectiveScore                float64                `json:"effective_score"`
	TestEfficacy                  float64                `json:"test_efficacy"`
	MutationCoverage              float64                `json:"mutation_coverage"`
	Actionable                    ActionableSummary      `json:"actionable"`
	DenominatorHealth             DenominatorHealth      `json:"denominator_health"`
	HighRiskSurvivors             int                    `json:"high_risk_survivors"`
	SuppressionReportOnly         int                    `json:"suppression_report_only"`
	SuppressionLowerPriority      int                    `json:"suppression_lower_priority"`
	SuppressionSuppressed         int                    `json:"suppression_suppressed"`
	SuppressionQuarantineRequired int                    `json:"suppression_quarantine_required"`
	NewSurvivors                  int                    `json:"new_survivors"`
	LongStandingSurvivors         int                    `json:"long_standing_survivors"`
	PlatformSensitiveSurvivors    int                    `json:"platform_sensitive_survivors"`
	NonProgressTimeouts           int                    `json:"non_progress_timeouts"`
	TimeoutRiskStats              map[string]int         `json:"timeout_risk_statistics,omitempty"`
	EquivalentRiskStats           map[string]int         `json:"equivalent_risk_statistics,omitempty"`
	SemanticGroupStats            map[string]int         `json:"semantic_group_statistics,omitempty"`
	MutatorStats                  map[string]MutatorStat `json:"mutator_statistics,omitempty"`
}

type ActionableSummary struct {
	RawScore                    float64 `json:"raw_score"`
	ActionableScore             float64 `json:"actionable_score"`
	Survivors                   int     `json:"survivors"`
	ActionableSurvivors         int     `json:"actionable_survivors"`
	TrueActionableSurvivors     int     `json:"true_actionable_survivors"`
	EquivalentRiskSurvivors     int     `json:"equivalent_risk_survivors"`
	PlatformSensitiveSurvivors  int     `json:"platform_sensitive_survivors"`
	NonProgressTimeouts         int     `json:"non_progress_timeouts"`
	SemanticGroupReviewUnits    int     `json:"semantic_group_review_units"`
	CollapsedSemanticDuplicates int     `json:"collapsed_semantic_duplicates"`
}

type DenominatorHealth struct {
	Generated        int      `json:"generated"`
	Covered          int      `json:"covered"`
	Executed         int      `json:"executed"`
	Effective        int      `json:"effective"`
	ScoreDenominator int      `json:"score_denominator"`
	Killed           int      `json:"killed"`
	Survived         int      `json:"survived"`
	NotCovered       int      `json:"not_covered"`
	TimedOut         int      `json:"timed_out"`
	MemoryKilled     int      `json:"memory_killed"`
	CompileError     int      `json:"compile_error"`
	Skipped          int      `json:"skipped"`
	SkippedResource  int      `json:"skipped_resource"`
	PendingBudget    int      `json:"pending_budget"`
	Ignored          int      `json:"ignored"`
	Quarantined      int      `json:"quarantined"`
	Healthy          bool     `json:"healthy"`
	Warnings         []string `json:"warnings,omitempty"`
}

type MutatorStat struct {
	Total           int    `json:"total"`
	Killed          int    `json:"killed"`
	Survived        int    `json:"survived"`
	NotCovered      int    `json:"not_covered"`
	TimedOut        int    `json:"timed_out"`
	MemoryKilled    int    `json:"memory_killed"`
	CompileError    int    `json:"compile_error"`
	Skipped         int    `json:"skipped"`
	SkippedResource int    `json:"skipped_resource"`
	PendingBudget   int    `json:"pending_budget"`
	Ignored         int    `json:"ignored"`
	Quarantined     int    `json:"quarantined"`
	Cached          int    `json:"cached"`
	Recommendation  string `json:"recommendation,omitempty"`
	EquivalentRisk  string `json:"equivalent_risk,omitempty"`
}

type GateCheckStatus string

const (
	GateCheckDisabled GateCheckStatus = "disabled"
	GateCheckSkipped  GateCheckStatus = "skipped"
	GateCheckPassed   GateCheckStatus = "passed"
	GateCheckFailed   GateCheckStatus = "failed"
)

type GateCheck struct {
	Name    string          `json:"name"`
	Status  GateCheckStatus `json:"status"`
	Summary string          `json:"summary,omitempty"`
}

type GateEvaluation struct {
	Evaluated    bool        `json:"evaluated"`
	Passed       bool        `json:"passed"`
	FailedChecks []string    `json:"failed_checks,omitempty"`
	Checks       []GateCheck `json:"checks,omitempty"`
}

type BaselineComparison struct {
	Enabled       bool     `json:"enabled"`
	Available     bool     `json:"available,omitempty"`
	Regression    bool     `json:"regression"`
	NewSurvivors  []string `json:"new_survivors"`
	PreviousScore float64  `json:"previous_score"`
	CurrentScore  float64  `json:"current_score"`
}

type CacheStats struct {
	Hits   int `json:"hits"`
	Misses int `json:"misses"`
}

type QuarantineStats struct {
	Active        int    `json:"active"`
	Expired       int    `json:"expired"`
	Path          string `json:"path,omitempty"`
	ExpireAfter   string `json:"expire_after,omitempty"`
	RequireReason bool   `json:"require_reason,omitempty"`
	RequireOwner  bool   `json:"require_owner,omitempty"`
	RequireIssue  bool   `json:"require_issue,omitempty"`
	FailOnExpired bool   `json:"fail_on_expired,omitempty"`
	MaxRenewals   int    `json:"max_renewals,omitempty"`
}

type HistoryStats struct {
	Enabled                bool               `json:"enabled"`
	Path                   string             `json:"path,omitempty"`
	UpdatedAt              string             `json:"updated_at,omitempty"`
	LoadedMutants          int                `json:"loaded_mutants"`
	UpdatedMutants         int                `json:"updated_mutants"`
	NewSurvivors           int                `json:"new_survivors"`
	LongStandingSurvivors  int                `json:"long_standing_survivors"`
	OperatorUsefulSurvivor map[string]float64 `json:"operator_useful_survivor_yield,omitempty"`
	Runs                   []HistoryRun       `json:"runs,omitempty"`
}

type HistoryRun struct {
	RunAt                   string             `json:"run_at"`
	RawScore                float64            `json:"raw_score"`
	ActionableScore         float64            `json:"actionable_score"`
	Survived                int                `json:"survived"`
	TrueActionableSurvivors int                `json:"true_actionable_survivors"`
	NewSurvivors            int                `json:"new_survivors"`
	LongStandingSurvivors   int                `json:"long_standing_survivors"`
	SurvivorAgeNew          int                `json:"survivor_age_new"`
	SurvivorAgeAging        int                `json:"survivor_age_aging"`
	SurvivorAgeLongStanding int                `json:"survivor_age_long_standing"`
	TimedOut                int                `json:"timed_out"`
	NonProgressTimeouts     int                `json:"non_progress_timeouts"`
	OperatorUsefulSurvivor  map[string]float64 `json:"operator_useful_survivor,omitempty"`
}

type RunResult struct {
	SchemaVersion       string             `json:"schema_version"`
	Summary             Summary            `json:"summary"`
	Environment         Environment        `json:"environment"`
	Slice               SliceMetadata      `json:"slice,omitempty"`
	Checkpoint          Checkpoint         `json:"checkpoint,omitempty"`
	StoppedReason       string             `json:"stopped_reason,omitempty"`
	LastCompletedMutant string             `json:"last_completed_mutant,omitempty"`
	Failure             *Failure           `json:"failure,omitempty"`
	Thresholds          map[string]any     `json:"thresholds"`
	Gate                GateEvaluation     `json:"gate"`
	Baseline            BaselineComparison `json:"baseline"`
	Cache               CacheStats         `json:"cache"`
	Quarantine          QuarantineStats    `json:"quarantine"`
	History             HistoryStats       `json:"history"`
	Mutants             []MutantResult     `json:"mutants"`
}

type SliceMetadata struct {
	Enabled              bool   `json:"enabled"`
	SliceBy              string `json:"slice_by,omitempty"`
	ShardIndex           int    `json:"shard_index,omitempty"`
	ShardCount           int    `json:"shard_count,omitempty"`
	GroupCount           int    `json:"group_count,omitempty"`
	SelectedGroups       int    `json:"selected_groups,omitempty"`
	MaxFilesPerRun       int    `json:"max_files_per_run,omitempty"`
	SelectedFiles        int    `json:"selected_files,omitempty"`
	MaxMutantsPerPackage int    `json:"max_mutants_per_package,omitempty"`
	SelectedMutants      int    `json:"selected_mutants,omitempty"`
}

type Failure struct {
	Kind                  string               `json:"kind"`
	Message               string               `json:"message"`
	CorrelationID         string               `json:"correlation_id,omitempty"`
	Command               []string             `json:"command,omitempty"`
	Targets               []string             `json:"targets,omitempty"`
	DebugArtifact         string               `json:"debug_artifact,omitempty"`
	PartialReportPresent  bool                 `json:"partial_report_present,omitempty"`
	PartialSummaryPresent bool                 `json:"partial_summary_present,omitempty"`
	RunnerResult          *FailureRunnerResult `json:"runner_result,omitempty"`
}

type FailureRunnerResult struct {
	Status       Status   `json:"status,omitempty"`
	StatusReason string   `json:"status_reason,omitempty"`
	Command      []string `json:"command,omitempty"`
	Output       string   `json:"output,omitempty"`
}

type Checkpoint struct {
	Fingerprint         string `json:"fingerprint"`
	Mutants             int    `json:"mutants"`
	IncludesFileDigests bool   `json:"includes_file_digests"`
	Reason              string `json:"reason,omitempty"`
}

type Environment struct {
	OS              string            `json:"os"`
	Arch            string            `json:"arch"`
	GoVersion       string            `json:"go_version,omitempty"`
	ToolVersion     string            `json:"tool_version,omitempty"`
	WorkingDir      string            `json:"working_dir,omitempty"`
	TempDir         string            `json:"temp_dir,omitempty"`
	TempRoot        string            `json:"temp_root,omitempty"`
	Isolation       string            `json:"isolation,omitempty"`
	Workers         int               `json:"workers,omitempty"`
	TestTimeout     string            `json:"test_timeout,omitempty"`
	Budget          string            `json:"budget,omitempty"`
	GoFlags         string            `json:"go_flags,omitempty"`
	GoMaxProcs      string            `json:"go_max_procs,omitempty"`
	GoMemLimit      string            `json:"go_mem_limit,omitempty"`
	CI              string            `json:"ci,omitempty"`
	WSL             bool              `json:"wsl,omitempty"`
	CGroup          string            `json:"cgroup,omitempty"`
	WindowsOneDrive bool              `json:"windows_onedrive,omitempty"`
	Warnings        []string          `json:"warnings,omitempty"`
	Extra           map[string]string `json:"extra,omitempty"`
}

type ProgressEvent struct {
	SchemaVersion string        `json:"schema_version"`
	Time          time.Time     `json:"time"`
	Completed     int           `json:"completed"`
	Total         int           `json:"total"`
	MutantID      string        `json:"mutant_id,omitempty"`
	Status        Status        `json:"status,omitempty"`
	Elapsed       time.Duration `json:"elapsed"`
	Remaining     int           `json:"remaining"`
	ETA           string        `json:"eta,omitempty"`
	ActiveMutant  string        `json:"active_mutant,omitempty"`
	Message       string        `json:"message"`
}

type RunRequest struct {
	Targets []string `json:"targets"`
	DryRun  bool     `json:"dry_run"`
}

type AffectedRequest struct {
	Targets []string `json:"targets"`
	Scope   string   `json:"scope"`
	Since   string   `json:"since"`
}

type AffectedResult struct {
	Modules          []string `json:"modules"`
	Packages         []string `json:"packages"`
	Files            []string `json:"files"`
	EstimatedMutants int      `json:"estimated_mutants"`
}

type ExplainRequest struct {
	MutantID string `json:"mutant_id"`
	Format   string `json:"format"`
}

type ExplainResult struct {
	MutantID    string `json:"mutant_id"`
	Explanation string `json:"explanation"`
	Suggestion  string `json:"suggestion"`
}

type ResultRef struct {
	ID string `json:"id"`
}

type CachedResult struct {
	Key    string       `json:"key"`
	Result MutantResult `json:"result"`
}

type Scheduler interface {
	Submit(ctx context.Context, job MutantJob) (ResultRef, error)
}

type Runner interface {
	Run(ctx context.Context, job MutantJob) (MutantResult, error)
}

type TestPlan struct {
	Command        []string `json:"command"`
	Reason         string   `json:"reason"`
	CoversMutant   bool     `json:"covers_mutant"`
	CoverageSource string   `json:"coverage_source"`
}

type TestSelector interface {
	Select(ctx context.Context, mutant Mutant) (TestPlan, error)
}

type Store interface {
	Get(ctx context.Context, key string) (CachedResult, bool, error)
	Put(ctx context.Context, result MutantResult) error
}
