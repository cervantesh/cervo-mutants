package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cervantesh/cervo-mutants/pkg/engine"
)

func TestGoldenPublicReportFormats(t *testing.T) {
	run := goldenReportFixture()

	jsonData, err := JSON(run)
	if err != nil {
		t.Fatalf("JSON returned error: %v", err)
	}
	junitData, err := JUnit(run)
	if err != nil {
		t.Fatalf("JUnit returned error: %v", err)
	}
	sarifData, err := SARIF(run)
	if err != nil {
		t.Fatalf("SARIF returned error: %v", err)
	}

	cases := []struct {
		name string
		path string
		data []byte
	}{
		{name: "json", path: filepath.Join("testdata", "public-report.json.golden"), data: jsonData},
		{name: "junit", path: filepath.Join("testdata", "public-report.junit.xml.golden"), data: junitData},
		{name: "html", path: filepath.Join("testdata", "public-report.index.html.golden"), data: []byte(HTML(run))},
		{name: "sarif", path: filepath.Join("testdata", "public-report.sarif.golden"), data: sarifData},
		{name: "github-summary", path: filepath.Join("testdata", "public-report.github-summary.md.golden"), data: []byte(GitHubSummary(run))},
		{name: "test-recommendations", path: filepath.Join("testdata", "public-report.test-recommendations.md.golden"), data: []byte(TestRecommendations(run))},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertGoldenBytes(t, tc.path, tc.data)
		})
	}
}

