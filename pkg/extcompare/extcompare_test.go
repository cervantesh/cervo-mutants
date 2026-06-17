package extcompare

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCervoReportNormalizesMetrics(t *testing.T) {
	path := writeJSON(t, `{
  "summary": {"total": 3, "killed": 1, "survived": 1, "not_covered": 1, "score": 50}
}`)

	result, err := ParseCervo(path)
	if err != nil {
		t.Fatalf("ParseCervo returned error: %v", err)
	}
	if result.Tool != "cervo-mutants" || result.Total != 3 || result.Killed != 1 || result.Survived != 1 || result.NotCovered != 1 || result.Score != 50 {
		t.Fatalf("unexpected normalized result: %+v", result)
	}
}

func TestParseGremlinsReportNormalizesMetrics(t *testing.T) {
	path := writeJSON(t, `{
  "mutants_total": 4,
  "mutants_killed": 2,
  "mutants_lived": 1,
  "mutants_not_covered": 1,
  "mutants_not_viable": 0,
  "test_efficacy": 66.6667
}`)

	result, err := ParseGremlins(path)
	if err != nil {
		t.Fatalf("ParseGremlins returned error: %v", err)
	}
	if result.Tool != "gremlins" || result.Total != 4 || result.Killed != 2 || result.Survived != 1 || result.NotCovered != 1 {
		t.Fatalf("unexpected normalized result: %+v", result)
	}
	if result.Status != "ok" || result.TestEfficacy == 0 || result.DenominatorHealth.Effective != 3 {
		t.Fatalf("expected status, efficacy, and denominator health: %+v", result)
	}
}

func TestParseGremlinsReportClassifiesAllTimedOutAndPoorDenominatorHealth(t *testing.T) {
	path := writeJSON(t, `{
  "mutants_total": 0,
  "mutants_killed": 0,
  "mutants_lived": 0,
  "mutants_not_covered": 5,
  "test_efficacy": 0,
  "files": [
    {"mutations": [{"status": "TIMED OUT"}, {"status": "TIMED OUT"}]}
  ]
}`)

	result, err := ParseGremlins(path)
	if err != nil {
		t.Fatalf("ParseGremlins returned error: %v", err)
	}
	if result.Status != "all_timed_out" || result.TimedOut != 2 {
		t.Fatalf("unexpected Gremlins status: %+v", result)
	}
}

func TestParseGremlinsReportClassifiesNotCoveredOnly(t *testing.T) {
	path := writeJSON(t, `{"mutants_total":0,"mutants_killed":0,"mutants_lived":0,"mutants_not_covered":3}`)
	result, err := ParseGremlins(path)
	if err != nil {
		t.Fatalf("ParseGremlins returned error: %v", err)
	}
	if result.Status != "not_covered_only" || len(result.Notes) == 0 {
		t.Fatalf("unexpected Gremlins not-covered status: %+v", result)
	}
}

func TestNormalizeGremlinsPackageRootTargetMarksNotComparable(t *testing.T) {
	effective, notComparable := NormalizeGremlinsTarget("./...", "gremlins-package-root")
	if effective != "." || !notComparable {
		t.Fatalf("effective=%q notComparable=%t, want . true", effective, notComparable)
	}
	result := ApplyTargetMode(ToolResult{Tool: "gremlins", Status: "ok"}, "./...", effective, "gremlins-package-root", notComparable)
	if result.Target != "./..." || result.EffectiveTarget != "." || result.TargetMode != "gremlins-package-root" || !result.NotComparable || len(result.Notes) == 0 {
		t.Fatalf("target metadata not applied: %+v", result)
	}
}

