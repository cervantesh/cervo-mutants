package mutator

import (
	"go/ast"
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

func TestAggressiveProfileAddsSemanticTriageMetadata(t *testing.T) {
	src := `package sample

import (
	"os"
	"sort"
)

func Review(path string, xs []int) {
	for i := 0; i < len(xs); i++ {
		_ = xs[i]
	}
	_ = os.MkdirAll(path, 0o755)
	sort.Slice(xs, func(i, j int) bool {
		return xs[i] < xs[j]
	})
	if len(xs) > 0 {
		_ = xs[0]
	}
}
`

	mutants, err := Generate("sample", "sample.go", []byte(src), ProfileAggressive)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	assertSemanticMutant(t, mutants, func(m Mutant) bool {
		return m.Operator == "inc-dec" && m.NonProgressRisk == "high" && containsTag(m.SemanticTags, "non-progress-loop-risk")
	}, "expected high-risk non-progress loop mutant")
	assertSemanticMutant(t, mutants, func(m Mutant) bool {
		return m.PlatformSensitive && m.Operator == "numeric-literals" && m.Original == "0o755"
	}, "expected permission-mode platform-sensitive mutant")
	assertSemanticMutant(t, mutants, func(m Mutant) bool {
		return m.GroupLabel == "sort comparator boundary" && m.SemanticGroup != ""
	}, "expected sort comparator semantic group")
	assertSemanticMutant(t, mutants, func(m Mutant) bool {
		return m.GroupLabel == "len boundary" && m.SemanticGroup != ""
	}, "expected len boundary semantic group")
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

func TestSemanticHelperBranches(t *testing.T) {
	src := `package sample

import (
	"os"
	"sort"
)

func Review(path string, xs []int) {
	for i := 0; i < len(xs); i++ {
		_ = xs[i]
	}
	for j := len(xs) - 1; j >= 0; j-- {
		_ = xs[j]
	}
	for k := 0; len(xs) > k; k += 2 {
		_ = xs[k]
	}
	_ = os.MkdirAll(path, 0o755)
	sort.Slice(xs, func(i, j int) bool {
		return xs[i] < xs[j]
	})
	if len(xs) > 0 {
		_ = xs[0]
	}
}
`

	ctx, file := parseMutationContext(t, src)
	incDecs := findIncDecs(file)
	if len(incDecs) < 2 {
		t.Fatalf("expected at least two inc/dec nodes, got %d", len(incDecs))
	}
	if risk := nonProgressLoopRisk(ctx.parents, incDecs[0], opIncDec); risk != "high" {
		t.Fatalf("nonProgressLoopRisk inc++ = %q, want high", risk)
	}
	if risk := nonProgressLoopRisk(ctx.parents, incDecs[1], opIncDec); risk != "high" {
		t.Fatalf("nonProgressLoopRisk dec-- = %q, want high", risk)
	}

	assign := findAssignByToken(file, token.ADD_ASSIGN)
	if assign == nil {
		t.Fatal("expected += assignment in fixture")
	}
	if risk := nonProgressLoopRisk(ctx.parents, assign, opAssignmentArithmetic); risk != "high" {
		t.Fatalf("nonProgressLoopRisk += = %q, want high", risk)
	}
	if risk := nonProgressLoopRisk(ctx.parents, assign, opIncDec); risk != "" {
		t.Fatalf("nonProgressLoopRisk with wrong operator = %q, want empty", risk)
	}

	permLit := findBasicLit(file, "0o755")
	if permLit == nil {
		t.Fatal("expected permission literal")
	}
	if !permissionModeMutation(ctx.parents, permLit, opNumericLiterals) {
		t.Fatal("permissionModeMutation should flag os.MkdirAll mode literal")
	}
	if permissionModeMutation(ctx.parents, permLit, opBooleanLiterals) {
		t.Fatal("permissionModeMutation should ignore unrelated operator")
	}
	call, argIndex, ok := enclosingCallForNode(ctx.parents, permLit)
	if !ok || argIndex != 1 || selectorName(call.Fun) != "os.MkdirAll" {
		t.Fatalf("enclosingCallForNode mismatch: ok=%v arg=%d call=%v", ok, argIndex, call)
	}

	sortExpr := findBinaryContaining(ctx.fset, file, "xs[i] < xs[j]")
	if sortExpr == nil {
		t.Fatal("expected sort comparator expression")
	}
	group, label, reason := semanticGroup(ctx, sortExpr, opConditionalsBoundary)
	if group == "" || label != "sort comparator boundary" || reason == "" {
		t.Fatalf("sort comparator semantic group mismatch: %q %q %q", group, label, reason)
	}
	sortCall := findCallByName(file, "sort.Slice")
	if sortCall == nil || !isSortComparatorCall(ctx.parents, sortExpr, sortCall) {
		t.Fatal("sort comparator should be detected inside sort.Slice closure")
	}

	lenExpr := findBinaryContaining(ctx.fset, file, "len(xs) > 0")
	if lenExpr == nil {
		t.Fatal("expected len boundary expression")
	}
	group, label, reason = semanticGroup(ctx, lenExpr, opSliceMapLenBoundary)
	if group == "" || label != "len boundary" || reason == "" {
		t.Fatalf("len semantic group mismatch: %q %q %q", group, label, reason)
	}
	if !isLenComparison(lenExpr) || !isLenCall(lenExpr.X) {
		t.Fatalf("len helper mismatch for %s", FormatNode(ctx.fset, lenExpr))
	}
	if direction, ok := loopConditionDirection(lenExpr, "missing"); ok || direction != "" {
		t.Fatalf("loopConditionDirection should not match unknown ident: %q %v", direction, ok)
	}
}

func TestDirectionAndSelectorHelpers(t *testing.T) {
	leftCases := []struct {
		op   token.Token
		want string
		ok   bool
	}{
		{op: token.LSS, want: "ascending", ok: true},
		{op: token.LEQ, want: "ascending", ok: true},
		{op: token.GTR, want: "descending", ok: true},
		{op: token.GEQ, want: "descending", ok: true},
		{op: token.EQL, want: "", ok: false},
	}
	for _, tc := range leftCases {
		got, ok := directionFromComparison(tc.op, true)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("directionFromComparison(%v, true) = (%q, %v), want (%q, %v)", tc.op, got, ok, tc.want, tc.ok)
		}
	}

	rightCases := []struct {
		op   token.Token
		want string
		ok   bool
	}{
		{op: token.GTR, want: "ascending", ok: true},
		{op: token.GEQ, want: "ascending", ok: true},
		{op: token.LSS, want: "descending", ok: true},
		{op: token.LEQ, want: "descending", ok: true},
		{op: token.EQL, want: "", ok: false},
	}
	for _, tc := range rightCases {
		got, ok := directionFromComparison(tc.op, false)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("directionFromComparison(%v, false) = (%q, %v), want (%q, %v)", tc.op, got, ok, tc.want, tc.ok)
		}
	}

	if directionForAssign(token.ADD_ASSIGN) != "ascending" || directionForAssign(token.SUB_ASSIGN) != "descending" || directionForAssign(token.ASSIGN) != "" {
		t.Fatal("directionForAssign branches changed")
	}
	if directionForIncDec(token.INC) != "ascending" || directionForIncDec(token.DEC) != "descending" || directionForIncDec(token.ADD) != "" {
		t.Fatal("directionForIncDec branches changed")
	}
	if oppositeDirection("ascending") != "descending" || oppositeDirection("descending") != "ascending" || oppositeDirection("sideways") != "" {
		t.Fatal("oppositeDirection branches changed")
	}
	if boundaryReplacement(token.LSS) != "<=" || boundaryReplacement(token.LEQ) != "<" || boundaryReplacement(token.GTR) != ">=" || boundaryReplacement(token.GEQ) != ">" {
		t.Fatal("boundaryReplacement branches changed")
	}

	tags := appendUnique([]string{"one"}, "")
	tags = appendUnique(tags, "one")
	tags = appendUnique(tags, "two")
	if strings.Join(tags, ",") != "one,two" {
		t.Fatalf("appendUnique result = %v", tags)
	}

	if selectorName(&ast.Ident{Name: "local"}) != "local" {
		t.Fatal("selectorName should return plain ident names")
	}
	if selectorName(&ast.SelectorExpr{X: &ast.BasicLit{}, Sel: &ast.Ident{Name: "Field"}}) != "Field" {
		t.Fatal("selectorName should fall back to selector field name")
	}
	if selectorName(&ast.BasicLit{}) != "" || identName(&ast.BasicLit{}) != "" {
		t.Fatal("selectorName/identName should ignore unsupported expressions")
	}
	if identName(&ast.Ident{Name: "value"}) != "value" {
		t.Fatal("identName should return identifier names")
	}
}

