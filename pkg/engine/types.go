package engine

import (
	"context"
	"time"
)

type Status string

const (
	StatusKilled       Status = "killed"
	StatusSurvived     Status = "survived"
	StatusTimedOut     Status = "timed_out"
	StatusCompileError Status = "compile_error"
	StatusSkipped      Status = "skipped"
	StatusNotCovered   Status = "not_covered"
	StatusIgnored      Status = "ignored"
	StatusQuarantined  Status = "quarantined"
	StatusCached       Status = "cached"
)

type Mutant struct {
	ID          string   `json:"id"`
	Module      string   `json:"module"`
	Package     string   `json:"package"`
	File        string   `json:"file"`
	Line        int      `json:"line"`
	Function    string   `json:"function"`
	Operator    string   `json:"operator"`
	Original    string   `json:"original"`
	Mutated     string   `json:"mutated"`
	StartOffset int      `json:"start_offset"`
	EndOffset   int      `json:"end_offset"`
	Diff        string   `json:"unified_diff"`
	Fingerprint string   `json:"fingerprint"`
	Hint        string   `json:"hint"`
	Description string   `json:"description"`
	NearbyTests []string `json:"nearby_tests,omitempty"`
}

type MutantJob struct {
	ID          string   `json:"id"`
	Mutant      Mutant   `json:"mutant"`
	WorkDir     string   `json:"work_dir"`
	TestCommand []string `json:"test_command"`
	Timeout     string   `json:"timeout"`
}

type MutantResult struct {
	MutantID     string        `json:"mutant_id"`
	Status       Status        `json:"status"`
	Duration     time.Duration `json:"duration"`
	TestCommand  []string      `json:"selected_tests"`
	StatusReason string        `json:"status_reason"`
	Output       string        `json:"output"`
	Mutant       Mutant        `json:"mutant"`
}

type Summary struct {
	Total             int                    `json:"total"`
	Killed            int                    `json:"killed"`
	Survived          int                    `json:"survived"`
	NotCovered        int                    `json:"not_covered"`
	TimedOut          int                    `json:"timed_out"`
	CompileError      int                    `json:"compile_error"`
	Skipped           int                    `json:"skipped"`
	Ignored           int                    `json:"ignored"`
	Quarantined       int                    `json:"quarantined"`
	Cached            int                    `json:"cached"`
	ExpiredQuarantine int                    `json:"expired_quarantine"`
	Score             float64                `json:"score"`
	EffectiveScore    float64                `json:"effective_score"`
	TestEfficacy      float64                `json:"test_efficacy"`
	MutationCoverage  float64                `json:"mutation_coverage"`
	MutatorStats      map[string]MutatorStat `json:"mutator_statistics,omitempty"`
}

type MutatorStat struct {
	Total        int `json:"total"`
	Killed       int `json:"killed"`
	Survived     int `json:"survived"`
	NotCovered   int `json:"not_covered"`
	TimedOut     int `json:"timed_out"`
	CompileError int `json:"compile_error"`
	Skipped      int `json:"skipped"`
	Ignored      int `json:"ignored"`
	Quarantined  int `json:"quarantined"`
	Cached       int `json:"cached"`
}

type BaselineComparison struct {
	Enabled       bool     `json:"enabled"`
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
	Active  int `json:"active"`
	Expired int `json:"expired"`
}

type RunResult struct {
	SchemaVersion string             `json:"schema_version"`
	Summary       Summary            `json:"summary"`
	Thresholds    map[string]any     `json:"thresholds"`
	Baseline      BaselineComparison `json:"baseline"`
	Cache         CacheStats         `json:"cache"`
	Quarantine    QuarantineStats    `json:"quarantine"`
	Mutants       []MutantResult     `json:"mutants"`
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
	Command      []string `json:"command"`
	Reason       string   `json:"reason"`
	CoversMutant bool     `json:"covers_mutant"`
}

type TestSelector interface {
	Select(ctx context.Context, mutant Mutant) (TestPlan, error)
}

type Store interface {
	Get(ctx context.Context, key string) (CachedResult, bool, error)
	Put(ctx context.Context, result MutantResult) error
}
