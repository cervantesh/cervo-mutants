package pool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type benchmarkSession struct {
	ctx             context.Context
	opts            BenchmarkOptions
	corpus          BenchmarkCorpus
	entries         []BenchmarkEntry
	runner          CommandRunner
	gitBinary       string
	cervoBinary     string
	summaryFilePath string
	resumed         map[string]BenchmarkResult
	results         []BenchmarkResult
}

func RunBenchmark(ctx context.Context, opts BenchmarkOptions) (RunSummary[BenchmarkResult], error) {
	session, err := newBenchmarkSession(ctx, opts)
	if err != nil {
		return RunSummary[BenchmarkResult]{}, err
	}
	return session.run()
}

func newBenchmarkSession(ctx context.Context, opts BenchmarkOptions) (*benchmarkSession, error) {
	corpus, err := LoadBenchmarkCorpus(opts.CorpusPath)
	if err != nil {
		return nil, err
	}
	entries := FilterBenchmarkEntries(corpus, opts.Names, opts.Limit)
	if err := os.MkdirAll(opts.WorkRoot, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(opts.OutputRoot, 0o755); err != nil {
		return nil, err
	}
	runner := opts.Runner
	if runner == nil {
		runner = RealCommandRunner{}
	}
	gitBinary, err := requiredBinary("git", defaultPath(opts.GitBinary, "git"))
	if err != nil {
		return nil, err
	}
	cervoBinary, err := requiredBinary("cervomut", opts.CervoBinary)
	if err != nil {
		return nil, err
	}

	session := &benchmarkSession{
		ctx:             ctx,
		opts:            opts,
		corpus:          corpus,
		entries:         entries,
		runner:          runner,
		gitBinary:       gitBinary,
		cervoBinary:     cervoBinary,
		summaryFilePath: summaryPath(opts.OutputRoot),
		resumed:         map[string]BenchmarkResult{},
		results:         make([]BenchmarkResult, 0, len(entries)),
	}
	if err := session.loadResumedResults(); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *benchmarkSession) run() (RunSummary[BenchmarkResult], error) {
	for _, entry := range s.entries {
		result, err := s.resultForEntry(entry)
		if err != nil {
			return s.partialSummary(), err
		}
		if err := s.recordResult(result); err != nil {
			return s.partialSummary(), err
		}
	}
	return s.finish()
}

func (s *benchmarkSession) loadResumedResults() error {
	if !s.opts.Resume {
		return nil
	}
	summary, ok, err := loadBenchmarkSummary(s.summaryFilePath)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	for _, result := range summary.Results {
		s.resumed[result.Name] = result
	}
	return nil
}

func (s *benchmarkSession) resultForEntry(entry BenchmarkEntry) (BenchmarkResult, error) {
	if result, ok := s.resumedResult(entry); ok {
		return result, nil
	}
	return s.runEntry(entry)
}

func (s *benchmarkSession) resumedResult(entry BenchmarkEntry) (BenchmarkResult, bool) {
	result, ok := s.resumed[entry.Name]
	if !ok {
		return BenchmarkResult{}, false
	}
	if !containsNote(result.Notes, "resumed from existing summary") {
		result.Notes = append(result.Notes, "resumed from existing summary")
	}
	return result, true
}

func (s *benchmarkSession) runEntry(entry BenchmarkEntry) (BenchmarkResult, error) {
	result := newBenchmarkResult(entry)
	started := time.Now()
	repoDir := s.repoDir(entry)
	outputDir := s.outputDir(entry)
	reportDir := filepath.Join(outputDir, "cervomut")

	if ok, err := s.ensureClone(entry, repoDir, outputDir, &result); err != nil {
		return result, err
	} else if !ok {
		s.finishEntry(&result, started)
		return result, nil
	}

	if ok, err := s.ensureCheckout(entry, repoDir, outputDir, &result); err != nil {
		return result, err
	} else if !ok {
		s.finishEntry(&result, started)
		return result, nil
	}

	if ok, err := s.runBaseline(entry, repoDir, outputDir, &result); err != nil {
		return result, err
	} else if !ok {
		s.finishEntry(&result, started)
		return result, nil
	}

	if ok, err := s.runDryRun(entry, repoDir, reportDir, outputDir, &result); err != nil {
		return result, err
	} else if !ok {
		s.finishEntry(&result, started)
		return result, nil
	}

	if ok, err := s.runMutation(entry, repoDir, reportDir, outputDir, &result); err != nil {
		return result, err
	} else if !ok {
		s.finishEntry(&result, started)
		return result, nil
	}

	s.applyRunReport(reportDir, entry, &result)
	s.finishEntry(&result, started)
	return result, nil
}

func (s *benchmarkSession) ensureClone(entry BenchmarkEntry, repoDir, outputDir string, result *BenchmarkResult) (bool, error) {
	result.Clone = "existing"
	if _, statErr := os.Stat(repoDir); statErr == nil {
		return true, nil
	}
	cloneExit, err := runSimpleCommand(s.ctx, s.runner, CommandSpec{
		Path:    s.gitBinary,
		Args:    []string{"clone", "--depth", "1", entry.URL, repoDir},
		Dir:     s.opts.WorkRoot,
		LogPath: filepath.Join(outputDir, "clone.log"),
		Timeout: time.Duration(benchmarkCloneTimeoutSeconds(entry)) * time.Second,
	})
	if err != nil {
		return false, err
	}
	if cloneExit != 0 {
		result.Clone = "failed"
		result.Notes = append(result.Notes, "clone exit "+strconv.Itoa(cloneExit))
		return false, nil
	}
	result.Clone = "ok"
	return true, nil
}

func (s *benchmarkSession) ensureCheckout(entry BenchmarkEntry, repoDir, outputDir string, result *BenchmarkResult) (bool, error) {
	result.Checkout = "skipped"
	if strings.TrimSpace(entry.Ref) == "" {
		return true, nil
	}
	ok, err := checkoutBenchmarkRef(s.ctx, s.runner, s.gitBinary, repoDir, entry, outputDir)
	if err != nil {
		return false, err
	}
	if !ok {
		result.Checkout = "failed"
		result.Notes = append(result.Notes, "checkout failed for ref "+entry.Ref)
		return false, nil
	}
	result.Checkout = "ok"
	return true, nil
}

func (s *benchmarkSession) runBaseline(entry BenchmarkEntry, repoDir, outputDir string, result *BenchmarkResult) (bool, error) {
	started := time.Now()
	exitCode, err := runSimpleCommand(s.ctx, s.runner, CommandSpec{
		Path:    "go",
		Args:    []string{"test", entry.Target},
		Dir:     repoDir,
		LogPath: filepath.Join(outputDir, "baseline.log"),
		Timeout: time.Duration(benchmarkBaselineTimeoutSeconds(entry)) * time.Second,
	})
	if err != nil {
		return false, err
	}
	result.BaselineExit = intPtr(exitCode)
	result.Metrics.BaselineSeconds = seconds(started)
	if exitCode != 0 {
		result.Notes = append(result.Notes, "baseline exit "+strconv.Itoa(exitCode))
		return false, nil
	}
	return true, nil
}

func (s *benchmarkSession) runDryRun(entry BenchmarkEntry, repoDir, reportDir, outputDir string, result *BenchmarkResult) (bool, error) {
	started := time.Now()
	exitCode, err := runSimpleCommand(s.ctx, s.runner, CommandSpec{
		Path:    s.cervoBinary,
		Args:    benchmarkRunArgs(entry, reportDir, true),
		Dir:     repoDir,
		LogPath: filepath.Join(outputDir, "dry-run.log"),
		Timeout: time.Duration(benchmarkDryRunTimeoutSeconds(entry)) * time.Second,
	})
	if err != nil {
		return false, err
	}
	result.DryRunExit = intPtr(exitCode)
	result.Metrics.DryRunSeconds = seconds(started)
	if exitCode != 0 {
		result.Notes = append(result.Notes, "dry-run exit "+strconv.Itoa(exitCode))
		return false, nil
	}
	return true, nil
}

func (s *benchmarkSession) runMutation(entry BenchmarkEntry, repoDir, reportDir, outputDir string, result *BenchmarkResult) (bool, error) {
	started := time.Now()
	exitCode, err := runSimpleCommand(s.ctx, s.runner, CommandSpec{
		Path:    s.cervoBinary,
		Args:    benchmarkRunArgs(entry, reportDir, false),
		Dir:     repoDir,
		LogPath: filepath.Join(outputDir, "mutation.log"),
		Timeout: time.Duration(benchmarkMutationTimeoutSeconds(entry)) * time.Second,
	})
	if err != nil {
		return false, err
	}
	result.MutationExit = intPtr(exitCode)
	result.Metrics.MutationSeconds = seconds(started)
	if exitCode != 0 {
		result.Notes = append(result.Notes, "mutation exit "+strconv.Itoa(exitCode))
		return false, nil
	}
	return true, nil
}

func (s *benchmarkSession) applyRunReport(reportDir string, entry BenchmarkEntry, result *BenchmarkResult) {
	report, reportPath, partial, err := loadBenchmarkRunResult(reportDir)
	if err != nil {
		result.Notes = append(result.Notes, err.Error())
		return
	}
	result.ReportPath = reportPath
	result.PartialReportUsed = partial
	if partial {
		result.Notes = append(result.Notes, "partial report used")
	}
	populateBenchmarkMetrics(result, report)
	result.Checks = evaluateBenchmarkThresholds(result.Metrics, entry.Thresholds)
	if benchmarkHasFailedCheck(result.Checks) {
		result.Status = "fail"
	} else {
		result.Status = "pass"
	}
	if len(result.Checks) == 0 {
		result.Notes = append(result.Notes, "no benchmark thresholds configured")
	}
}

func (s *benchmarkSession) recordResult(result BenchmarkResult) error {
	s.results = append(s.results, result)
	return writeBenchmarkSummary(s.summaryFilePath, s.opts.CorpusPath, s.corpus, s.results)
}

func (s *benchmarkSession) finish() (RunSummary[BenchmarkResult], error) {
	summary := buildBenchmarkSummary(s.opts.CorpusPath, s.corpus, s.results)
	if err := writeJSON(s.summaryFilePath, summary); err != nil {
		return s.partialSummary(), err
	}
	run := s.partialSummary()
	if summary.Totals.Errored > 0 && summary.Totals.Failed > 0 {
		return run, fmt.Errorf("benchmark threshold failed for %d entries and benchmark execution failed for %d entries; summary=%s", summary.Totals.Failed, summary.Totals.Errored, s.summaryFilePath)
	}
	if summary.Totals.Failed > 0 {
		return run, fmt.Errorf("benchmark threshold failed for %d entries; summary=%s", summary.Totals.Failed, s.summaryFilePath)
	}
	if summary.Totals.Errored > 0 {
		return run, fmt.Errorf("benchmark execution failed for %d entries; summary=%s", summary.Totals.Errored, s.summaryFilePath)
	}
	return run, nil
}

func (s *benchmarkSession) partialSummary() RunSummary[BenchmarkResult] {
	return RunSummary[BenchmarkResult]{
		Results:     s.results,
		SummaryPath: s.summaryFilePath,
	}
}

func (s *benchmarkSession) repoDir(entry BenchmarkEntry) string {
	return filepath.Join(s.opts.WorkRoot, entry.Name)
}

func (s *benchmarkSession) outputDir(entry BenchmarkEntry) string {
	return filepath.Join(s.opts.OutputRoot, entry.Name)
}

func newBenchmarkResult(entry BenchmarkEntry) BenchmarkResult {
	return BenchmarkResult{
		Name:          entry.Name,
		URL:           entry.URL,
		Ref:           entry.Ref,
		Target:        entry.Target,
		Size:          entry.Size,
		ResourceClass: entry.ResourceClass,
		Policy:        benchmarkPolicy(entry),
		MaxMutants:    benchmarkMaxMutants(entry),
		Workers:       benchmarkWorkers(entry),
		Clone:         "pending",
		Thresholds:    entry.Thresholds,
		Status:        "error",
	}
}

func (s *benchmarkSession) finishEntry(result *BenchmarkResult, started time.Time) {
	result.ElapsedSeconds = seconds(started)
}
