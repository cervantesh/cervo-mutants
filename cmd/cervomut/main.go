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

	"gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/baseline"
	"gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/config"
	"gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/daemon"
	"gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/doctor"
	"gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/engine"
	evalpkg "gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/eval"
	"gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/extcompare"
	"gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/mutator"
	"gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/report"
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
	if _, err := os.Stat("cervomut.yaml"); err == nil {
		return fmt.Errorf("cervomut.yaml already exists")
	}
	return os.WriteFile("cervomut.yaml", []byte(defaultConfigYAML()), 0o644)
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
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "only discover mutants")
	scope := fs.String("scope", "", "scope mode")
	since := fs.String("since", "", "git base")
	budget := fs.Duration("budget", 0, "run budget")
	testTimeout := fs.Duration("test-timeout", 0, "per-mutant go test timeout")
	maxMutants := fs.Int("max-mutants", 0, "max mutants")
	sample := fs.String("sample", "", "sampling mode")
	reportFormats := fs.String("report", "", "comma-separated report formats")
	out := fs.String("out", "", "report output directory")
	workers := fs.Int("workers", 0, "parallel mutation workers")
	isolation := fs.String("isolation", "", "isolation backend: temp-workdir or overlay")
	policy := fs.String("policy", "", "policy preset: ci-fast, ci-balanced, comparison-safe, nightly, or campaign")
	profile := fs.String("profile", "", "mutator profile")
	prefilter := fs.Bool("coverage-prefilter", false, "use coverage profile as a prefilter")
	resume := fs.Bool("resume", false, "resume from partial-mutation-report.json in the output directory")
	maxProcessMemory := fs.Int("max-process-memory-mb", 0, "best-effort process-tree memory cap in MB")
	if err := fs.Parse(reorderFlags(args, map[string]bool{
		"scope": true, "since": true, "budget": true, "test-timeout": true, "max-mutants": true, "sample": true, "report": true, "out": true, "workers": true, "isolation": true, "policy": true, "profile": true, "max-process-memory-mb": true,
	})); err != nil {
		return err
	}
	_ = since
	cfg := loadConfigIfPresent()
	if *policy != "" {
		cfg.Policy = *policy
		cfg = config.ApplyPolicy(cfg)
	}
	if *scope != "" {
		cfg.Scope.Mode = *scope
	}
	if *profile != "" {
		cfg.Mutators.Profile = *profile
	}
	if *prefilter {
		cfg.Selection.Prefilter = true
	}
	if *resume {
		cfg.Execution.Resume = true
	}
	if *maxProcessMemory > 0 {
		cfg.Execution.Resources.MaxProcessMemoryMB = *maxProcessMemory
	}
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
	if *reportFormats != "" {
		cfg.Reports.Formats = strings.Split(*reportFormats, ",")
	}
	if *out != "" {
		cfg.Reports.Output = *out
		cfg.Cache.Path = filepath.Join(*out, "cache")
		cfg.Selection.CoverageProfile = filepath.Join(*out, "coverage.out")
		cfg.Selection.TimingsPath = filepath.Join(*out, "timings.json")
		cfg.History.Path = filepath.Join(*out, "history.json")
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	result, err := engine.New(cfg).Run(context.Background(), engine.RunRequest{Targets: fs.Args(), DryRun: *dryRun})
	if err != nil {
		return err
	}
	if *dryRun {
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
	testTimeout := fs.Duration("test-timeout", 0, "per-mutant go test timeout")
	maxMutants := fs.Int("max-mutants", 0, "max mutants")
	sample := fs.String("sample", "", "sampling mode")
	workers := fs.Int("workers", 0, "parallel mutation workers")
	isolation := fs.String("isolation", "", "isolation backend: temp-workdir or overlay")
	policy := fs.String("policy", "", "policy preset: ci-fast, ci-balanced, comparison-safe, nightly, or campaign")
	resume := fs.Bool("resume", false, "resume from partial-mutation-report.json in the output directory")
	maxProcessMemory := fs.Int("max-process-memory-mb", 0, "best-effort process-tree memory cap in MB")
	if err := fs.Parse(reorderFlags(args, map[string]bool{
		"out": true, "framework": true, "budget": true, "test-timeout": true, "max-mutants": true, "sample": true, "workers": true, "isolation": true, "policy": true, "max-process-memory-mb": true,
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
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)
	cervo := fs.String("cervomut", "", "cervo-mutant mutation-report.json")
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
		return err
	}
	var results []extcompare.ToolResult
	if *cervo != "" {
		result, err := extcompare.ParseCervo(*cervo)
		if err != nil {
			return err
		}
		target := *cervoTarget
		effective := *cervoEffectiveTarget
		notComparable := false
		if target != "" && effective == "" {
			effective, notComparable = extcompare.NormalizeTarget(target, *cervoTargetMode)
		} else if target != "" && effective != "" && target != effective {
			notComparable = true
		}
		if target != "" || effective != "" {
			result = extcompare.ApplyTargetMode(result, target, effective, *cervoTargetMode, notComparable)
		}
		results = append(results, result)
	}
	if *gremlins != "" {
		result, err := extcompare.ParseGremlins(*gremlins)
		if err != nil {
			return err
		}
		target := *gremlinsTarget
		effective := *gremlinsEffectiveTarget
		notComparable := false
		if target != "" && effective == "" {
			effective, notComparable = extcompare.NormalizeGremlinsTarget(target, *gremlinsTargetMode)
		} else if target != "" && effective != "" && target != effective {
			notComparable = true
		}
		if target != "" || effective != "" {
			result = extcompare.ApplyTargetMode(result, target, effective, *gremlinsTargetMode, notComparable)
		}
		results = append(results, result)
	}
	if *gomu != "" {
		result, err := extcompare.ParseGomu(*gomu)
		if err != nil {
			return err
		}
		results = append(results, result)
	}
	if *goMutesting != "" {
		result, err := extcompare.ParseGoMutesting(*goMutesting)
		if err != nil {
			return err
		}
		results = append(results, result)
	}
	if len(results) == 0 {
		return fmt.Errorf("compare requires at least one tool report")
	}
	if err := extcompare.Write(*out, results); err != nil {
		return err
	}
	fmt.Printf("Tool comparison written to %s\n", *out)
	return nil
}

func cmdBaseline(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("baseline requires update or compare")
	}
	cfg := loadConfigIfPresent()
	switch args[0] {
	case "update":
		data, err := os.ReadFile(filepath.Join(cfg.Reports.Output, "mutation-report.json"))
		if err != nil {
			return err
		}
		var result engine.RunResult
		if err := json.Unmarshal(data, &result); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(cfg.Baseline.Path), 0o755); err != nil {
			return err
		}
		return baseline.Save(cfg.Baseline.Path, result)
	case "compare":
		prev, ok, err := baseline.Load(cfg.Baseline.Path)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("baseline not found")
		}
		data, err := os.ReadFile(filepath.Join(cfg.Reports.Output, "mutation-report.json"))
		if err != nil {
			return err
		}
		var current engine.RunResult
		if err := json.Unmarshal(data, &current); err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(baseline.Compare(prev, current))
	default:
		return fmt.Errorf("unknown baseline command %q", args[0])
	}
}

func cmdReport(args []string) error {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	out := fs.String("out", "", "report output directory")
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
	data, err := os.ReadFile(filepath.Join(cfg.Reports.Output, "mutation-report.json"))
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
	out := fs.String("out", "", "report output directory")
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
	if _, err := os.Stat("cervomut.yaml"); err == nil {
		if cfg, err := config.Load("cervomut.yaml"); err == nil {
			return cfg
		}
	}
	return config.Defaults()
}

func loadLastRun(cfg config.Config) (engine.RunResult, error) {
	data, err := os.ReadFile(filepath.Join(cfg.Reports.Output, "mutation-report.json"))
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
