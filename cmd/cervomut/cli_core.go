package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cervantesh/cervo-mutants/pkg/config"
	"github.com/cervantesh/cervo-mutants/pkg/doctor"
	"github.com/cervantesh/cervo-mutants/pkg/engine"
	"github.com/cervantesh/cervo-mutants/pkg/mutator"
	"github.com/cervantesh/cervo-mutants/pkg/report"
)

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

func cmdReport(args []string) error {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	out := fs.String("out", "", reportOutputDirectoryDoc)
	actionableOnly := fs.Bool("actionable-only", false, "show only actionable survivor views while preserving the raw report model")
	if err := fs.Parse(reorderFlags(args, map[string]bool{"out": true})); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("report requires summary, survivors, sarif, github-summary, or open")
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
		fmt.Print(report.SurvivorsWithOptions(result, report.SurvivorsOptions{ActionableOnly: *actionableOnly}))
	case "sarif":
		data, err := report.SARIF(result)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	case "github-summary":
		fmt.Print(report.GitHubSummary(result))
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

func loadConfigIfPresentStrict() (config.Config, error) {
	if _, err := os.Stat(configFileName); err == nil {
		return config.Load(configFileName)
	} else if !os.IsNotExist(err) {
		return config.Config{}, err
	}
	return config.Defaults(), nil
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
  slice_by: ""
  shard_index: 0
  shard_count: 0
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
  temp_root: ""
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
  max_mutants_per_package: 0
  max_files_per_run: 0
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
  actionable_only: false
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

func currentExecutable() string {
	path, err := os.Executable()
	if err != nil {
		return "cervomut"
	}
	return path
}

func splitList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
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
