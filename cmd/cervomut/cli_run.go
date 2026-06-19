package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/cervantesh/cervo-mutants/pkg/config"
	"github.com/cervantesh/cervo-mutants/pkg/engine"
	"github.com/cervantesh/cervo-mutants/pkg/report"
)

func cmdRun(args []string) (err error) {
	opts, targets, err := parseRunOptions(args)
	if err != nil {
		return err
	}
	cfg, err := loadConfigIfPresentStrict()
	if err != nil {
		cfg = config.Defaults()
		applyRunOutput(&cfg, opts.out)
		return finalizeStructuredFailure("run", args, targets, cfg, "config_error", err, "")
	}
	applyRunOptions(&cfg, opts)
	defer func() {
		if recovered := recover(); recovered != nil {
			err = finalizeStructuredFailure("run", args, targets, cfg, "internal_error", fmt.Errorf("unexpected panic: %v", recovered), trimCLIStack(string(debug.Stack())))
		}
	}()
	if err := cfg.Validate(); err != nil {
		return finalizeStructuredFailure("run", args, targets, cfg, "config_error", err, "")
	}
	result, err := runEngineFn(cfg, engine.RunRequest{Targets: targets, DryRun: opts.dryRun})
	if err != nil {
		return finalizeStructuredFailure("run", args, targets, cfg, classifyStructuredFailure(err), err, stackFromError(err))
	}
	if err := writeRunResultFn(cfg, result, opts.dryRun); err != nil {
		if strings.Contains(err.Error(), "threshold") {
			return err
		}
		return finalizeStructuredFailure("run", args, targets, cfg, classifyStructuredFailure(err), err, stackFromError(err))
	}
	return nil
}

type runOptions struct {
	dryRun           bool
	actionableOnly   bool
	scope            string
	sliceBy          string
	shardIndex       int
	shardCount       int
	budget           flagDuration
	testTimeout      flagDuration
	maxMutants       int
	maxFilesPerRun   int
	maxMutantsPerPkg int
	sample           string
	reportFormats    string
	out              string
	workers          int
	isolation        string
	tempRoot         string
	policy           string
	profile          string
	prefilter        bool
	resume           bool
	maxProcessMemory int
}

type flagDuration struct {
	value time.Duration
	set   bool
}

func parseRunOptions(args []string) (runOptions, []string, error) {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	opts := runOptions{}
	fs.BoolVar(&opts.dryRun, "dry-run", false, "only discover mutants")
	fs.BoolVar(&opts.actionableOnly, "actionable-only", false, "show only actionable survivor views while preserving the raw report model")
	scope := fs.String("scope", "", "scope mode")
	sliceBy := fs.String("slice-by", "", "large-repo slicing key: mutant, package, file, function, or operator")
	shard := fs.String("shard", "", "deterministic shard in the form index/count")
	since := fs.String("since", "", "git base")
	budget := fs.Duration("budget", 0, "run budget")
	testTimeout := fs.Duration(flagTestTimeout, 0, "per-mutant go test timeout")
	maxMutants := fs.Int(flagMaxMutants, 0, "max mutants")
	maxFilesPerRun := fs.Int("max-files-per-run", 0, "limit the run to the first N files after deterministic ordering")
	maxMutantsPerPackage := fs.Int("max-mutants-per-package", 0, "limit mutants kept per package after slicing")
	sample := fs.String("sample", "", "sampling mode")
	reportFormats := fs.String("report", "", "comma-separated report formats")
	out := fs.String("out", "", reportOutputDirectoryDoc)
	workers := fs.Int("workers", 0, "parallel mutation workers")
	isolation := fs.String("isolation", "", "isolation backend: temp-workdir or overlay")
	tempRoot := fs.String("temp-root", "", "temporary work root override")
	policy := fs.String("policy", "", "policy preset: ci-fast, ci-balanced, comparison-safe, nightly, or campaign")
	profile := fs.String("profile", "", "mutator profile")
	prefilter := fs.Bool("coverage-prefilter", false, "use coverage profile as a prefilter")
	resume := fs.Bool("resume", false, "resume from partial-mutation-report.json in the output directory")
	maxProcessMemory := fs.Int(flagMaxProcessMemoryMB, 0, "best-effort process-tree memory cap in MB")
	if err := fs.Parse(reorderFlags(args, map[string]bool{
		"scope": true, "slice-by": true, "shard": true, "since": true, "budget": true, flagTestTimeout: true, flagMaxMutants: true, "max-files-per-run": true, "max-mutants-per-package": true, "sample": true, "report": true, "out": true, "workers": true, "isolation": true, "temp-root": true, "policy": true, "profile": true, flagMaxProcessMemoryMB: true,
	})); err != nil {
		return runOptions{}, nil, err
	}
	opts.scope = *scope
	opts.sliceBy = *sliceBy
	if *shard != "" {
		index, count, err := parseShard(*shard)
		if err != nil {
			return runOptions{}, nil, err
		}
		opts.shardIndex = index
		opts.shardCount = count
	}
	_ = since
	opts.budget = flagDuration{value: *budget, set: *budget > 0}
	opts.testTimeout = flagDuration{value: *testTimeout, set: *testTimeout > 0}
	opts.maxMutants = *maxMutants
	opts.maxFilesPerRun = *maxFilesPerRun
	opts.maxMutantsPerPkg = *maxMutantsPerPackage
	opts.sample = *sample
	opts.reportFormats = *reportFormats
	opts.out = *out
	opts.workers = *workers
	opts.isolation = *isolation
	opts.tempRoot = *tempRoot
	opts.policy = *policy
	opts.profile = *profile
	opts.prefilter = *prefilter
	opts.resume = *resume
	opts.maxProcessMemory = *maxProcessMemory
	return opts, fs.Args(), nil
}

