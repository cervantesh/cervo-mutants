package extcompare

import (
	"math"
	"testing"
	"testing/quick"
)

func FuzzParseKeyValueText(f *testing.F) {
	for _, seed := range []struct {
		tool string
		text string
	}{
		{tool: "gomu", text: "total=4 killed=3 survived=1 score=75"},
		{tool: "go-mutesting", text: "The mutation score is 100.00%: 1 killed, 0 survived, 1 total"},
		{tool: "gremlins", text: "mutants: 0"},
		{tool: "custom", text: ""},
	} {
		f.Add(seed.tool, seed.text)
	}

	f.Fuzz(func(t *testing.T, tool, text string) {
		result := parseKeyValueText(tool, text)
		if result.Tool != tool {
			t.Fatalf("tool = %q, want %q", result.Tool, tool)
		}
		if !result.Completed {
			t.Fatalf("parseKeyValueText should always mark results completed: %+v", result)
		}
		if math.IsNaN(result.Score) || math.IsInf(result.Score, 0) {
			t.Fatalf("score should stay finite: %+v", result)
		}
		if result.DenominatorHealth.Effective < 0 || result.DenominatorHealth.ScoreDenominator < 0 {
			t.Fatalf("denominator health should stay non-negative: %+v", result.DenominatorHealth)
		}
	})
}

func FuzzNormalizeTarget(f *testing.F) {
	for _, seed := range []struct {
		target string
		mode   string
	}{
		{target: "./...", mode: "package-root"},
		{target: "./...", mode: "gremlins-package-root"},
		{target: "./pkg", mode: "manifest"},
		{target: ".", mode: "package-root"},
	} {
		f.Add(seed.target, seed.mode)
	}

	f.Fuzz(func(t *testing.T, target, mode string) {
		effective, notComparable := NormalizeTarget(target, mode)
		gremlinsEffective, gremlinsNotComparable := NormalizeGremlinsTarget(target, mode)
		if effective != gremlinsEffective || notComparable != gremlinsNotComparable {
			t.Fatalf("NormalizeTarget/NormalizeGremlinsTarget diverged: %q/%q => %q,%t vs %q,%t", target, mode, effective, notComparable, gremlinsEffective, gremlinsNotComparable)
		}
		if target == "./..." && (mode == "package-root" || mode == "gremlins-package-root") {
			if effective != "." || !notComparable {
				t.Fatalf("package-root normalization mismatch: target=%q mode=%q effective=%q notComparable=%t", target, mode, effective, notComparable)
			}
			return
		}
		if effective != target || notComparable {
			t.Fatalf("non-package-root normalization should be identity: target=%q mode=%q effective=%q notComparable=%t", target, mode, effective, notComparable)
		}
	})
}

func TestBuildComparabilityQuick(t *testing.T) {
	if err := quick.Check(func(target string) bool {
		if target == "" {
			target = "."
		}
		results := []ToolResult{
			{Tool: "one", Target: target, EffectiveTarget: target, TargetMode: "manifest"},
			{Tool: "two", Target: target, EffectiveTarget: target, TargetMode: "manifest"},
		}
		comp := BuildComparability(results)
		return comp.ApplesToApples && comp.ManifestEquivalent && len(comp.Warnings) == 0
	}, nil); err != nil {
		t.Fatalf("BuildComparability property failed: %v", err)
	}
}
