package pool

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRunCampaignExecutesJobsAndWritesSummary(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "repos.json")
	corpusPath := filepath.Join(dir, "corpus.json")
	if err := os.WriteFile(manifestPath, []byte(`{"schema_version":"1","repos":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(corpusPath, []byte(`{"schema_version":"1","entries":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	campaignPath := filepath.Join(dir, "campaign.json")
	if err := os.WriteFile(campaignPath, []byte(`{
  "schema_version": "1",
  "tracking_issue": "#89",
  "description": "pool campaign test",
  "jobs": [
    {"name":"smoke-pass","kind":"smoke","manifest_path":"repos.json","names":["cobra"],"run_mutation":true,"max_mutants":7,"workers":3},
    {"name":"compare-pass","kind":"compare","manifest_path":"repos.json","tools":["cervomut","gremlins"],"compare_target_mode":"package-root"},
    {"name":"benchmark-pass","kind":"benchmark","corpus_path":"corpus.json","names":["cobra-doc"],"limit":1}
  ]
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	workRoot := filepath.Join(dir, "work")
	outputRoot := filepath.Join(dir, "out")
	smokeCalled := false
	compareCalled := false
	benchmarkCalled := false
	run, err := RunCampaign(context.Background(), CampaignOptions{
		Path:              campaignPath,
		WorkRoot:          workRoot,
		OutputRoot:        outputRoot,
		CervoBinary:       "cervomut.exe",
		GitBinary:         "git.exe",
		GremlinsBinary:    "gremlins.exe",
		GomuBinary:        "gomu.exe",
		GoMutestingBinary: "go-mutesting.exe",
		SmokeRunner: func(_ context.Context, opts SmokeOptions) (RunSummary[SmokeResult], error) {
			smokeCalled = true
			if opts.ManifestPath != manifestPath {
				t.Fatalf("smoke manifest = %q, want %q", opts.ManifestPath, manifestPath)
			}
			if opts.WorkRoot != filepath.Join(workRoot, "smoke-pass") {
				t.Fatalf("smoke work root = %q", opts.WorkRoot)
			}
			if !opts.RunMutation || opts.MaxMutants != 7 || opts.Workers != 3 {
				t.Fatalf("unexpected smoke opts: %+v", opts)
			}
			return RunSummary[SmokeResult]{
				Results:     []SmokeResult{{Name: "cobra"}},
				SummaryPath: filepath.Join(opts.WorkRoot, "summary.json"),
			}, nil
		},
		CompareRunner: func(_ context.Context, opts CompareOptions) (RunSummary[CompareResult], error) {
			compareCalled = true
			if opts.ManifestPath != manifestPath {
				t.Fatalf("compare manifest = %q, want %q", opts.ManifestPath, manifestPath)
			}
			if opts.WorkRoot != filepath.Join(workRoot, "compare-pass") {
				t.Fatalf("compare work root = %q", opts.WorkRoot)
			}
			if opts.OutputRoot != filepath.Join(outputRoot, "compare-pass") {
				t.Fatalf("compare output root = %q", opts.OutputRoot)
			}
			if opts.CompareTargetMode != "package-root" || len(opts.Tools) != 2 {
				t.Fatalf("unexpected compare opts: %+v", opts)
			}
			return RunSummary[CompareResult]{
				Results:     []CompareResult{{Repo: "cobra", Tool: "cervomut"}},
				SummaryPath: filepath.Join(opts.OutputRoot, "summary.json"),
				Artifacts: map[string]string{
					"study_json": filepath.Join(opts.OutputRoot, "comparison-study.json"),
				},
			}, nil
		},
		BenchmarkRunner: func(_ context.Context, opts BenchmarkOptions) (RunSummary[BenchmarkResult], error) {
			benchmarkCalled = true
			if opts.CorpusPath != corpusPath {
				t.Fatalf("benchmark corpus = %q, want %q", opts.CorpusPath, corpusPath)
			}
			if opts.WorkRoot != filepath.Join(workRoot, "benchmark-pass") {
				t.Fatalf("benchmark work root = %q", opts.WorkRoot)
			}
			if opts.OutputRoot != filepath.Join(outputRoot, "benchmark-pass") {
				t.Fatalf("benchmark output root = %q", opts.OutputRoot)
			}
			if len(opts.Names) != 1 || opts.Names[0] != "cobra-doc" || opts.Limit != 1 {
				t.Fatalf("unexpected benchmark opts: %+v", opts)
			}
			return RunSummary[BenchmarkResult]{
				Results:     []BenchmarkResult{{Name: "cobra-doc"}},
				SummaryPath: filepath.Join(opts.OutputRoot, "summary.json"),
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("RunCampaign returned error: %v", err)
	}
	if !smokeCalled || !compareCalled || !benchmarkCalled {
		t.Fatalf("runner calls smoke=%v compare=%v benchmark=%v", smokeCalled, compareCalled, benchmarkCalled)
	}
	if run.SummaryPath != filepath.Join(outputRoot, "campaign-summary.json") {
		t.Fatalf("summary path = %q", run.SummaryPath)
	}
	summary, ok, err := loadCampaignSummary(run.SummaryPath)
	if err != nil {
		t.Fatalf("loadCampaignSummary returned error: %v", err)
	}
	if !ok {
		t.Fatal("campaign summary was not written")
	}
	if summary.Totals.Jobs != 3 || summary.Totals.Succeeded != 3 || summary.Totals.Failed != 0 {
		t.Fatalf("unexpected totals: %+v", summary.Totals)
	}
	if len(summary.Results) != 3 {
		t.Fatalf("unexpected result count: %d", len(summary.Results))
	}
	if summary.Results[1].Artifacts["study_json"] == "" {
		t.Fatalf("compare artifacts missing: %+v", summary.Results[1].Artifacts)
	}
}

func TestRunCampaignResumeSkipsCompletedJobs(t *testing.T) {
	dir := t.TempDir()
	campaignPath := filepath.Join(dir, "campaign.json")
	if err := os.WriteFile(campaignPath, []byte(`{
  "schema_version": "1",
  "jobs": [
    {"name":"smoke-pass","kind":"smoke","manifest_path":"repos.json"},
    {"name":"compare-pass","kind":"compare","manifest_path":"repos.json"}
  ]
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	workRoot := filepath.Join(dir, "work")
	summaryPath := filepath.Join(dir, "out", "campaign-summary.json")
	if err := writeCampaignSummary(summaryPath, campaignPath, CampaignManifest{}, []CampaignJobResult{{
		Name:        "smoke-pass",
		Kind:        "smoke",
		Status:      "ok",
		ResumeKey:   campaignJobResumeKey(dir, workRoot, filepath.Join(dir, "out"), CampaignJob{Name: "smoke-pass", Kind: "smoke", ManifestPath: "repos.json"}, "smoke-pass"),
		SummaryPath: filepath.Join(dir, "cached-smoke-summary.json"),
	}}); err != nil {
		t.Fatal(err)
	}

	compareCalled := false
	run, err := RunCampaign(context.Background(), CampaignOptions{
		Path:       campaignPath,
		WorkRoot:   workRoot,
		OutputRoot: filepath.Join(dir, "out"),
		Resume:     true,
		SmokeRunner: func(_ context.Context, _ SmokeOptions) (RunSummary[SmokeResult], error) {
			t.Fatal("smoke runner should not be called for resumed job")
			return RunSummary[SmokeResult]{}, nil
		},
		CompareRunner: func(_ context.Context, opts CompareOptions) (RunSummary[CompareResult], error) {
			compareCalled = true
			return RunSummary[CompareResult]{
				Results:     []CompareResult{{Repo: "cobra", Tool: "cervomut"}},
				SummaryPath: filepath.Join(opts.OutputRoot, "summary.json"),
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("RunCampaign returned error: %v", err)
	}
	if !compareCalled {
		t.Fatal("compare runner was not called")
	}
	if len(run.Results) != 2 || !run.Results[0].Resumed {
		t.Fatalf("unexpected resumed results: %+v", run.Results)
	}
	if !containsNote(run.Results[0].Notes, "resumed from existing campaign summary") {
		t.Fatalf("resume note missing: %+v", run.Results[0].Notes)
	}
	summary, ok, err := loadCampaignSummary(run.SummaryPath)
	if err != nil || !ok {
		t.Fatalf("loadCampaignSummary ok=%v err=%v", ok, err)
	}
	if summary.Totals.Resumed != 1 || summary.Totals.Succeeded != 2 {
		t.Fatalf("unexpected totals after resume: %+v", summary.Totals)
	}
}

func TestRunCampaignResumeRejectsStaleJobDefinition(t *testing.T) {
	dir := t.TempDir()
	campaignPath := filepath.Join(dir, "campaign.json")
	if err := os.WriteFile(campaignPath, []byte(`{
  "schema_version": "1",
  "jobs": [
    {"name":"smoke-pass","kind":"smoke","manifest_path":"repos.json","workers":2}
  ]
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	summaryPath := filepath.Join(dir, "out", "campaign-summary.json")
	if err := writeCampaignSummary(summaryPath, campaignPath, CampaignManifest{}, []CampaignJobResult{{
		Name:      "smoke-pass",
		Kind:      "smoke",
		Status:    "ok",
		ResumeKey: "stale-key",
	}}); err != nil {
		t.Fatal(err)
	}

	smokeCalled := false
	run, err := RunCampaign(context.Background(), CampaignOptions{
		Path:       campaignPath,
		OutputRoot: filepath.Join(dir, "out"),
		Resume:     true,
		SmokeRunner: func(_ context.Context, opts SmokeOptions) (RunSummary[SmokeResult], error) {
			smokeCalled = true
			return RunSummary[SmokeResult]{
				Results:     []SmokeResult{{Name: "cobra"}},
				SummaryPath: filepath.Join(opts.WorkRoot, "summary.json"),
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("RunCampaign returned error: %v", err)
	}
	if !smokeCalled {
		t.Fatal("smoke runner should execute when resume key is stale")
	}
	if len(run.Results) != 1 || run.Results[0].Resumed {
		t.Fatalf("unexpected stale resume reuse: %+v", run.Results)
	}
}

func TestRunCampaignDisabledJobDoesNotReuseCachedResult(t *testing.T) {
	dir := t.TempDir()
	campaignPath := filepath.Join(dir, "campaign.json")
	if err := os.WriteFile(campaignPath, []byte(`{
  "schema_version": "1",
  "jobs": [
    {"name":"smoke-pass","kind":"smoke","manifest_path":"repos.json","enabled":false}
  ]
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	summaryPath := filepath.Join(dir, "out", "campaign-summary.json")
	if err := writeCampaignSummary(summaryPath, campaignPath, CampaignManifest{}, []CampaignJobResult{{
		Name:      "smoke-pass",
		Kind:      "smoke",
		Status:    "ok",
		ResumeKey: campaignJobResumeKey(dir, "", filepath.Join(dir, "out"), CampaignJob{Name: "smoke-pass", Kind: "smoke", ManifestPath: "repos.json"}, "smoke-pass"),
	}}); err != nil {
		t.Fatal(err)
	}

	run, err := RunCampaign(context.Background(), CampaignOptions{
		Path:       campaignPath,
		OutputRoot: filepath.Join(dir, "out"),
		Resume:     true,
		SmokeRunner: func(_ context.Context, _ SmokeOptions) (RunSummary[SmokeResult], error) {
			t.Fatal("disabled job should not run")
			return RunSummary[SmokeResult]{}, nil
		},
	})
	if err != nil {
		t.Fatalf("RunCampaign returned error: %v", err)
	}
	if len(run.Results) != 1 || run.Results[0].Resumed || run.Results[0].Status != "skipped" {
		t.Fatalf("disabled job reused cached result: %+v", run.Results)
	}
}

func TestRunCampaignRecordsFailuresAndContinues(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "repos.json")
	corpusPath := filepath.Join(dir, "corpus.json")
	if err := os.WriteFile(manifestPath, []byte(`{"schema_version":"1","repos":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(corpusPath, []byte(`{"schema_version":"1","entries":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	campaignPath := filepath.Join(dir, "campaign.json")
	if err := os.WriteFile(campaignPath, []byte(`{
  "schema_version": "1",
  "jobs": [
    {"name":"smoke-fail","kind":"smoke","manifest_path":"repos.json"},
    {"name":"benchmark-pass","kind":"benchmark","corpus_path":"corpus.json"}
  ]
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	run, err := RunCampaign(context.Background(), CampaignOptions{
		Path: campaignPath,
		SmokeRunner: func(_ context.Context, _ SmokeOptions) (RunSummary[SmokeResult], error) {
			return RunSummary[SmokeResult]{}, errors.New("smoke exploded")
		},
		BenchmarkRunner: func(_ context.Context, opts BenchmarkOptions) (RunSummary[BenchmarkResult], error) {
			return RunSummary[BenchmarkResult]{
				Results:     []BenchmarkResult{{Name: "cobra-doc"}},
				SummaryPath: filepath.Join(opts.OutputRoot, "summary.json"),
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("RunCampaign returned error: %v", err)
	}
	if len(run.Results) != 2 {
		t.Fatalf("unexpected result count: %d", len(run.Results))
	}
	if run.Results[0].Status != "failed" || run.Results[0].Error != "smoke exploded" {
		t.Fatalf("unexpected failed result: %+v", run.Results[0])
	}
	if run.Results[1].Status != "ok" {
		t.Fatalf("unexpected second result: %+v", run.Results[1])
	}
	summary, ok, err := loadCampaignSummary(run.SummaryPath)
	if err != nil || !ok {
		t.Fatalf("loadCampaignSummary ok=%v err=%v", ok, err)
	}
	if summary.Totals.Failed != 1 || summary.Totals.Succeeded != 1 {
		t.Fatalf("unexpected failure totals: %+v", summary.Totals)
	}
}
