package engine

import (
	"github.com/cervantesh/cervo-mutants/pkg/config"
	"github.com/cervantesh/cervo-mutants/pkg/mutator"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCorruptCacheAndBaselineBranches(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Defaults()
	cfg.Cache.Path = filepath.Join(dir, "cache")
	cfg.Baseline.Path = filepath.Join(dir, "baseline.json")
	e := New(cfg)
	session := e.newRunSession()
	if err := os.MkdirAll(cfg.Cache.Path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.Cache.Path, "bad.json"), []byte("{bad json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := session.getCached("bad"); err == nil {
		t.Fatal("getCached accepted malformed JSON")
	}
	if err := os.WriteFile(cfg.Baseline.Path, []byte("{bad json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := session.loadBaseline(); err == nil {
		t.Fatal("loadBaseline accepted malformed JSON")
	}
}

func TestNewWithOptionsUsesCustomMutantGeneratorAndSuppressionEvaluator(t *testing.T) {
	dir := writeFixture(t)
	cfg := config.Defaults()
	isolateArtifacts(&cfg, dir)
	customGenerator := mutator.GeneratorFunc(func(pkg, filename string, src []byte, profile string) ([]mutator.Mutant, error) {
		return []mutator.Mutant{{
			File:             filename,
			Line:             4,
			Operator:         "custom-op",
			Original:         ">=",
			Mutated:          "<",
			StartOffset:      45,
			EndOffset:        47,
			Diff:             "--- calc.go\n+++ calc.go\n@@\n-return n >= 0\n+return n < 0\n",
			Hint:             "custom hint",
			Description:      "custom mutation",
			EquivalentRisk:   "low",
			Recommendation:   "custom",
			CompileErrorRisk: "low",
		}}, nil
	})
	customSuppression := SuppressionEvaluatorFunc(func(mutant mutator.Mutant) []SuppressionAudit {
		return []SuppressionAudit{{Name: "custom-suppression", Action: config.SuppressionReportOnly, Reason: "custom audit", EvidenceLevel: "heuristic"}}
	})
	e := NewWithOptions(cfg,
		WithMutantGenerator(customGenerator),
		WithSuppressionEvaluator(ChainSuppressionEvaluators(DefaultSuppressionEvaluator(cfg), customSuppression)),
	)

	mutants, err := e.discoverMutants([]string{dir})
	if err != nil {
		t.Fatalf("discoverMutants returned error: %v", err)
	}
	if len(mutants) == 0 {
		t.Fatal("discoverMutants returned no mutants")
	}
	if mutants[0].Operator != "custom-op" {
		t.Fatalf("custom generator was not used: %+v", mutants[0])
	}
	if len(mutants[0].SuppressionAudit) != 1 || mutants[0].SuppressionAudit[0].Name != "custom-suppression" {
		t.Fatalf("custom suppression evaluator was not used: %+v", mutants[0].SuppressionAudit)
	}
}

func TestEngineApplySurvivorRankingUsesCustomRanker(t *testing.T) {
	cfg := config.Defaults()
	e := NewWithOptions(cfg, WithSurvivorRanker(SurvivorRankerFunc(func(goos string, results []MutantResult) []SurvivorRanking {
		return []SurvivorRanking{{
			MutantID:            "custom",
			SurvivorRank:        9,
			RankScore:           211.5,
			RankReason:          "custom ranking",
			Actionability:       "high",
			SuggestedTestScope:  "./custom",
			SuggestedSkipReason: "custom skip",
			SemanticGroupSize:   3,
			NearestTests:        []string{"pkg/custom_test.go"},
			TestRecommendation: &TestRecommendation{
				Priority:       "high",
				Strategy:       "custom-strategy",
				Summary:        "custom recommendation",
				CandidateTests: []string{"pkg/custom_test.go"},
			},
		}}
	})))
	results := []MutantResult{
		{MutantID: "custom", Status: StatusSurvived, Mutant: Mutant{ID: "custom"}},
		{MutantID: "other", Status: StatusKilled, Mutant: Mutant{ID: "other"}},
	}

	e.applySurvivorRanking(results)

	if results[0].SurvivorRank != 9 || results[0].RankScore != 211.5 || results[0].RankReason != "custom ranking" {
		t.Fatalf("custom ranker did not apply ranking metadata: %+v", results[0])
	}
	if results[0].SuggestedTestScope != "./custom" || results[0].SuggestedSkipReason != "custom skip" || results[0].SemanticGroupSize != 3 {
		t.Fatalf("custom ranker did not apply survivor context: %+v", results[0])
	}
	if results[0].TestRecommendation == nil || results[0].TestRecommendation.Strategy != "custom-strategy" {
		t.Fatalf("custom ranker did not attach test recommendation: %+v", results[0].TestRecommendation)
	}
	if results[1].SurvivorRank != 0 {
		t.Fatalf("non-ranked mutants should remain unchanged: %+v", results[1])
	}
}

func TestEngineHelperBranches(t *testing.T) {
	assertGlobBranches(t)
	assertClassificationBranches(t)
	assertEnvironmentBranches(t)
	assertEngineTargetBranches(t)
}

func TestOwnershipRouteMatchesPackageFileAndDefault(t *testing.T) {
	cfg := config.Defaults()
	cfg.Ownership.Default = config.OwnershipTarget{Owner: "default-owner", Team: "default-team"}
	cfg.Ownership.Rules = []config.OwnershipRule{
		{Name: "pkg-fs", Package: "./pkg/fs", Owner: "fs-owner", Team: "platform"},
		{Name: "cmd-glob", File: "cmd/**/*.go", Owner: "cli-owner", Contact: "@cli"},
	}
	e := New(cfg)
	dir := t.TempDir()

	route := e.ownershipRoute(dir, "./pkg/fs", filepath.Join(dir, "pkg", "fs", "fs.go"))
	if route == nil || route.Owner != "fs-owner" || route.Team != "platform" || route.Rule != "pkg-fs" {
		t.Fatalf("package ownership route mismatch: %+v", route)
	}

	route = e.ownershipRoute(dir, "./cmd/cervomut", filepath.Join(dir, "cmd", "cervomut", "main.go"))
	if route == nil || route.Owner != "cli-owner" || route.Contact != "@cli" || route.Rule != "cmd-glob" {
		t.Fatalf("file ownership route mismatch: %+v", route)
	}

	route = e.ownershipRoute(dir, "./pkg/other", filepath.Join(dir, "pkg", "other", "other.go"))
	if route == nil || route.Owner != "default-owner" || route.Rule != "default" {
		t.Fatalf("default ownership route mismatch: %+v", route)
	}
}

func TestLoadStoresAndPriorityHelpers(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Defaults()
	cfg.Cache.Path = filepath.Join(dir, "cache")
	cfg.Baseline.Path = filepath.Join(dir, "baseline.json")
	cfg.Quarantine.Path = filepath.Join(dir, "quarantine.json")
	cfg.Quarantine.FailOnExpired = false
	e := New(cfg)

	assertCacheStore(t, e)
	assertBaselineStore(t, e, cfg.Baseline.Path)
	assertQuarantineLoad(t, e, cfg.Quarantine.Path)
	assertPriorityHelpers(t)
}