func parseMutationContext(t *testing.T, src string) (mutationContext, *ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile returned error: %v", err)
	}
	mutants := []Mutant{}
	return mutationContext{
		mutants:  &mutants,
		fset:     fset,
		pkg:      "sample",
		filename: "sample.go",
		src:      []byte(src),
		fn:       "Review",
		profile:  ProfileAggressive,
		parents:  buildParentIndex(file),
	}, file
}

func findIncDecs(root ast.Node) []*ast.IncDecStmt {
	var nodes []*ast.IncDecStmt
	ast.Inspect(root, func(node ast.Node) bool {
		if stmt, ok := node.(*ast.IncDecStmt); ok {
			nodes = append(nodes, stmt)
		}
		return true
	})
	return nodes
}

func findAssignByToken(root ast.Node, tok token.Token) *ast.AssignStmt {
	var found *ast.AssignStmt
	ast.Inspect(root, func(node ast.Node) bool {
		stmt, ok := node.(*ast.AssignStmt)
		if ok && stmt.Tok == tok {
			found = stmt
			return false
		}
		return true
	})
	return found
}

func findBasicLit(root ast.Node, value string) *ast.BasicLit {
	var found *ast.BasicLit
	ast.Inspect(root, func(node ast.Node) bool {
		lit, ok := node.(*ast.BasicLit)
		if ok && lit.Value == value {
			found = lit
			return false
		}
		return true
	})
	return found
}

