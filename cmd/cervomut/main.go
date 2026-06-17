package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/cervantesh/cervo-mutants/pkg/baseline"
	"github.com/cervantesh/cervo-mutants/pkg/config"
	"github.com/cervantesh/cervo-mutants/pkg/daemon"
	"github.com/cervantesh/cervo-mutants/pkg/doctor"
	"github.com/cervantesh/cervo-mutants/pkg/engine"
	evalpkg "github.com/cervantesh/cervo-mutants/pkg/eval"
	"github.com/cervantesh/cervo-mutants/pkg/extcompare"
	"github.com/cervantesh/cervo-mutants/pkg/mutator"
	"github.com/cervantesh/cervo-mutants/pkg/report"
)

const (
	configFileName           = "cervomut.yaml"
	mutationReportFileName   = "mutation-report.json"
	flagTestTimeout          = "test-timeout"
	flagMaxMutants           = "max-mutants"
	flagMaxProcessMemoryMB   = "max-process-memory-mb"
	reportOutputDirectoryDoc = "report output directory"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCode(err))
	}
}

func run(args []string) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			stack := strings.TrimSpace(string(debug.Stack()))
			err = fmt.Errorf("internal_error: unexpected panic: %v\n%s", recovered, stack)
		}
	}()
	if len(args) == 0 {
		usage()
		return nil
	}
	switch args[0] {
	case "help", "--help", "-h":
		usage()
		return nil
	case "init":
		return cmdInit()
	case "doctor":
		return cmdDoctor()
	case "affected":
		return cmdAffected(args[1:])
	case "run":
		return cmdRun(args[1:])
	case "fast":
		return cmdFast(args[1:])
	case "eval":
		return cmdEval(args[1:])
	case "compare":
		return cmdCompare(args[1:])
	case "baseline":
		return cmdBaseline(args[1:])
	case "report":
		return cmdReport(args[1:])
	case "show":
		return cmdShow(args[1:])
	case "explain":
		return cmdExplain(args[1:])
	case "list-mutators":
		return cmdListMutators()
	case "daemon", "worker":
		return daemon.ServeJSONLines(context.Background(), os.Stdin, os.Stdout, daemon.WorkerRunner{MaxOutputBytes: 12000})
	default:
		usage()
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func usage() {
	fmt.Println("usage: cervomut <init|doctor|affected|run|fast|eval|compare|baseline|report|show|explain|list-mutators|daemon|worker>")
}

func cmdInit() error {
	if _, err := os.Stat(configFileName); err == nil {
		return fmt.Errorf("%s already exists", configFileName)
	}
	return os.WriteFile(configFileName, []byte(defaultConfigYAML()), 0o644)
}

func cmdDoctor() error {
	checks := doctor.Run(context.Background())
	ok := true
	for _, check := range checks {
		status := "ok"
		if !check.OK {
			status = "fail"
			ok = false
		} else if check.Severity == "warn" {
			status = "warn"
		}
		fmt.Printf("%s %s %s", status, check.Name, check.Message)
		if !strings.HasSuffix(check.Message, "\n") {
			fmt.Println()
		}
	}
	if !ok {
		return fmt.Errorf("doctor found failing checks")
	}
	return nil
}

func cmdAffected(args []string) error {
	fs := flag.NewFlagSet("affected", flag.ContinueOnError)
	scope := fs.String("scope", "all", "scope mode")
	since := fs.String("since", "origin/main", "git base")
	if err := fs.Parse(reorderFlags(args, map[string]bool{"scope": true, "since": true})); err != nil {
		return err
	}
	cfg := loadConfigIfPresent()
	result, err := engine.New(cfg).Affected(context.Background(), engine.AffectedRequest{Targets: fs.Args(), Scope: *scope, Since: *since})
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}

func cmdRun(args []string) error {
	opts, targets, err := parseRunOptions(args)
	if err != nil {
		return err
	}
	cfg := loadConfigIfPresent()
	applyRunOptions(&cfg, opts)
	if err := cfg.Validate(); err != nil {
		return err
	}
	result, err := engine.New(cfg).Run(context.Background(), engine.RunRequest{Targets: targets, DryRun: opts.dryRun})
	if err != nil {
		return err
	}
	return writeRunResult(cfg, result, opts.dryRun)
}

type runOptions struct {
	dryRun           bool
	scope            string
	budget           flagDuration
	testTimeout      flagDuration
	maxMutants       int
	sample           string
	reportFormats    string
	out              string
	workers          int
	isolation        string
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
	scope := fs.String("scope", "", "scope mode")
	since := fs.String("since", "", "git base")
	budget := fs.Duration("budget", 0, "run budget")
	testTimeout := fs.Duration(flagTestTimeout, 0, "per-mutant go test timeout")
	maxMutants := fs.Int(flagMaxMutants, 0, "max mutants")
	sample := fs.String("sample", "", "sampling mode")
	reportFormats := fs.String("report", "", "comma-separated report formats")
	out := fs.String("out", "", reportOutputDirectoryDoc)
	workers := fs.Int("workers", 0, "parallel mutation workers")
	isolation := fs.String("isolation", "", "isolation backend: temp-workdir or overlay")
	policy := fs.String("policy", "", "policy preset: ci-fast, ci-balanced, comparison-safe, nightly, or campaign")
	profile := fs.String("profile", "", "mutator profile")
	prefilter := fs.Bool("coverage-prefilter", false, "use coverage profile as a prefilter")
	resume := fs.Bool("resume", false, "resume from partial-mutation-report.json in the output directory")
	maxProcessMemory := fs.Int(flagMaxProcessMemoryMB, 0, "best-effort process-tree memory cap in MB")
	if err := fs.Parse(reorderFlags(args, map[string]bool{
		"scope": true, "since": true, "budget": true, flagTestTimeout: true, flagMaxMutants: true, "sample": true, "report": true, "out": true, "workers": true, "isolation": true, "policy": true, "profile": true, flagMaxProcessMemoryMB: true,
	})); err != nil {
		return runOptions{}, nil, err
	}
	opts.scope = *scope
	_ = since
	opts.budget = flagDuration{value: *budget, set: *budget > 0}
	opts.testTimeout = flagDuration{value: *testTimeout, set: *testTimeout > 0}
	opts.maxMutants = *maxMutants
	opts.sample = *sample
	opts.reportFormats = *reportFormats
	opts.out = *out
	opts.workers = *workers
	opts.isolation = *isolation
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
	setString(&cfg.Mutators.Profile, opts.profile)
	setString(&cfg.Limits.Sample, opts.sample)
	setString(&cfg.Execution.Isolation, opts.isolation)
	if opts.prefilter {
		cfg.Selection.Prefilter = true
	}
	if opts.resume {
		cfg.Execution.Resume = true
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
	if err := report.WriteFormats(cfg.Reports.Output, result, cfg.Reports.Formats); err != nil {
		return err
	}
	fmt.Print(report.Summary(result))
	if cfg.CI.FailUnder > 0 && int(result.Summary.Score) < cfg.CI.FailUnder {
		return fmt.Errorf("mutation score %.2f below threshold %d", result.Summary.Score, cfg.CI.FailUnder)
	}
	return nil
}

func cmdFast(args []string) error {
	next := append([]string{"--policy", "ci-fast", "--report", "summary,json,junit"}, args...)
	return cmdRun(next)
}

func cmdEval(args []string) error {
	fs := flag.NewFlagSet("eval", flag.ContinueOnError)
	out := fs.String("out", ".cervomut/evaluation", "evaluation output directory")
	framework := fs.String("framework", "cervosoft", "evaluation framework")
	budget := fs.Duration("budget", 0, "run budget")
	testTimeout := fs.Duration(flagTestTimeout, 0, "per-mutant go test timeout")
	maxMutants := fs.Int(flagMaxMutants, 0, "max mutants")
	sample := fs.String("sample", "", "sampling mode")
	workers := fs.Int("workers", 0, "parallel mutation workers")
	isolation := fs.String("isolation", "", "isolation backend: temp-workdir or overlay")
	policy := fs.String("policy", "", "policy preset: ci-fast, ci-balanced, comparison-safe, nightly, or campaign")
	resume := fs.Bool("resume", false, "resume from partial-mutation-report.json in the output directory")
	maxProcessMemory := fs.Int(flagMaxProcessMemoryMB, 0, "best-effort process-tree memory cap in MB")
	if err := fs.Parse(reorderFlags(args, map[string]bool{
		"out": true, "framework": true, "budget": true, flagTestTimeout: true, flagMaxMutants: true, "sample": true, "workers": true, "isolation": true, "policy": true, flagMaxProcessMemoryMB: true,
	})); err != nil {
		return err
	}
	cfg := loadConfigIfPresent()
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
	cfg.Reports.Output = *out
	cfg.Cache.Path = filepath.Join(*out, "cache")
	cfg.Selection.CoverageProfile = filepath.Join(*out, "coverage.out")
	cfg.Selection.TimingsPath = filepath.Join(*out, "timings.json")
	cfg.History.Path = filepath.Join(*out, "history.json")
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
	if err := cfg.Validate(); err != nil {
		return err
	}
	targets := fs.Args()
	runResult, err := engine.New(cfg).Run(context.Background(), engine.RunRequest{Targets: targets})
	if err != nil {
		return err
	}
	if err := report.WriteFormats(cfg.Reports.Output, runResult, cfg.Reports.Formats); err != nil {
		return err
	}
	evaluation := evalpkg.Build(evalpkg.BuildRequest{
		Tool:       "cervo-mutant",
		Target:     strings.Join(targets, " "),
		Commit:     currentCommit(),
		Command:    append([]string{"cervomut", "eval"}, args...),
		Framework:  *framework,
		Run:        runResult,
		ManualMode: true,
	})
	if err := evalpkg.Write(*out, evaluation); err != nil {
		return err
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
	cervo := fs.String("cervomut", "", "cervo-mutant "+mutationReportFileName)
	cervoTarget := fs.String("cervomut-target", "", "original manifest target used for CervoMutant comparison")
	cervoEffectiveTarget := fs.String("cervomut-effective-target", "", "effective target passed to CervoMutant")
	cervoTargetMode := fs.String("cervomut-target-mode", "manifest", "CervoMutant target normalization mode: manifest or package-root")
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

func cmdBaseline(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("baseline requires update or compare")
	}
	cfg := loadConfigIfPresent()
	switch args[0] {
	case "update":
		return updateBaseline(cfg)
	case "compare":
		return compareBaselineCommand(cfg)
	default:
		return fmt.Errorf("unknown baseline command %q", args[0])
	}
}

func updateBaseline(cfg config.Config) error {
	result, err := readRunReport(filepath.Join(cfg.Reports.Output, mutationReportFileName))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cfg.Baseline.Path), 0o755); err != nil {
		return err
	}
	return baseline.Save(cfg.Baseline.Path, result)
}

func compareBaselineCommand(cfg config.Config) error {
	prev, ok, err := baseline.Load(cfg.Baseline.Path)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("baseline not found")
	}
	current, err := readRunReport(filepath.Join(cfg.Reports.Output, mutationReportFileName))
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(baseline.Compare(prev, current))
}