func TestBuildComparabilitySeparatesToolApplesFromManifestEquivalence(t *testing.T) {
	cervo := ApplyTargetMode(ToolResult{Tool: "cervo-mutants"}, "./...", ".", "package-root", true)
	gremlins := ApplyTargetMode(ToolResult{Tool: "gremlins"}, "./...", ".", "package-root", true)

	comp := BuildComparability([]ToolResult{cervo, gremlins})
	if !comp.ApplesToApples {
		t.Fatalf("expected package-root run to be apples-to-apples between tools: %+v", comp)
	}
	if comp.ManifestEquivalent {
		t.Fatalf("expected package-root run to be marked different from manifest: %+v", comp)
	}
	if strings.Join(comp.Warnings, ",") != "effective_target_differs_from_manifest" {
		t.Fatalf("unexpected warnings: %+v", comp)
	}
}

func TestBuildComparabilityDetectsEffectiveTargetMismatch(t *testing.T) {
	cervo := ApplyTargetMode(ToolResult{Tool: "cervo-mutants"}, "./...", "./...", "manifest", false)
	gremlins := ApplyTargetMode(ToolResult{Tool: "gremlins"}, "./...", ".", "gremlins-package-root", true)

	comp := BuildComparability([]ToolResult{cervo, gremlins})
	if comp.ApplesToApples {
		t.Fatalf("expected mismatched targets to be non-comparable: %+v", comp)
	}
	warnings := strings.Join(comp.Warnings, ",")
	for _, want := range []string{"effective_target_mismatch", "target_mode_mismatch", "effective_target_differs_from_manifest"} {
		if !strings.Contains(warnings, want) {
			t.Fatalf("missing warning %q: %+v", want, comp)
		}
	}
}

func TestBuildComparabilityDetectsMissingTargetMetadataAndDedupesWarnings(t *testing.T) {
	comp := BuildComparability([]ToolResult{{Tool: "a"}, {Tool: "b"}})
	if comp.ApplesToApples || comp.ManifestEquivalent || strings.Count(strings.Join(comp.Warnings, ","), "target_metadata_missing") != 1 {
		t.Fatalf("unexpected missing-target comparability: %+v", comp)
	}
}

func TestDenominatorHealthWarnsWhenScoreHidesTimeouts(t *testing.T) {
	result := ToolResult{Tool: "gremlins", Status: "ok", Total: 3, Killed: 3, TimedOut: 1244, NotCovered: 37, Score: 100}
	result.DenominatorHealth = denominatorHealth(result)
	warnings := strings.Join(result.DenominatorHealth.Warnings, ",")
	for _, want := range []string{"timed_out_exceeds_effective", "not_covered_exceeds_effective", "high_score_poor_denominator_health"} {
		if !strings.Contains(warnings, want) {
			t.Fatalf("warnings missing %q: %+v", want, result.DenominatorHealth)
		}
	}
}

func TestParseGomuReportAcceptsJSONStatusResults(t *testing.T) {
	path := writeJSON(t, `{
  "totalMutants": 5,
  "results": [
    {"status": "KILLED"},
    {"status": "KILLED"},
    {"status": "SURVIVED"},
    {"status": "ERROR"},
    {"status": "NOT_VIABLE"}
  ]
}`)

	result, err := ParseGomu(path)
	if err != nil {
		t.Fatalf("ParseGomu returned error: %v", err)
	}
	if result.Tool != "gomu" || result.Total != 5 || result.Killed != 2 || result.Survived != 1 || result.Errors != 1 || result.NotViable != 1 {
		t.Fatalf("unexpected normalized result: %+v", result)
	}
}

func TestParseGoMutestingReportAcceptsJSONStats(t *testing.T) {
	path := writeJSON(t, `{
  "stats": {
    "totalMutantsCount": 4,
    "killedCount": 3,
    "escapedCount": 1,
    "notCoveredCount": 0,
    "errorCount": 0,
    "skippedCount": 0,
    "timeOutCount": 0,
    "msi": 0.75
  }
}`)

	result, err := ParseGoMutesting(path)
	if err != nil {
		t.Fatalf("ParseGoMutesting returned error: %v", err)
	}
	if result.Tool != "go-mutesting" || result.Total != 4 || result.Killed != 3 || result.Survived != 1 || result.Score != 75 {
		t.Fatalf("unexpected normalized result: %+v", result)
	}
}

