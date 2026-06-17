package mutator

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestConservativeFastMutatorsStayHighSignal(t *testing.T) {
	src := `package sample

func Check(n int, ready bool, p *int) bool {
	if n < 1 && ready && p == nil {
		return n + 1 > 0
	}
	return false
}
`

	mutants, err := Generate("sample", "sample.go", []byte(src), ProfileConservativeFast)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	operators := operatorSet(mutants)
	for _, want := range []string{"conditionals-negation", "conditionals-boundary", "arithmetic-basic"} {
		if !operators[want] {
			t.Fatalf("conservative-fast missing %s: %+v", want, mutants)
		}
	}
	for _, noisy := range []string{"logical", "boolean-literals", "nil-checks"} {
		if operators[noisy] {
			t.Fatalf("conservative-fast generated noisy operator %s: %+v", noisy, mutants)
		}
	}
}

func TestGremlinsCompatibleProfileMatchesConservativeFastOperators(t *testing.T) {
	src := `package sample

func Check(n int) bool {
	return n + 1 >= 2
}
`
	fast, err := Generate("sample", "sample.go", []byte(src), ProfileConservativeFast)
	if err != nil {
		t.Fatalf("Generate conservative-fast returned error: %v", err)
	}
	compatible, err := Generate("sample", "sample.go", []byte(src), ProfileGremlinsCompatible)
	if err != nil {
		t.Fatalf("Generate gremlins-compatible returned error: %v", err)
	}
	if len(fast) != len(compatible) {
		t.Fatalf("gremlins-compatible generated %d mutants, want %d", len(compatible), len(fast))
	}
	if operatorSet(fast)["logical"] || operatorSet(compatible)["logical"] {
		t.Fatal("fast profiles should not include logical mutants")
	}
}

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

	assertActionableMutants(t, mutants)
	assertConservativeOperators(t, mutants)
}

func assertActionableMutants(t *testing.T, mutants []Mutant) {
	t.Helper()
	seen := map[string]bool{}
	for _, mutant := range mutants {
		if mutant.ID == "" || mutant.Operator == "" || mutant.Diff == "" || mutant.Line == 0 {
			t.Fatalf("mutant missing actionable fields: %+v", mutant)
		}
		if mutant.Description == "" {
			t.Fatalf("mutant missing description: %+v", mutant)
		}
		if mutant.EquivalentRisk == "" || mutant.Recommendation == "" {
			t.Fatalf("mutant missing governance fields: %+v", mutant)
		}
		if mutant.CompileErrorRisk == "" {
			t.Fatalf("mutant missing compile-error risk: %+v", mutant)
		}
		if seen[mutant.ID] {
			t.Fatalf("duplicate mutant ID: %s", mutant.ID)
		}
		seen[mutant.ID] = true
	}
}

func assertConservativeOperators(t *testing.T, mutants []Mutant) {
	t.Helper()
	foundConditional := false
	foundLogical := false
	foundBoolean := false
	for _, mutant := range mutants {
		foundConditional = foundConditional || mutant.Operator == "conditionals-negation"
		foundLogical = foundLogical || mutant.Operator == "logical"
		foundBoolean = foundBoolean || mutant.Operator == "boolean-literals"
		if strings.Contains(mutant.Diff, "--- sample.go") && strings.Contains(mutant.Diff, "+++ sample.go") {
			continue
		}
		t.Fatalf("mutant diff is not unified enough: %q", mutant.Diff)
	}
	if !foundConditional || !foundLogical || !foundBoolean {
		t.Fatalf("missing expected operators: conditionals=%v logical=%v boolean=%v", foundConditional, foundLogical, foundBoolean)
	}
	if operatorSet(mutants)["nil-checks"] {
		t.Fatalf("conservative should not generate nil-checks after Cobra noise study: %+v", mutants)
	}
}

func TestDefaultProfileAddsNilChecks(t *testing.T) {
	src := `package sample

func Check(p *int, n int) bool {
	n += 2
	n--
	if n == 2 {
		return true
	}
	return p == nil
}
`
	mutants, err := Generate("sample", "sample.go", []byte(src), ProfileDefault)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if !operatorSet(mutants)["nil-checks"] {
		t.Fatalf("default profile missing nil-checks: %+v", mutants)
	}
	if !operatorSet(mutants)["numeric-literals"] || !operatorSet(mutants)["return-bool-literals"] {
		t.Fatalf("default profile missing controlled literal/return operators: %+v", operatorSet(mutants))
	}
	if !operatorSet(mutants)["assignment-arithmetic"] || !operatorSet(mutants)["inc-dec"] {
		t.Fatalf("default profile missing assignment/counter operators: %+v", operatorSet(mutants))
	}
}

