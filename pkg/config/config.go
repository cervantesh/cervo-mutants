package config

import (
	"errors"
	"os"
	"runtime"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version     int         `yaml:"version" json:"version"`
	Policy      string      `yaml:"policy" json:"policy"`
	Scope       Scope       `yaml:"scope" json:"scope"`
	Tests       Tests       `yaml:"tests" json:"tests"`
	Mutators    Mutators    `yaml:"mutators" json:"mutators"`
	Execution   Execution   `yaml:"execution" json:"execution"`
	Selection   Selection   `yaml:"selection" json:"selection"`
	Suppression Suppression `yaml:"suppression" json:"suppression"`
	History     History     `yaml:"history" json:"history"`
	Cache       Cache       `yaml:"cache" json:"cache"`
	Baseline    Baseline    `yaml:"baseline" json:"baseline"`
	Limits      Limits      `yaml:"limits" json:"limits"`
	CI          CI          `yaml:"ci" json:"ci"`
	Ignore      Ignore      `yaml:"ignore" json:"ignore"`
	Quarantine  Quarantine  `yaml:"quarantine" json:"quarantine"`
	Reports     Reports     `yaml:"reports" json:"reports"`
}

type Scope struct {
	Mode    string   `yaml:"mode" json:"mode"`
	Since   string   `yaml:"since" json:"since"`
	Include []string `yaml:"include" json:"include"`
	Exclude []string `yaml:"exclude" json:"exclude"`
}

type Tests struct {
	Command          []string      `yaml:"command" json:"command"`
	Timeout          time.Duration `yaml:"timeout" json:"timeout"`
	NoTests          string        `yaml:"no_tests" json:"no_tests"`
	BaselineRequired bool          `yaml:"baseline_required" json:"baseline_required"`
}

type Mutators struct {
	Profile  string   `yaml:"profile" json:"profile"`
	Enabled  []string `yaml:"enabled" json:"enabled"`
	Disabled []string `yaml:"disabled" json:"disabled"`
}

type Execution struct {
	Workers            int           `yaml:"workers" json:"workers"`
	Isolation          string        `yaml:"isolation" json:"isolation"`
	Budget             time.Duration `yaml:"budget" json:"budget"`
	FailFast           bool          `yaml:"fail_fast" json:"fail_fast"`
	Resume             bool          `yaml:"resume" json:"resume"`
	CheckpointIncludes []string      `yaml:"checkpoint_includes" json:"checkpoint_includes"`
	Resources          Resources     `yaml:"resources" json:"resources"`
}

type Resources struct {
	MaxProcessMemoryMB int `yaml:"max_process_memory_mb" json:"max_process_memory_mb"`
	MaxProcesses       int `yaml:"max_processes" json:"max_processes"`
}

type Selection struct {
	Mode            string `yaml:"mode" json:"mode"`
	Prefilter       bool   `yaml:"prefilter" json:"prefilter"`
	UseTimings      bool   `yaml:"use_timings" json:"use_timings"`
	CoverageProfile string `yaml:"coverage_profile" json:"coverage_profile"`
	TimingsPath     string `yaml:"timings_path" json:"timings_path"`
}

type Suppression struct {
	Enabled bool              `yaml:"enabled" json:"enabled"`
	Rules   []SuppressionRule `yaml:"rules" json:"rules"`
}

type SuppressionRule struct {
	Name           string `yaml:"name" json:"name"`
	Operator       string `yaml:"operator" json:"operator"`
	File           string `yaml:"file" json:"file"`
	Original       string `yaml:"original" json:"original"`
	Mutated        string `yaml:"mutated" json:"mutated"`
	EquivalentRisk string `yaml:"equivalent_risk" json:"equivalent_risk"`
	Action         string `yaml:"action" json:"action"`
	Reason         string `yaml:"reason" json:"reason"`
	Evidence       string `yaml:"evidence" json:"evidence"`
	Reviewers      int    `yaml:"reviewers" json:"reviewers"`
}

type History struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Path    string `yaml:"path" json:"path"`
}

type Cache struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Path    string `yaml:"path" json:"path"`
	Mode    string `yaml:"mode" json:"mode"`
}

type Baseline struct {
	Enabled               bool   `yaml:"enabled" json:"enabled"`
	Path                  string `yaml:"path" json:"path"`
	FailOnRegression      bool   `yaml:"fail_on_regression" json:"fail_on_regression"`
	FailOnNewSurvivors    bool   `yaml:"fail_on_new_survivors" json:"fail_on_new_survivors"`
	BaselineMutationScore int    `yaml:"baseline_mutation_score" json:"baseline_mutation_score"`
}

type Limits struct {
	MaxMutants int    `yaml:"max_mutants" json:"max_mutants"`
	Sample     string `yaml:"sample" json:"sample"`
	Seed       int64  `yaml:"seed" json:"seed"`
}

