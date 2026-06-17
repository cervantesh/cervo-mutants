package pool

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/cervantesh/cervo-mutants/internal/testharness"
)

func TestRunCompareUsesPartialCervoReportAndResume(t *testing.T) {
	fixture := testharness.NewDir(t)
	manifestPath := fixture.WriteJSON(t, "manifest.json", Manifest{SchemaVersion: "1", Repos: []Repo{{
		Name:   "pflag",
		URL:    "https://example.com/pflag.git",
		Target: "./...",
		Lane:   "validation",
		Domain: "cli",
	}}})
	workRoot := filepath.Join(fixture.Root, "work")
	repoDir := filepath.Join(workRoot, "pflag")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}

	runner := &fakeRunner{run: func(spec CommandSpec) (CommandResult, error) {
		out := flagValue(spec.Args, "--out")
		if out != "" {
			reportPath := filepath.Join(out, "partial-mutation-report.json")
			if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
				return CommandResult{}, err
			}
			if err := os.WriteFile(reportPath, []byte(`{"summary":{"total":20,"killed":12,"survived":8,"not_covered":0,"timed_out":0,"compile_error":0,"skipped":0,"score":60,"test_efficacy":60}}`), 0o644); err != nil {
				return CommandResult{}, err
			}
		}
		return CommandResult{ExitCode: 0}, nil
	}}

	run, err := RunCompare(context.Background(), CompareOptions{
		ManifestPath:      manifestPath,
		WorkRoot:          workRoot,
		OutputRoot:        filepath.Join(fixture.Root, "out"),
		Names:             []string{"pflag"},
		Tools:             []string{"cervomut"},
		CompareTargetMode: "package-root",
		Workers:           2,
		TimeoutSeconds:    60,
		CervoBinary:       "cervomut",
		GremlinsBinary:    "gremlins",
		GomuBinary:        "gomu",
		GoMutestingBinary: "go-mutesting",
		MemoryPollSeconds: 1,
		MemoryWaitSeconds: 1,
		Runner:            runner,
	})
	if err != nil {
		t.Fatalf("RunCompare returned error: %v", err)
	}
	if len(run.Results) != 1 {
		t.Fatalf("results = %d, want 1", len(run.Results))
	}
	got := run.Results[0]
	if got.EffectiveTarget != "." || got.TargetMode != "package-root" {
		t.Fatalf("target normalization mismatch: %+v", got)
	}
	if !got.PartialReportUsed || got.Total != 20 || got.Killed != 12 || got.Survived != 8 {
		t.Fatalf("partial report not used: %+v", got)
	}
	if len(runner.specs) != 1 {
		t.Fatalf("command count = %d, want 1", len(runner.specs))
	}

	secondRunner := &fakeRunner{run: func(spec CommandSpec) (CommandResult, error) {
		return CommandResult{ExitCode: 0}, nil
	}}
	secondRun, err := RunCompare(context.Background(), CompareOptions{
		ManifestPath:      manifestPath,
		WorkRoot:          workRoot,
		OutputRoot:        filepath.Join(fixture.Root, "out"),
		Names:             []string{"pflag"},
		Tools:             []string{"cervomut"},
		CompareTargetMode: "package-root",
		Workers:           2,
		TimeoutSeconds:    60,
		CervoBinary:       "cervomut",
		Resume:            true,
		Runner:            secondRunner,
	})
	if err != nil {
		t.Fatalf("resume RunCompare returned error: %v", err)
	}
	if len(secondRun.Results) != 1 {
		t.Fatalf("resume results = %d, want 1", len(secondRun.Results))
	}
	if len(secondRunner.specs) != 0 {
		t.Fatalf("resume should skip existing result, ran %d commands", len(secondRunner.specs))
	}
}

func TestRunCompareClassifiesGremlinsPanicFromLog(t *testing.T) {
	fixture := testharness.NewDir(t)
	manifestPath := fixture.WriteJSON(t, "manifest.json", Manifest{SchemaVersion: "1", Repos: []Repo{{
		Name:   "cobra",
		URL:    "https://example.com/cobra.git",
		Target: "./doc",
		Lane:   "tuning",
		Domain: "cli",
	}}})
	workRoot := filepath.Join(fixture.Root, "work")
	repoDir := filepath.Join(workRoot, "cobra")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}

	runner := &fakeRunner{run: func(spec CommandSpec) (CommandResult, error) {
		if err := os.MkdirAll(filepath.Dir(spec.LogPath), 0o755); err != nil {
			return CommandResult{}, err
		}
		if err := os.WriteFile(spec.LogPath, []byte("panic: simulated"), 0o644); err != nil {
			return CommandResult{}, err
		}
		return CommandResult{ExitCode: 0}, nil
	}}

	run, err := RunCompare(context.Background(), CompareOptions{
		ManifestPath:      manifestPath,
		WorkRoot:          workRoot,
		OutputRoot:        filepath.Join(fixture.Root, "out"),
		Names:             []string{"cobra"},
		Tools:             []string{"gremlins"},
		CompareTargetMode: "manifest",
		Workers:           2,
		TimeoutSeconds:    60,
		GremlinsBinary:    "gremlins",
		CervoBinary:       "cervomut",
		GomuBinary:        "gomu",
		GoMutestingBinary: "go-mutesting",
		Runner:            runner,
	})
	if err != nil {
		t.Fatalf("RunCompare returned error: %v", err)
	}
	if len(run.Results) != 1 {
		t.Fatalf("results = %d, want 1", len(run.Results))
	}
	if run.Results[0].Status != "panic" {
		t.Fatalf("status = %q, want panic", run.Results[0].Status)
	}
}

func TestRunCompareRecordsMissingRepo(t *testing.T) {
	fixture := testharness.NewDir(t)
	manifestPath := fixture.WriteJSON(t, "manifest.json", Manifest{SchemaVersion: "1", Repos: []Repo{{
		Name:   "missing",
		URL:    "https://example.com/missing.git",
		Target: "./...",
	}}})

	run, err := RunCompare(context.Background(), CompareOptions{
		ManifestPath:      manifestPath,
		WorkRoot:          fixture.Path("work"),
		OutputRoot:        fixture.Path("out"),
		Names:             []string{"missing"},
		Tools:             []string{"cervomut"},
		CompareTargetMode: "manifest",
		CervoBinary:       "cervomut",
		GremlinsBinary:    "gremlins",
		GomuBinary:        "gomu",
		GoMutestingBinary: "go-mutesting",
	})
	if err != nil {
		t.Fatalf("RunCompare returned error: %v", err)
	}
	if len(run.Results) != 1 {
		t.Fatalf("results = %d, want 1", len(run.Results))
	}
	if run.Results[0].Status != "missing_repo" || run.Results[0].Tool != "all" {
		t.Fatalf("missing repo classification mismatch: %+v", run.Results[0])
	}
	if _, err := os.Stat(run.SummaryPath); err != nil {
		t.Fatalf("summary missing: %v", err)
	}
}