func goldenReportFixture() engine.RunResult {
	return engine.RunResult{
		SchemaVersion:       "1",
		StoppedReason:       "completed",
		LastCompletedMutant: "pkg/review.go:27:logical:abc123def456",
		Summary: engine.Summary{
			Total:            4,
			Killed:           1,
			Survived:         2,
			NotCovered:       1,
			TimedOut:         1,
			MemoryKilled:     0,
			CompileError:     0,
			GeneratedMutants: 5,
			CoveredMutants:   4,
			ExecutedMutants:  4,
			EffectiveMutants: 3,
			ScoreDenominator: 4,
			Score:            50,
			EffectiveScore:   33.33,
			TestEfficacy:     33.33,
			MutationCoverage: 80,
			Actionable: engine.ActionableSummary{
				RawScore:                    50,
				ActionableScore:             50,
				Survivors:                   2,
				ActionableSurvivors:         1,
				TrueActionableSurvivors:     1,
				EquivalentRiskSurvivors:     1,
				PlatformSensitiveSurvivors:  1,
				NonProgressTimeouts:         1,
				SemanticGroupReviewUnits:    1,
				CollapsedSemanticDuplicates: 0,
			},
			HighRiskSurvivors:             1,
			SuppressionReportOnly:         1,
			SuppressionLowerPriority:      1,
			SuppressionSuppressed:         0,
			SuppressionQuarantineRequired: 1,
			NewSurvivors:                  1,
			LongStandingSurvivors:         1,
			PlatformSensitiveSurvivors:    1,
			NonProgressTimeouts:           1,
			TimeoutRiskStats:              map[string]int{"high": 1},
			EquivalentRiskStats:           map[string]int{"high": 1, "medium": 2},
			SemanticGroupStats:            map[string]int{"sort comparator boundary": 2},
			DenominatorHealth: engine.DenominatorHealth{
				Generated:        5,
				Covered:          4,
				Executed:         4,
				Effective:        3,
				ScoreDenominator: 4,
				Killed:           1,
				Survived:         2,
				NotCovered:       1,
				TimedOut:         1,
				Healthy:          false,
				Warnings:         []string{"timed_out_exceeds_effective"},
			},
			MutatorStats: map[string]engine.MutatorStat{
				"conditionals-boundary": {Total: 2, Survived: 2, Recommendation: "fast-ci", EquivalentRisk: "high"},
				"logical":               {Total: 1, Killed: 1, Recommendation: "conservative", EquivalentRisk: "medium"},
				"numeric-literals":      {Total: 1, NotCovered: 1, Recommendation: "nightly", EquivalentRisk: "medium"},
			},
		},
		Environment: engine.Environment{
			OS:              "windows",
			Arch:            "amd64",
			GoVersion:       "go1.25.6",
			ToolVersion:     "v0.2.0",
			WorkingDir:      `C:\workspace\cervo-mutants`,
			TempDir:         `C:\Users\tester\AppData\Local\Temp\cervomut-123`,
			TempRoot:        `C:\Users\tester\AppData\Local\CervoMutants\tmp`,
			Isolation:       "overlay",
			Workers:         2,
			TestTimeout:     "30s",
			Budget:          "10m0s",
			GoFlags:         "-count=1 -p=1",
			GoMaxProcs:      "2",
			GoMemLimit:      "2GiB",
			CI:              "github-actions",
			WindowsOneDrive: true,
			Warnings:        []string{"workspace path contains spaces", "temp root overridden away from OneDrive-backed temp"},
			Extra:           map[string]string{"campaign": "nightly"},
		},
		Slice: engine.SliceMetadata{
			Enabled:              true,
			SliceBy:              "package",
			ShardIndex:           2,
			ShardCount:           4,
			GroupCount:           12,
			SelectedGroups:       3,
			MaxFilesPerRun:       5,
			SelectedFiles:        2,
			MaxMutantsPerPackage: 10,
			SelectedMutants:      4,
		},
		Checkpoint: engine.Checkpoint{
			Fingerprint:         "abc123checkpoint",
			Mutants:             4,
			IncludesFileDigests: true,
			Reason:              "final",
		},
		Thresholds: map[string]any{
			"fail_under": 75,
			"failed":     false,
		},
		Baseline: engine.BaselineComparison{
			Enabled:       true,
			Regression:    true,
			NewSurvivors:  []string{"pkg/review.go:19:conditionals-boundary:123"},
			PreviousScore: 66.67,
			CurrentScore:  50,
		},
		Cache: engine.CacheStats{
			Hits:   2,
			Misses: 2,
		},
		Quarantine: engine.QuarantineStats{
			Active:  1,
			Expired: 0,
		},
		History: engine.HistoryStats{
			Enabled:                true,
			Path:                   ".cervomut/history.json",
			LoadedMutants:          3,
			UpdatedMutants:         4,
			NewSurvivors:           1,
			LongStandingSurvivors:  1,
			OperatorUsefulSurvivor: map[string]float64{"conditionals-boundary": 0.5, "logical": 0.2},
		},
		Mutants: []engine.MutantResult{
			{
				MutantID:        "pkg/review.go:19:conditionals-boundary:123456789abc",
				Status:          engine.StatusSurvived,
				Duration:        3500 * time.Millisecond,
				FailureKind:     "",
				MemoryPeakBytes: 1048576,
				TestCommand:     []string{"go", "test", "./pkg/review"},
				StatusReason:    "tests passed with mutant applied",
				SelectionReason: "coverage profile matched mutant line",
				CoverageSource:  "coverage-mode",
				Output:          "ok",
				Mutant: engine.Mutant{
					ID:                  "pkg/review.go:19:conditionals-boundary:123456789abc",
					Module:              "fixture",
					Package:             "./pkg/review",
					File:                "pkg/review.go",
					Line:                19,
					Function:            "ReviewIDs",
					Operator:            "conditionals-boundary",
					Original:            "<",
					Mutated:             "<=",
					StartOffset:         120,
					EndOffset:           121,
					Diff:                "--- pkg/review.go\n+++ pkg/review.go\n@@\n-return ids[i] < ids[j]\n+return ids[i] <= ids[j]\n",
					Fingerprint:         "123456789abcdef0",
					Hint:                "Add a strict ordering assertion.",
					Description:         "Changed < to <= in ReviewIDs.",
					NearbyTests:         []string{"pkg/review_test.go", "pkg/review_sort_test.go"},
					EquivalentRisk:      "high",
					Recommendation:      "fast-ci",
					CompileErrorRisk:    "low",
					SemanticTags:        []string{"equivalence-risk-group", "sort-comparator-boundary"},
					SemanticGroup:       "sort-boundary:pkg/review.go:19",
					GroupLabel:          "sort comparator boundary",
					GroupReason:         "Boundary mutations inside sort comparator closures often collapse into one review decision.",
					SuggestedSkipReason: "review once for this semantic group before treating each survivor independently",
					SuppressionAudit: []engine.SuppressionAudit{{
						Name:          "equivalence-audit",
						Action:        "lower-priority",
						Reason:        "semantic grouping",
						EvidenceLevel: "suspected",
						ReviewerCount: 1,
					}},
				},
				SurvivorRank:       1,
				RankScore:          140,
				RankReason:         "risk=high recommendation=fast-ci nearby_tests=2",
				Actionability:      "high",
				SuggestedTestScope: "./pkg/review",
				TestRecommendation: &engine.TestRecommendation{
					Priority:            "high",
					Strategy:            "tighten-branch-assertions",
					Summary:             "Start with `pkg/review_sort_test.go` while this survivor is new: Add a strict ordering assertion.",
					CandidateTests:      []string{"pkg/review_sort_test.go", "pkg/review_test.go"},
					SuggestedAssertions: []string{"Add a strict ordering assertion."},
					Rationale: []string{
						"operator=conditionals-boundary -> branch and boundary assertions usually kill this operator family",
						"coverage_source=coverage-mode -> the mutant was matched by coverage data, so the next test should usually be an assertion upgrade",
						"nearby_tests=2 -> start with pkg/review_sort_test.go",
						"history=new_survivor -> fix the closest nearby test while the regression is still fresh",
						"semantic_group=sort comparator boundary -> one good review can collapse 2 similar survivors",
					},
				},
				SuggestedSkipReason: "review once for this semantic group before treating each survivor independently",
				NearestTests:        []string{"pkg/review_test.go", "pkg/review_sort_test.go"},
				SemanticGroupSize:   2,
				PreviousStatus:      engine.StatusKilled,
				FirstSeen:           "2026-06-10T00:00:00Z",
				LastSeen:            "2026-06-17T00:00:00Z",
				SurvivorAgeRuns:     1,
				HistoryStatus:       "new_survivor",
				OperatorYield:       0.5,
			},
			{
				MutantID:        "pkg/review.go:27:logical:abc123def456",
				Status:          engine.StatusKilled,
				Duration:        240 * time.Millisecond,
				MemoryPeakBytes: 786432,
				TestCommand:     []string{"go", "test", "./pkg/review", "-run", "TestReviewIDs"},
				StatusReason:    "test TestReviewIDs failed",
				SelectionReason: "package selected from mutant file",
				CoverageSource:  "package-mode-prefilter",
				Output:          "FAIL\tpkg/review",
				Mutant: engine.Mutant{
					ID:               "pkg/review.go:27:logical:abc123def456",
					Module:           "fixture",
					Package:          "./pkg/review",
					File:             "pkg/review.go",
					Line:             27,
					Function:         "ReviewIDs",
					Operator:         "logical",
					Original:         "&&",
					Mutated:          "||",
					StartOffset:      222,
					EndOffset:        224,
					Diff:             "--- pkg/review.go\n+++ pkg/review.go\n@@\n-if ready && strict {\n+if ready || strict {\n",
					Fingerprint:      "abc123def4567890",
					Hint:             "Assert combined-branch behavior.",
					Description:      "Changed && to || in ReviewIDs.",
					NearbyTests:      []string{"pkg/review_test.go"},
					EquivalentRisk:   "medium",
					Recommendation:   "conservative",
					CompileErrorRisk: "low",
				},
				PreviousStatus:  engine.StatusSurvived,
				FirstSeen:       "2026-06-01T00:00:00Z",
				LastSeen:        "2026-06-17T00:00:00Z",
				SurvivorAgeRuns: 6,
				HistoryStatus:   "resolved_survivor",
				OperatorYield:   0.2,
			},
			{
				MutantID:        "pkg/fs.go:41:numeric-literals:ff00ff00aa11",
				Status:          engine.StatusSurvived,
				Duration:        1100 * time.Millisecond,
				MemoryPeakBytes: 512000,
				TestCommand:     []string{"go", "test", "./pkg/fs"},
				StatusReason:    "tests passed on Windows permission-mode mutation",
				SelectionReason: "coverage profile matched mutant file",
				CoverageSource:  "coverage-mode-file-fallback",
				Output:          "ok",
				Mutant: engine.Mutant{
					ID:                  "pkg/fs.go:41:numeric-literals:ff00ff00aa11",
					Module:              "fixture",
					Package:             "./pkg/fs",
					File:                "pkg/fs.go",
					Line:                41,
					Function:            "EnsureDir",
					Operator:            "numeric-literals",
					Original:            "0o755",
					Mutated:             "0",
					StartOffset:         88,
					EndOffset:           93,
					Diff:                "--- pkg/fs.go\n+++ pkg/fs.go\n@@\n-os.MkdirAll(path, 0o755)\n+os.MkdirAll(path, 0)\n",
					Fingerprint:         "ff00ff00aa112233",
					Hint:                "Review on the target OS before treating permission-mode survivors as actionable.",
					Description:         "Changed 0o755 to 0 in EnsureDir.",
					NearbyTests:         []string{"pkg/fs_test.go"},
					EquivalentRisk:      "medium",
					Recommendation:      "nightly",
					CompileErrorRisk:    "low",
					SemanticTags:        []string{"platform-sensitive"},
					PlatformSensitive:   true,
					SuggestedSkipReason: "review on the target OS before treating this permission-mode mutant as actionable",
				},
				SurvivorRank:       2,
				RankScore:          95,
				RankReason:         "risk=medium recommendation=nightly platform-sensitive windows",
				Actionability:      "medium",
				SuggestedTestScope: "./pkg/fs",
				TestRecommendation: &engine.TestRecommendation{
					Priority:            "medium",
					Strategy:            "review-platform-contract",
					Summary:             "Promote `pkg/fs_test.go` into a named regression: Review on the target OS before treating permission-mode survivors as actionable.",
					CandidateTests:      []string{"pkg/fs_test.go"},
					SuggestedAssertions: []string{"Review on the target OS before treating permission-mode survivors as actionable.", "Keep the assertion file-local; fallback coverage usually means the package-level run is too broad.", "Exercise the permission-mode behavior on Windows explicitly before treating the survivor as actionable.", "Give the fix a dedicated regression case name so future runs expose the contract immediately."},
					Rationale: []string{
						"operator=numeric-literals -> exact value checks usually kill this operator family",
						"coverage_source=coverage-mode-file-fallback -> the mutant was reached through fallback coverage rather than a tight file-level match",
						"nearby_tests=1 -> start with pkg/fs_test.go",
						"history=long_standing_survivor -> this mutant has survived 7 runs; convert the next fix into a named regression",
						"goos=windows -> permission-mode mutations need target-OS verification before escalation",
					},
				},
				SuggestedSkipReason: "review on the target OS before treating this permission-mode mutant as actionable",
				NearestTests:        []string{"pkg/fs_test.go"},
				SemanticGroupSize:   1,
				PreviousStatus:      engine.StatusSurvived,
				FirstSeen:           "2026-05-20T00:00:00Z",
				LastSeen:            "2026-06-17T00:00:00Z",
				SurvivorAgeRuns:     7,
				HistoryStatus:       "long_standing_survivor",
				OperatorYield:       0.4,
			},
			{
				MutantID:        "pkg/loop.go:12:inc-dec:998877665544",
				Status:          engine.StatusTimedOut,
				FailureKind:     "non_progress_loop",
				Duration:        2800 * time.Millisecond,
				MemoryPeakBytes: 1280000,
				TestCommand:     []string{"go", "test", "./pkg/loop"},
				StatusReason:    "loop variable stopped making progress",
				SelectionReason: "package selected from mutant file",
				CoverageSource:  "coverage-mode",
				Output:          "timed out after 3s",
				Mutant: engine.Mutant{
					ID:                  "pkg/loop.go:12:inc-dec:998877665544",
					Module:              "fixture",
					Package:             "./pkg/loop",
					File:                "pkg/loop.go",
					Line:                12,
					Function:            "Walk",
					Operator:            "inc-dec",
					Original:            "i++",
					Mutated:             "i--",
					StartOffset:         44,
					EndOffset:           47,
					Diff:                "--- pkg/loop.go\n+++ pkg/loop.go\n@@\n-for i := 0; i < len(xs); i++ {\n+for i := 0; i < len(xs); i-- {\n",
					Fingerprint:         "9988776655443322",
					Hint:                "Check loop termination behavior.",
					Description:         "Changed i++ to i-- in Walk.",
					NearbyTests:         []string{"pkg/loop_test.go"},
					EquivalentRisk:      "medium",
					Recommendation:      "review",
					CompileErrorRisk:    "low",
					SemanticTags:        []string{"non-progress-loop-risk"},
					NonProgressRisk:     "high",
					SuggestedSkipReason: "reviewed-skip or quarantine if timeout confirms a non-progress loop",
				},
				SuggestedSkipReason: "reviewed-skip or quarantine if timeout confirms a non-progress loop",
			},
			{
				MutantID:        "pkg/fs.go:50:numeric-literals:001122334455",
				Status:          engine.StatusNotCovered,
				Duration:        100 * time.Millisecond,
				MemoryPeakBytes: 128000,
				TestCommand:     []string{"go", "test", "./pkg/fs"},
				StatusReason:    "coverage profile did not match mutant file",
				SelectionReason: "coverage prefilter did not match mutant file",
				CoverageSource:  "package-mode-prefilter",
				Output:          "",
				Mutant: engine.Mutant{
					ID:               "pkg/fs.go:50:numeric-literals:001122334455",
					Module:           "fixture",
					Package:          "./pkg/fs",
					File:             "pkg/fs.go",
					Line:             50,
					Function:         "EnsureDir",
					Operator:         "numeric-literals",
					Original:         "2",
					Mutated:          "1",
					StartOffset:      144,
					EndOffset:        145,
					Diff:             "--- pkg/fs.go\n+++ pkg/fs.go\n@@\n-return 2\n+return 1\n",
					Fingerprint:      "0011223344556677",
					Hint:             "Add direct assertions for the uncovered branch.",
					Description:      "Changed 2 to 1 in EnsureDir.",
					NearbyTests:      []string{"pkg/fs_test.go"},
					EquivalentRisk:   "medium",
					Recommendation:   "nightly",
					CompileErrorRisk: "low",
				},
			},
		},
	}
}

func assertGoldenBytes(t *testing.T, path string, got []byte) {
	t.Helper()
	got = normalizeGoldenNewlines(got)
	if os.Getenv("UPDATE_GOLDENS") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create golden dir: %v", err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("write golden file: %v", err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden file %s: %v", path, err)
	}
	want = normalizeGoldenNewlines(want)
	if string(got) != string(want) {
		t.Fatalf("golden mismatch for %s\nwant:\n%s\n\ngot:\n%s", path, want, got)
	}
}

func normalizeGoldenNewlines(data []byte) []byte {
	return []byte(strings.ReplaceAll(string(data), "\r\n", "\n"))
}

func TestGoldenFixtureJSONIsStable(t *testing.T) {
	run := goldenReportFixture()
	data, err := json.Marshal(run)
	if err != nil {
		t.Fatalf("fixture should be valid JSON: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("fixture JSON should not be empty")
	}
}