func readRunReport(path string) (engine.RunResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return engine.RunResult{}, err
	}
	var result engine.RunResult
	return result, json.Unmarshal(data, &result)
}

func cmdReport(args []string) error {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	out := fs.String("out", "", reportOutputDirectoryDoc)
	if err := fs.Parse(reorderFlags(args, map[string]bool{"out": true})); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("report requires summary, survivors, or open")
	}
	cfg := loadConfigIfPresent()
	if *out != "" {
		cfg.Reports.Output = *out
	}
	data, err := os.ReadFile(filepath.Join(cfg.Reports.Output, mutationReportFileName))
	if err != nil {
		return err
	}
	var result engine.RunResult
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	action := fs.Arg(0)
	switch action {
	case "summary":
		fmt.Print(report.Summary(result))
	case "survivors":
		fmt.Print(report.Survivors(result))
	case "open":
		path := filepath.Join(cfg.Reports.Output, "index.html")
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", path).Start()
	default:
		return fmt.Errorf("unknown report command %q", action)
	}
	return nil
}

func cmdShow(args []string) error {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	out := fs.String("out", "", reportOutputDirectoryDoc)
	if err := fs.Parse(reorderFlags(args, map[string]bool{"out": true})); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("show requires mutant id")
	}
	cfg := loadConfigIfPresent()
	if *out != "" {
		cfg.Reports.Output = *out
	}
	result, err := loadLastRun(cfg)
	if err != nil {
		return err
	}
	id := fs.Arg(0)
	for _, mutant := range result.Mutants {
		if mutant.MutantID == id || mutant.Mutant.ID == id {
			data, _ := json.MarshalIndent(mutant, "", "  ")
			fmt.Println(string(data))
			return nil
		}
	}
	return fmt.Errorf("mutant %q not found", id)
}

