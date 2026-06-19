package pool

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cervantesh/cervo-mutants/internal/testharness"
	"github.com/cervantesh/cervo-mutants/pkg/engine"
)

func TestPoolManifestAndFileHelpersErrorBranches(t *testing.T) {
	fixture := testharness.NewDir(t)

	if _, err := LoadManifest(fixture.Path("missing-manifest.json")); err == nil {
		t.Fatal("LoadManifest accepted a missing file")
	}
	invalidManifestPath := fixture.WriteFile(t, "invalid-manifest.json", "{")
	if _, err := LoadManifest(invalidManifestPath); err == nil {
		t.Fatal("LoadManifest accepted invalid JSON")
	}

	dirPath := fixture.Path("existing-dir")
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(dirPath, map[string]string{"status": "bad-target"}); err == nil {
		t.Fatal("writeJSON accepted a directory target")
	}

	if err := copyFile(fixture.Path("missing-source.txt"), fixture.Path("out", "copied.txt")); err == nil {
		t.Fatal("copyFile accepted a missing source file")
	}
}

func TestBenchmarkHelperFallbackBranches(t *testing.T) {
	fixture := testharness.NewDir(t)

	if _, err := LoadBenchmarkCorpus(fixture.Path("missing-corpus.json")); err == nil {
		t.Fatal("LoadBenchmarkCorpus accepted a missing file")
	}
	invalidCorpusPath := fixture.WriteFile(t, "invalid-corpus.json", "{")
	if _, err := LoadBenchmarkCorpus(invalidCorpusPath); err == nil {
		t.Fatal("LoadBenchmarkCorpus accepted invalid JSON")
	}

	if _, ok, err := loadBenchmarkSummary(fixture.Path("missing-summary.json")); err != nil || ok {
		t.Fatalf("loadBenchmarkSummary missing file = ok=%v err=%v, want ok=false err=nil", ok, err)
	}
	invalidSummaryPath := fixture.WriteFile(t, "invalid-summary.json", "{")
	if _, ok, err := loadBenchmarkSummary(invalidSummaryPath); err == nil || ok {
		t.Fatalf("loadBenchmarkSummary invalid JSON = ok=%v err=%v, want ok=false err!=nil", ok, err)
	}

	run := engine.RunResult{
		Summary: engine.Summary{
			Killed:       1,
			Survived:     2,
			TimedOut:     1,
			CompileError: 1,
		},
		Mutants: []engine.MutantResult{
			{MutantID: "k", Status: engine.StatusKilled, MemoryPeakBytes: 1 * 1024 * 1024},
			{MutantID: "s", Status: engine.StatusSurvived, MemoryPeakBytes: 3 * 1024 * 1024},
			{MutantID: "t", Status: engine.StatusTimedOut},
			{MutantID: "m", Status: engine.StatusMemoryKilled},
			{MutantID: "c", Status: engine.StatusCompileError},
			{MutantID: "n", Status: engine.StatusNotCovered},
		},
	}

	if got := benchmarkGeneratedMutants(run); got != len(run.Mutants) {
		t.Fatalf("benchmarkGeneratedMutants len fallback = %d, want %d", got, len(run.Mutants))
	}
	run.Summary.Total = 9
	if got := benchmarkGeneratedMutants(run); got != 9 {
		t.Fatalf("benchmarkGeneratedMutants total fallback = %d, want 9", got)
	}
	run.Summary.GeneratedMutants = 11
	if got := benchmarkGeneratedMutants(run); got != 11 {
		t.Fatalf("benchmarkGeneratedMutants summary value = %d, want 11", got)
	}

	run.Summary.ExecutedMutants = 0
	if got := benchmarkExecutedMutants(run); got != 5 {
		t.Fatalf("benchmarkExecutedMutants counted = %d, want 5", got)
	}
	run.Summary.ExecutedMutants = 7
	if got := benchmarkExecutedMutants(run); got != 7 {
		t.Fatalf("benchmarkExecutedMutants summary value = %d, want 7", got)
	}

	run.Summary.EffectiveMutants = 0
	if got := benchmarkEffectiveMutants(run); got != 3 {
		t.Fatalf("benchmarkEffectiveMutants fallback = %d, want 3", got)
	}
	run.Summary.EffectiveMutants = 8
	if got := benchmarkEffectiveMutants(run); got != 8 {
		t.Fatalf("benchmarkEffectiveMutants summary value = %d, want 8", got)
	}

	run.Summary.ScoreDenominator = 0
	if got := benchmarkScoreDenominator(run); got != 10 {
		t.Fatalf("benchmarkScoreDenominator fallback = %d, want 10", got)
	}
	run.Summary.ScoreDenominator = 12
	if got := benchmarkScoreDenominator(run); got != 12 {
		t.Fatalf("benchmarkScoreDenominator summary value = %d, want 12", got)
	}

	if got := benchmarkPeakMemoryMB(nil); got != 0 {
		t.Fatalf("benchmarkPeakMemoryMB nil = %.2f, want 0", got)
	}
	if got := benchmarkPeakMemoryMB(run.Mutants); got < 2.9 || got > 3.1 {
		t.Fatalf("benchmarkPeakMemoryMB = %.2f, want about 3", got)
	}

	if got := benchmarkMutantsPerSecond(0, 5); got != 0 {
		t.Fatalf("benchmarkMutantsPerSecond zero executed = %.3f, want 0", got)
	}
	if got := benchmarkMutantsPerSecond(10, 0); got < 9999 {
		t.Fatalf("benchmarkMutantsPerSecond zero elapsed = %.3f, want adjusted positive fallback", got)
	}

	unsupported := benchmarkCheck("unknown", 1, 2, "!=")
	if unsupported.Status != "fail" || unsupported.Message != "unsupported comparator" {
		t.Fatalf("benchmarkCheck unsupported comparator = %+v", unsupported)
	}

	summary := buildBenchmarkSummary("corpus.json", BenchmarkCorpus{
		TrackingIssue: "#257",
		Description:   "helper coverage",
	}, []BenchmarkResult{
		{Status: "pass", Checks: []BenchmarkCheck{{Status: "pass"}}},
		{Status: "fail", Notes: []string{"resumed from existing summary"}, Checks: []BenchmarkCheck{{Status: "fail"}}},
		{Status: "error"},
	})
	if summary.Totals.Entries != 3 || summary.Totals.Passed != 1 || summary.Totals.Failed != 1 || summary.Totals.Errored != 1 {
		t.Fatalf("benchmark summary totals mismatch: %+v", summary.Totals)
	}
	if summary.Totals.Resumed != 1 || summary.Totals.ChecksPassed != 1 || summary.Totals.ChecksFailed != 1 {
		t.Fatalf("benchmark summary check totals mismatch: %+v", summary.Totals)
	}
}

