package pool

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cervantesh/cervo-mutants/internal/testharness"
)

func TestRunSmokeWritesSummaryAndParsesMutationReport(t *testing.T) {
	fixture := testharness.NewDir(t)
	manifestPath := fixture.WriteJSON(t, "manifest.json", Manifest{SchemaVersion: "1", Repos: []Repo{{
		Name:   "cobra",
		URL:    "https://example.com/cobra.git",
		Target: "./doc",
		Lane:   "tuning",
		Domain: "cli",
	}}})

	runner := &fakeRunner{run: func(spec CommandSpec) (CommandResult, error) {
		switch spec.Path {
		case "git":
			dest := spec.Args[len(spec.Args)-1]
			if err := os.MkdirAll(dest, 0o755); err != nil {
				return CommandResult{}, err
			}
		case "cervomut":
			out := flagValue(spec.Args, "--out")
			if strings.Contains(strings.Join(spec.Args, " "), "ci-balanced") {
				reportPath := filepath.Join(out, "mutation-report.json")
				if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
					return CommandResult{}, err
				}
				if err := os.WriteFile(reportPath, []byte(`{"summary":{"total":10,"killed":7,"survived":2,"not_covered":1,"timed_out":0,"compile_error":0,"skipped":0,"score":77.7,"test_efficacy":77.7}}`), 0o644); err != nil {
					return CommandResult{}, err
				}
			}
		}
		return CommandResult{ExitCode: 0}, nil
	}}

	run, err := RunSmoke(context.Background(), SmokeOptions{
		ManifestPath:           manifestPath,
		WorkRoot:               filepath.Join(fixture.Root, "work"),
		Names:                  []string{"cobra"},
		RunMutation:            true,
		MaxMutants:             10,
		Workers:                2,
		CloneTimeoutSeconds:    60,
		TestTimeoutSeconds:     60,
		DryRunTimeoutSeconds:   60,
		MutationTimeoutSeconds: 60,
		CervoBinary:            "cervomut",
		GitBinary:              "git",
		Runner:                 runner,
	})
	if err != nil {
		t.Fatalf("RunSmoke returned error: %v", err)
	}
	if len(run.Results) != 1 {
		t.Fatalf("results = %d, want 1", len(run.Results))
	}
	got := run.Results[0]
	if got.Clone != "ok" {
		t.Fatalf("clone = %q, want ok", got.Clone)
	}
	if got.Mutants == nil || *got.Mutants != 10 || got.Killed == nil || *got.Killed != 7 || got.Survived == nil || *got.Survived != 2 || got.NotCovered == nil || *got.NotCovered != 1 {
		t.Fatalf("mutation metrics not parsed: %+v", got)
	}
	if _, err := os.Stat(run.SummaryPath); err != nil {
		t.Fatalf("summary missing: %v", err)
	}
	if len(runner.specs) != 4 {
		t.Fatalf("command count = %d, want 4", len(runner.specs))
	}
}

func TestRunSmokeHandlesCloneFailureAndExistingRepo(t *testing.T) {
	t.Run("clone failure", func(t *testing.T) {
		fixture := testharness.NewDir(t)
		manifestPath := fixture.WriteJSON(t, "manifest.json", Manifest{SchemaVersion: "1", Repos: []Repo{{
			Name:   "cobra",
			URL:    "https://example.com/cobra.git",
			Target: "./doc",
		}}})
		runner := &fakeRunner{run: func(spec CommandSpec) (CommandResult, error) {
			if spec.Path == "git" {
				return CommandResult{ExitCode: 9}, nil
			}
			return CommandResult{ExitCode: 0}, nil
		}}

		run, err := RunSmoke(context.Background(), SmokeOptions{
			ManifestPath:         manifestPath,
			WorkRoot:             fixture.Path("work"),
			Names:                []string{"cobra"},
			CervoBinary:          "cervomut",
			GitBinary:            "git",
			CloneTimeoutSeconds:  1,
			TestTimeoutSeconds:   1,
			DryRunTimeoutSeconds: 1,
			Runner:               runner,
		})
		if err != nil {
			t.Fatalf("RunSmoke returned error: %v", err)
		}
		if len(run.Results) != 1 || run.Results[0].Clone != "failed" || !strings.Contains(run.Results[0].Notes, "clone exit 9") {
			t.Fatalf("clone failure result mismatch: %+v", run.Results)
		}
		if len(runner.specs) != 1 {
			t.Fatalf("clone failure should only run git once, ran %d commands", len(runner.specs))
		}
	})

	t.Run("existing repo skips clone", func(t *testing.T) {
		fixture := testharness.NewDir(t)
		manifestPath := fixture.WriteJSON(t, "manifest.json", Manifest{SchemaVersion: "1", Repos: []Repo{{
			Name:   "cobra",
			URL:    "https://example.com/cobra.git",
			Target: "./doc",
		}}})
		repoDir := fixture.Path("work", "cobra")
		if err := os.MkdirAll(repoDir, 0o755); err != nil {
			t.Fatal(err)
		}
		runner := &fakeRunner{run: func(spec CommandSpec) (CommandResult, error) {
			return CommandResult{ExitCode: 0}, nil
		}}

		run, err := RunSmoke(context.Background(), SmokeOptions{
			ManifestPath:         manifestPath,
			WorkRoot:             fixture.Path("work"),
			Names:                []string{"cobra"},
			CervoBinary:          "cervomut",
			GitBinary:            "git",
			MaxMutants:           1,
			Workers:              1,
			CloneTimeoutSeconds:  1,
			TestTimeoutSeconds:   1,
			DryRunTimeoutSeconds: 1,
			Runner:               runner,
		})
		if err != nil {
			t.Fatalf("RunSmoke returned error: %v", err)
		}
		if len(run.Results) != 1 || run.Results[0].Clone != "ok" {
			t.Fatalf("existing repo result mismatch: %+v", run.Results)
		}
		if len(runner.specs) != 2 {
			t.Fatalf("existing repo should skip clone and run baseline+dry-run, ran %d commands", len(runner.specs))
		}
		for _, spec := range runner.specs {
			if spec.Path == "git" {
				t.Fatalf("existing repo should not run git clone: %+v", runner.specs)
			}
		}
	})
}