func applyRunOptions(cfg *config.Config, opts runOptions) {
	if opts.policy != "" {
		cfg.Policy = opts.policy
		*cfg = config.ApplyPolicy(*cfg)
	}
	applyRunOverrides(cfg, opts)
	applyRunOutput(cfg, opts.out)
}

func applyRunOverrides(cfg *config.Config, opts runOptions) {
	setString(&cfg.Scope.Mode, opts.scope)
	setString(&cfg.Scope.SliceBy, opts.sliceBy)
	setString(&cfg.Mutators.Profile, opts.profile)
	setString(&cfg.Limits.Sample, opts.sample)
	setString(&cfg.Execution.Isolation, opts.isolation)
	setString(&cfg.Execution.TempRoot, opts.tempRoot)
	if opts.prefilter {
		cfg.Selection.Prefilter = true
	}
	if opts.resume {
		cfg.Execution.Resume = true
	}
	if opts.maxFilesPerRun > 0 {
		cfg.Limits.MaxFilesPerRun = opts.maxFilesPerRun
	}
	if opts.maxMutantsPerPkg > 0 {
		cfg.Limits.MaxMutantsPerPackage = opts.maxMutantsPerPkg
	}
	if opts.shardCount > 0 {
		cfg.Scope.ShardIndex = opts.shardIndex
		cfg.Scope.ShardCount = opts.shardCount
	}
	if opts.maxProcessMemory > 0 {
		cfg.Execution.Resources.MaxProcessMemoryMB = opts.maxProcessMemory
	}
	if opts.budget.set {
		cfg.Execution.Budget = opts.budget.value
	}
	if opts.testTimeout.set {
		cfg.Tests.Timeout = opts.testTimeout.value
	}
	if opts.maxMutants > 0 {
		cfg.Limits.MaxMutants = opts.maxMutants
	}
	if opts.workers > 0 {
		cfg.Execution.Workers = opts.workers
	}
	if opts.reportFormats != "" {
		cfg.Reports.Formats = strings.Split(opts.reportFormats, ",")
	}
	if opts.actionableOnly {
		cfg.Reports.ActionableOnly = true
	}
}

func applyRunOutput(cfg *config.Config, out string) {
	if out == "" {
		return
	}
	cfg.Reports.Output = out
	cfg.Cache.Path = filepath.Join(out, "cache")
	cfg.Selection.CoverageProfile = filepath.Join(out, "coverage.out")
	cfg.Selection.TimingsPath = filepath.Join(out, "timings.json")
	cfg.History.Path = filepath.Join(out, "history.json")
}

func setString(target *string, value string) {
	if value != "" {
		*target = value
	}
}

func writeRunResult(cfg config.Config, result engine.RunResult, dryRun bool) error {
	if dryRun {
		data, _ := report.JSON(result)
		fmt.Println(string(data))
		return nil
	}
	if err := report.WriteFormatsWithOptions(cfg.Reports.Output, result, cfg.Reports.Formats, report.WriteOptions{ActionableOnly: cfg.Reports.ActionableOnly}); err != nil {
		return err
	}
	fmt.Print(report.Summary(result))
	if cfg.Reports.ActionableOnly {
		fmt.Print(report.SurvivorsWithOptions(result, report.SurvivorsOptions{ActionableOnly: true}))
	}
	if cfg.CI.FailUnder > 0 && int(result.Summary.Score) < cfg.CI.FailUnder {
		return fmt.Errorf("mutation score %.2f below threshold %d", result.Summary.Score, cfg.CI.FailUnder)
	}
	return nil
}

