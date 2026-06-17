package pool

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cervantesh/cervo-mutants/internal/testharness"
)

func TestBuildToolSpecsAndCompareHelpers(t *testing.T) {
	repo := Repo{Name: "cobra", Target: "./...", URL: "https://example.com/cobra.git"}
	opts := CompareOptions{
		WorkRoot:                   t.TempDir(),
		CompareTargetMode:          "package-root",
		GremlinsTargetMode:         "package-root",
		Workers:                    3,
		GomuWorkers:                5,
		GoMutestingWorkers:         7,
		TimeoutSeconds:             90,
		GremlinsTimeoutCoefficient: 4,
		CervoBinary:                "cervomut",
		GremlinsBinary:             "gremlins",
		GomuBinary:                 "gomu",
		GoMutestingBinary:          "go-mutesting",
		GoMemoryLimit:              "2GiB",
		GoMaxProcs:                 4,
		GoFlags:                    "-count=1",
	}

	specs := buildToolSpecs(repo, t.TempDir(), opts)
	if len(specs) != 4 {
		t.Fatalf("buildToolSpecs = %d, want 4", len(specs))
	}
	if specs[0].name != "cervomut" || specs[0].effectiveTarget != "." || specs[0].targetMode != "package-root" {
		t.Fatalf("unexpected cervomut spec: %+v", specs[0])
	}
	if specs[1].name != "gremlins" || specs[1].effectiveTarget != "." || specs[1].targetMode != "package-root" {
		t.Fatalf("unexpected gremlins spec: %+v", specs[1])
	}
	if value := flagValue(specs[0].args, "--out"); value == "" {
		t.Fatalf("cervomut args missing --out: %v", specs[0].args)
	}
	gremlins := strings.Join(gremlinsArgs("./...", t.TempDir(), opts), " ")
	if !strings.Contains(gremlins, "--timeout-coefficient 4") {
		t.Fatalf("gremlinsArgs missing timeout coefficient: %s", gremlins)
	}
	if got := comparisonTarget("./...", "package-root"); got != "." {
		t.Fatalf("comparisonTarget = %q, want .", got)
	}
	if got := gremlinsMode(opts); got != "package-root" {
		t.Fatalf("gremlinsMode = %q", got)
	}
	if got := targetModeForTool("gomu", CompareOptions{}); got != "manifest" {
		t.Fatalf("targetModeForTool fallback = %q", got)
	}
	if !hasCompareResult([]CompareResult{{Repo: "cobra", Tool: "gremlins"}}, "cobra", "gremlins") {
		t.Fatal("hasCompareResult should match existing result")
	}

	env := goEnvOverrides(opts)
	text := strings.Join(env, " ")
	for _, want := range []string{"GOMEMLIMIT=2GiB", "GOMAXPROCS=4", "GOFLAGS=-count=1"} {
		if !strings.Contains(text, want) {
			t.Fatalf("goEnvOverrides missing %q: %v", want, env)
		}
	}
}

func TestAdjustMemoryThresholdsUsesMonitorTotals(t *testing.T) {
	opts := CompareOptions{
		MinFreeMemoryMB:       512,
		MinFreeCommitMB:       256,
		KillBelowFreeMemoryMB: 256,
		KillBelowFreeCommitMB: 128,
		MaxUsedMemoryMB:       7000,
		MaxCommittedMemoryMB:  12000,
	}
	monitor := &sequenceMonitor{statuses: []MemoryStatus{{
		TotalMemoryMB: 8192,
		FreeMemoryMB:  4096,
		TotalCommitMB: 16384,
		FreeCommitMB:  8192,
	}}}

	adjustMemoryThresholds(&opts, monitor)

	if opts.MinFreeMemoryMB != 1192 || opts.KillBelowFreeMemoryMB != 1192 {
		t.Fatalf("memory thresholds not raised: %+v", opts)
	}
	if opts.MinFreeCommitMB != 4384 || opts.KillBelowFreeCommitMB != 4384 {
		t.Fatalf("commit thresholds not raised: %+v", opts)
	}
}

