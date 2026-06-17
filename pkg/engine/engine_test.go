package engine

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/cervantesh/cervo-mutants/pkg/config"
	"github.com/cervantesh/cervo-mutants/pkg/mutator"
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
	cfg.Baseline.Path = filepath.Join(dir, ".cervomut", "baseline.json")
	cfg.Quarantine.Path = filepath.Join(dir, ".cervomut", "quarantine.json")
	cfg.History.Path = filepath.Join(dir, ".cervomut", "history.json")
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
	if !strings.Contains(string(data), `"environment"`) {
		t.Fatalf("report missing environment metadata: %s", data)
	}
	if _, err := os.Stat(filepath.Join(cfg.Reports.Output, "partial-mutation-report.json")); err != nil {
		t.Fatalf("partial report was not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cfg.Reports.Output, "partial-summary.json")); err != nil {
		t.Fatalf("partial summary was not written: %v", err)
	}
	progress, err := os.ReadFile(filepath.Join(cfg.Reports.Output, "progress.jsonl"))
	if err != nil {
		t.Fatalf("progress stream was not written: %v", err)
	}
	if !strings.Contains(string(progress), `"schema_version":"1"`) || !strings.Contains(string(progress), `"completed"`) {
		t.Fatalf("progress stream missing expected fields: %s", progress)
	}
	if !strings.Contains(string(progress), `"eta"`) || !strings.Contains(string(progress), `"active_mutant"`) {
		t.Fatalf("progress stream missing eta/active mutant fields: %s", progress)
	}
}

func TestRunUsesParallelWorkers(t *testing.T) {
	dir := writeFixture(t)
	cfg := config.Defaults()
	cfg.Tests.Command = []string{"go", "test", "./..."}
	cfg.Tests.Timeout = 10_000_000_000
	cfg.Execution.Workers = 2
	cfg.Limits.MaxMutants = 2
	isolateArtifacts(&cfg, dir)

	result, err := New(cfg).Run(context.Background(), RunRequest{Targets: []string{dir}})
	if err != nil {
		t.Fatalf("parallel Run returned error: %v", err)
	}
	if len(result.Mutants) != 2 {
		t.Fatalf("mutants = %d, want 2", len(result.Mutants))
	}
	for _, mutant := range result.Mutants {
		if mutant.Status == "" || mutant.MutantID == "" {
			t.Fatalf("parallel result missing status/id: %+v", mutant)
		}
	}
}

func TestWritePartialResultsUsesAtomicReplacement(t *testing.T) {
	cfg := config.Defaults()
	cfg.Reports.Output = t.TempDir()
	engine := New(cfg)

	engine.writePartialResults([]MutantResult{{
		MutantID: "old",
		Status:   StatusSurvived,
		Mutant:   Mutant{ID: "old", Operator: "boolean-literal"},
	}})
	engine.writePartialResults([]MutantResult{{
		MutantID: "new",
		Status:   StatusKilled,
		Mutant:   Mutant{ID: "new", Operator: "nil-check"},
	}})

	path := filepath.Join(cfg.Reports.Output, "partial-mutation-report.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("partial report was not written: %v", err)
	}
	var run RunResult
	if err := json.Unmarshal(data, &run); err != nil {
		t.Fatalf("partial report is not valid JSON: %v\n%s", err, data)
	}
	if len(run.Mutants) != 1 || run.Mutants[0].MutantID != "new" {
		t.Fatalf("partial report was not atomically replaced: %+v", run.Mutants)
	}
	leftovers, err := filepath.Glob(filepath.Join(cfg.Reports.Output, ".partial-mutation-report.json.*.tmp"))
	if err != nil {
		t.Fatalf("glob temp files: %v", err)
	}
	if len(leftovers) != 0 {
		t.Fatalf("atomic write left temp files: %v", leftovers)
	}
	summaryData, err := os.ReadFile(filepath.Join(cfg.Reports.Output, "partial-summary.json"))
	if err != nil {
		t.Fatalf("partial summary was not written: %v", err)
	}
	if !strings.Contains(string(summaryData), `"timeout_risk_statistics"`) {
		t.Fatalf("partial summary missing timeout risk stats: %s", summaryData)
	}
}

func TestRunCanResumeFromPartialCheckpoint(t *testing.T) {
	dir := writeFixture(t)
	cfg := config.Defaults()
	cfg.Tests.Command = []string{"go", "test", "./..."}
	cfg.Tests.Timeout = 10_000_000_000
	cfg.Execution.Workers = 1
	cfg.Limits.MaxMutants = 1
	isolateArtifacts(&cfg, dir)

	first, err := New(cfg).Run(context.Background(), RunRequest{Targets: []string{dir}})
	if err != nil {
		t.Fatalf("first Run returned error: %v", err)
	}
	if len(first.Mutants) != 1 {
		t.Fatalf("first mutants = %d, want 1", len(first.Mutants))
	}
	cfg.Execution.Resume = true
	second, err := New(cfg).Run(context.Background(), RunRequest{Targets: []string{dir}})
	if err != nil {
		t.Fatalf("resume Run returned error: %v", err)
	}
	if len(second.Mutants) != 1 {
		t.Fatalf("second mutants = %d, want 1", len(second.Mutants))
	}
	if second.Mutants[0].Status != StatusCached {
		t.Fatalf("resumed status = %q, want cached", second.Mutants[0].Status)
	}
	if second.Mutants[0].PreviousStatus == "" {
		t.Fatal("resumed result did not preserve previous status")
	}
	if !strings.Contains(second.Mutants[0].StatusReason, "partial checkpoint") {
		t.Fatalf("resume reason = %q", second.Mutants[0].StatusReason)
	}
	if second.Summary.Cached != 1 || second.Summary.ExecutedMutants == 0 {
		t.Fatalf("cached result was not counted in summary: %+v", second.Summary)
	}
}

func TestResumeRejectsIncompatiblePartialCheckpoint(t *testing.T) {
	dir := writeFixture(t)
	cfg := config.Defaults()
	cfg.Tests.Command = []string{"go", "test", "./..."}
	cfg.Tests.Timeout = 10_000_000_000
	cfg.Execution.Workers = 1
	cfg.Limits.MaxMutants = 1
	isolateArtifacts(&cfg, dir)

	if _, err := New(cfg).Run(context.Background(), RunRequest{Targets: []string{dir}}); err != nil {
		t.Fatalf("first Run returned error: %v", err)
	}
	cfg.Execution.Resume = true
	cfg.Tests.Command = []string{"go", "test", "."}
	_, err := New(cfg).Run(context.Background(), RunRequest{Targets: []string{dir}})
	if err == nil {
		t.Fatal("resume succeeded with incompatible checkpoint")
	}
	if !strings.Contains(err.Error(), "fingerprint mismatch") {
		t.Fatalf("error = %v, want fingerprint mismatch", err)
	}
}

