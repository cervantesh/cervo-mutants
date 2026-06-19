package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cervantesh/cervo-mutants/pkg/engine"
	reportpkg "github.com/cervantesh/cervo-mutants/pkg/report"
)

func TestBuildWaveResultFromReportArtifacts(t *testing.T) {
	reportDir := filepath.Join(t.TempDir(), "wave")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatalf("create report dir: %v", err)
	}

	runResult := engine.RunResult{
		Summary: engine.Summary{
			Total:            4,
			Killed:           2,
			Survived:         1,
			NotCovered:       1,
			TimedOut:         0,
			CompileError:     0,
			Score:            66.7,
			MutationCoverage: 75,
			TestEfficacy:     66.7,
			Actionable: engine.ActionableSummary{
				ActionableScore:             80,
				ActionableSurvivors:         1,
				TrueActionableSurvivors:     1,
				EquivalentRiskSurvivors:     0,
				SemanticGroupReviewUnits:    1,
				CollapsedSemanticDuplicates: 1,
			},
			DenominatorHealth: engine.DenominatorHealth{
				Generated: 4,
				Covered:   4,
				Executed:  4,
				Effective: 3,
				Warnings:  []string{"partial denominator"},
			},
			SemanticGroupStats: map[string]int{"group-1": 2},
		},
		Environment: engine.Environment{
			OS:        "linux",
			GoVersion: "go1.25.6",
		},
		Mutants: []engine.MutantResult{
			{
				MutantID:           "m1",
				TestRecommendation: &engine.TestRecommendation{Priority: "high"},
				Mutant:             engine.Mutant{SemanticGroup: "group-1"},
			},
			{
				MutantID:           "m2",
				TestRecommendation: &engine.TestRecommendation{Priority: "medium"},
				Mutant:             engine.Mutant{SemanticGroup: "group-1"},
			},
			{
				MutantID:           "m3",
				TestRecommendation: &engine.TestRecommendation{Priority: "high"},
				Mutant:             engine.Mutant{},
			},
		},
	}
	writeJSONForTest(t, filepath.Join(reportDir, "mutation-report.json"), runResult)
	writeJSONForTest(t, filepath.Join(reportDir, "semantic-triage-ledger.json"), reportpkg.TriageLedger{
		SchemaVersion: "1",
		Entries: []reportpkg.TriageLedgerEntry{
			{MutantID: "m1"},
			{MutantID: "m2"},
		},
	})
	writeJSONForTest(t, filepath.Join(reportDir, "governance-review.json"), reportpkg.GovernanceReview{
		SchemaVersion: "1",
		QuarantineTemplates: []reportpkg.GovernanceQuarantineTemplate{
			{MutantID: "m1", Status: engine.StatusTimedOut},
		},
		SuppressionTemplates: []reportpkg.GovernanceSuppressionTemplate{
			{MutantID: "m2", Status: engine.StatusSurvived},
		},
	})

	result, err := buildWaveResult(waveMetadata{
		GeneratedAt:        "2026-06-19T20:15:00Z",
		Name:               "apimachinery-resource",
		Repository:         "kubernetes/apimachinery",
		InstallPath:        "github-action@test",
		ActionRef:          "abc123",
		Ref:                "main",
		Target:             "./pkg/api/resource",
		Profile:            "ci-balanced",
		GoVersion:          "1.26.0",
		GoVersionTarget:    "1.26.0",
		GoVersionActionMin: "1.25.6",
		Policy:             "ci-balanced",
		CoveragePrefilter:  true,
		ReportDir:          reportDir,
		JobStatus:          "success",
		MaxMutants:         10,
		Workers:            2,
	})
	if err != nil {
		t.Fatalf("buildWaveResult returned error: %v", err)
	}
	if result.ReportKind != "full" {
		t.Fatalf("report kind = %q, want full", result.ReportKind)
	}
	if result.Summary == nil || result.Summary.Killed != 2 || result.Summary.ActionableScore == nil || *result.Summary.ActionableScore != 80 {
		t.Fatalf("unexpected summary: %+v", result.Summary)
	}
	if result.Triage.RecommendationEntries != 3 || result.Triage.RecommendationReviewUnits != 2 || result.Triage.CollapsedRecommendationDupes != 1 {
		t.Fatalf("unexpected recommendation triage: %+v", result.Triage)
	}
	if result.Triage.LedgerEntries != 2 || result.Triage.GovernanceTotalSuggestions != 2 {
		t.Fatalf("unexpected ledger/governance triage: %+v", result.Triage)
	}
	if result.Triage.GovernanceSuggestionsByStatus[string(engine.StatusTimedOut)] != 1 || result.Triage.GovernanceSuggestionsByStatus[string(engine.StatusSurvived)] != 1 {
		t.Fatalf("unexpected governance suggestions by status: %+v", result.Triage.GovernanceSuggestionsByStatus)
	}
	if result.DenominatorHealth == nil || result.DenominatorHealth.Generated != 4 || len(result.DenominatorHealth.Warnings) != 1 {
		t.Fatalf("unexpected denominator health: %+v", result.DenominatorHealth)
	}
}