func TestCampaignValidationAndHelperBranches(t *testing.T) {
	fixture := testharness.NewDir(t)

	if _, err := LoadCampaignManifest(fixture.Path("missing-campaign.json")); err == nil {
		t.Fatal("LoadCampaignManifest accepted a missing file")
	}
	invalidCampaignPath := fixture.WriteFile(t, "invalid-campaign.json", "{")
	if _, err := LoadCampaignManifest(invalidCampaignPath); err == nil {
		t.Fatal("LoadCampaignManifest accepted invalid JSON")
	}

	if _, ok, err := loadCampaignSummary(fixture.Path("missing-campaign-summary.json")); err != nil || ok {
		t.Fatalf("loadCampaignSummary missing file = ok=%v err=%v, want ok=false err=nil", ok, err)
	}
	invalidSummaryPath := fixture.WriteFile(t, "invalid-campaign-summary.json", "{")
	if _, ok, err := loadCampaignSummary(invalidSummaryPath); err == nil || ok {
		t.Fatalf("loadCampaignSummary invalid JSON = ok=%v err=%v, want ok=false err!=nil", ok, err)
	}

	baseDir := fixture.Root
	disabled := false
	enabled := true
	if campaignJobEnabled(CampaignJob{Enabled: &disabled}) {
		t.Fatal("campaignJobEnabled accepted disabled job")
	}
	if !campaignJobEnabled(CampaignJob{}) || !campaignJobEnabled(CampaignJob{Enabled: &enabled}) {
		t.Fatal("campaignJobEnabled rejected enabled/default job")
	}

	if canResumeCampaignJob(CampaignJobResult{Status: "failed", ResumeKey: "same"}, "same") {
		t.Fatal("canResumeCampaignJob accepted failed result")
	}
	if canResumeCampaignJob(CampaignJobResult{Status: "ok"}, "same") {
		t.Fatal("canResumeCampaignJob accepted blank resume key")
	}
	if !canResumeCampaignJob(CampaignJobResult{Status: "ok", ResumeKey: "same"}, " same ") {
		t.Fatal("canResumeCampaignJob rejected matching trimmed resume key")
	}

	if got := campaignRootPath("  "+fixture.Path("cli-root")+"  ", "manifest-root", baseDir, "fallback"); got != filepath.Clean(fixture.Path("cli-root")) {
		t.Fatalf("campaignRootPath CLI value = %q", got)
	}
	if got := campaignRootPath("", "manifest-root", baseDir, "fallback"); got != filepath.Join(baseDir, "manifest-root") {
		t.Fatalf("campaignRootPath manifest value = %q", got)
	}
	if got := campaignRootPath("", "", baseDir, "fallback"); got != "fallback" {
		t.Fatalf("campaignRootPath fallback = %q, want fallback", got)
	}

	if got := campaignJobWorkRoot("default-work", baseDir, CampaignJob{}, "job-a"); got != filepath.Join("default-work", "job-a") {
		t.Fatalf("campaignJobWorkRoot default = %q", got)
	}
	if got := campaignJobWorkRoot("default-work", baseDir, CampaignJob{WorkRoot: "custom-work"}, "job-a"); got != filepath.Join(baseDir, "custom-work") {
		t.Fatalf("campaignJobWorkRoot custom = %q", got)
	}

	smokeOutput := campaignJobOutputRoot("smoke", "default-out", baseDir, CampaignJob{OutputRoot: "ignored-for-smoke"}, "smoke-job", "smoke-work")
	if smokeOutput != "smoke-work" {
		t.Fatalf("campaignJobOutputRoot smoke = %q, want smoke-work", smokeOutput)
	}
	if got := campaignJobOutputRoot("compare", "default-out", baseDir, CampaignJob{OutputRoot: "custom-out"}, "compare-job", "compare-work"); got != filepath.Join(baseDir, "custom-out") {
		t.Fatalf("campaignJobOutputRoot custom = %q", got)
	}
	if got := campaignJobOutputRoot("compare", "default-out", baseDir, CampaignJob{}, "compare-job", "compare-work"); got != filepath.Join("default-out", "compare-job") {
		t.Fatalf("campaignJobOutputRoot default = %q", got)
	}

	if got := resolveCampaignPath(baseDir, "  "); got != "" {
		t.Fatalf("resolveCampaignPath blank = %q, want empty", got)
	}
	if got := resolveCampaignPath(baseDir, "relative.json"); got != filepath.Join(baseDir, "relative.json") {
		t.Fatalf("resolveCampaignPath relative = %q", got)
	}
	if got := resolveCampaignPath(baseDir, fixture.Path("absolute.json")); got != filepath.Clean(fixture.Path("absolute.json")) {
		t.Fatalf("resolveCampaignPath absolute = %q", got)
	}

	values := copyStringMap(map[string]string{"one": "1"})
	values["one"] = "changed"
	if values["one"] != "changed" {
		t.Fatalf("copyStringMap copy write failed: %+v", values)
	}
	if got := copyStringMap(nil); got != nil {
		t.Fatalf("copyStringMap nil = %+v, want nil", got)
	}

	campaignPath := fixture.WriteFile(t, "campaign.json", `{
  "schema_version": "1",
  "jobs": [
    {"name":"smoke-missing","kind":"smoke"},
    {"name":"compare-missing","kind":"compare"},
    {"name":"benchmark-missing","kind":"benchmark"},
    {"name":"unknown-job","kind":"weird"}
  ]
}`)
	run, err := RunCampaign(context.Background(), CampaignOptions{
		Path:       campaignPath,
		WorkRoot:   fixture.Path("work"),
		OutputRoot: fixture.Path("out"),
	})
	if err != nil {
		t.Fatalf("RunCampaign returned error: %v", err)
	}
	if len(run.Results) != 4 {
		t.Fatalf("RunCampaign results = %d, want 4", len(run.Results))
	}
	wantErrors := []string{
		"manifest_path is required for smoke jobs",
		"manifest_path is required for compare jobs",
		"corpus_path is required for benchmark jobs",
		`unsupported campaign job kind "weird"`,
	}
	for idx, want := range wantErrors {
		if run.Results[idx].Status != "failed" || run.Results[idx].Error != want {
			t.Fatalf("RunCampaign result %d = %+v, want error %q", idx, run.Results[idx], want)
		}
		if run.Results[idx].ElapsedSeconds < 0 {
			t.Fatalf("RunCampaign elapsed seconds should be non-negative: %+v", run.Results[idx])
		}
	}
	summary, ok, err := loadCampaignSummary(run.SummaryPath)
	if err != nil || !ok {
		t.Fatalf("loadCampaignSummary after validation run ok=%v err=%v", ok, err)
	}
	if summary.Totals.Jobs != 4 || summary.Totals.Failed != 4 {
		t.Fatalf("campaign summary totals mismatch: %+v", summary.Totals)
	}

	// Keep a tiny direct use of seconds() in this file so helper timing stays exercised
	// even when the validation run returns almost immediately on fast hosts.
	if got := seconds(time.Now().Add(-1100 * time.Millisecond)); got < 1 {
		t.Fatalf("seconds helper = %.3f, want elapsed value >= 1", got)
	}
}
