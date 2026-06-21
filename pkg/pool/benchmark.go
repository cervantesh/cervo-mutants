package pool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cervantesh/cervo-mutants/pkg/engine"
)

const (
	defaultBenchmarkPolicy                 = "comparison-safe"
	defaultBenchmarkWorkers                = 2
	defaultBenchmarkMaxMutants             = 25
	defaultBenchmarkCloneTimeoutSeconds    = 180
	defaultBenchmarkBaselineTimeoutSeconds = 120
	defaultBenchmarkDryRunTimeoutSeconds   = 120
	defaultBenchmarkMutationTimeoutSeconds = 300
)

type BenchmarkCorpus struct {
	SchemaVersion string           `json:"schema_version"`
	TrackingIssue string           `json:"tracking_issue,omitempty"`
	Description   string           `json:"description,omitempty"`
	Entries       []BenchmarkEntry `json:"entries"`
}

type BenchmarkEntry struct {
	Name                   string              `json:"name"`
	URL                    string              `json:"url"`
	Ref                    string              `json:"ref,omitempty"`
	Target                 string              `json:"target"`
	Size                   string              `json:"size,omitempty"`
	ResourceClass          string              `json:"resource_class,omitempty"`
	Policy                 string              `json:"policy,omitempty"`
	MaxMutants             int                 `json:"max_mutants,omitempty"`
	Workers                int                 `json:"workers,omitempty"`
	CloneTimeoutSeconds    int                 `json:"clone_timeout_seconds,omitempty"`
	BaselineTimeoutSeconds int                 `json:"baseline_timeout_seconds,omitempty"`
	DryRunTimeoutSeconds   int                 `json:"dry_run_timeout_seconds,omitempty"`
	MutationTimeoutSeconds int                 `json:"mutation_timeout_seconds,omitempty"`
	Thresholds             BenchmarkThresholds `json:"thresholds,omitempty"`
	Notes                  string              `json:"notes,omitempty"`
}

type BenchmarkThresholds struct {
	MaxBaselineSeconds  float64 `json:"max_baseline_seconds,omitempty"`
	MaxDryRunSeconds    float64 `json:"max_dry_run_seconds,omitempty"`
	MaxMutationSeconds  float64 `json:"max_mutation_seconds,omitempty"`
	MaxPeakMemoryMB     float64 `json:"max_peak_memory_mb,omitempty"`
	MinMutantsPerSecond float64 `json:"min_mutants_per_second,omitempty"`
	MinExecutedMutants  float64 `json:"min_executed_mutants,omitempty"`
}

type BenchmarkOptions struct {
	CorpusPath  string
	WorkRoot    string
	OutputRoot  string
	Names       []string
	Limit       int
	Resume      bool
	CervoBinary string
	GitBinary   string
	Runner      CommandRunner
}

type BenchmarkMetrics struct {
	BaselineSeconds  float64 `json:"baseline_seconds,omitempty"`
	DryRunSeconds    float64 `json:"dry_run_seconds,omitempty"`
	MutationSeconds  float64 `json:"mutation_seconds,omitempty"`
	GeneratedMutants int     `json:"generated_mutants,omitempty"`
	ExecutedMutants  int     `json:"executed_mutants,omitempty"`
	EffectiveMutants int     `json:"effective_mutants,omitempty"`
	ScoreDenominator int     `json:"score_denominator,omitempty"`
	MaxPeakMemoryMB  float64 `json:"max_peak_memory_mb,omitempty"`
	MutantsPerSecond float64 `json:"mutants_per_second,omitempty"`
}

type BenchmarkCheck struct {
	Name       string  `json:"name"`
	Status     string  `json:"status"`
	Comparator string  `json:"comparator"`
	Actual     float64 `json:"actual"`
	Threshold  float64 `json:"threshold"`
	Message    string  `json:"message,omitempty"`
}

