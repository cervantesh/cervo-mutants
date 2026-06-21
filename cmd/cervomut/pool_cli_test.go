package main

import (
	"context"
	"strings"
	"testing"

	"github.com/cervantesh/cervo-mutants/pkg/pool"
)

func TestPoolSmokeCommandDispatches(t *testing.T) {
	app := newCLIApp()
	called := false
	app.deps.runPoolSmoke = func(_ context.Context, opts pool.SmokeOptions) (pool.RunSummary[pool.SmokeResult], error) {
		called = true
		if opts.ManifestPath != "manifest.json" || opts.WorkRoot != "work" || opts.MaxMutants != 7 || opts.Workers != 3 || opts.CervoBinary != "cervomut.exe" {
			t.Fatalf("unexpected smoke options: %+v", opts)
		}
		if len(opts.Names) != 2 || opts.Names[0] != "cobra" || opts.Names[1] != "pflag" {
			t.Fatalf("unexpected names: %+v", opts.Names)
		}
		return pool.RunSummary[pool.SmokeResult]{SummaryPath: "work/summary.json"}, nil
	}

	if err := app.run([]string{"pool", "smoke", "--manifest", "manifest.json", "--work-root", "work", "--names", "cobra,pflag", "--max-mutants", "7", "--workers", "3", "--cervomutants", "cervomut.exe"}); err != nil {
		t.Fatalf("pool smoke returned error: %v", err)
	}
	if !called {
		t.Fatal("pool smoke handler was not called")
	}
}

func TestPoolCompareCommandDispatches(t *testing.T) {
	app := newCLIApp()
	called := false
	app.deps.runPoolCompare = func(_ context.Context, opts pool.CompareOptions) (pool.RunSummary[pool.CompareResult], error) {
		called = true
		if opts.ManifestPath != "manifest.json" || opts.WorkRoot != "work" || opts.OutputRoot != "out" || opts.CompareTargetMode != "package-root" {
			t.Fatalf("unexpected compare options: %+v", opts)
		}
		if len(opts.Tools) != 2 || opts.Tools[0] != "cervomut" || opts.Tools[1] != "gremlins" {
			t.Fatalf("unexpected tools: %+v", opts.Tools)
		}
		return pool.RunSummary[pool.CompareResult]{
			SummaryPath: "out/summary.json",
			Artifacts: map[string]string{
				"study_json":       "out/comparison-study.json",
				"summary_markdown": "out/comparison-summary.md",
			},
		}, nil
	}

	output := captureStdout(t, func() {
		if err := app.run([]string{"pool", "compare", "--manifest", "manifest.json", "--work-root", "work", "--output-root", "out", "--tools", "cervomut,gremlins", "--compare-target-mode", "package-root"}); err != nil {
			t.Fatalf("pool compare returned error: %v", err)
		}
	})
	if !strings.Contains(output, "Pool comparison raw summary: out/summary.json") {
		t.Fatalf("raw summary output missing:\n%s", output)
	}
	if !strings.Contains(output, "Pool comparison study JSON: out/comparison-study.json") {
		t.Fatalf("study json output missing:\n%s", output)
	}
	if !strings.Contains(output, "Pool comparison summary markdown: out/comparison-summary.md") {
		t.Fatalf("summary markdown output missing:\n%s", output)
	}
	if !called {
		t.Fatal("pool compare handler was not called")
	}
}

func TestPoolBenchmarkCommandDispatches(t *testing.T) {
	app := newCLIApp()
	called := false
	app.deps.runPoolBenchmark = func(_ context.Context, opts pool.BenchmarkOptions) (pool.RunSummary[pool.BenchmarkResult], error) {
		called = true
		if opts.CorpusPath != "corpus.json" || opts.WorkRoot != "work" || opts.OutputRoot != "out" || !opts.Resume || opts.CervoBinary != "cervomut.exe" {
			t.Fatalf("unexpected benchmark options: %+v", opts)
		}
		if len(opts.Names) != 2 || opts.Names[0] != "cobra-doc" || opts.Names[1] != "logrus" {
			t.Fatalf("unexpected benchmark names: %+v", opts.Names)
		}
		return pool.RunSummary[pool.BenchmarkResult]{SummaryPath: "out/summary.json"}, nil
	}

	if err := app.run([]string{"pool", "benchmark", "--corpus", "corpus.json", "--work-root", "work", "--output-root", "out", "--names", "cobra-doc,logrus", "--resume", "--cervomutants", "cervomut.exe"}); err != nil {
		t.Fatalf("pool benchmark returned error: %v", err)
	}
	if !called {
		t.Fatal("pool benchmark handler was not called")
	}
}

func TestPoolCampaignCommandDispatches(t *testing.T) {
	app := newCLIApp()
	called := false
	app.deps.runPoolCampaign = func(_ context.Context, opts pool.CampaignOptions) (pool.RunSummary[pool.CampaignJobResult], error) {
		called = true
		if opts.Path != "campaign.json" || opts.WorkRoot != "work" || opts.OutputRoot != "out" || !opts.Resume || opts.CervoBinary != "cervomut.exe" {
			t.Fatalf("unexpected campaign options: %+v", opts)
		}
		if opts.GremlinsBinary != "gremlins.exe" || opts.GomuBinary != "gomu.exe" || opts.GoMutestingBinary != "go-mutesting.exe" {
			t.Fatalf("unexpected campaign binary options: %+v", opts)
		}
		return pool.RunSummary[pool.CampaignJobResult]{SummaryPath: "out/campaign-summary.json"}, nil
	}

	output := captureStdout(t, func() {
		if err := app.run([]string{"pool", "campaign", "--file", "campaign.json", "--work-root", "work", "--output-root", "out", "--resume", "--cervomutants", "cervomut.exe", "--gremlins", "gremlins.exe", "--gomu", "gomu.exe", "--go-mutesting", "go-mutesting.exe"}); err != nil {
			t.Fatalf("pool campaign returned error: %v", err)
		}
	})
	if !strings.Contains(output, "Pool campaign summary: out/campaign-summary.json") {
		t.Fatalf("campaign output missing:\n%s", output)
	}
	if !called {
		t.Fatal("pool campaign handler was not called")
	}
}
