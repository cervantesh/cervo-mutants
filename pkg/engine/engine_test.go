package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cervantesh/cervo-mutants/pkg/config"
)

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
	if result.Gate.Evaluated {
		t.Fatalf("dry-run should not evaluate gate: %+v", result.Gate)
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
	cfg.Tests.Timeout = 10 * time.Second
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

func TestRunHandlesOneDriveStyleModulePathWithSpaces(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "OneDrive - Personal", "Documents", "Workspace", "cobra doc")
	writeFixtureFiles(t, dir)
	before, err := os.ReadFile(filepath.Join(dir, "calc.go"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Defaults()
	cfg.Tests.Command = []string{"go", "test", "./..."}
	cfg.Tests.Timeout = 10 * time.Second
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
	cfg.Tests.Timeout = 10 * time.Second
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
	e := New(cfg)
	e.deps.discoverMutants = func(_ *Engine, _ []string) ([]Mutant, error) {
		panic("discover panic")
	}

	_, err := e.Run(context.Background(), RunRequest{Targets: []string{"./..."}})
	assertPanicError(t, err, "discover panic")
}

func TestRunRecoversBaselinePanic(t *testing.T) {
	cfg := config.Defaults()
	isolateArtifacts(&cfg, t.TempDir())
	e := New(cfg)
	e.deps.discoverMutants = func(_ *Engine, _ []string) ([]Mutant, error) {
		return []Mutant{{ID: "m1"}}, nil
	}
	e.deps.runBaseline = func(_ *runSession, _ context.Context, _ []string) (MutantResult, error) {
		panic("baseline panic")
	}

	_, err := e.Run(context.Background(), RunRequest{Targets: []string{"./..."}})
	assertPanicError(t, err, "baseline panic")
}

func TestRunRecoversMutantExecutionPanic(t *testing.T) {
	cfg := config.Defaults()
	isolateArtifacts(&cfg, t.TempDir())
	e := New(cfg)
	e.deps.discoverMutants = func(_ *Engine, _ []string) ([]Mutant, error) {
		return []Mutant{{ID: "m1"}}, nil
	}
	e.deps.runBaseline = func(_ *runSession, _ context.Context, _ []string) (MutantResult, error) {
		return MutantResult{}, nil
	}
	e.deps.runMutants = func(_ *runSession, _ context.Context, _ []Mutant, _ map[string]bool) ([]MutantResult, error) {
		panic("mutant panic")
	}

	_, err := e.Run(context.Background(), RunRequest{Targets: []string{"./..."}})
	assertPanicError(t, err, "mutant panic")
}

func TestRunRecoversReportPanic(t *testing.T) {
	cfg := config.Defaults()
	isolateArtifacts(&cfg, t.TempDir())
	e := New(cfg)
	e.deps.discoverMutants = func(_ *Engine, _ []string) ([]Mutant, error) {
		return []Mutant{{ID: "m1"}}, nil
	}
	e.deps.runBaseline = func(_ *runSession, _ context.Context, _ []string) (MutantResult, error) {
		return MutantResult{}, nil
	}
	e.deps.runMutants = func(_ *runSession, _ context.Context, _ []Mutant, _ map[string]bool) ([]MutantResult, error) {
		return []MutantResult{{MutantID: "m1", Status: StatusKilled, Mutant: Mutant{ID: "m1"}}}, nil
	}
	e.deps.writeReports = func(_ *Engine, _ RunResult) error {
		panic("report panic")
	}

	_, err := e.Run(context.Background(), RunRequest{Targets: []string{"./..."}})
	assertPanicError(t, err, "report panic")
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