func TestBuildWaveResultUsesFailureDebugWhenReportMissing(t *testing.T) {
	reportDir := filepath.Join(t.TempDir(), "wave")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatalf("create report dir: %v", err)
	}
	writeJSONForTest(t, filepath.Join(reportDir, "failure-debug.json"), map[string]any{
		"schema_version": "1",
		"kind":           "runner_error",
		"message":        "runner_error: baseline tests failed before mutation",
		"correlation_id": "cid-278",
		"runner_result": map[string]any{
			"status":        "compile_error",
			"status_reason": "baseline compile failed",
			"command":       []string{"go", "test", "./pkg/api/resource"},
			"output":        "go: go.mod requires go >= 1.26.0",
		},
	})

	result, err := buildWaveResult(waveMetadata{
		GeneratedAt:        "2026-06-19T20:16:00Z",
		Name:               "apimachinery-resource",
		Repository:         "kubernetes/apimachinery",
		InstallPath:        "github-action@test",
		ActionRef:          "abc123",
		Ref:                "main",
		Target:             "./pkg/api/resource",
		Profile:            "ci-balanced",
		GoVersion:          "1.26.0",
		GoVersionActionMin: "1.25.6",
		Policy:             "ci-balanced",
		ReportDir:          reportDir,
		JobStatus:          "failure",
	})
	if err != nil {
		t.Fatalf("buildWaveResult returned error: %v", err)
	}
	if result.ReportKind != "missing" || result.Summary != nil {
		t.Fatalf("unexpected missing-report result: %+v", result)
	}
	if result.Failure == nil || result.Failure.Kind != "runner_error" || result.Failure.RunnerResult == nil {
		t.Fatalf("unexpected recovered failure: %+v", result.Failure)
	}
	markdown := renderWaveResultMarkdown(result)
	for _, want := range []string{"- Report: missing", "- Failure: `runner_error`", "- Runner detail: status=`compile_error`"} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("wave markdown missing %q:\n%s", want, markdown)
		}
	}
}

