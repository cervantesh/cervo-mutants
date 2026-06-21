package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/cervantesh/cervo-mutants/pkg/config"
	"github.com/cervantesh/cervo-mutants/pkg/engine"
	evalpkg "github.com/cervantesh/cervo-mutants/pkg/eval"
	"github.com/cervantesh/cervo-mutants/pkg/extcompare"
	"github.com/cervantesh/cervo-mutants/pkg/pool"
	"github.com/cervantesh/cervo-mutants/pkg/report"
)

func (app *cliApp) cmdEval(args []string) (err error) {
	fs := flag.NewFlagSet("eval", flag.ContinueOnError)
	out := fs.String("out", ".cervomut/evaluation", "evaluation output directory")
	framework := fs.String("framework", "generic-go", "evaluation framework")
	sliceBy := fs.String("slice-by", "", "large-repo slicing key: mutant, package, file, function, or operator")
	shard := fs.String("shard", "", "deterministic shard in the form index/count")
	budget := fs.Duration("budget", 0, "run budget")
	testTimeout := fs.Duration(flagTestTimeout, 0, "per-mutant go test timeout")
	maxMutants := fs.Int(flagMaxMutants, 0, "max mutants")
	maxFilesPerRun := fs.Int("max-files-per-run", 0, "limit the run to the first N files after deterministic ordering")
	maxMutantsPerPackage := fs.Int("max-mutants-per-package", 0, "limit mutants kept per package after slicing")
	sample := fs.String("sample", "", "sampling mode")
	workers := fs.Int("workers", 0, "parallel mutation workers")
	isolation := fs.String("isolation", "", "isolation backend: temp-workdir or overlay")
	tempRoot := fs.String("temp-root", "", "temporary work root override")
	policy := fs.String("policy", "", "policy preset: ci-fast, ci-balanced, comparison-safe, nightly, or campaign")
	resume := fs.Bool("resume", false, "resume from partial-mutation-report.json in the output directory")
	maxProcessMemory := fs.Int(flagMaxProcessMemoryMB, 0, "best-effort process-tree memory cap in MB")
	if err := fs.Parse(reorderFlags(args, map[string]bool{
		"out": true, "framework": true, "slice-by": true, "shard": true, "budget": true, flagTestTimeout: true, flagMaxMutants: true, "max-files-per-run": true, "max-mutants-per-package": true, "sample": true, "workers": true, "isolation": true, "temp-root": true, "policy": true, flagMaxProcessMemoryMB: true,
	})); err != nil {
		return err
	}
	cfg, err := loadConfigIfPresentStrict()
	if err != nil {
		cfg = config.Defaults()
		applyRunOutput(&cfg, *out)
		return finalizeStructuredFailure("eval", args, fs.Args(), cfg, "config_error", err, "")
	}
	if *policy != "" {
		cfg.Policy = *policy
		cfg = config.ApplyPolicy(cfg)
	}
	if *resume {
		cfg.Execution.Resume = true
	}
	if *maxProcessMemory > 0 {
		cfg.Execution.Resources.MaxProcessMemoryMB = *maxProcessMemory
	}
	applyRunOutput(&cfg, *out)
	if *budget > 0 {
		cfg.Execution.Budget = *budget
	}
	if *testTimeout > 0 {
		cfg.Tests.Timeout = *testTimeout
	}
	if *maxMutants > 0 {
		cfg.Limits.MaxMutants = *maxMutants
	}
	if *sample != "" {
		cfg.Limits.Sample = *sample
	}
	if *workers > 0 {
		cfg.Execution.Workers = *workers
	}
	if *isolation != "" {
		cfg.Execution.Isolation = *isolation
	}
	if *sliceBy != "" {
		cfg.Scope.SliceBy = *sliceBy
	}
	if *shard != "" {
		index, count, err := parseShard(*shard)
		if err != nil {
			return err
		}
		cfg.Scope.ShardIndex = index
		cfg.Scope.ShardCount = count
	}
	if *maxFilesPerRun > 0 {
		cfg.Limits.MaxFilesPerRun = *maxFilesPerRun
	}
	if *maxMutantsPerPackage > 0 {
		cfg.Limits.MaxMutantsPerPackage = *maxMutantsPerPackage
	}
	if *tempRoot != "" {
		cfg.Execution.TempRoot = *tempRoot
	}
	targets := fs.Args()
	defer func() {
		if recovered := recover(); recovered != nil {
			err = finalizeStructuredFailure("eval", args, targets, cfg, "internal_error", fmt.Errorf("unexpected panic: %v", recovered), trimCLIStack(string(debug.Stack())))
		}
	}()
	if err := cfg.Validate(); err != nil {
		return finalizeStructuredFailure("eval", args, targets, cfg, "config_error", err, "")
	}
	runResult, err := app.deps.runEngine(cfg, engine.RunRequest{Targets: targets})
	if err != nil {
		return finalizeStructuredFailure("eval", args, targets, cfg, classifyStructuredFailure(err), err, stackFromError(err))
	}
	if err := report.WriteFormats(cfg.Reports.Output, runResult, cfg.Reports.Formats); err != nil {
		return finalizeStructuredFailure("eval", args, targets, cfg, classifyStructuredFailure(err), err, stackFromError(err))
	}
	evaluation := app.deps.buildEval(evalpkg.BuildRequest{
		Tool:       "cervo-mutants",
		Target:     strings.Join(targets, " "),
		Commit:     currentCommit(),
		Command:    append([]string{"cervomut", "eval"}, args...),
		Framework:  *framework,
		Run:        runResult,
		ManualMode: true,
	})
	if err := app.deps.writeEval(*out, evaluation); err != nil {
		return finalizeStructuredFailure("eval", args, targets, cfg, "internal_error", err, stackFromError(err))
	}
	fmt.Printf("Evaluation written to %s\n", *out)
	return nil
}