func TestParseToolMetricsReadsReportsAndCopiesArtifacts(t *testing.T) {
	fixture := testharness.NewDir(t)
	repoOut := fixture.Path("repo-out")
	reportRoot := fixture.Path("tool-reports")

	cervoPartial := testharness.Dir{Root: reportRoot}.WriteFile(t, "cervo/partial-mutation-report.json", `{"summary":{"total":5,"killed":3,"survived":2,"not_covered":0,"timed_out":0,"compile_error":0,"skipped":0,"score":60}}`)
	metrics, err := parseToolMetrics(toolSpec{
		parser:        "cervo",
		report:        fixture.Path("repo-out", "cervo", "mutation-report.json"),
		partialReport: cervoPartial,
	}, repoOut, fixture.Path("logs", "cervo.log"))
	if err != nil {
		t.Fatalf("parseToolMetrics cervo returned error: %v", err)
	}
	if !metrics.PartialReportUsed || metrics.Total != 5 || metrics.Killed != 3 {
		t.Fatalf("unexpected cervo metrics: %+v", metrics)
	}

	gremlinsReport := testharness.Dir{Root: reportRoot}.WriteFile(t, "gremlins/report.json", `{"mutants_total":4,"mutants_killed":2,"mutants_lived":1,"mutants_not_covered":1,"test_efficacy":66.7}`)
	metrics, err = parseToolMetrics(toolSpec{parser: "gremlins", report: gremlinsReport}, repoOut, fixture.Path("logs", "gremlins.log"))
	if err != nil {
		t.Fatalf("parseToolMetrics gremlins returned error: %v", err)
	}
	if metrics.Total != 4 || metrics.Status != "ok" {
		t.Fatalf("unexpected gremlins metrics: %+v", metrics)
	}

	gomuReport := testharness.Dir{Root: reportRoot}.WriteFile(t, "gomu/mutation-report.json", `{"totalMutants":3,"results":[{"status":"KILLED"},{"status":"SURVIVED"},{"status":"ERROR"}]}`)
	metrics, err = parseToolMetrics(toolSpec{parser: "gomu", report: gomuReport}, repoOut, fixture.Path("logs", "gomu.log"))
	if err != nil {
		t.Fatalf("parseToolMetrics gomu returned error: %v", err)
	}
	if metrics.Total != 3 || metrics.Killed != 1 || metrics.Errors != 1 {
		t.Fatalf("unexpected gomu metrics: %+v", metrics)
	}
	if _, err := os.Stat(filepath.Join(repoOut, "gomu-mutation-report.json")); err != nil {
		t.Fatalf("gomu artifact copy missing: %v", err)
	}

	goMutestingReport := testharness.Dir{Root: reportRoot}.WriteFile(t, "go-mutesting/report.json", `{"stats":{"totalMutantsCount":4,"killedCount":2,"escapedCount":1,"notCoveredCount":1,"errorCount":0,"skippedCount":0,"timeOutCount":0,"msi":0.5}}`)
	metrics, err = parseToolMetrics(toolSpec{parser: "go-mutesting", report: goMutestingReport}, repoOut, fixture.Path("logs", "go-mutesting.log"))
	if err != nil {
		t.Fatalf("parseToolMetrics go-mutesting returned error: %v", err)
	}
	if metrics.Total != 4 || metrics.NotCovered != 1 || metrics.Score != 50 {
		t.Fatalf("unexpected go-mutesting metrics: %+v", metrics)
	}
	if _, err := os.Stat(filepath.Join(repoOut, "go-mutesting-report.json")); err != nil {
		t.Fatalf("go-mutesting artifact copy missing: %v", err)
	}
}

func TestClassifyCompareStatus(t *testing.T) {
	fixture := testharness.NewDir(t)

	status, note := classifyCompareStatus("cervo", 124, compareMetrics{PartialReportUsed: true}, fixture.Path("missing.log"), fixture.Path("missing.json"))
	if status != "partial_timeout" || !strings.Contains(note, "partial CervoMutants report used") {
		t.Fatalf("unexpected cervo partial timeout: status=%q note=%q", status, note)
	}

	logPath := fixture.WriteFile(t, "panic.log", "panic: boom")
	status, note = classifyCompareStatus("gremlins", 0, compareMetrics{}, logPath, fixture.Path("missing.json"))
	if status != "panic" || !strings.Contains(note, "panic") {
		t.Fatalf("unexpected gremlins panic classification: status=%q note=%q", status, note)
	}

	logPath = fixture.WriteFile(t, "no-results.log", "No results to report")
	status, note = classifyCompareStatus("gremlins", 1, compareMetrics{}, logPath, fixture.Path("missing.json"))
	if status != "no_results" || !strings.Contains(note, "no covered") {
		t.Fatalf("unexpected gremlins no-results classification: status=%q note=%q", status, note)
	}

	reportPath := fixture.WriteFile(t, "gremlins.json", "{}")
	status, note = classifyCompareStatus("gremlins", 0, compareMetrics{Status: "all_timed_out"}, fixture.Path("ok.log"), reportPath)
	if status != "all_timed_out" || !strings.Contains(note, "all observed mutations timed out") {
		t.Fatalf("unexpected gremlins timed-out classification: status=%q note=%q", status, note)
	}

	if got := appendNote("one", "two"); got != "one; two" {
		t.Fatalf("appendNote = %q", got)
	}
}