func TestResumeRejectsSourceChangedAfterCheckpoint(t *testing.T) {
	dir := writeFixture(t)
	cfg := config.Defaults()
	cfg.Tests.Command = []string{"go", "test", "./..."}
	cfg.Tests.Timeout = 10_000_000_000
	cfg.Execution.Workers = 1
	cfg.Limits.MaxMutants = 1
	isolateArtifacts(&cfg, dir)

	if _, err := New(cfg).Run(context.Background(), RunRequest{Targets: []string{dir}}); err != nil {
		t.Fatalf("first Run returned error: %v", err)
	}
	path := filepath.Join(dir, "calc_test.go")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, []byte("\n// checkpoint invalidation\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg.Execution.Resume = true
	_, err = New(cfg).Run(context.Background(), RunRequest{Targets: []string{dir}})
	if err == nil {
		t.Fatal("resume succeeded after source/test file changed")
	}
	if !strings.Contains(err.Error(), "fingerprint mismatch") {
		t.Fatalf("error = %v, want fingerprint mismatch", err)
	}
}

func TestResumeRejectsConfiguredFixtureChangedAfterCheckpoint(t *testing.T) {
	dir := writeFixture(t)
	fixtureDir := filepath.Join(dir, "testdata")
	if err := os.MkdirAll(fixtureDir, 0o700); err != nil {
		t.Fatal(err)
	}
	fixturePath := filepath.Join(fixtureDir, "case.json")
	if err := os.WriteFile(fixturePath, []byte(`{"case":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Defaults()
	cfg.Tests.Command = []string{"go", "test", "./..."}
	cfg.Tests.Timeout = 10_000_000_000
	cfg.Execution.Workers = 1
	cfg.Limits.MaxMutants = 1
	isolateArtifacts(&cfg, dir)

	if _, err := New(cfg).Run(context.Background(), RunRequest{Targets: []string{dir}}); err != nil {
		t.Fatalf("first Run returned error: %v", err)
	}
	if err := os.WriteFile(fixturePath, []byte(`{"case":2}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg.Execution.Resume = true
	_, err := New(cfg).Run(context.Background(), RunRequest{Targets: []string{dir}})
	if err == nil {
		t.Fatal("resume succeeded after configured fixture changed")
	}
	if !strings.Contains(err.Error(), "fingerprint mismatch") {
		t.Fatalf("error = %v, want fingerprint mismatch", err)
	}
}

func TestRunHandlesOneDriveStyleModulePathWithSpaces(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "OneDrive - Personal", "Documents", "Workspace", "cobra doc")
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
	if result.EffectiveMutants != 2 || result.ScoreDenominator != 2 {
		t.Fatalf("effective denominator not populated: %+v", result)
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

func TestSummarizeSeparatesTestEfficacyFromDenominatorHealth(t *testing.T) {
	result := summarize([]MutantResult{
		{Status: StatusKilled, Mutant: Mutant{Operator: "conditionals-negation"}},
		{Status: StatusTimedOut, Mutant: Mutant{Operator: "arithmetic-basic"}},
		{Status: StatusTimedOut, Mutant: Mutant{Operator: "arithmetic-basic"}},
		{Status: StatusNotCovered, Mutant: Mutant{Operator: "logical"}},
		{Status: StatusNotCovered, Mutant: Mutant{Operator: "logical"}},
	})

	if result.Score < 33.3333 || result.Score > 33.3334 {
		t.Fatalf("score = %.4f, want %.4f", result.Score, 100.0/3.0)
	}
	if result.TestEfficacy != 100 {
		t.Fatalf("test efficacy = %.2f, want 100 over killed+survived", result.TestEfficacy)
	}
	if result.EffectiveMutants != 1 || result.ScoreDenominator != 3 {
		t.Fatalf("unexpected denominators: %+v", result)
	}
	warnings := strings.Join(result.DenominatorHealth.Warnings, ",")
	for _, want := range []string{"timed_out_exceeds_effective", "not_covered_exceeds_effective", "high_score_poor_denominator_health"} {
		if !strings.Contains(warnings, want) {
			t.Fatalf("denominator warnings missing %q: %+v", want, result.DenominatorHealth)
		}
	}
	if result.DenominatorHealth.Healthy {
		t.Fatalf("denominator health should not be healthy: %+v", result.DenominatorHealth)
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
	covered := e.coverageMentions(Mutant{Module: dir, File: filepath.Join(dir, "calc.go"), Line: 1})
	uncovered := e.coverageMentions(Mutant{Module: dir, File: filepath.Join(dir, "calc.go"), Line: 3})
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

	plan := New(cfg).selectTests(Mutant{Module: dir, Package: "./target", File: filepath.Join(dir, "calc.go"), Line: 4})
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

	plan := New(cfg).selectTests(Mutant{Module: dir, Package: "./target", File: filepath.Join(dir, "calc.go"), Line: 4})
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

func TestSuppressionRuleCanIgnoreMutantBeforeExecution(t *testing.T) {
	cfg := config.Defaults()
	cfg.Suppression.Rules = []config.SuppressionRule{{
		Name:      "known-equivalent-conditional",
		Operator:  "conditionals-boundary",
		Action:    "suppress",
		Reason:    "Audited as equivalent in generated comparison wrappers.",
		Evidence:  "confirmed",
		Reviewers: 1,
	}}
	mutant := Mutant{
		ID:               "m-suppressed",
		Module:           t.TempDir(),
		Package:          ".",
		File:             "calc.go",
		Line:             3,
		Operator:         "conditionals-boundary",
		SuppressionAudit: New(cfg).suppressionAudit(mutator.Mutant{Operator: "conditionals-boundary", EquivalentRisk: "medium"}),
	}

	results, err := New(cfg).runMutantsSerial(context.Background(), []Mutant{mutant}, map[string]bool{})
	if err != nil {
		t.Fatalf("runMutantsSerial returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	if results[0].Status != StatusIgnored {
		t.Fatalf("status = %q, want %q", results[0].Status, StatusIgnored)
	}
	if !strings.Contains(results[0].StatusReason, "known-equivalent-conditional") {
		t.Fatalf("status reason does not name suppression rule: %q", results[0].StatusReason)
	}
}

func TestSerialRunnerHandlesQuarantineAndBudgetBranches(t *testing.T) {
	cfg := config.Defaults()
	cfg.Execution.Budget = time.Nanosecond
	e := New(cfg)
	start := time.Now()
	for time.Since(start) < time.Nanosecond {
	}
	mutants := []Mutant{
		{ID: "q", Operator: "conditionals-negation"},
		{ID: "budget", Operator: "conditionals-negation"},
	}
	results, err := e.runMutantsSerial(context.Background(), mutants, map[string]bool{"q": true})
	if err != nil {
		t.Fatalf("runMutantsSerial returned error: %v", err)
	}
	if results[0].Status != StatusQuarantined || results[1].Status != StatusPendingBudget || results[1].FailureKind != "budget_exhausted" {
		t.Fatalf("unexpected serial statuses: %+v", results)
	}
}

func TestRunTestClassifiesPassFailureAndTimeout(t *testing.T) {
	dir := writeFixture(t)
	cfg := config.Defaults()
	cfg.Tests.Timeout = 10 * time.Second
	e := New(cfg)
	pass, err := e.runTest(context.Background(), MutantJob{Mutant: Mutant{ID: "pass"}, WorkDir: dir, TestCommand: []string{"go", "test", "."}})
	if err != nil {
		t.Fatalf("pass runTest returned error: %v", err)
	}
	if pass.Status != StatusSurvived {
		t.Fatalf("pass status = %q", pass.Status)
	}

	fail, err := e.runTest(context.Background(), MutantJob{Mutant: Mutant{ID: "fail"}, WorkDir: dir, TestCommand: []string{"go", "test", "./missing"}})
	if err != nil {
		t.Fatalf("fail runTest returned error: %v", err)
	}
	if fail.Status != StatusKilled || !strings.Contains(fail.Output, "missing") {
		t.Fatalf("fail result = %+v", fail)
	}

	cfg.Tests.Timeout = time.Nanosecond
	timeout, err := New(cfg).runTest(context.Background(), MutantJob{Mutant: Mutant{ID: "timeout"}, WorkDir: dir, TestCommand: []string{"go", "test", "."}})
	if err != nil {
		t.Fatalf("timeout runTest returned error: %v", err)
	}
	if timeout.Status != StatusTimedOut {
		t.Fatalf("timeout status = %q", timeout.Status)
	}

	if runtime.GOOS != "windows" {
		cfg := config.Defaults()
		cfg.Execution.Resources.MaxProcessMemoryMB = 64
		resourceSkipped, err := New(cfg).runTest(context.Background(), MutantJob{Mutant: Mutant{ID: "resource"}, WorkDir: dir, TestCommand: []string{"go", "test", "."}})
		if err != nil {
			t.Fatalf("resource-limited runTest returned error: %v", err)
		}
		if resourceSkipped.Status != StatusSurvived || resourceSkipped.FailureKind != "" {
			t.Fatalf("resource-limited result = %+v", resourceSkipped)
		}
	}
}

func TestEnvironmentWarnsWhenProcessLimitsAreBestEffort(t *testing.T) {
	cfg := config.Defaults()
	cfg.Execution.Resources.MaxProcessMemoryMB = 64
	env := New(cfg).environment(1)
	if runtime.GOOS == "windows" {
		for _, warning := range env.Warnings {
			if strings.Contains(warning, "process resource limits are not enforced on this platform") {
				t.Fatalf("unexpected non-Windows process-limit warning on Windows: %+v", env.Warnings)
			}
		}
		return
	}
	found := false
	for _, warning := range env.Warnings {
		if strings.Contains(warning, "process resource limits are not enforced on this platform") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected best-effort process-limit warning, got %+v", env.Warnings)
	}
}

func TestPrepareMutationTempWorkdirAndOverlayBranches(t *testing.T) {
	dir := writeFixture(t)
	source := filepath.Join(dir, "calc.go")
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Index(string(data), ">=")
	mutant := Mutant{
		ID:          "m-prepare",
		Module:      dir,
		Package:     ".",
		File:        source,
		Original:    ">=",
		Mutated:     ">",
		StartOffset: start,
		EndOffset:   start + len(">="),
	}
	cfg := config.Defaults()
	cfg.Execution.Isolation = "temp-workdir"
	workdir, command, cleanup, err := New(cfg).prepareMutation(mutant, []string{"go", "test", "."})
	if err != nil {
		t.Fatalf("prepareMutation temp-workdir returned error: %v", err)
	}
	defer cleanup()
	if workdir == dir || strings.Join(command, " ") != "go test ." {
		t.Fatalf("unexpected temp workdir/command: %s %#v", workdir, command)
	}
	mutated, err := os.ReadFile(filepath.Join(workdir, "calc.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mutated), "n > 0") {
		t.Fatalf("mutant was not applied in temp workdir: %s", mutated)
	}

	cfg.Execution.Isolation = "overlay"
	workdir, command, cleanup, err = New(cfg).prepareMutation(mutant, []string{"go", "test", "."})
	if err != nil {
		t.Fatalf("prepareMutation overlay returned error: %v", err)
	}
	defer cleanup()
	if workdir != dir || !containsArg(command, "-overlay") {
		t.Fatalf("unexpected overlay workdir/command: %s %#v", workdir, command)
	}

	bad := mutant
	bad.File = filepath.Join(t.TempDir(), "outside.go")
	if _, _, cleanup, err := New(cfg).prepareMutation(bad, []string{"go", "test", "."}); err == nil {
		cleanup()
		t.Fatal("prepareMutation accepted outside file")
	}
}

func TestRunHandlesBaselineOptionalAndDiscoveryErrors(t *testing.T) {
	dir := writeFixture(t)
	cfg := config.Defaults()
	cfg.Tests.Command = []string{"go", "test", "./missing"}
	cfg.Tests.BaselineRequired = false
	cfg.Limits.MaxMutants = 1
	isolateArtifacts(&cfg, dir)
	if _, err := New(cfg).Run(context.Background(), RunRequest{Targets: []string{dir}}); err != nil {
		t.Fatalf("run with optional broken baseline returned error: %v", err)
	}

	badDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(badDir, "go.mod"), []byte("module bad\n\ngo 1.25.6\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(badDir, "bad.go"), []byte("package bad\nfunc broken("), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(config.Defaults()).Run(context.Background(), RunRequest{Targets: []string{badDir}, DryRun: true}); err == nil {
		t.Fatal("Run accepted invalid Go source")
	}
}

func TestRunErrorBranchesForQuarantineBaselineAndReports(t *testing.T) {
	dir := writeFixture(t)
	cfg := config.Defaults()
	cfg.Tests.Command = []string{"go", "test", "."}
	cfg.Limits.MaxMutants = 1
	isolateArtifacts(&cfg, dir)
	if err := os.MkdirAll(filepath.Dir(cfg.Quarantine.Path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg.Quarantine.Path, []byte("{bad json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(cfg).Run(context.Background(), RunRequest{Targets: []string{dir}}); err == nil {
		t.Fatal("Run accepted malformed quarantine file")
	}

	cfg = config.Defaults()
	cfg.Tests.Command = []string{"go", "test", "./missing"}
	cfg.Tests.BaselineRequired = true
	isolateArtifacts(&cfg, dir)
	if _, err := New(cfg).Run(context.Background(), RunRequest{Targets: []string{dir}}); err == nil {
		t.Fatal("Run accepted required failing baseline")
	}

	cfg = config.Defaults()
	cfg.Tests.Command = []string{"go", "test", "."}
	cfg.Limits.MaxMutants = 1
	isolateArtifacts(&cfg, dir)
	cfg.Reports.Output = filepath.Join(dir, "calc.go")
	if _, err := New(cfg).Run(context.Background(), RunRequest{Targets: []string{dir}}); err == nil {
		t.Fatal("Run accepted report output path that is a file")
	}
}

func TestRunRecoversDiscoveryPanic(t *testing.T) {
	cfg := config.Defaults()
	isolateArtifacts(&cfg, t.TempDir())
	restoreRunHooks(t)
	discoverMutantsForRun = func(_ *Engine, _ []string) ([]Mutant, error) {
		panic("discover panic")
	}

	_, err := New(cfg).Run(context.Background(), RunRequest{Targets: []string{"./..."}})
	assertPanicError(t, err, "discover panic")
}

func TestRunRecoversBaselinePanic(t *testing.T) {
	cfg := config.Defaults()
	isolateArtifacts(&cfg, t.TempDir())
	restoreRunHooks(t)
	discoverMutantsForRun = func(_ *Engine, _ []string) ([]Mutant, error) {
		return []Mutant{{ID: "m1"}}, nil
	}
	runBaselineForRun = func(_ *Engine, _ context.Context, _ []string) (MutantResult, error) {
		panic("baseline panic")
	}

	_, err := New(cfg).Run(context.Background(), RunRequest{Targets: []string{"./..."}})
	assertPanicError(t, err, "baseline panic")
}

func TestRunRecoversMutantExecutionPanic(t *testing.T) {
	cfg := config.Defaults()
	isolateArtifacts(&cfg, t.TempDir())
	restoreRunHooks(t)
	discoverMutantsForRun = func(_ *Engine, _ []string) ([]Mutant, error) {
		return []Mutant{{ID: "m1"}}, nil
	}
	runBaselineForRun = func(_ *Engine, _ context.Context, _ []string) (MutantResult, error) {
		return MutantResult{}, nil
	}
	runMutantsForRun = func(_ *Engine, _ context.Context, _ []Mutant, _ map[string]bool) ([]MutantResult, error) {
		panic("mutant panic")
	}

	_, err := New(cfg).Run(context.Background(), RunRequest{Targets: []string{"./..."}})
	assertPanicError(t, err, "mutant panic")
}

func TestRunRecoversReportPanic(t *testing.T) {
	cfg := config.Defaults()
	isolateArtifacts(&cfg, t.TempDir())
	restoreRunHooks(t)
	discoverMutantsForRun = func(_ *Engine, _ []string) ([]Mutant, error) {
		return []Mutant{{ID: "m1"}}, nil
	}
	runBaselineForRun = func(_ *Engine, _ context.Context, _ []string) (MutantResult, error) {
		return MutantResult{}, nil
	}
	runMutantsForRun = func(_ *Engine, _ context.Context, _ []Mutant, _ map[string]bool) ([]MutantResult, error) {
		return []MutantResult{{MutantID: "m1", Status: StatusKilled, Mutant: Mutant{ID: "m1"}}}, nil
	}
	writeReportsForRun = func(_ *Engine, _ RunResult) error {
		panic("report panic")
	}

	_, err := New(cfg).Run(context.Background(), RunRequest{Targets: []string{"./..."}})
	assertPanicError(t, err, "report panic")
}

func TestPartialCheckpointErrorBranches(t *testing.T) {
	dir := writeFixture(t)
	cfg := config.Defaults()
	isolateArtifacts(&cfg, dir)
	e := New(cfg)
	if completed, err := e.loadPartialResults(nil); err != nil || len(completed) != 0 {
		t.Fatalf("missing partial results = %+v err=%v", completed, err)
	}
	if err := os.MkdirAll(cfg.Reports.Output, 0o700); err != nil {
		t.Fatal(err)
	}
	partial := filepath.Join(cfg.Reports.Output, "partial-mutation-report.json")
	if err := os.WriteFile(partial, []byte("{bad json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := e.loadPartialResults(nil); err == nil {
		t.Fatal("loadPartialResults accepted malformed JSON")
	}
	if err := os.WriteFile(partial, []byte(`{"schema_version":"1","checkpoint":{},"mutants":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := e.loadPartialResults(nil); err == nil {
		t.Fatal("loadPartialResults accepted missing fingerprint")
	}
}

func TestResumeWithoutCheckpointUsesConfiguredWorkerPath(t *testing.T) {
	mutants := []Mutant{{ID: "q1"}, {ID: "q2"}}
	quarantined := map[string]bool{"q1": true, "q2": true}

	cfg := config.Defaults()
	cfg.Execution.Resume = true
	cfg.Execution.Workers = 1
	cfg.Reports.Output = t.TempDir()
	results, err := New(cfg).runMutantsWithResume(context.Background(), mutants, quarantined)
	if err != nil {
		t.Fatalf("serial resume without checkpoint returned error: %v", err)
	}
	if len(results) != 2 || results[0].Status != StatusQuarantined {
		t.Fatalf("serial resume results: %+v", results)
	}

	cfg.Execution.Workers = 2
	cfg.Reports.Output = t.TempDir()
	results, err = New(cfg).runMutantsWithResume(context.Background(), mutants, quarantined)
	if err != nil {
		t.Fatalf("parallel resume without checkpoint returned error: %v", err)
	}
	if len(results) != 2 || results[1].Status != StatusQuarantined {
		t.Fatalf("parallel resume results: %+v", results)
	}
}

func TestRunMutantCacheAndErrorBranches(t *testing.T) {
	dir := writeFixture(t)
	cfg := config.Defaults()
	cfg.Tests.Command = []string{"go", "test", "."}
	cfg.Cache.Path = filepath.Join(dir, ".cervomut", "cache")
	cfg.Selection.Mode = "package"
	cfg.Reports.Output = filepath.Join(dir, ".cervomut", "reports")
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
	e := New(cfg)
	key, err := e.cacheKey(mutant, e.selectTests(mutant))
	if err != nil {
		t.Fatal(err)
	}
	if err := e.putCached(key, MutantResult{MutantID: mutant.ID, Status: StatusKilled, Mutant: mutant}); err != nil {
		t.Fatal(err)
	}
	cached, err := e.runMutant(context.Background(), mutant)
	if err != nil {
		t.Fatalf("runMutant cached returned error: %v", err)
	}
	if cached.Status != StatusCached || cached.PreviousStatus != StatusKilled {
		t.Fatalf("cached result not reused: %+v", cached)
	}

	missing := mutant
	missing.File = filepath.Join(dir, "missing.go")
	if _, err := e.runMutant(context.Background(), missing); err == nil {
		t.Fatal("runMutant accepted missing source file")
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
	basePlan := New(baseCfg).selectTests(mutant)
	baseKey, err := New(baseCfg).cacheKey(mutant, basePlan)
	if err != nil {
		t.Fatal(err)
	}

	slicedCfg := baseCfg
	slicedCfg.Scope.SliceBy = "package"
	slicedCfg.Scope.ShardIndex = 1
	slicedCfg.Scope.ShardCount = 4
	slicedKey, err := New(slicedCfg).cacheKey(mutant, basePlan)
	if err != nil {
		t.Fatal(err)
	}
	if baseKey == slicedKey {
		t.Fatalf("cache key should change when slice config changes: %q", baseKey)
	}
}

func TestLoadCorruptCacheAndBaselineBranches(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Defaults()
	cfg.Cache.Path = filepath.Join(dir, "cache")
	cfg.Baseline.Path = filepath.Join(dir, "baseline.json")
	e := New(cfg)
	if err := os.MkdirAll(cfg.Cache.Path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.Cache.Path, "bad.json"), []byte("{bad json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := e.getCached("bad"); err == nil {
		t.Fatal("getCached accepted malformed JSON")
	}
	if err := os.WriteFile(cfg.Baseline.Path, []byte("{bad json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := e.loadBaseline(); err == nil {
		t.Fatal("loadBaseline accepted malformed JSON")
	}
}

func TestWriteReportsAndTimingNoopBranches(t *testing.T) {
	cfg := config.Defaults()
	cfg.Reports.Output = ""
	if err := New(cfg).writeReports(RunResult{}); err != nil {
		t.Fatalf("writeReports with empty output returned error: %v", err)
	}
	cfg.Selection.UseTimings = false
	New(cfg).recordTiming("m", time.Millisecond)
	cfg.Selection.UseTimings = true
	cfg.Selection.TimingsPath = ""
	New(cfg).recordTiming("m", time.Millisecond)
}

func TestHistoryTracksNewAndLongStandingSurvivors(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Defaults()
	cfg.History.Path = filepath.Join(dir, "history.json")
	e := New(cfg)
	first := []MutantResult{{
		MutantID: "m-history",
		Status:   StatusSurvived,
		Mutant:   Mutant{Operator: "conditionals-negation"},
	}}

	stats := e.applyHistory(first)
	if stats.NewSurvivors != 1 {
		t.Fatalf("new survivors = %d, want 1", stats.NewSurvivors)
	}
	if first[0].HistoryStatus != "new_survivor" || first[0].SurvivorAgeRuns != 1 {
		t.Fatalf("first run history not populated: %+v", first[0])
	}

	second := []MutantResult{{
		MutantID: "m-history",
		Status:   StatusSurvived,
		Mutant:   Mutant{Operator: "conditionals-negation"},
	}}
	stats = e.applyHistory(second)
	if stats.LongStandingSurvivors != 1 {
		t.Fatalf("long-standing survivors = %d, want 1", stats.LongStandingSurvivors)
	}
	if second[0].PreviousStatus != StatusSurvived || second[0].HistoryStatus != "long_standing_survivor" || second[0].SurvivorAgeRuns != 2 {
		t.Fatalf("second run history not populated: %+v", second[0])
	}
}

func TestHistoryDisabledAndMixedStatuses(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Defaults()
	cfg.History.Enabled = false
	stats := New(cfg).applyHistory([]MutantResult{{MutantID: "m-disabled", Status: StatusSurvived}})
	if stats.Enabled {
		t.Fatalf("history should be disabled: %+v", stats)
	}

	cfg = config.Defaults()
	cfg.History.Path = filepath.Join(dir, "history.json")
	e := New(cfg)
	results := []MutantResult{
		{MutantID: "k", Status: StatusKilled, Mutant: Mutant{Operator: "op"}},
		{MutantID: "n", Status: StatusNotCovered, Mutant: Mutant{Operator: "op"}},
		{MutantID: "c", Status: StatusCompileError, Mutant: Mutant{Operator: "op"}},
		{MutantID: "t", Status: StatusTimedOut, Mutant: Mutant{Operator: "op"}},
	}
	stats = e.applyHistory(results)
	if stats.UpdatedMutants != len(results) || stats.OperatorUsefulSurvivor["op"] != 0 {
		t.Fatalf("mixed history stats not populated: %+v", stats)
	}
	for _, result := range results {
		if result.HistoryStatus != "seen" || result.FirstSeen == "" || result.LastSeen == "" {
			t.Fatalf("history fields not populated: %+v", result)
		}
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

func TestBudgetSchedulingUsesTimeoutRiskWithinSameRecommendation(t *testing.T) {
	cfg := config.Defaults()
	cfg.Execution.Budget = time.Minute
	e := New(cfg)
	mutants := []Mutant{
		{ID: "slow", Recommendation: "default", Operator: "loop-control"},
		{ID: "fast", Recommendation: "default", Operator: "conditionals-negation"},
		{ID: "medium", Recommendation: "default", Operator: "arithmetic-basic"},
	}

	e.scheduleMutants(mutants)

	got := []string{mutants[0].ID, mutants[1].ID, mutants[2].ID}
	want := []string{"fast", "medium", "slow"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("scheduled IDs = %v, want %v", got, want)
	}
}

func TestSummarizeIncludesTimeoutRiskStats(t *testing.T) {
	result := summarize([]MutantResult{
		{MutantID: "fast", Status: StatusKilled, Mutant: Mutant{ID: "fast", Operator: "conditionals-negation"}},
		{MutantID: "slow", Status: StatusTimedOut, Mutant: Mutant{ID: "slow", Operator: "loop-control"}},
	})
	if result.TimeoutRiskStats["low"] != 1 || result.TimeoutRiskStats["very_high"] != 1 {
		t.Fatalf("unexpected timeout risk stats: %+v", result.TimeoutRiskStats)
	}
}

func TestSummarizeCoversCachedAndSuppressionStatusBranches(t *testing.T) {
	result := summarize([]MutantResult{
		{Status: StatusCached, PreviousStatus: StatusKilled, Mutant: Mutant{Operator: "cached-killed"}},
		{Status: StatusCached, PreviousStatus: StatusSurvived, Mutant: Mutant{Operator: "cached-survived", EquivalentRisk: "high"}},
		{Status: StatusCached, PreviousStatus: StatusNotCovered, Mutant: Mutant{Operator: "cached-not-covered"}},
		{Status: StatusCached, PreviousStatus: StatusCompileError, Mutant: Mutant{Operator: "cached-compile"}},
		{Status: StatusCached, PreviousStatus: StatusTimedOut, Mutant: Mutant{Operator: "cached-timeout"}},
		{Status: StatusCached, PreviousStatus: StatusMemoryKilled, Mutant: Mutant{Operator: "cached-memory"}},
		{Status: StatusCompileError, Mutant: Mutant{Operator: "compile"}},
		{Status: StatusSkippedResource, Mutant: Mutant{Operator: "resource-skip"}},
		{Status: StatusPendingBudget, Mutant: Mutant{Operator: "budget-pending"}},
		{Status: StatusIgnored, Mutant: Mutant{Operator: "ignored"}},
		{Status: StatusQuarantined, Mutant: Mutant{Operator: "quarantined", SuppressionAudit: []SuppressionAudit{
			{Action: config.SuppressionReportOnly},
			{Action: config.SuppressionLowerPriority},
			{Action: "suppress"},
			{Action: "quarantine-required"},
		}}},
	})
	if result.Cached != 6 || result.Killed != 1 || result.Survived != 1 || result.NotCovered != 1 || result.CompileError != 2 || result.TimedOut != 1 || result.MemoryKilled != 1 || result.SkippedResource != 1 || result.PendingBudget != 1 {
		t.Fatalf("cached/status branches not summarized: %+v", result)
	}
	if result.SuppressionReportOnly != 1 || result.SuppressionLowerPriority != 1 || result.SuppressionSuppressed != 1 || result.SuppressionQuarantineRequired != 1 {
		t.Fatalf("suppression audit branches not summarized: %+v", result)
	}
}

func TestRunStopMetadata(t *testing.T) {
	reason, last := runStopMetadata([]MutantResult{
		{MutantID: "done", Status: StatusKilled},
		{MutantID: "later", Status: StatusPendingBudget},
	})
	if reason != "budget_exhausted" || last != "done" {
		t.Fatalf("budget stop metadata = %q %q", reason, last)
	}
	reason, last = runStopMetadata([]MutantResult{
		{MutantID: "a", Status: StatusSkippedResource},
		{MutantID: "b", Status: StatusSkippedResource},
	})
	if reason != "resource_limits_unavailable" || last != "" {
		t.Fatalf("resource stop metadata = %q %q", reason, last)
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

func TestAffectedAndExplainPublicAPIs(t *testing.T) {
	dir := writeFixture(t)
	cfg := config.Defaults()
	isolateArtifacts(&cfg, dir)
	e := New(cfg)

	affected, err := e.Affected(context.Background(), AffectedRequest{Targets: []string{dir}})
	if err != nil {
		t.Fatalf("Affected returned error: %v", err)
	}
	if len(affected.Modules) != 1 || len(affected.Files) == 0 || affected.EstimatedMutants == 0 {
		t.Fatalf("unexpected affected result: %+v", affected)
	}

	explained, err := e.Explain(context.Background(), ExplainRequest{MutantID: "m1", Format: "text"})
	if err != nil {
		t.Fatalf("Explain returned error: %v", err)
	}
	if explained.MutantID != "m1" || explained.Explanation == "" || explained.Suggestion == "" {
		t.Fatalf("unexpected explanation: %+v", explained)
	}
	if _, err := e.Explain(context.Background(), ExplainRequest{}); err == nil {
		t.Fatal("Explain accepted empty mutant id")
	}
}

func TestEngineHelperBranches(t *testing.T) {
	assertGlobBranches(t)
	assertClassificationBranches(t)
	assertEnvironmentBranches(t)
	assertEngineTargetBranches(t)
}

func assertGlobBranches(t *testing.T) {
	t.Helper()
	if !globMatch("testdata/**", "testdata/case.json") {
		t.Fatal("globMatch should match recursive suffix pattern")
	}
	if !globMatch("testdata/**", "testdata") {
		t.Fatal("globMatch should match recursive suffix root")
	}
	if !globMatch("pkg/**/*.go", "pkg/deep/calc.go") {
		t.Fatal("globMatch should match middle recursive pattern")
	}
	if globMatch("pkg/**/case.go", "cmd/case.go") {
		t.Fatal("globMatch middle pattern matched wrong prefix")
	}
	if globMatch("pkg/**/case.go", "pkg/deep/case.go/extra") {
		t.Fatal("globMatch middle pattern matched wrong tail")
	}
	if !globMatch("**/*.go", "calc.go") {
		t.Fatal("globMatch should match recursive prefix pattern")
	}
	if globMatch("pkg/*.go", "cmd/main.go") {
		t.Fatal("globMatch matched unrelated path")
	}
	if !suppressionFileMatches("pkg/*.go", "pkg/calc.go") || !suppressionFileMatches("calc.go", "pkg/calc.go") {
		t.Fatal("suppressionFileMatches should support glob and suffix matches")
	}
}

func assertClassificationBranches(t *testing.T) {
	t.Helper()
	if classifyFailure("panic: boom", nil) != "test_panic" {
		t.Fatal("panic output should classify as test_panic")
	}
	if classifyFailure("undefined: Symbol", nil) != "compile_error" {
		t.Fatal("undefined output should classify as compile_error")
	}
	if classifyFailure("cannot find go", nil) != "environment_error" {
		t.Fatal("missing binary output should classify as environment_error")
	}
	if classifyFailure("", errors.New("runner failed")) != "runner_error" {
		t.Fatal("plain runner error should classify as runner_error")
	}
	noopCleanup()
	noopProcessLimitCleanup()
	if !fallbackCoverageMentions("calc.go:1.1,2.1 1 1", "calc.go", "calc.go") {
		t.Fatal("fallbackCoverageMentions should detect raw coverage line")
	}
	if compacted := compactedResults([]MutantResult{{}, {MutantID: "m1"}}); len(compacted) != 1 || compacted[0].MutantID != "m1" {
		t.Fatalf("compactedResults = %+v", compacted)
	}
}

func assertEnvironmentBranches(t *testing.T) {
	t.Helper()
	cfg := config.Defaults()
	cfg.Execution.Resources.MaxProcessMemoryMB = 64
	cfg.Execution.Budget = time.Minute
	env := New(cfg).environment(2)
	if env.Extra["max_process_memory_mb"] != "64" || env.Budget != "1m0s" {
		t.Fatalf("environment did not expose limits: %+v", env)
	}
	if New(config.Defaults()).workerCount(1) != 1 {
		t.Fatal("workerCount should cap workers to mutant count")
	}
}

func assertEngineTargetBranches(t *testing.T) {
	t.Helper()
	if targets := New(config.Defaults()).runTargets(nil); len(targets) == 0 {
		t.Fatal("runTargets should fall back to configured scope")
	}
	if _, err := New(config.Defaults()).discoverMutants([]string{filepath.Join(t.TempDir(), "missing")}); err == nil {
		t.Fatal("discoverMutants accepted missing target")
	}
	wd, err := moduleForTargets(nil)
	if err != nil || wd == "" {
		t.Fatalf("moduleForTargets(nil) = %q err=%v", wd, err)
	}
	if got := effectiveWorkerCount("windows", config.IsolationTempWorkdir, 4, 10); got != 2 {
		t.Fatalf("effectiveWorkerCount windows temp-workdir = %d, want 2", got)
	}
	if got := effectiveWorkerCount("windows", "overlay", 4, 10); got != 4 {
		t.Fatalf("effectiveWorkerCount windows overlay = %d, want 4", got)
	}
	if got := effectiveWorkerCount("linux", config.IsolationTempWorkdir, 4, 1); got != 1 {
		t.Fatalf("effectiveWorkerCount linux mutant cap = %d, want 1", got)
	}
	plan := effectiveTestCommandEnv("windows", config.IsolationTempWorkdir, 2, []string{"go", "test", "./..."}, []string{"PATH=C:\\Windows\\System32"})
	if !plan.Applied || plan.GOMAXPROCS != "2" || !strings.Contains(plan.GoFlags, "-p=1") {
		t.Fatalf("effectiveTestCommandEnv did not apply conservative settings: %+v", plan)
	}
	plan = effectiveTestCommandEnv("windows", "overlay", 1, []string{"go", "test", "./..."}, []string{"PATH=C:\\Windows\\System32"})
	if plan.Applied {
		t.Fatalf("effectiveTestCommandEnv should not apply for already-conservative overlay run: %+v", plan)
	}
	plan = effectiveTestCommandEnv("windows", config.IsolationTempWorkdir, 2, []string{"echo", "ok"}, []string{"PATH=C:\\Windows\\System32"})
	if plan.Applied {
		t.Fatalf("effectiveTestCommandEnv should ignore non-go-test commands: %+v", plan)
	}
}

func TestSelectionPatchAndRunTestErrorBranches(t *testing.T) {
	cfg := config.Defaults()
	cfg.Tests.Command = nil
	e := New(cfg)
	plan := e.selectTests(Mutant{ID: "m1"})
	if len(plan.Command) != 3 || plan.Command[0] != "go" || plan.Reason != "all tests selected" {
		t.Fatalf("default selectTests plan = %+v", plan)
	}
	if _, err := e.runTest(context.Background(), MutantJob{}); err == nil {
		t.Fatal("runTest accepted empty command")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "calc.go")
	if err := os.WriteFile(path, []byte("package p\nconst n = 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	mutant := Mutant{File: path, StartOffset: -1, EndOffset: 2, Original: "1", Mutated: "2"}
	if err := applyDiffReplacement(path, mutant); err == nil {
		t.Fatal("applyDiffReplacement accepted invalid offsets")
	}
	mutant = Mutant{File: path, StartOffset: 0, EndOffset: len("package p"), Original: "missing", Mutated: "2"}
	if err := applyDiffReplacement(path, mutant); err == nil {
		t.Fatal("applyDiffReplacement accepted missing original token")
	}
	if err := applyDiffReplacement(filepath.Join(dir, "missing.go"), mutant); err == nil {
		t.Fatal("applyDiffReplacement accepted missing file")
	}
	if got := withOverlayFlag([]string{"echo", "ok"}, "overlay.json"); strings.Join(got, " ") != "echo ok" {
		t.Fatalf("withOverlayFlag changed non-go command: %v", got)
	}
}

func TestCheckpointHelperBranches(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Defaults()
	cfg.Execution.CheckpointIncludes = []string{"", "fixtures/*.txt", "data/**"}
	e := New(cfg)
	textPath := filepath.Join(dir, "fixtures", "case.txt")
	if err := os.MkdirAll(filepath.Dir(textPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(textPath, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if fingerprint, ok := e.checkpointFileFingerprint(dir, textPath, "case.txt"); !ok || !strings.Contains(fingerprint, "fixtures/case.txt:") {
		t.Fatalf("checkpointFileFingerprint = %q ok=%t", fingerprint, ok)
	}
	if _, ok := e.checkpointFileFingerprint(dir, filepath.Join(dir, "missing.txt"), "missing.txt"); ok {
		t.Fatal("missing file should not fingerprint")
	}
	if e.checkpointIncludesFile(dir, filepath.Join(dir, "ignored.bin"), "ignored.bin") {
		t.Fatal("unexpected checkpoint include for ignored file")
	}
	if action := checkpointDirAction(fakeDirEntry{name: "vendor", dir: true}); action != filepath.SkipDir {
		t.Fatalf("vendor checkpoint action = %v, want SkipDir", action)
	}
	if !skipCheckpointWalkEntry(nil, errors.New("walk failed")) || !skipCheckpointWalkEntry(fakeDirEntry{name: "dir", dir: true}, nil) {
		t.Fatal("skipCheckpointWalkEntry should skip errors and directories")
	}
	results := []MutantResult{{MutantID: "empty"}, {MutantID: "m1", Mutant: Mutant{ID: "m1", File: textPath}}}
	if checkpoint := e.checkpointFromResults(results, "partial"); checkpoint.Reason != "partial" || checkpoint.Mutants != 1 {
		t.Fatalf("checkpointFromResults = %+v", checkpoint)
	}
	e.setCheckpointScope([]Mutant{{ID: "scoped", File: textPath}})
	if checkpoint := e.checkpointFromResults(nil, "scoped"); checkpoint.Mutants != 1 {
		t.Fatalf("checkpointFromResults scoped = %+v", checkpoint)
	}
}

type fakeDirEntry struct {
	name string
	dir  bool
}

func (f fakeDirEntry) Name() string               { return f.name }
func (f fakeDirEntry) IsDir() bool                { return f.dir }
func (f fakeDirEntry) Type() os.FileMode          { return 0 }
func (f fakeDirEntry) Info() (os.FileInfo, error) { return nil, errors.New("no info") }

func TestParallelWorkerAndCollectorErrorBranches(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	jobs := make(chan indexedMutant, 1)
	done := make(chan indexedResult, 1)
	startParallelWorkers(ctx, 1, jobs, done, func(context.Context, Mutant) (MutantResult, error) {
		return MutantResult{MutantID: "unexpected"}, nil
	})
	jobs <- indexedMutant{index: 0, mutant: Mutant{ID: "m1"}}
	close(jobs)
	item := <-done
	if !errors.Is(item.err, context.Canceled) {
		t.Fatalf("worker err = %v, want canceled", item.err)
	}

	cfg := config.Defaults()
	cfg.Reports.Output = t.TempDir()
	e := New(cfg)
	failed := make(chan indexedResult, 1)
	failed <- indexedResult{index: 0, err: errors.New("boom")}
	close(failed)
	_, err := e.collectParallelResults(failed, []MutantResult{{MutantID: "m1"}}, 1, time.Now(), func() {})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("collectParallelResults err = %v, want boom", err)
	}
}

func TestCompareBaselineDetectsRegressionAndNewSurvivors(t *testing.T) {
	previous := RunResult{
		Summary: Summary{Score: 80},
		Mutants: []MutantResult{
			{MutantID: "old", Status: StatusSurvived},
			{MutantID: "killed-before", Status: StatusKilled},
		},
	}
	current := RunResult{
		Summary: Summary{Score: 70},
		Mutants: []MutantResult{
			{MutantID: "old", Status: StatusSurvived},
			{MutantID: "killed-before", Status: StatusSurvived},
			{MutantID: "brand-new", Status: StatusSurvived},
		},
	}

	comparison := compareBaseline(previous, current)
	if !comparison.Enabled || !comparison.Regression {
		t.Fatalf("expected enabled regression comparison: %+v", comparison)
	}
	if strings.Join(comparison.NewSurvivors, ",") != "killed-before,brand-new" {
		t.Fatalf("new survivors = %+v", comparison.NewSurvivors)
	}
}

func TestParallelRunnerHandlesPreExecutionOutcomes(t *testing.T) {
	cfg := config.Defaults()
	cfg.Suppression.Rules = []config.SuppressionRule{{
		Name:      "confirmed-equivalent",
		Operator:  "logical",
		Action:    "suppress",
		Reason:    "confirmed equivalent",
		Evidence:  "confirmed",
		Reviewers: 1,
	}}
	e := New(cfg)
	mutants := []Mutant{
		{ID: "quarantined"},
		{ID: "suppressed", Operator: "logical", SuppressionAudit: []SuppressionAudit{{Name: "confirmed-equivalent", Action: "suppress", Reason: "confirmed equivalent"}}},
		{ID: "also-quarantined"},
	}

	results, err := e.runMutantsParallel(context.Background(), mutants, map[string]bool{"quarantined": true, "also-quarantined": true}, 2)
	if err != nil {
		t.Fatalf("runMutantsParallel returned error: %v", err)
	}
	statuses := []Status{results[0].Status, results[1].Status, results[2].Status}
	want := []Status{StatusQuarantined, StatusIgnored, StatusQuarantined}
	if strings.Join(statusStrings(statuses), ",") != strings.Join(statusStrings(want), ",") {
		t.Fatalf("statuses = %+v, want %+v", statuses, want)
	}
}

func TestLoadStoresAndPriorityHelpers(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Defaults()
	cfg.Cache.Path = filepath.Join(dir, "cache")
	cfg.Baseline.Path = filepath.Join(dir, "baseline.json")
	cfg.Quarantine.Path = filepath.Join(dir, "quarantine.json")
	cfg.Quarantine.FailOnExpired = false
	e := New(cfg)

	assertCacheStore(t, e)
	assertBaselineStore(t, e, cfg.Baseline.Path)
	assertQuarantineLoad(t, e, cfg.Quarantine.Path)
	assertPriorityHelpers(t)
}

func assertCacheStore(t *testing.T, e *Engine) {
	t.Helper()
	if _, ok, err := e.getCached("missing"); err != nil || ok {
		t.Fatalf("missing cache = ok %t err %v", ok, err)
	}
	if err := e.putCached("hit", MutantResult{MutantID: "m1", Status: StatusKilled}); err != nil {
		t.Fatalf("putCached returned error: %v", err)
	}
	if cached, ok, err := e.getCached("hit"); err != nil || !ok || cached.MutantID != "m1" {
		t.Fatalf("cached = %+v ok=%t err=%v", cached, ok, err)
	}
}

func assertBaselineStore(t *testing.T, e *Engine, path string) {
	t.Helper()
	if _, ok, err := e.loadBaseline(); err != nil || ok {
		t.Fatalf("missing baseline = ok %t err %v", ok, err)
	}
	baseline := RunResult{SchemaVersion: "1", Summary: Summary{Score: 90}}
	data, _ := json.Marshal(baseline)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if loaded, ok, err := e.loadBaseline(); err != nil || !ok || loaded.Summary.Score != 90 {
		t.Fatalf("loaded baseline = %+v ok=%t err=%v", loaded, ok, err)
	}
}

func assertQuarantineLoad(t *testing.T, e *Engine, path string) {
	t.Helper()
	entries := []map[string]any{{
		"mutant_id":  "m-active",
		"reason":     "temporary",
		"owner":      "qa",
		"issue":      "cervantesh/CervoMutants#31",
		"created_at": time.Now().Add(-time.Hour).Format(time.RFC3339),
		"expires_at": time.Now().Add(time.Hour).Format(time.RFC3339),
	}}
	quarantineData, _ := json.Marshal(entries)
	if err := os.WriteFile(path, quarantineData, 0o600); err != nil {
		t.Fatal(err)
	}
	active, expired, err := e.loadQuarantine()
	if err != nil {
		t.Fatalf("loadQuarantine returned error: %v", err)
	}
	if !active["m-active"] || expired != 0 {
		t.Fatalf("unexpected quarantine state active=%+v expired=%d", active, expired)
	}
}

func assertPriorityHelpers(t *testing.T) {
	t.Helper()
	for risk, want := range map[string]int{"low": 0, "medium": 1, "high": 2, "other": 3} {
		if got := riskPriority(risk); got != want {
			t.Fatalf("riskPriority(%q) = %d, want %d", risk, got, want)
		}
	}
	for action, want := range map[string]int{"report-only": 0, "lower-priority": 1, "quarantine-required": 2, "suppress": 3, "none": -1} {
		if got := suppressionPriority(action); got != want {
			t.Fatalf("suppressionPriority(%q) = %d, want %d", action, got, want)
		}
	}
	if hasProcessLimits(config.Resources{}) {
		t.Fatal("empty resource limits should not enable process limits")
	}
	if !hasProcessLimits(config.Resources{MaxProcessMemoryMB: 1}) || !hasProcessLimits(config.Resources{MaxProcesses: 1}) {
		t.Fatal("configured memory or process cap should enable process limits")
	}
	noopProcessLimitCleanup()
}

func statusStrings(statuses []Status) []string {
	values := make([]string, 0, len(statuses))
	for _, status := range statuses {
		values = append(values, string(status))
	}
	return values
}

func restoreRunHooks(t *testing.T) {
	t.Helper()
	oldDiscover := discoverMutantsForRun
	oldBaseline := runBaselineForRun
	oldRunMutants := runMutantsForRun
	oldWriteReports := writeReportsForRun
	t.Cleanup(func() {
		discoverMutantsForRun = oldDiscover
		runBaselineForRun = oldBaseline
		runMutantsForRun = oldRunMutants
		writeReportsForRun = oldWriteReports
	})
}

func assertPanicError(t *testing.T, err error, recovered string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected panic error")
	}
	var panicErr *PanicError
	if !errors.As(err, &panicErr) {
		t.Fatalf("error = %T %v, want PanicError", err, err)
	}
	if panicErr.Stage != "run" {
		t.Fatalf("panic stage = %q, want run", panicErr.Stage)
	}
	if !strings.Contains(panicErr.Recovered, recovered) {
		t.Fatalf("recovered = %q, want substring %q", panicErr.Recovered, recovered)
	}
	if panicErr.Stack == "" {
		t.Fatal("panic stack trace was empty")
	}
}