type BenchmarkResult struct {
	Name              string              `json:"name"`
	URL               string              `json:"url"`
	Ref               string              `json:"ref,omitempty"`
	Target            string              `json:"target"`
	Size              string              `json:"size,omitempty"`
	ResourceClass     string              `json:"resource_class,omitempty"`
	Policy            string              `json:"policy"`
	MaxMutants        int                 `json:"max_mutants"`
	Workers           int                 `json:"workers"`
	Clone             string              `json:"clone"`
	Checkout          string              `json:"checkout,omitempty"`
	BaselineExit      *int                `json:"baseline_exit,omitempty"`
	DryRunExit        *int                `json:"dry_run_exit,omitempty"`
	MutationExit      *int                `json:"mutation_exit,omitempty"`
	Status            string              `json:"status"`
	Thresholds        BenchmarkThresholds `json:"thresholds,omitempty"`
	Metrics           BenchmarkMetrics    `json:"metrics"`
	Checks            []BenchmarkCheck    `json:"checks,omitempty"`
	ReportPath        string              `json:"report_path,omitempty"`
	PartialReportUsed bool                `json:"partial_report_used,omitempty"`
	Notes             []string            `json:"notes,omitempty"`
	ElapsedSeconds    float64             `json:"elapsed_seconds"`
}

type BenchmarkTotals struct {
	Entries      int `json:"entries"`
	Passed       int `json:"passed"`
	Failed       int `json:"failed"`
	Errored      int `json:"errored"`
	Resumed      int `json:"resumed"`
	ChecksPassed int `json:"checks_passed"`
	ChecksFailed int `json:"checks_failed"`
}

type BenchmarkSummaryFile struct {
	SchemaVersion string            `json:"schema_version"`
	CorpusPath    string            `json:"corpus_path"`
	TrackingIssue string            `json:"tracking_issue,omitempty"`
	Description   string            `json:"description,omitempty"`
	GeneratedAt   time.Time         `json:"generated_at"`
	Totals        BenchmarkTotals   `json:"totals"`
	Results       []BenchmarkResult `json:"results"`
}

func LoadBenchmarkCorpus(path string) (BenchmarkCorpus, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return BenchmarkCorpus{}, err
	}
	var corpus BenchmarkCorpus
	if err := json.Unmarshal(data, &corpus); err != nil {
		return BenchmarkCorpus{}, err
	}
	return corpus, nil
}

func FilterBenchmarkEntries(corpus BenchmarkCorpus, names []string, limit int) []BenchmarkEntry {
	entries := append([]BenchmarkEntry(nil), corpus.Entries...)
	if len(names) > 0 {
		wanted := map[string]bool{}
		for _, name := range names {
			if name = strings.TrimSpace(name); name != "" {
				wanted[name] = true
			}
		}
		filtered := make([]BenchmarkEntry, 0, len(entries))
		for _, entry := range entries {
			if wanted[entry.Name] {
				filtered = append(filtered, entry)
			}
		}
		entries = filtered
	}
	if limit > 0 && limit < len(entries) {
		entries = entries[:limit]
	}
	return entries
}

func checkoutBenchmarkRef(ctx context.Context, runner CommandRunner, gitBinary, repoDir string, entry BenchmarkEntry, outputDir string) (bool, error) {
	fetchExit, err := runSimpleCommand(ctx, runner, CommandSpec{
		Path:    gitBinary,
		Args:    []string{"fetch", "--depth", "1", "origin", entry.Ref},
		Dir:     repoDir,
		LogPath: filepath.Join(outputDir, "checkout-fetch.log"),
		Timeout: time.Duration(benchmarkCloneTimeoutSeconds(entry)) * time.Second,
	})
	if err != nil {
		return false, err
	}
	if fetchExit != 0 {
		return false, nil
	}
	checkoutExit, err := runSimpleCommand(ctx, runner, CommandSpec{
		Path:    gitBinary,
		Args:    []string{"checkout", "--force", "FETCH_HEAD"},
		Dir:     repoDir,
		LogPath: filepath.Join(outputDir, "checkout.log"),
		Timeout: time.Duration(benchmarkCloneTimeoutSeconds(entry)) * time.Second,
	})
	if err != nil {
		return false, err
	}
	return checkoutExit == 0, nil
}

func benchmarkRunArgs(entry BenchmarkEntry, reportDir string, dryRun bool) []string {
	args := []string{"run", entry.Target}
	if dryRun {
		args = append(args, "--dry-run")
	}
	args = append(args,
		"--policy", benchmarkPolicy(entry),
		"--max-mutants", strconv.Itoa(benchmarkMaxMutants(entry)),
		"--workers", strconv.Itoa(benchmarkWorkers(entry)),
		"--report", "summary,json",
		"--out", reportDir,
	)
	return args
}

func benchmarkPolicy(entry BenchmarkEntry) string {
	if strings.TrimSpace(entry.Policy) != "" {
		return entry.Policy
	}
	return defaultBenchmarkPolicy
}

