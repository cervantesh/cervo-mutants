package engine

import (
	"context"
	"encoding/json"
	"github.com/cervantesh/cervo-mutants/internal/testharness"
	"github.com/cervantesh/cervo-mutants/pkg/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCoverageSelectionUsesCoverageProfileAndRecordsTiming(t *testing.T) {
	dir := writeFixture(t)
	cfg := config.Defaults()
	cfg.Tests.Command = []string{"go", "test", "./..."}
	cfg.Tests.Timeout = 10_000_000_000
	cfg.Selection.Mode = "coverage"
	isolateArtifacts(&cfg, dir)
	cfg.Limits.MaxMutants = 1

	result, err := New(cfg).Run(context.Background(), RunRequest{Targets: []string{dir}})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(result.Mutants) != 1 {
		t.Fatalf("mutants = %d, want 1", len(result.Mutants))
	}
	if got := strings.Join(result.Mutants[0].TestCommand, " "); got != "go test ." {
		t.Fatalf("coverage selection command = %q, want package command", got)
	}
	if _, err := os.Stat(cfg.Selection.CoverageProfile); err != nil {
		t.Fatalf("coverage profile was not written: %v", err)
	}
	data, err := os.ReadFile(cfg.Selection.TimingsPath)
	if err != nil {
		t.Fatalf("timing history was not written: %v", err)
	}
	var timings map[string]int64
	if err := json.Unmarshal(data, &timings); err != nil {
		t.Fatalf("timing history is not JSON: %v", err)
	}
	if timings[result.Mutants[0].MutantID] <= 0 {
		t.Fatalf("timing not recorded for mutant: %#v", timings)
	}
}