func cmdExplain(args []string) error {
	fs := flag.NewFlagSet("explain", flag.ContinueOnError)
	format := fs.String("format", "text", "text or json")
	if err := fs.Parse(reorderFlags(args, map[string]bool{"format": true})); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("explain requires mutant id")
	}
	cfg := loadConfigIfPresent()
	explained, err := engine.New(cfg).Explain(context.Background(), engine.ExplainRequest{MutantID: fs.Arg(0), Format: *format})
	if err != nil {
		return err
	}
	if *format == "json" {
		return json.NewEncoder(os.Stdout).Encode(explained)
	}
	fmt.Printf("%s\n%s\n", explained.Explanation, explained.Suggestion)
	return nil
}

func cmdListMutators() error {
	return json.NewEncoder(os.Stdout).Encode(mutator.Definitions())
}

func loadConfigIfPresent() config.Config {
	if _, err := os.Stat(configFileName); err == nil {
		if cfg, err := config.Load(configFileName); err == nil {
			return cfg
		}
	}
	return config.Defaults()
}

func loadLastRun(cfg config.Config) (engine.RunResult, error) {
	data, err := os.ReadFile(filepath.Join(cfg.Reports.Output, mutationReportFileName))
	if err != nil {
		return engine.RunResult{}, err
	}
	var result engine.RunResult
	return result, json.Unmarshal(data, &result)
}

