package engine

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/cervantesh/cervo-mutants/pkg/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWritePartialResultsUsesAtomicReplacement(t *testing.T) {
	cfg := config.Defaults()
	cfg.Reports.Output = t.TempDir()
	engine := New(cfg)
	session := engine.newRunSession()

	session.writePartialResults([]MutantResult{{
		MutantID: "old",
		Status:   StatusSurvived,
		Mutant:   Mutant{ID: "old", Operator: "boolean-literal"},
	}})
	session.writePartialResults([]MutantResult{{
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
	cfg.Ownership.Rules = []config.OwnershipRule{{Name: "calc-owner", File: "calc.go", Owner: "fresh-owner"}}
	isolateArtifacts(&cfg, dir)

	first, err := New(cfg).Run(context.Background(), RunRequest{Targets: []string{dir}})
	if err != nil {
		t.Fatalf("first Run returned error: %v", err)
	}
	if len(first.Mutants) != 1 {
		t.Fatalf("first mutants = %d, want 1", len(first.Mutants))
	}
	partialPath := filepath.Join(cfg.Reports.Output, "partial-mutation-report.json")
	partialData, err := os.ReadFile(partialPath)
	if err != nil {
		t.Fatalf("ReadFile partial checkpoint returned error: %v", err)
	}
	var partialRun RunResult
	if err := json.Unmarshal(partialData, &partialRun); err != nil {
		t.Fatalf("Unmarshal partial checkpoint returned error: %v", err)
	}
	partialRun.Mutants[0].Mutant.Ownership = &OwnershipRoute{Owner: "stale-owner", Rule: "stale"}
	updatedPartial, err := json.MarshalIndent(partialRun, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent partial checkpoint returned error: %v", err)
	}
	if err := os.WriteFile(partialPath, updatedPartial, 0o600); err != nil {
		t.Fatalf("WriteFile partial checkpoint returned error: %v", err)
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
	if second.Mutants[0].Mutant.Ownership == nil || second.Mutants[0].Mutant.Ownership.Owner != "fresh-owner" {
		t.Fatalf("resume did not refresh ownership metadata: %+v", second.Mutants[0].Mutant.Ownership)
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

func TestPartialCheckpointErrorBranches(t *testing.T) {
	dir := writeFixture(t)
	cfg := config.Defaults()
	isolateArtifacts(&cfg, dir)
	e := New(cfg)
	session := e.newRunSession()
	if completed, err := session.loadPartialResults(nil); err != nil || len(completed) != 0 {
		t.Fatalf("missing partial results = %+v err=%v", completed, err)
	}
	if err := os.MkdirAll(cfg.Reports.Output, 0o700); err != nil {
		t.Fatal(err)
	}
	partial := filepath.Join(cfg.Reports.Output, "partial-mutation-report.json")
	if err := os.WriteFile(partial, []byte("{bad json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := session.loadPartialResults(nil); err == nil {
		t.Fatal("loadPartialResults accepted malformed JSON")
	}
	if err := os.WriteFile(partial, []byte(`{"schema_version":"1","checkpoint":{},"mutants":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := session.loadPartialResults(nil); err == nil {
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
	serialEngine := New(cfg)
	serialSession := serialEngine.newRunSession()
	results, err := serialSession.runMutantsWithResume(context.Background(), mutants, quarantined)
	if err != nil {
		t.Fatalf("serial resume without checkpoint returned error: %v", err)
	}
	if len(results) != 2 || results[0].Status != StatusQuarantined {
		t.Fatalf("serial resume results: %+v", results)
	}

	cfg.Execution.Workers = 2
	cfg.Reports.Output = t.TempDir()
	parallelEngine := New(cfg)
	parallelSession := parallelEngine.newRunSession()
	results, err = parallelSession.runMutantsWithResume(context.Background(), mutants, quarantined)
	if err != nil {
		t.Fatalf("parallel resume without checkpoint returned error: %v", err)
	}
	if len(results) != 2 || results[1].Status != StatusQuarantined {
		t.Fatalf("parallel resume results: %+v", results)
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
	session := e.newRunSession()
	results := []MutantResult{{MutantID: "empty"}, {MutantID: "m1", Mutant: Mutant{ID: "m1", File: textPath}}}
	if checkpoint := session.checkpointFromResults(results, "partial"); checkpoint.Reason != "partial" || checkpoint.Mutants != 1 {
		t.Fatalf("checkpointFromResults = %+v", checkpoint)
	}
	session.setCheckpointScope([]Mutant{{ID: "scoped", File: textPath}})
	if checkpoint := session.checkpointFromResults(nil, "scoped"); checkpoint.Mutants != 1 {
		t.Fatalf("checkpointFromResults scoped = %+v", checkpoint)
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