func findBinaryContaining(fset *token.FileSet, root ast.Node, want string) *ast.BinaryExpr {
	var found *ast.BinaryExpr
	ast.Inspect(root, func(node ast.Node) bool {
		expr, ok := node.(*ast.BinaryExpr)
		if ok && normalizeExpr(FormatNode(fset, expr)) == normalizeExpr(want) {
			found = expr
			return false
		}
		return true
	})
	return found
}

func findCallByName(root ast.Node, want string) *ast.CallExpr {
	var found *ast.CallExpr
	ast.Inspect(root, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if ok && selectorName(call.Fun) == want {
			found = call
			return false
		}
		return true
	})
	return found
}

func TestChainGeneratorsAppendsCustomMutantsAndDedupes(t *testing.T) {
	base := GeneratorFunc(func(pkg, filename string, src []byte, profile string) ([]Mutant, error) {
		return []Mutant{{
			ID:       "builtin-id",
			File:     filename,
			Line:     8,
			Operator: "logical",
			Original: "&&",
			Mutated:  "||",
		}}, nil
	})
	custom := GeneratorFunc(func(pkg, filename string, src []byte, profile string) ([]Mutant, error) {
		return []Mutant{
			{
				File:     filename,
				Line:     8,
				Operator: "logical",
				Original: "&&",
				Mutated:  "||",
			},
			{
				File:     filename,
				Line:     11,
				Operator: "custom-op",
				Original: "1",
				Mutated:  "2",
			},
		}, nil
	})

	mutants, err := ChainGenerators(base, nil, custom).Generate("sample", "sample.go", []byte("package sample"), ProfileDefault)
	if err != nil {
		t.Fatalf("ChainGenerators returned error: %v", err)
	}
	if len(mutants) != 2 {
		t.Fatalf("ChainGenerators produced %d mutants, want 2: %+v", len(mutants), mutants)
	}
	if mutants[1].Operator != "custom-op" {
		t.Fatalf("expected chained custom mutant to be retained, got %+v", mutants)
	}
}

func TestGeneratorMutantKeyFallsBackToID(t *testing.T) {
	if key := generatorMutantKey(Mutant{ID: "only-id"}); key != "id:only-id" {
		t.Fatalf("generatorMutantKey fallback = %q, want id fallback", key)
	}
}

func operatorSet(mutants []Mutant) map[string]bool {
	operators := map[string]bool{}
	for _, mutant := range mutants {
		operators[mutant.Operator] = true
	}
	return operators
}

func assertSemanticMutant(t *testing.T, mutants []Mutant, match func(Mutant) bool, message string) {
	t.Helper()
	for _, mutant := range mutants {
		if match(mutant) {
			return
		}
	}
	t.Fatalf("%s: %+v", message, mutants)
}

func containsTag(tags []string, want string) bool {
	for _, tag := range tags {
		if tag == want {
			return true
		}
	}
	return false
}
