package mutator

import (
	"strings"
	"testing"
)

func TestConservativeMutatorsGenerateStableActionableMutants(t *testing.T) {
	src := `package sample

func Check(n int, ready bool, p *int) bool {
	if n == 1 && ready && p == nil {
		return true
	}
	return false
}
`

	mutants, err := Generate("sample", "sample.go", []byte(src), ProfileConservative)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if len(mutants) < 4 {
		t.Fatalf("generated %d mutants, want at least 4", len(mutants))
	}

	seen := map[string]bool{}
	for _, mutant := range mutants {
		if mutant.ID == "" || mutant.Operator == "" || mutant.Diff == "" || mutant.Line == 0 {
			t.Fatalf("mutant missing actionable fields: %+v", mutant)
		}
		if seen[mutant.ID] {
			t.Fatalf("duplicate mutant ID: %s", mutant.ID)
		}
		seen[mutant.ID] = true
	}

	foundConditional := false
	foundLogical := false
	foundNil := false
	foundBoolean := false
	for _, mutant := range mutants {
		foundConditional = foundConditional || mutant.Operator == "conditionals"
		foundLogical = foundLogical || mutant.Operator == "logical"
		foundNil = foundNil || mutant.Operator == "nil-checks"
		foundBoolean = foundBoolean || mutant.Operator == "boolean-literals"
		if strings.Contains(mutant.Diff, "--- sample.go") && strings.Contains(mutant.Diff, "+++ sample.go") {
			continue
		}
		t.Fatalf("mutant diff is not unified enough: %q", mutant.Diff)
	}
	if !foundConditional || !foundLogical || !foundNil || !foundBoolean {
		t.Fatalf("missing expected operators: conditionals=%v logical=%v nil=%v boolean=%v", foundConditional, foundLogical, foundNil, foundBoolean)
	}
}

func TestInlineIgnoreRequiresReasonWhenConfigured(t *testing.T) {
	src := `package sample

func Check(n int) bool {
	// cervomut:ignore conditionals reason="covered by generated contract"
	return n == 1
}
`
	mutants, err := Generate("sample", "sample.go", []byte(src), ProfileConservative)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	for _, mutant := range mutants {
		if mutant.Operator == "conditionals" {
			t.Fatalf("conditionals mutant was not ignored: %+v", mutant)
		}
	}

	bad := []byte(strings.Replace(src, ` reason="covered by generated contract"`, "", 1))
	if _, err := ValidateInlineIgnores("sample.go", bad, true); err == nil {
		t.Fatal("ValidateInlineIgnores accepted ignore without reason")
	}
}