type CI struct {
	FailUnder          int  `yaml:"fail_under" json:"fail_under"`
	FailOnTimeout      bool `yaml:"fail_on_timeout" json:"fail_on_timeout"`
	FailOnCompileError bool `yaml:"fail_on_compile_error" json:"fail_on_compile_error"`
}

type Ignore struct {
	Files    []string `yaml:"files" json:"files"`
	Packages []string `yaml:"packages" json:"packages"`
	Mutators []string `yaml:"mutators" json:"mutators"`
	Comments struct {
		Enabled       bool `yaml:"enabled" json:"enabled"`
		RequireReason bool `yaml:"require_reason" json:"require_reason"`
	} `yaml:"comments" json:"comments"`
}

type Quarantine struct {
	Enabled       bool          `yaml:"enabled" json:"enabled"`
	Path          string        `yaml:"path" json:"path"`
	ExpireAfter   time.Duration `yaml:"expire_after" json:"expire_after"`
	RequireReason bool          `yaml:"require_reason" json:"require_reason"`
	RequireOwner  bool          `yaml:"require_owner" json:"require_owner"`
	RequireIssue  bool          `yaml:"require_issue" json:"require_issue"`
	FailOnExpired bool          `yaml:"fail_on_expired" json:"fail_on_expired"`
	MaxRenewals   int           `yaml:"max_renewals" json:"max_renewals"`
}

type Reports struct {
	Output            string   `yaml:"output" json:"output"`
	Formats           []string `yaml:"formats" json:"formats"`
	Detail            string   `yaml:"detail" json:"detail"`
	IncludeDiff       bool     `yaml:"include_diff" json:"include_diff"`
	IncludeTestOutput string   `yaml:"include_test_output" json:"include_test_output"`
	MaxOutputBytes    int      `yaml:"max_output_bytes" json:"max_output_bytes"`
}