func defaultConfigYAML() string {
	return `version: 1
policy: ""
scope:
  mode: all
  since: origin/main
  include: ["./..."]
  exclude: ["**/*_generated.go", "**/vendor/**"]
tests:
  command: ["go", "test", "./..."]
  timeout: 30s
  no_tests: warn
  baseline_required: true
mutators:
  profile: conservative
execution:
  workers: 4
  isolation: temp-workdir
  budget: 0s
  fail_fast: false
  resume: false
  checkpoint_includes: ["testdata/**", "fixtures/**"]
  resources:
    max_process_memory_mb: 0
    max_processes: 0
selection:
  mode: package
  prefilter: false
  use_timings: true
  coverage_profile: .cervomut/coverage.out
  timings_path: .cervomut/timings.json
suppression:
  enabled: true
  rules:
    - name: audit-high-equivalent-risk
      equivalent_risk: high
      action: report-only
      reason: High equivalent-mutant risk must be visible before suppression is allowed.
      evidence: heuristic
    - name: lower-priority-loop-control
      operator: loop-control
      action: lower-priority
      reason: Loop-control mutants are high-signal but often require manual review.
    - name: lower-priority-broad-literals
      operator: literals
      action: lower-priority
      reason: Broad literal mutants often need equivalence review before CI gating.
    - name: lower-priority-broad-returns
      operator: returns
      action: lower-priority
      reason: Broad return mutants can duplicate narrower return-bool-literal signal.
history:
  enabled: true
  path: .cervomut/history.json
cache:
  enabled: true
  path: .cervomut/cache
  mode: incremental
baseline:
  enabled: true
  path: .cervomut/baseline.json
  fail_on_regression: true
  fail_on_new_survivors: true
limits:
  max_mutants: 0
  sample: none
  seed: 0
ci:
  fail_under: 0
  fail_on_timeout: true
  fail_on_compile_error: false
ignore:
  files: ["**/*_generated.go"]
  packages: []
  mutators: []
  comments:
    enabled: true
    require_reason: true
quarantine:
  enabled: true
  path: .cervomut/quarantine.json
  expire_after: 720h
  require_reason: true
  require_owner: true
  require_issue: true
  fail_on_expired: true
  max_renewals: 1
reports:
  output: .cervomut/reports
  formats: [summary, json, junit, html]
  detail: standard
  include_diff: true
  include_test_output: failed-only
  max_output_bytes: 12000
`
}

func currentCommit() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func exitCode(err error) int {
	msg := err.Error()
	if strings.Contains(msg, "baseline tests failed") {
		return 3
	}
	if strings.Contains(msg, "threshold") {
		return 1
	}
	return 2
}

func reorderFlags(args []string, takesValue map[string]bool) []string {
	var flags []string
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positional = append(positional, arg)
			continue
		}
		flags = append(flags, arg)
		name := strings.TrimLeft(arg, "-")
		if eq := strings.Index(name, "="); eq >= 0 {
			continue
		}
		if takesValue[name] && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return append(flags, positional...)
}