func TestBuildWaveSummaryAndMarkdown(t *testing.T) {
	artifactsDir := filepath.Join(t.TempDir(), "artifacts")
	repoADir := filepath.Join(artifactsDir, "external-wave-a")
	repoBDir := filepath.Join(artifactsDir, "external-wave-b")
	if err := os.MkdirAll(repoADir, 0o755); err != nil {
		t.Fatalf("create repoA dir: %v", err)
	}
	if err := os.MkdirAll(repoBDir, 0o755); err != nil {
		t.Fatalf("create repoB dir: %v", err)
	}
	writeJSONForTest(t, filepath.Join(repoADir, "wave-result.json"), waveResult{
		Name:               "repo-a",
		Repository:         "owner/repo-a",
		InstallPath:        "github-action@test",
		ActionRef:          "abc123",
		Target:             "./...",
		Profile:            "ci-fast",
		GoVersion:          "1.25.6",
		GoVersionActionMin: "1.25.6",
		Policy:             "ci-fast",
		ReportKind:         "full",
		Summary: &waveResultSummary{
			Killed:          3,
			Survived:        1,
			NotCovered:      0,
			TimedOut:        0,
			CompileError:    0,
			Score:           75,
			ActionableScore: nullableFloat64(80),
		},
		Triage: waveTriage{
			ActionableReviewUnits:        1,
			SemanticGroupCount:           1,
			RecommendationEntries:        2,
			RecommendationReviewUnits:    1,
			CollapsedRecommendationDupes: 1,
			LedgerEntries:                1,
			GovernanceSuggestionsByStatus: map[string]int{
				"survived": 1,
			},
			GovernanceTotalSuggestions: 1,
		},
		DenominatorHealth: &engine.DenominatorHealth{
			Generated: 4,
			Covered:   4,
			Executed:  4,
			Effective: 4,
			Warnings:  []string{"warn"},
		},
		Failure: nil,
	})
	writeJSONForTest(t, filepath.Join(repoBDir, "wave-result.json"), waveResult{
		Name:               "repo-b",
		Repository:         "owner/repo-b",
		InstallPath:        "github-action@test",
		ActionRef:          "abc123",
		Target:             "./pkg",
		Profile:            "ci-balanced",
		GoVersion:          "1.26.0",
		GoVersionActionMin: "1.25.6",
		Policy:             "ci-balanced",
		ReportKind:         "missing",
		Triage: waveTriage{
			GovernanceSuggestionsByStatus: map[string]int{},
		},
		Failure: &engine.Failure{
			Kind:    "runner_error",
			Message: "baseline failed",
		},
	})

	summary, err := buildWaveSummary("#278", "docs/evaluations/wave.json", "abc123", "github-action@test", artifactsDir, "2026-06-19T20:17:00Z")
	if err != nil {
		t.Fatalf("buildWaveSummary returned error: %v", err)
	}
	if summary.Aggregate.Selected != 2 || summary.Aggregate.Reports != 1 || summary.Aggregate.MissingReports != 1 {
		t.Fatalf("unexpected aggregate selection counts: %+v", summary.Aggregate)
	}
	if summary.Aggregate.Generated != 4 || summary.Aggregate.WarningRepos != 1 {
		t.Fatalf("unexpected aggregate denominator counts: %+v", summary.Aggregate)
	}
	if summary.Aggregate.FailedReports != 1 || summary.Aggregate.FailureKinds["runner_error"] != 1 {
		t.Fatalf("unexpected failure aggregate: %+v", summary.Aggregate)
	}
	if summary.Triage.RecommendationEntries != 2 || summary.Triage.GovernanceTotalSuggestions != 1 {
		t.Fatalf("unexpected triage aggregate: %+v", summary.Triage)
	}
	markdown := renderWaveSummaryMarkdown(summary)
	for _, want := range []string{"# External GitHub Action Wave Summary", "- Failure kinds: `runner_error=1`", "## repo-a", "## repo-b"} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("summary markdown missing %q:\n%s", want, markdown)
		}
	}
}

func TestWaveHelperCommandsWriteJSONAndMarkdown(t *testing.T) {
	reportDir := filepath.Join(t.TempDir(), "wave")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatalf("create report dir: %v", err)
	}
	writeJSONForTest(t, filepath.Join(reportDir, "failure-debug.json"), map[string]any{
		"schema_version": "1",
		"kind":           "runner_error",
		"message":        "baseline failed",
		"correlation_id": "cid-278",
	})

	var resultOut bytes.Buffer
	if err := cmdBuildWaveResult([]string{
		"--generated-at", "2026-06-19T20:18:00Z",
		"--name", "repo-a",
		"--repository", "owner/repo-a",
		"--install-path", "github-action@test",
		"--action-ref", "abc123",
		"--ref", "main",
		"--target", "./...",
		"--profile", "ci-fast",
		"--go-version", "1.25.6",
		"--go-version-action-min", "1.25.6",
		"--policy", "ci-fast",
		"--report-dir", reportDir,
		"--job-status", "failure",
	}, &resultOut); err != nil {
		t.Fatalf("cmdBuildWaveResult returned error: %v", err)
	}
	var result waveResult
	if err := json.Unmarshal(resultOut.Bytes(), &result); err != nil {
		t.Fatalf("cmdBuildWaveResult did not emit valid JSON: %v", err)
	}
	if result.ReportKind != "missing" || result.Failure == nil {
		t.Fatalf("unexpected wave result command output: %+v", result)
	}

	writeJSONForTest(t, filepath.Join(reportDir, "wave-result.json"), result)
	var markdownOut bytes.Buffer
	if err := cmdRenderWaveResultMarkdown([]string{"--path", filepath.Join(reportDir, "wave-result.json")}, &markdownOut); err != nil {
		t.Fatalf("cmdRenderWaveResultMarkdown returned error: %v", err)
	}
	if !strings.Contains(markdownOut.String(), "## repo-a") {
		t.Fatalf("wave result markdown missing repo heading:\n%s", markdownOut.String())
	}
}