func Defaults() Config {
	workers := runtime.NumCPU()
	if runtime.GOOS == "windows" && workers > 4 {
		workers = 4
	}
	cfg := Config{
		Version: 1,
		Scope: Scope{
			Mode:    "all",
			Since:   "origin/main",
			Include: []string{"./..."},
			Exclude: []string{"**/*_generated.go", "**/vendor/**"},
		},
		Tests: Tests{
			Command:          []string{"go", "test", "./..."},
			Timeout:          30 * time.Second,
			NoTests:          "warn",
			BaselineRequired: true,
		},
		Mutators:  Mutators{Profile: "conservative"},
		Execution: Execution{Workers: workers, Isolation: "temp-workdir", CheckpointIncludes: []string{"testdata/**", "fixtures/**"}},
		Selection: Selection{Mode: "package", UseTimings: true, CoverageProfile: ".cervomut/coverage.out", TimingsPath: ".cervomut/timings.json"},
		Suppression: Suppression{Enabled: true, Rules: []SuppressionRule{
			{Name: "audit-high-equivalent-risk", EquivalentRisk: "high", Action: "report-only", Reason: "High equivalent-mutant risk must be visible before suppression is allowed.", Evidence: "heuristic"},
			{Name: "lower-priority-loop-control", Operator: "loop-control", Action: "lower-priority", Reason: "Loop-control mutants are high-signal but often require manual review."},
			{Name: "lower-priority-broad-literals", Operator: "literals", Action: "lower-priority", Reason: "Broad literal mutants often need equivalence review before CI gating."},
			{Name: "lower-priority-broad-returns", Operator: "returns", Action: "lower-priority", Reason: "Broad return mutants can duplicate narrower return-bool-literal signal."},
		}},
		History:  History{Enabled: true, Path: ".cervomut/history.json"},
		Cache:    Cache{Enabled: true, Path: ".cervomut/cache", Mode: "incremental"},
		Baseline: Baseline{Enabled: true, Path: ".cervomut/baseline.json", FailOnRegression: true, FailOnNewSurvivors: true},
		Limits:   Limits{Sample: "none"},
		CI:       CI{FailUnder: 0, FailOnTimeout: true},
		Quarantine: Quarantine{
			Enabled:       true,
			Path:          ".cervomut/quarantine.json",
			ExpireAfter:   30 * 24 * time.Hour,
			RequireReason: true,
			RequireOwner:  true,
			RequireIssue:  true,
			FailOnExpired: true,
			MaxRenewals:   1,
		},
		Reports: Reports{
			Output:            ".cervomut/reports",
			Formats:           []string{"summary", "json", "junit", "html"},
			Detail:            "standard",
			IncludeDiff:       true,
			IncludeTestOutput: "failed-only",
			MaxOutputBytes:    12000,
		},
	}
	cfg.Ignore.Files = []string{"**/*_generated.go"}
	cfg.Ignore.Comments.Enabled = true
	cfg.Ignore.Comments.RequireReason = true
	return cfg
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	cfg := Defaults()
	var header struct {
		Policy string `yaml:"policy"`
	}
	if err := yaml.Unmarshal(data, &header); err != nil {
		return Config{}, err
	}
	if header.Policy != "" {
		cfg.Policy = header.Policy
		cfg = ApplyPolicy(cfg)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (cfg Config) Validate() error {
	if cfg.Policy != "" && !oneOf(cfg.Policy, "ci-fast", "ci-balanced", "nightly", "campaign", "comparison-safe") {
		return errors.New("policy must be ci-fast, ci-balanced, nightly, campaign, or comparison-safe")
	}
	if !oneOf(cfg.Scope.Mode, "all", "changed", "packages") {
		return errors.New("scope.mode must be all, changed, or packages")
	}
	if !oneOf(cfg.Selection.Mode, "all", "package", "coverage") {
		return errors.New("selection.mode must be all, package, or coverage")
	}
	if !oneOf(cfg.Mutators.Profile, "gremlins-compatible", "conservative-fast", "conservative", "default", "aggressive") {
		return errors.New("mutators.profile must be gremlins-compatible, conservative-fast, conservative, default, or aggressive")
	}
	if !oneOf(cfg.Execution.Isolation, "temp-workdir", "overlay") {
		return errors.New("execution.isolation must be temp-workdir or overlay")
	}
	if !oneOf(cfg.Cache.Mode, "off", "read-only", "incremental") {
		return errors.New("cache.mode must be off, read-only, or incremental")
	}
	if !oneOf(cfg.Limits.Sample, "none", "random", "deterministic") {
		return errors.New("limits.sample must be none, random, or deterministic")
	}
	for _, rule := range cfg.Suppression.Rules {
		if rule.Name == "" || rule.Action == "" || rule.Reason == "" {
			return errors.New("suppression rules require name, action, and reason")
		}
		if !oneOf(rule.Action, "report-only", "lower-priority", "suppress", "quarantine-required") {
			return errors.New("suppression rule action must be report-only, lower-priority, suppress, or quarantine-required")
		}
		if rule.Evidence != "" && !oneOf(rule.Evidence, "heuristic", "sampled", "confirmed") {
			return errors.New("suppression rule evidence must be heuristic, sampled, or confirmed")
		}
		if rule.Action == "suppress" && (rule.Evidence != "confirmed" || rule.Reviewers < 1) {
			return errors.New("suppress rules require confirmed evidence and at least one reviewer")
		}
	}
	return nil
}

func ApplyPolicy(cfg Config) Config {
	switch cfg.Policy {
	case "ci-fast":
		cfg.Mutators.Profile = "conservative-fast"
		cfg.Selection.Mode = "coverage"
		cfg.Selection.Prefilter = true
		cfg.Execution.Isolation = "overlay"
		cfg.Tests.Timeout = 20 * time.Second
		cfg.Reports.Formats = []string{"summary", "json", "junit"}
	case "ci-balanced":
		cfg.Mutators.Profile = "conservative"
		cfg.Selection.Mode = "coverage"
		cfg.Selection.Prefilter = true
		cfg.Execution.Isolation = "overlay"
		cfg.Tests.Timeout = 45 * time.Second
		cfg.Reports.Formats = []string{"summary", "json", "junit"}
	case "comparison-safe":
		cfg.Mutators.Profile = "gremlins-compatible"
		cfg.Selection.Mode = "package"
		cfg.Selection.Prefilter = false
		cfg.Execution.Isolation = "overlay"
		cfg.Execution.Budget = 10 * time.Minute
		cfg.Execution.Workers = minInt(cfg.Execution.Workers, 2)
		cfg.Tests.Timeout = 20 * time.Second
		cfg.Limits.Sample = "deterministic"
		if cfg.Limits.MaxMutants == 0 {
			cfg.Limits.MaxMutants = 250
		}
		cfg.Reports.Formats = []string{"summary", "json", "junit"}
	case "nightly":
		cfg.Mutators.Profile = "default"
		cfg.Selection.Mode = "coverage"
		cfg.Selection.Prefilter = true
		cfg.Execution.Isolation = "overlay"
		cfg.Tests.Timeout = 90 * time.Second
		cfg.Reports.Formats = []string{"summary", "json", "junit", "html"}
	case "campaign":
		cfg.Mutators.Profile = "aggressive"
		cfg.Selection.Mode = "package"
		cfg.Selection.Prefilter = false
		cfg.Execution.Isolation = "temp-workdir"
		cfg.Tests.Timeout = 2 * time.Minute
		cfg.Reports.Formats = []string{"summary", "json", "junit", "html"}
	}
	return cfg
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}