func benchmarkWorkers(entry BenchmarkEntry) int {
	if entry.Workers > 0 {
		return entry.Workers
	}
	return defaultBenchmarkWorkers
}

func benchmarkMaxMutants(entry BenchmarkEntry) int {
	if entry.MaxMutants > 0 {
		return entry.MaxMutants
	}
	return defaultBenchmarkMaxMutants
}

func benchmarkCloneTimeoutSeconds(entry BenchmarkEntry) int {
	if entry.CloneTimeoutSeconds > 0 {
		return entry.CloneTimeoutSeconds
	}
	return defaultBenchmarkCloneTimeoutSeconds
}

func benchmarkBaselineTimeoutSeconds(entry BenchmarkEntry) int {
	if entry.BaselineTimeoutSeconds > 0 {
		return entry.BaselineTimeoutSeconds
	}
	return defaultBenchmarkBaselineTimeoutSeconds
}

func benchmarkDryRunTimeoutSeconds(entry BenchmarkEntry) int {
	if entry.DryRunTimeoutSeconds > 0 {
		return entry.DryRunTimeoutSeconds
	}
	return defaultBenchmarkDryRunTimeoutSeconds
}

func benchmarkMutationTimeoutSeconds(entry BenchmarkEntry) int {
	if entry.MutationTimeoutSeconds > 0 {
		return entry.MutationTimeoutSeconds
	}
	return defaultBenchmarkMutationTimeoutSeconds
}