type failureDebugArtifact struct {
	SchemaVersion string                      `json:"schema_version"`
	Kind          string                      `json:"kind"`
	Message       string                      `json:"message"`
	CorrelationID string                      `json:"correlation_id"`
	Command       []string                    `json:"command,omitempty"`
	Targets       []string                    `json:"targets,omitempty"`
	StackTrace    string                      `json:"stack_trace,omitempty"`
	RunnerResult  *engine.FailureRunnerResult `json:"runner_result,omitempty"`
}

func finalizeStructuredFailure(command string, args, targets []string, cfg config.Config, kind string, cause error, stack string) error {
	correlationID := newCorrelationID()
	outputDir := cfg.Reports.Output
	partialReportPresent := fileExists(filepath.Join(outputDir, "partial-mutation-report.json"))
	partialSummaryPresent := fileExists(filepath.Join(outputDir, "partial-summary.json"))
	debugArtifact := ""
	runnerResult := runnerFailureResultFromError(cause)
	if outputDir != "" {
		_ = os.MkdirAll(outputDir, 0o755)
		debugArtifact = failureDebugFileName
		debugData, err := json.MarshalIndent(failureDebugArtifact{
			SchemaVersion: "1",
			Kind:          kind,
			Message:       cause.Error(),
			CorrelationID: correlationID,
			Command:       append([]string{"cervomut", command}, args...),
			Targets:       append([]string{}, targets...),
			StackTrace:    stack,
			RunnerResult:  runnerResult,
		}, "", "  ")
		if err == nil {
			_ = os.WriteFile(filepath.Join(outputDir, failureDebugFileName), debugData, 0o644)
		}
		reportPath := filepath.Join(outputDir, mutationReportFileName)
		if !fileExists(reportPath) {
			runResult := engine.FailureResult(cfg, engine.Failure{
				Kind:                  kind,
				Message:               cause.Error(),
				CorrelationID:         correlationID,
				Command:               append([]string{"cervomut", command}, args...),
				Targets:               append([]string{}, targets...),
				DebugArtifact:         debugArtifact,
				PartialReportPresent:  partialReportPresent,
				PartialSummaryPresent: partialSummaryPresent,
				RunnerResult:          runnerResult,
			})
			if data, err := report.JSON(runResult); err == nil {
				_ = os.WriteFile(reportPath, data, 0o644)
			}
		}
	}
	return fmt.Errorf("%s: %v [correlation_id=%s]", kind, cause, correlationID)
}

func classifyStructuredFailure(err error) string {
	if err == nil {
		return ""
	}
	var panicErr *engine.PanicError
	if errors.As(err, &panicErr) {
		return "internal_error"
	}
	msg := strings.ToLower(err.Error())
	for _, kind := range []string{"internal_error", "discovery_error", "package_load_error", "config_error", "environment_error", "runner_error"} {
		if strings.HasPrefix(msg, kind+":") {
			return kind
		}
	}
	if strings.Contains(msg, "panic:") {
		return "internal_error"
	}
	if strings.Contains(msg, "config") || strings.Contains(msg, "invalid") {
		return "config_error"
	}
	if strings.Contains(msg, "discover") || strings.Contains(msg, "package") || strings.Contains(msg, "load") {
		return "discovery_error"
	}
	return "runner_error"
}

func stackFromError(err error) string {
	var panicErr *engine.PanicError
	if errors.As(err, &panicErr) {
		return trimCLIStack(panicErr.Stack)
	}
	return ""
}

func runnerFailureResultFromError(err error) *engine.FailureRunnerResult {
	var baselineErr *engine.BaselineFailureError
	if !errors.As(err, &baselineErr) {
		return nil
	}
	result := baselineErr.Result
	if result.Status == "" && result.StatusReason == "" && len(result.TestCommand) == 0 && result.Output == "" {
		return nil
	}
	return &engine.FailureRunnerResult{
		Status:       result.Status,
		StatusReason: result.StatusReason,
		Command:      append([]string{}, result.TestCommand...),
		Output:       trimFailureOutput(result.Output),
	}
}

func trimFailureOutput(output string) string {
	const maxBytes = 8192
	output = strings.TrimSpace(output)
	if len(output) <= maxBytes {
		return output
	}
	return output[:maxBytes]
}

func trimCLIStack(stack string) string {
	if stack == "" {
		return ""
	}
	lines := strings.Split(stack, "\n")
	if len(lines) > 64 {
		lines = lines[:64]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func newCorrelationID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func parseShard(value string) (int, int, error) {
	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid shard %q, want index/count", value)
	}
	index, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid shard index %q", parts[0])
	}
	count, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid shard count %q", parts[1])
	}
	if count < 1 || index < 1 || index > count {
		return 0, 0, fmt.Errorf("invalid shard %q, want 1 <= index <= count", value)
	}
	return index, count, nil
}

func cmdFast(args []string) error {
	next := append([]string{"--policy", "ci-fast", "--report", "summary,json,junit"}, args...)
	return cmdRun(next)
}
