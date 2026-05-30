package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultsAreAdoptionFriendlyAndAllIn(t *testing.T) {
	cfg := Defaults()

	if cfg.CI.FailUnder != 0 {
		t.Fatalf("FailUnder = %d, want 0", cfg.CI.FailUnder)
	}
	if !cfg.Baseline.Enabled || !cfg.Baseline.FailOnRegression || !cfg.Baseline.FailOnNewSurvivors {
		t.Fatalf("baseline defaults are not strict adoption defaults: %+v", cfg.Baseline)
	}
	if cfg.Execution.Isolation != "temp-workdir" {
		t.Fatalf("isolation = %q, want temp-workdir", cfg.Execution.Isolation)
	}
	if len(cfg.Execution.CheckpointIncludes) == 0 {
		t.Fatal("checkpoint includes should default to common fixture directories")
	}
	if cfg.Selection.Mode != "package" {
		t.Fatalf("selection mode = %q, want package", cfg.Selection.Mode)
	}
	if cfg.Mutators.Profile != "conservative" {
		t.Fatalf("mutator profile = %q, want conservative", cfg.Mutators.Profile)
	}
	if !cfg.Quarantine.Enabled || !cfg.Quarantine.RequireReason || !cfg.Quarantine.RequireOwner || !cfg.Quarantine.RequireIssue {
		t.Fatalf("quarantine defaults are not strict: %+v", cfg.Quarantine)
	}
	wantReports := []string{"summary", "json", "junit", "html"}
	if len(cfg.Reports.Formats) != len(wantReports) {
		t.Fatalf("report formats = %#v, want %#v", cfg.Reports.Formats, wantReports)
	}
	for i := range wantReports {
		if cfg.Reports.Formats[i] != wantReports[i] {
			t.Fatalf("report formats = %#v, want %#v", cfg.Reports.Formats, wantReports)
		}
	}
}

func TestLoadParsesYAMLAndValidatesEnums(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cervomut.yaml")
	err := os.WriteFile(path, []byte(`version: 1
scope:
  mode: changed
  since: origin/main
tests:
  command: ["go", "test", "./pkg/..."]
  timeout: 45s
mutators:
  profile: aggressive
execution:
  workers: 2
  budget: 10m
  checkpoint_includes: ["testdata/**", "golden/**"]
selection:
  mode: coverage
ci:
  fail_under: 80
`), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Scope.Mode != "changed" || cfg.Scope.Since != "origin/main" {
		t.Fatalf("scope not loaded: %+v", cfg.Scope)
	}
	if cfg.Tests.Command[2] != "./pkg/..." || cfg.Tests.Timeout != 45*time.Second {
		t.Fatalf("tests not loaded: %+v", cfg.Tests)
	}
	if cfg.Mutators.Profile != "aggressive" || cfg.Execution.Workers != 2 || cfg.Selection.Mode != "coverage" || cfg.CI.FailUnder != 80 {
		t.Fatalf("overrides not loaded: %+v", cfg)
	}
	if len(cfg.Execution.CheckpointIncludes) != 2 || cfg.Execution.CheckpointIncludes[1] != "golden/**" {
		t.Fatalf("checkpoint includes not loaded: %+v", cfg.Execution.CheckpointIncludes)
	}

	err = os.WriteFile(path, []byte("version: 1\nselection:\n  mode: impossible\n"), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load accepted invalid selection mode")
	}
}

func TestValidateAllowsOverlayIsolation(t *testing.T) {
	cfg := Defaults()
	cfg.Execution.Isolation = "overlay"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate rejected overlay isolation: %v", err)
	}

	cfg.Execution.Isolation = "shared-worktree"
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate accepted invalid isolation")
	}
}

func TestValidateAllowsConservativeFastProfile(t *testing.T) {
	cfg := Defaults()
	cfg.Mutators.Profile = "conservative-fast"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate rejected conservative-fast profile: %v", err)
	}

	cfg.Mutators.Profile = "gremlins-compatible"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate rejected gremlins-compatible profile: %v", err)
	}
}

func TestPolicyPresetsTuneMutationRuns(t *testing.T) {
	cfg := Defaults()
	cfg.Policy = "ci-fast"
	cfg = ApplyPolicy(cfg)

	if cfg.Mutators.Profile != "conservative-fast" {
		t.Fatalf("ci-fast profile = %q, want conservative-fast", cfg.Mutators.Profile)
	}
	if cfg.Selection.Mode != "coverage" || !cfg.Selection.Prefilter {
		t.Fatalf("ci-fast selection not coverage-prefiltered: %+v", cfg.Selection)
	}
	if cfg.Execution.Isolation != "overlay" {
		t.Fatalf("ci-fast isolation = %q, want overlay", cfg.Execution.Isolation)
	}

	cfg = Defaults()
	cfg.Policy = "comparison-safe"
	cfg = ApplyPolicy(cfg)
	if cfg.Mutators.Profile != "gremlins-compatible" || cfg.Execution.Budget != 10*time.Minute || cfg.Limits.Sample != "deterministic" || cfg.Limits.MaxMutants != 250 {
		t.Fatalf("comparison-safe preset not bounded/comparable: %+v", cfg)
	}
	if cfg.Execution.Workers > 2 || cfg.Tests.Timeout != 20*time.Second {
		t.Fatalf("comparison-safe resource limits not applied: workers=%d timeout=%s", cfg.Execution.Workers, cfg.Tests.Timeout)
	}

	cfg = Defaults()
	cfg.Policy = "campaign"
	cfg = ApplyPolicy(cfg)
	if cfg.Mutators.Profile != "aggressive" || cfg.Selection.Mode != "package" || cfg.Selection.Prefilter {
		t.Fatalf("campaign preset not exhaustive package mode: %+v", cfg)
	}
}

func TestLoadPolicyPresetKeepsExplicitOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cervomut.yaml")
	if err := os.WriteFile(path, []byte(`version: 1
policy: ci-fast
mutators:
  profile: conservative
execution:
  isolation: temp-workdir
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Policy != "ci-fast" {
		t.Fatalf("policy = %q, want ci-fast", cfg.Policy)
	}
	if cfg.Selection.Mode != "coverage" || !cfg.Selection.Prefilter {
		t.Fatalf("preset selection was not applied: %+v", cfg.Selection)
	}
	if cfg.Mutators.Profile != "conservative" {
		t.Fatalf("explicit profile override lost: %+v", cfg.Mutators)
	}
	if cfg.Execution.Isolation != "temp-workdir" {
		t.Fatalf("explicit isolation override lost: %+v", cfg.Execution)
	}
}

func TestValidateRejectsUnauditableSuppressionRules(t *testing.T) {
	cfg := Defaults()
	cfg.Suppression.Rules = []SuppressionRule{{Name: "bad", Action: "hide", Reason: "not auditable"}}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate accepted unsupported suppression action")
	}

	cfg = Defaults()
	cfg.Suppression.Rules = []SuppressionRule{{Name: "missing-reason", Action: "report-only"}}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate accepted suppression rule without reason")
	}
	cfg.Suppression.Rules = []SuppressionRule{{Name: "unsafe-suppress", Action: "suppress", Reason: "too weak", Evidence: "sampled"}}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate accepted suppress rule without confirmed evidence and reviewer")
	}
	cfg.Suppression.Rules = []SuppressionRule{{Name: "confirmed-suppress", Action: "suppress", Reason: "reviewed equivalent", Evidence: "confirmed", Reviewers: 1}}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate rejected audited suppress rule: %v", err)
	}
}
