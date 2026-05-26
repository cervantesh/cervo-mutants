package engine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/config"
)

func writeFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFixtureFiles(t, dir)
	return dir
}

func writeFixtureFiles(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module fixture\n\ngo 1.25.6\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "calc.go"), []byte(`package fixture

func IsPositiveOrZero(n int) bool {
	return n >= 0
}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "calc_test.go"), []byte(`package fixture

import "testing"

func TestIsPositiveOrZero(t *testing.T) {
	if !IsPositiveOrZero(1) {
		t.Fatal("want positive")
	}
}
`), 0o600); err != nil {
		t.Fatal(err)
	}
}

func isolateArtifacts(cfg *config.Config, dir string) {
	cfg.Reports.Output = filepath.Join(dir, ".cervomut", "reports")
	cfg.Cache.Path = filepath.Join(dir, ".cervomut", "cache")
	cfg.Selection.CoverageProfile = filepath.Join(dir, ".cervomut", "coverage.out")
	cfg.Selection.TimingsPath = filepath.Join(dir, ".cervomut", "timings.json")
}

func TestRunDryRunDiscoversMutantsWithoutChangingWorkspace(t *testing.T) {
	dir := writeFixture(t)
	before, err := os.ReadFile(filepath.Join(dir, "calc.go"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Defaults()
	cfg.Tests.Command = []string{"go", "test", "./..."}
	isolateArtifacts(&cfg, dir)

	result, err := New(cfg).Run(context.Background(), RunRequest{Targets: []string{dir}, DryRun: true})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Summary.Total == 0 {
		t.Fatal("dry-run discovered no mutants")
	}
	if result.Mutants[0].Mutant.Description == "" {
		t.Fatalf("mutant missing description: %+v", result.Mutants[0].Mutant)
	}
	if len(result.Mutants[0].Mutant.NearbyTests) == 0 {
		t.Fatalf("mutant missing nearby tests: %+v", result.Mutants[0].Mutant)
	}
	after, err := os.ReadFile(filepath.Join(dir, "calc.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("dry-run changed source workspace")
	}
	for _, mutant := range result.Mutants {
		if filepath.IsAbs(mutant.MutantID) || strings.Contains(mutant.MutantID, `\`) {
			t.Fatalf("mutant ID should be module-relative and slash-normalized, got %q", mutant.MutantID)
		}
		if strings.Contains(mutant.MutantID, ":\\") {
			t.Fatalf("mutant ID contains raw Windows drive path: %q", mutant.MutantID)
		}
	}
}

func TestRunClassifiesSurvivorAndWritesReports(t *testing.T) {
	dir := writeFixture(t)
	cfg := config.Defaults()
	cfg.Tests.Command = []string{"go", "test", "./..."}
	cfg.Tests.Timeout = 10_000_000_000
	isolateArtifacts(&cfg, dir)
	cfg.Execution.Workers = 1

	result, err := New(cfg).Run(context.Background(), RunRequest{Targets: []string{dir}})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Summary.Total == 0 {
		t.Fatal("run discovered no mutants")
	}
	if result.Summary.Survived == 0 {
		t.Fatalf("expected weak fixture test to leave a survivor: %+v", result.Summary)
	}
	reportPath := filepath.Join(cfg.Reports.Output, "mutation-report.json")
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("report was not written: %v", err)
	}
	if !strings.Contains(string(data), `"schema_version": "1"`) {
		t.Fatalf("report missing schema version: %s", data)
	}
}

func TestRunHandlesOneDriveStyleModulePathWithSpaces(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "OneDrive - Personal", "Documents", "CervoSoft", "cobra doc")
	writeFixtureFiles(t, dir)
	before, err := os.ReadFile(filepath.Join(dir, "calc.go"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Defaults()
	cfg.Tests.Command = []string{"go", "test", "./..."}
	cfg.Tests.Timeout = 10_000_000_000
	cfg.Limits.MaxMutants = 1
	isolateArtifacts(&cfg, dir)

	result, err := New(cfg).Run(context.Background(), RunRequest{Targets: []string{dir}})
	if err != nil {
		t.Fatalf("Run returned error for OneDrive-style path: %v", err)
	}
	if len(result.Mutants) != 1 {
		t.Fatalf("mutants = %d, want 1", len(result.Mutants))
	}
	if filepath.IsAbs(result.Mutants[0].MutantID) || strings.Contains(result.Mutants[0].MutantID, `\`) {
		t.Fatalf("mutant ID should not contain raw absolute Windows-style path: %q", result.Mutants[0].MutantID)
	}
	if _, err := os.Stat(filepath.Join(cfg.Reports.Output, "mutation-report.json")); err != nil {
		t.Fatalf("report missing for OneDrive-style path: %v", err)
	}
	after, err := os.ReadFile(filepath.Join(dir, "calc.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("run changed source workspace under OneDrive-style path")
	}
}

func TestRunCanUseGoOverlayIsolationWithoutChangingWorkspace(t *testing.T) {
	dir := writeFixture(t)
	before, err := os.ReadFile(filepath.Join(dir, "calc.go"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Defaults()
	cfg.Execution.Isolation = "overlay"
	cfg.Tests.Command = []string{"go", "test", "./..."}
	cfg.Tests.Timeout = 10_000_000_000
	cfg.Limits.MaxMutants = 1
	isolateArtifacts(&cfg, dir)

	result, err := New(cfg).Run(context.Background(), RunRequest{Targets: []string{dir}})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(result.Mutants) != 1 {
		t.Fatalf("mutants = %d, want 1", len(result.Mutants))
	}
	if !containsArg(result.Mutants[0].TestCommand, "-overlay") {
		t.Fatalf("overlay test command missing -overlay: %#v", result.Mutants[0].TestCommand)
	}
	after, err := os.ReadFile(filepath.Join(dir, "calc.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("overlay isolation changed source workspace")
	}
}

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

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
	first, err := e.cacheKeyForTest(mutants[0], TestPlan{Command: []string{"go", "test", "."}})
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
	second, err := e.cacheKeyForTest(mutants[0], TestPlan{Command: []string{"go", "test", "."}})
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("cache key did not change after relevant test file changed")
	}
}

func TestSummarizeReportsCoverageEfficacyAndMutatorStats(t *testing.T) {
	result := summarize([]MutantResult{
		{Status: StatusKilled, Mutant: Mutant{Operator: "conditionals-negation"}},
		{Status: StatusSurvived, Mutant: Mutant{Operator: "conditionals-negation"}},
		{Status: StatusNotCovered, Mutant: Mutant{Operator: "logical"}},
		{Status: StatusSkipped, Mutant: Mutant{Operator: "boolean"}},
	})

	if result.NotCovered != 1 {
		t.Fatalf("not covered = %d, want 1", result.NotCovered)
	}
	if result.GeneratedMutants != 4 || result.CoveredMutants != 2 || result.ExecutedMutants != 2 {
		t.Fatalf("decomposed counts not populated: %+v", result)
	}
	if result.Score != 50 {
		t.Fatalf("score = %.2f, want 50", result.Score)
	}
	if result.TestEfficacy != 50 {
		t.Fatalf("test efficacy = %.2f, want 50", result.TestEfficacy)
	}
	if result.MutationCoverage != 66.66666666666666 {
		t.Fatalf("mutation coverage = %.14f, want 66.66666666666666", result.MutationCoverage)
	}
	if result.MutatorStats["conditionals-negation"].Killed != 1 || result.MutatorStats["conditionals-negation"].Survived != 1 {
		t.Fatalf("conditionals stats not populated: %+v", result.MutatorStats["conditionals-negation"])
	}
	if result.MutatorStats["logical"].NotCovered != 1 {
		t.Fatalf("logical not-covered stats not populated: %+v", result.MutatorStats["logical"])
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
	result, err := e.runMutant(context.Background(), Mutant{
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

	result, err := New(cfg).runMutant(context.Background(), Mutant{
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

func TestBudgetSchedulingPrioritizesFastOperators(t *testing.T) {
	cfg := config.Defaults()
	cfg.Execution.Budget = 1
	e := New(cfg)
	mutants := []Mutant{
		{ID: "z", Recommendation: "aggressive"},
		{ID: "b", Recommendation: "fast-ci"},
		{ID: "a", Recommendation: "default"},
	}

	e.scheduleMutants(mutants)

	got := []string{mutants[0].ID, mutants[1].ID, mutants[2].ID}
	want := []string{"b", "a", "z"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("scheduled IDs = %v, want %v", got, want)
	}
}

func TestRankSurvivorsPrioritizesLowerEquivalentRisk(t *testing.T) {
	results := []MutantResult{
		{MutantID: "high", Status: StatusSurvived, Mutant: Mutant{EquivalentRisk: "high", Recommendation: "fast-ci"}},
		{MutantID: "low", Status: StatusSurvived, Mutant: Mutant{EquivalentRisk: "low", Recommendation: "conservative", NearbyTests: []string{"x_test.go"}}},
		{MutantID: "killed", Status: StatusKilled, Mutant: Mutant{EquivalentRisk: "low"}},
	}

	rankSurvivors(results)

	if results[1].SurvivorRank != 1 || results[0].SurvivorRank != 2 {
		t.Fatalf("unexpected survivor ranks: %+v", results)
	}
	if results[2].SurvivorRank != 0 {
		t.Fatalf("killed mutant should not be ranked: %+v", results[2])
	}
}
