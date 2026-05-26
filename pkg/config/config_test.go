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