func TestAggressiveProfileAddsLiteralAndReturnMutators(t *testing.T) {
	src := `package sample

func Answer() int {
	return 1
}

func Ready() bool {
	return true
}

func Count(xs []int) int {
	for i := 0; i < len(xs); i++ {
		return i
	}
	return 0
}
`

	conservative, err := Generate("sample", "sample.go", []byte(src), ProfileConservative)
	if err != nil {
		t.Fatalf("Generate conservative returned error: %v", err)
	}
	for _, mutant := range conservative {
		if mutant.Operator == "literals" || mutant.Operator == "returns" {
			t.Fatalf("conservative generated aggressive mutant: %+v", mutant)
		}
	}

	aggressive, err := Generate("sample", "sample.go", []byte(src), ProfileAggressive)
	if err != nil {
		t.Fatalf("Generate aggressive returned error: %v", err)
	}
	foundLiteral := false
	foundReturn := false
	foundLoop := false
	foundLen := false
	for _, mutant := range aggressive {
		foundLiteral = foundLiteral || mutant.Operator == "literals"
		foundReturn = foundReturn || mutant.Operator == "returns"
		foundLoop = foundLoop || mutant.Operator == "loop-control"
		foundLen = foundLen || mutant.Operator == "slice-map-len-boundary"
	}
	if !foundLiteral || !foundReturn || !foundLoop || !foundLen {
		t.Fatalf("aggressive profile missing operators: literal=%v return=%v loop=%v len=%v mutants=%+v", foundLiteral, foundReturn, foundLoop, foundLen, aggressive)
	}
}

func TestDefinitionsCarryGovernanceMetadata(t *testing.T) {
	for _, definition := range Definitions() {
		if definition.CompileErrorRisk == "" || definition.Reason == "" {
			t.Fatalf("definition missing governance metadata: %+v", definition)
		}
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
		if strings.HasPrefix(mutant.Operator, "conditionals-") {
			t.Fatalf("conditionals mutant was not ignored: %+v", mutant)
		}
	}

	bad := []byte(strings.Replace(src, ` reason="covered by generated contract"`, "", 1))
	if _, err := ValidateInlineIgnores("sample.go", bad, true); err == nil {
		t.Fatal("ValidateInlineIgnores accepted ignore without reason")
	}
}

func TestInlineIgnoreParserIgnoresStringLiterals(t *testing.T) {
	src := []byte(`package sample

const marker = "cervomut:ignore"
`)
	if _, err := ValidateInlineIgnores("sample.go", src, true); err != nil {
		t.Fatalf("ValidateInlineIgnores treated string literal as directive: %v", err)
	}
}

func TestFormatNodePrintsASTNode(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", `package sample

func Add(a, b int) int { return a + b }
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	text := FormatNode(fset, file.Decls[0])
	if !strings.Contains(text, "func Add") || !strings.Contains(text, "a + b") {
		t.Fatalf("unexpected formatted node: %s", text)
	}
}

func TestSmallMutationHelpersCoverFallbackBranches(t *testing.T) {
	if _, err := Generate("sample", "bad.go", []byte("package sample\nfunc broken("), ""); err == nil {
		t.Fatal("Generate accepted invalid Go")
	}
	if got := boundaryReplacement(token.EQL); got != "" {
		t.Fatalf("boundaryReplacement(EQL) = %q, want empty", got)
	}
	if operatorMatchesIgnore("logical", "conditionals") {
		t.Fatal("conditionals ignore should not match logical")
	}
	if _, err := replaceFirst("abc", "z", "x"); err == nil {
		t.Fatal("replaceFirst accepted missing token")
	}
	if parseInlineIgnoreOperator(`reason="all"`) != "*" {
		t.Fatal("reason-only inline ignore should apply to all operators")
	}
	if parseInlineIgnoreReason(`reason=unquoted`) != "unquoted" {
		t.Fatal("inline ignore should accept unquoted reason fallback")
	}
	if operatorEnabled("unknown", ProfileAggressive) {
		t.Fatal("unknown operator should not be enabled")
	}
	if equivalentRisk("conditionals-boundary") != "high" {
		t.Fatalf("conditionals-boundary equivalent risk = %q, want high", equivalentRisk("conditionals-boundary"))
	}
	if equivalentRisk("unknown") != "unknown" || recommendation("unknown") != "review" || compileErrorRisk("unknown") != "unknown" {
		t.Fatal("unknown operator metadata fallbacks changed")
	}
	if hint("unknown") == "" || description("", "unknown", "a", "b") == "" {
		t.Fatal("fallback hint/description should be populated")
	}
}

func operatorSet(mutants []Mutant) map[string]bool {
	operators := map[string]bool{}
	for _, mutant := range mutants {
		operators[mutant.Operator] = true
	}
	return operators
}