func TestCacheKeyChangesWhenGoModOrTestsChange(t *testing.T) {
	dir := writeFixture(t)
	cfg := config.Defaults()
	cfg.Tests.Command = []string{"go", "test", "./..."}
	isolateArtifacts(&cfg, dir)

	e := New(cfg)
	discovered, err := e.discoverForTest([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	mutants, err := e.generateMutants(discovered)
	if err != nil {
		t.Fatal(err)
	}
	if len(mutants) == 0 {
		t.Fatal("no mutants generated")
	}
	session := e.newRunSession()
	first, err := session.cacheKey(mutants[0], TestPlan{Command: []string{"go", "test", "."}})
	if err != nil {
		t.Fatal(err)
	}
	testPath := filepath.Join(dir, "calc_test.go")
	f, err := os.OpenFile(testPath, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("\n// new assertion coming later\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := session.cacheKey(mutants[0], TestPlan{Command: []string{"go", "test", "."}})
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("cache key did not change after relevant test file changed")
	}
}

func TestCoverageSelectionCanClassifyUncoveredMutantWithoutRunningAllTests(t *testing.T) {
	dir := writeFixture(t)
	cfg := config.Defaults()
	cfg.Tests.Command = []string{"go", "test", "./..."}
	cfg.Tests.Timeout = 10_000_000_000
	cfg.Selection.Mode = "coverage"
	isolateArtifacts(&cfg, dir)
	if err := os.MkdirAll(filepath.Dir(cfg.Selection.CoverageProfile), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg.Selection.CoverageProfile, []byte("mode: set\nother.go:1.1,1.2 1 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	e := New(cfg)
	session := e.newRunSession()
	result, err := session.runMutant(context.Background(), Mutant{
		ID:          "m-uncovered",
		Module:      dir,
		Package:     ".",
		File:        filepath.Join(dir, "calc.go"),
		Line:        3,
		Operator:    "conditionals-boundary",
		Original:    ">=",
		Mutated:     ">",
		StartOffset: 0,
		EndOffset:   1,
	})
	if err != nil {
		t.Fatalf("runMutant returned error: %v", err)
	}
	if result.Status != StatusNotCovered {
		t.Fatalf("status = %q, want %q", result.Status, StatusNotCovered)
	}
	if !strings.Contains(result.StatusReason, "coverage profile") {
		t.Fatalf("status reason should mention coverage profile: %q", result.StatusReason)
	}
}

func TestCoverageSelectionUsesLineRangesNotOnlyFilePresence(t *testing.T) {
	dir := writeFixture(t)
	cfg := config.Defaults()
	cfg.Selection.Mode = "coverage"
	isolateArtifacts(&cfg, dir)
	if err := os.MkdirAll(filepath.Dir(cfg.Selection.CoverageProfile), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg.Selection.CoverageProfile, []byte("mode: set\ncalc.go:1.1,2.1 1 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	e := New(cfg)
	session := e.newRunSession()
	covered := session.coverageMentions(Mutant{Module: dir, File: filepath.Join(dir, "calc.go"), Line: 1})
	uncovered := session.coverageMentions(Mutant{Module: dir, File: filepath.Join(dir, "calc.go"), Line: 3})
	if !covered {
		t.Fatal("coverage range should cover line 1")
	}
	if uncovered {
		t.Fatal("coverage range should not cover line 3 just because the same file appears")
	}
}

func TestCoverageSelectionFallsBackToPackageWhenFileCoveredButLineMissing(t *testing.T) {
	dir := writeFixture(t)
	cfg := config.Defaults()
	cfg.Tests.Command = []string{"go", "test", "-run", "Test", "./pkg/a", "./pkg/b"}
	cfg.Selection.Mode = "coverage"
	isolateArtifacts(&cfg, dir)
	if err := os.MkdirAll(filepath.Dir(cfg.Selection.CoverageProfile), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg.Selection.CoverageProfile, []byte("mode: set\ncalc.go:1.1,2.1 1 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	e := New(cfg)
	session := e.newRunSession()
	plan := session.selectTests(Mutant{Module: dir, Package: "./target", File: filepath.Join(dir, "calc.go"), Line: 4})
	if !plan.CoversMutant {
		t.Fatalf("covered file fallback should run package tests: %+v", plan)
	}
	if plan.CoverageSource != "coverage-mode-file-fallback" || !strings.Contains(plan.Reason, "package fallback") {
		t.Fatalf("unexpected fallback metadata: %+v", plan)
	}
	if got := strings.Join(plan.Command, " "); got != "go test -run Test ./target" {
		t.Fatalf("package scoped command = %q", got)
	}
}

func TestPackagePrefilterUsesFileCoverageBeforeReportingNotCovered(t *testing.T) {
	dir := writeFixture(t)
	cfg := config.Defaults()
	cfg.Tests.Command = []string{"go", "test", "./pkg/a", "./pkg/b"}
	cfg.Selection.Mode = "package"
	cfg.Selection.Prefilter = true
	isolateArtifacts(&cfg, dir)
	if err := os.MkdirAll(filepath.Dir(cfg.Selection.CoverageProfile), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg.Selection.CoverageProfile, []byte("mode: set\ncalc.go:1.1,2.1 1 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	e := New(cfg)
	session := e.newRunSession()
	plan := session.selectTests(Mutant{Module: dir, Package: "./target", File: filepath.Join(dir, "calc.go"), Line: 4})
	if !plan.CoversMutant {
		t.Fatalf("package prefilter should not reject a covered file: %+v", plan)
	}
	if got := strings.Join(plan.Command, " "); got != "go test ./target" {
		t.Fatalf("package scoped command = %q", got)
	}
}

func TestPackageSelectionCanPrefilterUncoveredMutants(t *testing.T) {
	dir := writeFixture(t)
	cfg := config.Defaults()
	cfg.Tests.Command = []string{"go", "test", "./..."}
	cfg.Selection.Mode = "package"
	cfg.Selection.Prefilter = true
	isolateArtifacts(&cfg, dir)
	if err := os.MkdirAll(filepath.Dir(cfg.Selection.CoverageProfile), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg.Selection.CoverageProfile, []byte("mode: set\nother.go:1.1,1.2 1 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	e := New(cfg)
	session := e.newRunSession()
	result, err := session.runMutant(context.Background(), Mutant{
		ID:          "m-package-prefilter",
		Module:      dir,
		Package:     ".",
		File:        filepath.Join(dir, "calc.go"),
		Line:        3,
		Operator:    "conditionals-boundary",
		Original:    ">=",
		Mutated:     ">",
		StartOffset: 0,
		EndOffset:   1,
	})
	if err != nil {
		t.Fatalf("runMutant returned error: %v", err)
	}
	if result.Status != StatusNotCovered {
		t.Fatalf("status = %q, want %q", result.Status, StatusNotCovered)
	}
	if !strings.Contains(result.StatusReason, "coverage profile") {
		t.Fatalf("status reason should mention coverage profile: %q", result.StatusReason)
	}
}

func TestRunNestedPackageTargetUsesRelativeCoverageProfileFromTargetDir(t *testing.T) {
	dir := testharness.WriteGoModuleTempDir(t, "nestedfixture", map[string]string{
		"nested/calc.go": `package nested

func IsPositiveOrZero(n int) bool {
	return n >= 0
}
`,
		"nested/calc_test.go": `package nested

import "testing"

func TestIsPositiveOrZero(t *testing.T) {
	if !IsPositiveOrZero(1) {
		t.Fatal("want positive")
	}
}
`,
	})
	t.Chdir(dir)

	cfg := config.Defaults()
	cfg.Tests.Command = []string{"go", "test", "./..."}
	cfg.Tests.Timeout = 10 * time.Second
	cfg.Execution.Workers = 1
	cfg.Execution.TempRoot = filepath.Join(dir, ".cervomut", "tmp")
	cfg.Reports.Output = filepath.Join(dir, ".cervomut", "reports")
	cfg.Cache.Path = filepath.Join(dir, ".cervomut", "cache")
	cfg.Baseline.Path = filepath.Join(dir, ".cervomut", "baseline.json")
	cfg.Quarantine.Path = filepath.Join(dir, ".cervomut", "quarantine.json")
	cfg.History.Path = filepath.Join(dir, ".cervomut", "history.json")
	cfg.Limits.MaxMutants = 1
	cfg.Mutators.Enabled = []string{"conditionals-boundary"}
	cfg.Mutators.Disabled = nil
	cfg.Selection.Prefilter = true

	result, err := New(cfg).Run(context.Background(), RunRequest{Targets: []string{"./nested"}})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Summary.Total == 0 {
		t.Fatal("run discovered no mutants")
	}
	if result.Summary.NotCovered != 0 {
		t.Fatalf("nested target should not be misclassified as not_covered when the relative coverage profile is written under the target dir: %+v", result.Summary)
	}
	if result.Summary.Survived == 0 {
		t.Fatalf("expected covered nested-package mutant to execute and survive weak tests: %+v", result.Summary)
	}
	if _, err := os.Stat(filepath.Join(dir, "nested", ".cervomut", "coverage.out")); err != nil {
		t.Fatalf("nested target coverage profile was not written under the target directory: %v", err)
	}
}

func TestApplySlicingCapsFilesAndPackageMutants(t *testing.T) {
	cfg := config.Defaults()
	cfg.Limits.MaxFilesPerRun = 1
	cfg.Limits.MaxMutantsPerPackage = 1
	mutants := []Mutant{
		{ID: "b", Package: "./pkg/a", File: filepath.Join("pkg", "a", "first.go")},
		{ID: "a", Package: "./pkg/a", File: filepath.Join("pkg", "a", "second.go")},
		{ID: "c", Package: "./pkg/b", File: filepath.Join("pkg", "b", "third.go")},
	}

	selected, meta := New(cfg).applySlicing(mutants)
	if len(selected) != 1 || selected[0].ID != "a" {
		t.Fatalf("unexpected sliced mutants: %+v", selected)
	}
	if !meta.Enabled || meta.SelectedFiles != 1 || meta.SelectedMutants != 1 || meta.MaxMutantsPerPackage != 1 {
		t.Fatalf("unexpected slice metadata: %+v", meta)
	}
}

func TestApplySlicingUsesDeterministicShardGroups(t *testing.T) {
	cfg := config.Defaults()
	cfg.Scope.SliceBy = "package"
	cfg.Scope.ShardIndex = 1
	cfg.Scope.ShardCount = 2
	mutants := []Mutant{
		{ID: "a1", Package: "./pkg/a", File: "pkg/a/one.go"},
		{ID: "a2", Package: "./pkg/a", File: "pkg/a/two.go"},
		{ID: "b1", Package: "./pkg/b", File: "pkg/b/one.go"},
		{ID: "c1", Package: "./pkg/c", File: "pkg/c/one.go"},
	}

	selected, meta := New(cfg).applySlicing(mutants)
	expectedPackages := map[string]bool{}
	for _, pkg := range []string{"./pkg/a", "./pkg/b", "./pkg/c"} {
		if shardForKey(pkg, 2) == 1 {
			expectedPackages[pkg] = true
		}
	}
	if meta.GroupCount != 3 || meta.SelectedGroups != len(expectedPackages) || !meta.Enabled {
		t.Fatalf("unexpected shard metadata: %+v", meta)
	}
	for _, mutant := range selected {
		if !expectedPackages[mutant.Package] {
			t.Fatalf("mutant from unexpected package shard: %+v", mutant)
		}
	}
}

func TestCacheKeyChangesWhenSliceConfigChanges(t *testing.T) {
	dir := writeFixture(t)
	mutant := Mutant{
		ID:          "m-cache",
		Module:      dir,
		Package:     ".",
		File:        filepath.Join(dir, "calc.go"),
		Line:        3,
		Operator:    "conditionals-boundary",
		Original:    ">=",
		Mutated:     ">",
		StartOffset: 0,
		EndOffset:   1,
		Fingerprint: "fp",
	}
	baseCfg := config.Defaults()
	baseCfg.Tests.Command = []string{"go", "test", "."}
	baseEngine := New(baseCfg)
	baseSession := baseEngine.newRunSession()
	basePlan := baseSession.selectTests(mutant)
	baseKey, err := baseSession.cacheKey(mutant, basePlan)
	if err != nil {
		t.Fatal(err)
	}

	slicedCfg := baseCfg
	slicedCfg.Scope.SliceBy = "package"
	slicedCfg.Scope.ShardIndex = 1
	slicedCfg.Scope.ShardCount = 4
	slicedEngine := New(slicedCfg)
	slicedSession := slicedEngine.newRunSession()
	slicedKey, err := slicedSession.cacheKey(mutant, basePlan)
	if err != nil {
		t.Fatal(err)
	}
	if baseKey == slicedKey {
		t.Fatalf("cache key should change when slice config changes: %q", baseKey)
	}
}