func loadBenchmarkRunResult(reportDir string) (engine.RunResult, string, bool, error) {
	candidates := []struct {
		name    string
		partial bool
	}{
		{name: "mutation-report.json"},
		{name: "partial-mutation-report.json", partial: true},
	}
	for _, candidate := range candidates {
		path := filepath.Join(reportDir, candidate.name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var result engine.RunResult
		if err := json.Unmarshal(data, &result); err != nil {
			return engine.RunResult{}, "", false, fmt.Errorf("parse %s: %w", path, err)
		}
		return result, path, candidate.partial, nil
	}
	return engine.RunResult{}, "", false, fmt.Errorf("missing mutation report in %s", reportDir)
}

func populateBenchmarkMetrics(result *BenchmarkResult, run engine.RunResult) {
	result.Metrics.GeneratedMutants = benchmarkGeneratedMutants(run)
	result.Metrics.ExecutedMutants = benchmarkExecutedMutants(run)
	result.Metrics.EffectiveMutants = benchmarkEffectiveMutants(run)
	result.Metrics.ScoreDenominator = benchmarkScoreDenominator(run)
	result.Metrics.MaxPeakMemoryMB = benchmarkPeakMemoryMB(run.Mutants)
	result.Metrics.MutantsPerSecond = benchmarkMutantsPerSecond(result.Metrics.ExecutedMutants, result.Metrics.MutationSeconds)
}

func benchmarkGeneratedMutants(run engine.RunResult) int {
	if run.Summary.GeneratedMutants > 0 {
		return run.Summary.GeneratedMutants
	}
	if run.Summary.Total > 0 {
		return run.Summary.Total
	}
	return len(run.Mutants)
}

func benchmarkExecutedMutants(run engine.RunResult) int {
	if run.Summary.ExecutedMutants > 0 {
		return run.Summary.ExecutedMutants
	}
	executed := 0
	for _, mutant := range run.Mutants {
		switch mutant.Status {
		case engine.StatusKilled, engine.StatusSurvived, engine.StatusTimedOut, engine.StatusMemoryKilled, engine.StatusCompileError:
			executed++
		}
	}
	return executed
}

func benchmarkEffectiveMutants(run engine.RunResult) int {
	if run.Summary.EffectiveMutants > 0 {
		return run.Summary.EffectiveMutants
	}
	return run.Summary.Killed + run.Summary.Survived
}

func benchmarkScoreDenominator(run engine.RunResult) int {
	if run.Summary.ScoreDenominator > 0 {
		return run.Summary.ScoreDenominator
	}
	return benchmarkEffectiveMutants(run) + run.Summary.TimedOut + run.Summary.CompileError
}

func benchmarkPeakMemoryMB(mutants []engine.MutantResult) float64 {
	var peak int64
	for _, mutant := range mutants {
		if mutant.MemoryPeakBytes > peak {
			peak = mutant.MemoryPeakBytes
		}
	}
	if peak == 0 {
		return 0
	}
	return float64(peak) / (1024 * 1024)
}

func benchmarkMutantsPerSecond(executed int, elapsedSeconds float64) float64 {
	if executed <= 0 {
		return 0
	}
	if elapsedSeconds <= 0 {
		elapsedSeconds = 0.001
	}
	return float64(executed) / elapsedSeconds
}

func evaluateBenchmarkThresholds(metrics BenchmarkMetrics, thresholds BenchmarkThresholds) []BenchmarkCheck {
	checks := make([]BenchmarkCheck, 0, 6)
	if thresholds.MaxBaselineSeconds > 0 {
		checks = append(checks, benchmarkCheck("baseline_seconds", metrics.BaselineSeconds, thresholds.MaxBaselineSeconds, "<="))
	}
	if thresholds.MaxDryRunSeconds > 0 {
		checks = append(checks, benchmarkCheck("dry_run_seconds", metrics.DryRunSeconds, thresholds.MaxDryRunSeconds, "<="))
	}
	if thresholds.MaxMutationSeconds > 0 {
		checks = append(checks, benchmarkCheck("mutation_seconds", metrics.MutationSeconds, thresholds.MaxMutationSeconds, "<="))
	}
	if thresholds.MaxPeakMemoryMB > 0 {
		checks = append(checks, benchmarkCheck("max_peak_memory_mb", metrics.MaxPeakMemoryMB, thresholds.MaxPeakMemoryMB, "<="))
	}
	if thresholds.MinMutantsPerSecond > 0 {
		checks = append(checks, benchmarkCheck("mutants_per_second", metrics.MutantsPerSecond, thresholds.MinMutantsPerSecond, ">="))
	}
	if thresholds.MinExecutedMutants > 0 {
		checks = append(checks, benchmarkCheck("executed_mutants", float64(metrics.ExecutedMutants), thresholds.MinExecutedMutants, ">="))
	}
	return checks
}

func benchmarkCheck(name string, actual, threshold float64, comparator string) BenchmarkCheck {
	status := "pass"
	message := fmt.Sprintf("%s %s %.2f", name, comparator, threshold)
	switch comparator {
	case "<=":
		if actual > threshold {
			status = "fail"
		}
	case ">=":
		if actual < threshold {
			status = "fail"
		}
	default:
		status = "fail"
		message = "unsupported comparator"
	}
	return BenchmarkCheck{
		Name:       name,
		Status:     status,
		Comparator: comparator,
		Actual:     actual,
		Threshold:  threshold,
		Message:    message,
	}
}

func benchmarkHasFailedCheck(checks []BenchmarkCheck) bool {
	for _, check := range checks {
		if check.Status == "fail" {
			return true
		}
	}
	return false
}

func buildBenchmarkSummary(corpusPath string, corpus BenchmarkCorpus, results []BenchmarkResult) BenchmarkSummaryFile {
	summary := BenchmarkSummaryFile{
		SchemaVersion: "1",
		CorpusPath:    corpusPath,
		TrackingIssue: corpus.TrackingIssue,
		Description:   corpus.Description,
		GeneratedAt:   time.Now().UTC(),
		Results:       results,
	}
	summary.Totals.Entries = len(results)
	for _, result := range results {
		if containsNote(result.Notes, "resumed from existing summary") {
			summary.Totals.Resumed++
		}
		switch result.Status {
		case "pass":
			summary.Totals.Passed++
		case "fail":
			summary.Totals.Failed++
		default:
			summary.Totals.Errored++
		}
		for _, check := range result.Checks {
			if check.Status == "fail" {
				summary.Totals.ChecksFailed++
			} else {
				summary.Totals.ChecksPassed++
			}
		}
	}
	return summary
}

func writeBenchmarkSummary(path, corpusPath string, corpus BenchmarkCorpus, results []BenchmarkResult) error {
	return writeJSON(path, buildBenchmarkSummary(corpusPath, corpus, results))
}

func loadBenchmarkSummary(path string) (BenchmarkSummaryFile, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return BenchmarkSummaryFile{}, false, nil
		}
		return BenchmarkSummaryFile{}, false, err
	}
	var summary BenchmarkSummaryFile
	if err := json.Unmarshal(data, &summary); err != nil {
		return BenchmarkSummaryFile{}, false, err
	}
	return summary, true, nil
}

func containsNote(notes []string, want string) bool {
	for _, note := range notes {
		if note == want {
			return true
		}
	}
	return false
}