func cmdCompare(args []string) error {
	opts, err := parseCompareOptions(args)
	if err != nil {
		return err
	}
	results, err := loadComparisonResults(opts)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		return fmt.Errorf("compare requires at least one tool report")
	}
	if err := extcompare.Write(opts.out, results); err != nil {
		return err
	}
	fmt.Printf("Tool comparison written to %s\n", opts.out)
	return nil
}

func (app *cliApp) cmdPool(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("pool requires smoke, compare, benchmark, or campaign")
	}
	switch args[0] {
	case "smoke":
		return app.cmdPoolSmoke(args[1:])
	case "compare":
		return app.cmdPoolCompare(args[1:])
	case "benchmark":
		return app.cmdPoolBenchmark(args[1:])
	case "campaign":
		return app.cmdPoolCampaign(args[1:])
	default:
		return fmt.Errorf("unknown pool command %q", args[0])
	}
}

func (app *cliApp) cmdPoolSmoke(args []string) error {
	fs := flag.NewFlagSet("pool smoke", flag.ContinueOnError)
	manifest := fs.String("manifest", "docs/evaluations/go-repo-pool-40.json", "repository pool manifest")
	workRoot := fs.String("work-root", filepath.Join(os.TempDir(), "cervomut-go-pool-40"), "working root for cloned repositories and reports")
	names := fs.String("names", "", "comma-separated repository names to include")
	limit := fs.Int("limit", 0, "limit repositories after filtering")
	runMutation := fs.Bool("run-mutation", false, "run bounded mutation after baseline and dry-run")
	maxMutants := fs.Int(flagMaxMutants, 10, "max mutants per target")
	workers := fs.Int("workers", 2, "parallel mutation workers")
	cloneTimeout := fs.Int("clone-timeout-seconds", 180, "git clone timeout in seconds")
	testTimeout := fs.Int("test-timeout-seconds", 120, "baseline go test timeout in seconds")
	dryRunTimeout := fs.Int("dry-run-timeout-seconds", 120, "cervomut dry-run timeout in seconds")
	mutationTimeout := fs.Int("mutation-timeout-seconds", 300, "mutation run timeout in seconds")
	cervoBinary := fs.String("cervomutants", currentExecutable(), "path to the cervomut binary used for nested runs")
	gitBinary := fs.String("git", "git", "path to git")
	if err := fs.Parse(reorderFlags(args, map[string]bool{
		"manifest": true, "work-root": true, "names": true, "limit": true, "run-mutation": true, flagMaxMutants: true, "workers": true, "clone-timeout-seconds": true, "test-timeout-seconds": true, "dry-run-timeout-seconds": true, "mutation-timeout-seconds": true, "cervomutants": true, "git": true,
	})); err != nil {
		return err
	}
	run, err := app.deps.runPoolSmoke(app.deps.background(), pool.SmokeOptions{
		ManifestPath:           *manifest,
		WorkRoot:               *workRoot,
		Names:                  splitList(*names),
		Limit:                  *limit,
		RunMutation:            *runMutation,
		MaxMutants:             *maxMutants,
		Workers:                *workers,
		CloneTimeoutSeconds:    *cloneTimeout,
		TestTimeoutSeconds:     *testTimeout,
		DryRunTimeoutSeconds:   *dryRunTimeout,
		MutationTimeoutSeconds: *mutationTimeout,
		CervoBinary:            *cervoBinary,
		GitBinary:              *gitBinary,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Pool smoke summary: %s\n", run.SummaryPath)
	return nil
}

func (app *cliApp) cmdPoolCompare(args []string) error {
	fs := flag.NewFlagSet("pool compare", flag.ContinueOnError)
	manifest := fs.String("manifest", "docs/evaluations/go-repo-pool-40.json", "repository pool manifest")
	workRoot := fs.String("work-root", filepath.Join(os.TempDir(), "cervomut-go-pool-40"), "working root containing checked-out repositories")
	outputRoot := fs.String("output-root", filepath.Join(os.TempDir(), "cervomut-tool-comparison-12"), "output directory for logs and summary")
	names := fs.String("names", "", "comma-separated repository names to include")
	tools := fs.String("tools", "", "comma-separated tools to run")
	workers := fs.Int("workers", 2, "parallel mutation workers")
	compareTargetMode := fs.String("compare-target-mode", "manifest", "target normalization mode: manifest or package-root")
	gremlinsTargetMode := fs.String("gremlins-target-mode", "manifest", "Gremlins target normalization mode: manifest or package-root")
	gremlinsTimeoutCoefficient := fs.Int("gremlins-timeout-coefficient", 1, "Gremlins timeout coefficient")
	gomuWorkers := fs.Int("gomu-workers", 1, "gomu workers")
	goMutestingWorkers := fs.Int("go-mutesting-workers", 1, "go-mutesting workers")
	timeoutSeconds := fs.Int("timeout-seconds", 600, "per tool/repo timeout in seconds")
	minFreeMemoryMB := fs.Int("min-free-memory-mb", 4096, "wait until this much physical memory is free before launch")
	minFreeCommitMB := fs.Int("min-free-commit-mb", 8192, "wait until this much commit headroom is free before launch")
	killBelowFreeMemoryMB := fs.Int("kill-below-free-memory-mb", 2048, "kill the tool when free physical memory drops below this value")
	killBelowFreeCommitMB := fs.Int("kill-below-free-commit-mb", 4096, "kill the tool when free commit headroom drops below this value")
	maxUsedMemoryMB := fs.Int("max-used-memory-mb", 0, "derive free-memory thresholds from a maximum used-memory cap")
	maxCommittedMemoryMB := fs.Int("max-committed-memory-mb", 0, "derive free-commit thresholds from a maximum committed-memory cap")
	maxProcessTreeMemoryMB := fs.Int("max-process-tree-memory-mb", 0, "best-effort process-tree memory cap in MB")
	memoryWaitSeconds := fs.Int("memory-wait-seconds", 900, "maximum seconds to wait for free memory before skipping")
	memoryPollSeconds := fs.Int("memory-poll-seconds", 5, "memory watchdog poll interval in seconds")
	goMemoryLimit := fs.String("go-memory-limit", "", "optional GOMEMLIMIT value for child processes")
	goMaxProcs := fs.Int("go-max-procs", 0, "optional GOMAXPROCS value for child processes")
	goFlags := fs.String("go-flags", "", "optional GOFLAGS value for child processes")
	resume := fs.Bool("resume", false, "resume using the existing summary.json")
	cervoBinary := fs.String("cervomutants", currentExecutable(), "path to the cervomut binary used for nested runs")
	gremlinsBinary := fs.String("gremlins", filepath.Join(os.TempDir(), "cervomut-study-cobra", "tools", "gremlins.exe"), "path to Gremlins")
	gomuBinary := fs.String("gomu", filepath.Join(os.TempDir(), "cervomut-study-cobra", "tools", "gomu-patched.exe"), "path to gomu")
	goMutestingBinary := fs.String("go-mutesting", filepath.Join(os.TempDir(), "cervomut-study-cobra", "tools", "go-mutesting-patched.exe"), "path to go-mutesting")
	if err := fs.Parse(reorderFlags(args, map[string]bool{
		"manifest": true, "work-root": true, "output-root": true, "names": true, "tools": true, "workers": true, "compare-target-mode": true, "gremlins-target-mode": true, "gremlins-timeout-coefficient": true, "gomu-workers": true, "go-mutesting-workers": true, "timeout-seconds": true, "min-free-memory-mb": true, "min-free-commit-mb": true, "kill-below-free-memory-mb": true, "kill-below-free-commit-mb": true, "max-used-memory-mb": true, "max-committed-memory-mb": true, "max-process-tree-memory-mb": true, "memory-wait-seconds": true, "memory-poll-seconds": true, "go-memory-limit": true, "go-max-procs": true, "go-flags": true, "resume": true, "cervomutants": true, "gremlins": true, "gomu": true, "go-mutesting": true,
	})); err != nil {
		return err
	}
	run, err := app.deps.runPoolCompare(app.deps.background(), pool.CompareOptions{
		ManifestPath:               *manifest,
		WorkRoot:                   *workRoot,
		OutputRoot:                 *outputRoot,
		Names:                      splitList(*names),
		Tools:                      splitList(*tools),
		Workers:                    *workers,
		CompareTargetMode:          *compareTargetMode,
		GremlinsTargetMode:         *gremlinsTargetMode,
		GremlinsTimeoutCoefficient: *gremlinsTimeoutCoefficient,
		GomuWorkers:                *gomuWorkers,
		GoMutestingWorkers:         *goMutestingWorkers,
		TimeoutSeconds:             *timeoutSeconds,
		MinFreeMemoryMB:            *minFreeMemoryMB,
		MinFreeCommitMB:            *minFreeCommitMB,
		KillBelowFreeMemoryMB:      *killBelowFreeMemoryMB,
		KillBelowFreeCommitMB:      *killBelowFreeCommitMB,
		MaxUsedMemoryMB:            *maxUsedMemoryMB,
		MaxCommittedMemoryMB:       *maxCommittedMemoryMB,
		MaxProcessTreeMemoryMB:     *maxProcessTreeMemoryMB,
		MemoryWaitSeconds:          *memoryWaitSeconds,
		MemoryPollSeconds:          *memoryPollSeconds,
		GoMemoryLimit:              *goMemoryLimit,
		GoMaxProcs:                 *goMaxProcs,
		GoFlags:                    *goFlags,
		Resume:                     *resume,
		CervoBinary:                *cervoBinary,
		GremlinsBinary:             *gremlinsBinary,
		GomuBinary:                 *gomuBinary,
		GoMutestingBinary:          *goMutestingBinary,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Pool comparison raw summary: %s\n", run.SummaryPath)
	if studyJSON := run.Artifacts["study_json"]; studyJSON != "" {
		fmt.Printf("Pool comparison study JSON: %s\n", studyJSON)
	}
	if summaryMarkdown := run.Artifacts["summary_markdown"]; summaryMarkdown != "" {
		fmt.Printf("Pool comparison summary markdown: %s\n", summaryMarkdown)
	}
	return nil
}

func (app *cliApp) cmdPoolBenchmark(args []string) error {
	fs := flag.NewFlagSet("pool benchmark", flag.ContinueOnError)
	corpus := fs.String("corpus", "docs/evaluations/benchmark-corpus.json", "benchmark corpus manifest")
	workRoot := fs.String("work-root", filepath.Join(os.TempDir(), "cervomut-benchmark-corpus"), "working root containing cloned benchmark repositories")
	outputRoot := fs.String("output-root", filepath.Join(os.TempDir(), "cervomut-benchmark-results"), "output directory for benchmark reports and summary")
	names := fs.String("names", "", "comma-separated benchmark names to include")
	limit := fs.Int("limit", 0, "limit benchmark entries after filtering")
	resume := fs.Bool("resume", false, "resume using the existing summary.json")
	cervoBinary := fs.String("cervomutants", currentExecutable(), "path to the cervomut binary used for nested benchmark runs")
	gitBinary := fs.String("git", "git", "path to git")
	if err := fs.Parse(reorderFlags(args, map[string]bool{
		"corpus": true, "work-root": true, "output-root": true, "names": true, "limit": true, "resume": true, "cervomutants": true, "git": true,
	})); err != nil {
		return err
	}
	run, err := app.deps.runPoolBenchmark(app.deps.background(), pool.BenchmarkOptions{
		CorpusPath:  *corpus,
		WorkRoot:    *workRoot,
		OutputRoot:  *outputRoot,
		Names:       splitList(*names),
		Limit:       *limit,
		Resume:      *resume,
		CervoBinary: *cervoBinary,
		GitBinary:   *gitBinary,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Pool benchmark summary: %s\n", run.SummaryPath)
	return nil
}

type compareOptions struct {
	cervo                   string
	cervoTarget             string
	cervoEffectiveTarget    string
	cervoTargetMode         string
	gremlins                string
	gremlinsTarget          string
	gremlinsEffectiveTarget string
	gremlinsTargetMode      string
	gomu                    string
	goMutesting             string
	out                     string
}

func parseCompareOptions(args []string) (compareOptions, error) {
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)
	opts := compareOptions{}
	cervo := fs.String("cervomut", "", "cervo-mutants "+mutationReportFileName)
	cervoTarget := fs.String("cervomut-target", "", "original manifest target used for CervoMutants comparison")
	cervoEffectiveTarget := fs.String("cervomut-effective-target", "", "effective target passed to CervoMutants")
	cervoTargetMode := fs.String("cervomut-target-mode", "manifest", "CervoMutants target normalization mode: manifest or package-root")
	gremlins := fs.String("gremlins", "", "Gremlins report JSON")
	gremlinsTarget := fs.String("gremlins-target", "", "original manifest target used for Gremlins comparison")
	gremlinsEffectiveTarget := fs.String("gremlins-effective-target", "", "effective target passed to Gremlins")
	gremlinsTargetMode := fs.String("gremlins-target-mode", "manifest", "Gremlins target normalization mode: manifest, package-root, or gremlins-package-root")
	gomu := fs.String("gomu", "", "gomu text or JSON summary")
	goMutesting := fs.String("go-mutesting", "", "go-mutesting text summary")
	out := fs.String("out", ".cervomut/evaluation/tool-comparison.json", "normalized comparison output")
	if err := fs.Parse(reorderFlags(args, map[string]bool{
		"cervomut": true, "cervomut-target": true, "cervomut-effective-target": true, "cervomut-target-mode": true, "gremlins": true, "gremlins-target": true, "gremlins-effective-target": true, "gremlins-target-mode": true, "gomu": true, "go-mutesting": true, "out": true,
	})); err != nil {
		return compareOptions{}, err
	}
	opts.cervo = *cervo
	opts.cervoTarget = *cervoTarget
	opts.cervoEffectiveTarget = *cervoEffectiveTarget
	opts.cervoTargetMode = *cervoTargetMode
	opts.gremlins = *gremlins
	opts.gremlinsTarget = *gremlinsTarget
	opts.gremlinsEffectiveTarget = *gremlinsEffectiveTarget
	opts.gremlinsTargetMode = *gremlinsTargetMode
	opts.gomu = *gomu
	opts.goMutesting = *goMutesting
	opts.out = *out
	return opts, nil
}

func loadComparisonResults(opts compareOptions) ([]extcompare.ToolResult, error) {
	var results []extcompare.ToolResult
	if opts.cervo != "" {
		result, err := parseTargetedReport(opts.cervo, opts.cervoTarget, opts.cervoEffectiveTarget, opts.cervoTargetMode, extcompare.ParseCervo, extcompare.NormalizeTarget)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	if opts.gremlins != "" {
		result, err := parseTargetedReport(opts.gremlins, opts.gremlinsTarget, opts.gremlinsEffectiveTarget, opts.gremlinsTargetMode, extcompare.ParseGremlins, extcompare.NormalizeGremlinsTarget)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	for _, external := range []struct {
		path  string
		parse func(string) (extcompare.ToolResult, error)
	}{{opts.gomu, extcompare.ParseGomu}, {opts.goMutesting, extcompare.ParseGoMutesting}} {
		if external.path == "" {
			continue
		}
		result, err := external.parse(external.path)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func parseTargetedReport(path, target, effective, mode string, parse func(string) (extcompare.ToolResult, error), normalize func(string, string) (string, bool)) (extcompare.ToolResult, error) {
	result, err := parse(path)
	if err != nil {
		return extcompare.ToolResult{}, err
	}
	notComparable := false
	if target != "" && effective == "" {
		effective, notComparable = normalize(target, mode)
	} else if target != "" && effective != "" && target != effective {
		notComparable = true
	}
	if target != "" || effective != "" {
		result = extcompare.ApplyTargetMode(result, target, effective, mode, notComparable)
	}
	return result, nil
}