func TestTextParsersAndTargetApplication(t *testing.T) {
	gomu := writeText(t, "mutants: 6\nkilled: 3\nsurvived: 1\nnot_covered: 2\ntimed_out: 4\nscore: 75\n")
	gomuResult, err := ParseGomu(gomu)
	if err != nil {
		t.Fatalf("ParseGomu text returned error: %v", err)
	}
	if gomuResult.Total != 6 || gomuResult.Killed != 3 || gomuResult.Survived != 1 || gomuResult.NotCovered != 2 || gomuResult.TimedOut != 4 || gomuResult.Score != 75 {
		t.Fatalf("unexpected gomu text metrics: %+v", gomuResult)
	}

	goMutesting := writeText(t, "The mutation score is 66.67%: 2 killed, 1 survived, 3 total, 5 timed out")
	goMutestingResult, err := ParseGoMutesting(goMutesting)
	if err != nil {
		t.Fatalf("ParseGoMutesting text returned error: %v", err)
	}
	if goMutestingResult.Total != 3 || goMutestingResult.Killed != 2 || goMutestingResult.Survived != 1 || goMutestingResult.TimedOut != 5 {
		t.Fatalf("unexpected go-mutesting text metrics: %+v", goMutestingResult)
	}

	applied := ApplyTarget(ToolResult{Tool: "x"}, "./...", ".", true)
	if applied.Target != "./..." || applied.EffectiveTarget != "." || applied.TargetMode != "manifest" || !applied.NotComparable {
		t.Fatalf("target not applied: %+v", applied)
	}
}

func TestGremlinsNoResultsAndIntFloatFields(t *testing.T) {
	path := writeJSON(t, `{"total":"0","killed":"0","survived":"0","score":"0"}`)
	result, err := ParseGremlins(path)
	if err != nil {
		t.Fatalf("ParseGremlins returned error: %v", err)
	}
	if result.Status != "no_results" {
		t.Fatalf("status = %q, want no_results", result.Status)
	}
}

func TestParserErrorAndFallbackBranches(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.json")
	if _, err := ParseCervo(missing); err == nil {
		t.Fatal("ParseCervo accepted missing file")
	}
	bad := writeText(t, "{bad json")
	if _, err := ParseGremlins(bad); err == nil {
		t.Fatal("ParseGremlins accepted malformed JSON")
	}
	gomu := writeText(t, "killed: 2\nsurvived: 2\n")
	gomuResult, err := ParseGomu(gomu)
	if err != nil {
		t.Fatalf("ParseGomu fallback returned error: %v", err)
	}
	if gomuResult.Score != 50 || gomuResult.MutationCoverage != 100 {
		t.Fatalf("unexpected gomu fallback metrics: %+v", gomuResult)
	}
	goMutesting := writeText(t, "2 killed, 2 survived, 4 total")
	goMutestingResult, err := ParseGoMutesting(goMutesting)
	if err != nil {
		t.Fatalf("ParseGoMutesting fallback returned error: %v", err)
	}
	if goMutestingResult.Score != 50 {
		t.Fatalf("go-mutesting fallback score = %.2f, want 50", goMutestingResult.Score)
	}
	if regexpMatch(`missing (\d+)`, "none") != "" {
		t.Fatal("regexpMatch should return empty string for no match")
	}
}

func TestWriteStudy(t *testing.T) {
	out := filepath.Join(t.TempDir(), "study.json")
	if err := Write(out, []ToolResult{{Tool: "cervo-mutants", Completed: true, Total: 1}}); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("study missing: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("study file is empty")
	}
}

func writeJSON(t *testing.T, text string) string {
	t.Helper()
	return writeText(t, text)
}

func writeText(t *testing.T, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "report.txt")
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
