package mutator

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strconv"
	"strings"
)

const (
	ProfileGremlinsCompatible = "gremlins-compatible"
	ProfileConservativeFast   = "conservative-fast"
	ProfileConservative       = "conservative"
	ProfileDefault            = "default"
	ProfileAggressive         = "aggressive"

	opConditionalsNegation   = "conditionals-negation"
	opConditionalsBoundary   = "conditionals-boundary"
	opArithmeticBasic        = "arithmetic-basic"
	opLogical                = "logical"
	opBooleanLiterals        = "boolean-literals"
	opStringEmptyLiterals    = "string-empty-literals"
	opNilChecks              = "nil-checks"
	opErrorReturns           = "error-returns"
	opNumericLiterals        = "numeric-literals"
	opReturnBoolLiterals     = "return-bool-literals"
	opAssignmentArithmetic   = "assignment-arithmetic"
	opIncDec                 = "inc-dec"
	opLiterals               = "literals"
	opReturns                = "returns"
	opLoopControl            = "loop-control"
	opSliceMapLenBoundary    = "slice-map-len-boundary"
	astBinaryExpr            = "ast.BinaryExpr"
	astBasicLit              = "ast.BasicLit"
	inlineIgnoreReasonPrefix = "reason="
)

type Definition struct {
	Name                 string   `json:"name"`
	Profile              string   `json:"profile"`
	Risk                 string   `json:"risk"`
	EquivalentMutantRisk string   `json:"equivalent_mutant_risk"`
	CompileErrorRisk     string   `json:"compile_error_risk"`
	ASTNodes             []string `json:"ast_nodes"`
	Example              string   `json:"example"`
	Reason               string   `json:"reason"`
}

type Mutant struct {
	ID               string `json:"id"`
	Module           string `json:"module"`
	Package          string `json:"package"`
	File             string `json:"file"`
	Line             int    `json:"line"`
	Function         string `json:"function"`
	Operator         string `json:"operator"`
	Original         string `json:"original"`
	Mutated          string `json:"mutated"`
	StartOffset      int    `json:"start_offset"`
	EndOffset        int    `json:"end_offset"`
	Diff             string `json:"unified_diff"`
	Fingerprint      string `json:"fingerprint"`
	Hint             string `json:"hint"`
	Description      string `json:"description"`
	EquivalentRisk   string `json:"equivalent_risk"`
	Recommendation   string `json:"recommendation"`
	CompileErrorRisk string `json:"compile_error_risk"`
}

func Definitions() []Definition {
	return []Definition{
		{Name: opConditionalsNegation, Profile: ProfileGremlinsCompatible, Risk: "low", EquivalentMutantRisk: "medium", CompileErrorRisk: "low", ASTNodes: []string{astBinaryExpr}, Example: "a == b -> a != b", Reason: "Fast branch behavior signal."},
		{Name: opConditionalsBoundary, Profile: ProfileGremlinsCompatible, Risk: "low", EquivalentMutantRisk: "medium", CompileErrorRisk: "low", ASTNodes: []string{astBinaryExpr}, Example: "a < b -> a <= b", Reason: "Fast boundary-condition signal."},
		{Name: opArithmeticBasic, Profile: ProfileGremlinsCompatible, Risk: "medium", EquivalentMutantRisk: "low", CompileErrorRisk: "medium", ASTNodes: []string{astBinaryExpr}, Example: "a + b -> a - b", Reason: "Numeric result signal for fast CI."},
		{Name: opConditionalsNegation, Profile: ProfileConservativeFast, Risk: "low", EquivalentMutantRisk: "medium", CompileErrorRisk: "low", ASTNodes: []string{astBinaryExpr}, Example: "a == b -> a != b", Reason: "Fast branch behavior signal."},
		{Name: opConditionalsBoundary, Profile: ProfileConservativeFast, Risk: "low", EquivalentMutantRisk: "medium", CompileErrorRisk: "low", ASTNodes: []string{astBinaryExpr}, Example: "a < b -> a <= b", Reason: "Fast boundary-condition signal."},
		{Name: opArithmeticBasic, Profile: ProfileConservativeFast, Risk: "medium", EquivalentMutantRisk: "low", CompileErrorRisk: "medium", ASTNodes: []string{astBinaryExpr}, Example: "a + b -> a - b", Reason: "Numeric result signal for fast CI."},
		{Name: opLogical, Profile: ProfileConservative, Risk: "low", EquivalentMutantRisk: "medium", CompileErrorRisk: "low", ASTNodes: []string{astBinaryExpr}, Example: "a && b -> a || b", Reason: "Captures missing boolean combination assertions."},
		{Name: opBooleanLiterals, Profile: ProfileConservative, Risk: "low", EquivalentMutantRisk: "low", CompileErrorRisk: "low", ASTNodes: []string{"ast.Ident"}, Example: "true -> false", Reason: "Simple branch outcome signal."},
		{Name: opStringEmptyLiterals, Profile: ProfileConservative, Risk: "medium", EquivalentMutantRisk: "medium", CompileErrorRisk: "low", ASTNodes: []string{astBasicLit}, Example: `"error" -> ""`, Reason: "Controlled string behavior signal with low compile risk."},
		{Name: opNilChecks, Profile: ProfileDefault, Risk: "medium", EquivalentMutantRisk: "high", CompileErrorRisk: "low", ASTNodes: []string{astBinaryExpr}, Example: "err == nil -> err != nil", Reason: "Important Go error-path signal but high equivalence risk."},
		{Name: opErrorReturns, Profile: ProfileDefault, Risk: "medium", EquivalentMutantRisk: "high", CompileErrorRisk: "medium", ASTNodes: []string{"ast.IfStmt"}, Example: "err == nil -> err != nil", Reason: "Controlled error-path mutation for nightly/default runs."},
		{Name: opNumericLiterals, Profile: ProfileDefault, Risk: "medium", EquivalentMutantRisk: "medium", CompileErrorRisk: "low", ASTNodes: []string{astBasicLit}, Example: "2 -> 1", Reason: "Controlled numeric literal signal for default runs."},
		{Name: opReturnBoolLiterals, Profile: ProfileDefault, Risk: "medium", EquivalentMutantRisk: "medium", CompileErrorRisk: "low", ASTNodes: []string{"ast.ReturnStmt"}, Example: "return true -> return false", Reason: "Return behavior signal without broad return rewrites."},
		{Name: opAssignmentArithmetic, Profile: ProfileDefault, Risk: "medium", EquivalentMutantRisk: "medium", CompileErrorRisk: "medium", ASTNodes: []string{"ast.AssignStmt"}, Example: "x += n -> x -= n", Reason: "Assignment-update signal from Go-heavy operator catalogs."},
		{Name: opIncDec, Profile: ProfileDefault, Risk: "medium", EquivalentMutantRisk: "medium", CompileErrorRisk: "low", ASTNodes: []string{"ast.IncDecStmt"}, Example: "i++ -> i--", Reason: "Loop and counter update signal without broad loop-control mutation."},
		{Name: opLiterals, Profile: ProfileAggressive, Risk: "high", EquivalentMutantRisk: "high", CompileErrorRisk: "medium", ASTNodes: []string{astBasicLit}, Example: "1 -> 0", Reason: "Broad campaign-only literal exploration."},
		{Name: opReturns, Profile: ProfileAggressive, Risk: "high", EquivalentMutantRisk: "high", CompileErrorRisk: "medium", ASTNodes: []string{"ast.ReturnStmt"}, Example: "return true -> return false", Reason: "Campaign-only return behavior exploration."},
		{Name: opLoopControl, Profile: ProfileAggressive, Risk: "high", EquivalentMutantRisk: "high", CompileErrorRisk: "medium", ASTNodes: []string{"ast.ForStmt"}, Example: "i < n -> i <= n", Reason: "Campaign-only loop boundary exploration."},
		{Name: opSliceMapLenBoundary, Profile: ProfileAggressive, Risk: "high", EquivalentMutantRisk: "medium", CompileErrorRisk: "low", ASTNodes: []string{astBinaryExpr}, Example: "len(xs) > 0 -> len(xs) >= 0", Reason: "Targets collection boundary assumptions."},
	}
}

type inlineIgnore struct {
	line     int
	operator string
	reason   string
}

type mutationContext struct {
	mutants  *[]Mutant
	fset     *token.FileSet
	pkg      string
	filename string
	src      []byte
	fn       string
	profile  string
	ignores  []inlineIgnore
}

func ValidateInlineIgnores(filename string, src []byte, requireReason bool) ([]inlineIgnore, error) {
	var ignores []inlineIgnore
	lines := strings.Split(string(src), "\n")
	for i, line := range lines {
		ignore, ok, err := parseInlineIgnore(filename, i, line, requireReason)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		ignores = append(ignores, ignore)
	}
	return ignores, nil
}

func parseInlineIgnore(filename string, index int, line string, requireReason bool) (inlineIgnore, bool, error) {
	idx := strings.Index(line, "cervomut:ignore")
	if idx < 0 {
		return inlineIgnore{}, false, nil
	}
	if commentIdx := strings.Index(line[:idx], "//"); commentIdx < 0 {
		return inlineIgnore{}, false, nil
	}
	rest := strings.TrimSpace(line[idx+len("cervomut:ignore"):])
	operator := parseInlineIgnoreOperator(rest)
	reason := parseInlineIgnoreReason(rest)
	if requireReason && reason == "" {
		return inlineIgnore{}, false, fmt.Errorf("%s:%d inline ignore requires reason", filename, index+1)
	}
	return inlineIgnore{line: index + 2, operator: operator, reason: reason}, true, nil
}

func parseInlineIgnoreOperator(rest string) string {
	fields := strings.Fields(rest)
	if len(fields) > 0 && !strings.HasPrefix(fields[0], inlineIgnoreReasonPrefix) {
		return fields[0]
	}
	return "*"
}

func parseInlineIgnoreReason(rest string) string {
	reasonIdx := strings.Index(rest, inlineIgnoreReasonPrefix)
	if reasonIdx < 0 {
		return ""
	}
	raw := strings.TrimSpace(rest[reasonIdx+len(inlineIgnoreReasonPrefix):])
	if parsed, err := strconv.Unquote(raw); err == nil {
		return parsed
	}
	return strings.Trim(raw, `"`)
}

func Generate(pkg, filename string, src []byte, profile string) ([]Mutant, error) {
	if profile == "" {
		profile = ProfileConservativeFast
	}
	ignores, err := ValidateInlineIgnores(filename, src, true)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	var mutants []Mutant
	var fn string
	ctx := mutationContext{mutants: &mutants, fset: fset, pkg: pkg, filename: filename, src: src, profile: profile, ignores: ignores}
	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.FuncDecl:
			prev := fn
			fn = n.Name.Name
			ctx.fn = fn
			ast.Inspect(n.Body, func(child ast.Node) bool {
				collectNode(ctx, child)
				return true
			})
			fn = prev
			ctx.fn = fn
			return false
		default:
			ctx.fn = fn
			collectNode(ctx, node)
		}
		return true
	})
	return mutants, nil
}

func collectNode(ctx mutationContext, node ast.Node) {
	switch n := node.(type) {
	case *ast.BinaryExpr:
		addBinaryMutants(ctx, n)
	case *ast.Ident:
		addIdentMutants(ctx, n)
	case *ast.BasicLit:
		addBasicLitMutants(ctx, n)
	case *ast.ReturnStmt:
		addReturnMutants(ctx, n)
	case *ast.ForStmt:
		if expr, ok := n.Cond.(*ast.BinaryExpr); ok {
			addLoopControlMutant(ctx, expr)
		}
	case *ast.AssignStmt:
		addAssignMutants(ctx, n)
	case *ast.IncDecStmt:
		addIncDecMutants(ctx, n)
	}
}

func addIdentMutants(ctx mutationContext, ident *ast.Ident) {
	switch ident.Name {
	case "true":
		addMutation(ctx, ident, opBooleanLiterals, "true", "false")
	case "false":
		addMutation(ctx, ident, opBooleanLiterals, "false", "true")
	}
}

func addBasicLitMutants(ctx mutationContext, lit *ast.BasicLit) {
	if lit.Kind == token.STRING && lit.Value != `""` {
		addMutation(ctx, lit, opStringEmptyLiterals, lit.Value, `""`)
		addMutation(ctx, lit, opLiterals, lit.Value, `""`)
		return
	}
	if lit.Kind != token.INT {
		return
	}
	mutated := "0"
	if lit.Value == "0" {
		mutated = "1"
	}
	addMutation(ctx, lit, opNumericLiterals, lit.Value, mutated)
	addMutation(ctx, lit, opLiterals, lit.Value, mutated)
}

func addReturnMutants(ctx mutationContext, stmt *ast.ReturnStmt) {
	for _, result := range stmt.Results {
		ident, ok := result.(*ast.Ident)
		if !ok {
			continue
		}
		addReturnIdentMutants(ctx, ident)
	}
}

func addReturnIdentMutants(ctx mutationContext, ident *ast.Ident) {
	switch ident.Name {
	case "true":
		addMutation(ctx, ident, opReturnBoolLiterals, "true", "false")
		addMutation(ctx, ident, opReturns, "true", "false")
	case "false":
		addMutation(ctx, ident, opReturnBoolLiterals, "false", "true")
		addMutation(ctx, ident, opReturns, "false", "true")
	}
}

func addAssignMutants(ctx mutationContext, stmt *ast.AssignStmt) {
	replacements := map[token.Token]string{
		token.ADD_ASSIGN: "-=",
		token.SUB_ASSIGN: "+=",
		token.MUL_ASSIGN: "/=",
		token.QUO_ASSIGN: "*=",
	}
	if mutated, ok := replacements[stmt.Tok]; ok {
		addMutation(ctx, stmt, opAssignmentArithmetic, stmt.Tok.String(), mutated)
	}
}

func addIncDecMutants(ctx mutationContext, stmt *ast.IncDecStmt) {
	switch stmt.Tok {
	case token.INC:
		addMutation(ctx, stmt, opIncDec, "++", "--")
	case token.DEC:
		addMutation(ctx, stmt, opIncDec, "--", "++")
	}
}

func addBinaryMutants(ctx mutationContext, expr *ast.BinaryExpr) {
	type candidate struct {
		operator    string
		replacement string
	}
	var candidates []candidate
	switch expr.Op {
	case token.EQL:
		candidates = append(candidates, candidate{operator: opConditionalsNegation, replacement: "!="})
	case token.NEQ:
		candidates = append(candidates, candidate{operator: opConditionalsNegation, replacement: "=="})
	case token.LSS:
		candidates = append(candidates,
			candidate{operator: opConditionalsBoundary, replacement: "<="},
			candidate{operator: opConditionalsNegation, replacement: ">="},
		)
	case token.LEQ:
		candidates = append(candidates,
			candidate{operator: opConditionalsBoundary, replacement: "<"},
			candidate{operator: opConditionalsNegation, replacement: ">"},
		)
	case token.GTR:
		candidates = append(candidates,
			candidate{operator: opConditionalsBoundary, replacement: ">="},
			candidate{operator: opConditionalsNegation, replacement: "<="},
		)
	case token.GEQ:
		candidates = append(candidates,
			candidate{operator: opConditionalsBoundary, replacement: ">"},
			candidate{operator: opConditionalsNegation, replacement: "<"},
		)
	case token.LAND:
		candidates = append(candidates, candidate{operator: opLogical, replacement: "||"})
	case token.LOR:
		candidates = append(candidates, candidate{operator: opLogical, replacement: "&&"})
	case token.ADD:
		candidates = append(candidates, candidate{operator: opArithmeticBasic, replacement: "-"})
	case token.SUB:
		candidates = append(candidates, candidate{operator: opArithmeticBasic, replacement: "+"})
	case token.MUL:
		candidates = append(candidates, candidate{operator: opArithmeticBasic, replacement: "/"})
	case token.QUO:
		candidates = append(candidates, candidate{operator: opArithmeticBasic, replacement: "*"})
	}
	if len(candidates) == 0 {
		return
	}
	if isNilCheck(expr) {
		candidates = []candidate{{operator: opNilChecks, replacement: candidates[0].replacement}}
	}
	if isLenComparison(expr) {
		candidates = append(candidates, candidate{operator: opSliceMapLenBoundary, replacement: boundaryReplacement(expr.Op)})
	}
	for _, mutation := range candidates {
		if mutation.replacement == "" {
			continue
		}
		addMutation(ctx, expr, mutation.operator, expr.Op.String(), mutation.replacement)
	}
}

func addLoopControlMutant(ctx mutationContext, expr *ast.BinaryExpr) {
	replacement := boundaryReplacement(expr.Op)
	if replacement == "" {
		return
	}
	addMutation(ctx, expr, opLoopControl, expr.Op.String(), replacement)
}

func boundaryReplacement(op token.Token) string {
	switch op {
	case token.LSS:
		return "<="
	case token.LEQ:
		return "<"
	case token.GTR:
		return ">="
	case token.GEQ:
		return ">"
	default:
		return ""
	}
}

func isLenComparison(expr *ast.BinaryExpr) bool {
	return isLenCall(expr.X) || isLenCall(expr.Y)
}

func isLenCall(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return false
	}
	ident, ok := call.Fun.(*ast.Ident)
	return ok && ident.Name == "len"
}

func isNilCheck(expr *ast.BinaryExpr) bool {
	if expr.Op != token.EQL && expr.Op != token.NEQ {
		return false
	}
	return isNil(expr.X) || isNil(expr.Y)
}

func isNil(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "nil"
}

func addMutation(ctx mutationContext, node ast.Node, operator, original, mutated string) {
	if !operatorEnabled(operator, ctx.profile) {
		return
	}
	pos := ctx.fset.Position(node.Pos())
	if ignored(pos.Line, operator, ctx.ignores) {
		return
	}
	start := ctx.fset.Position(node.Pos()).Offset
	end := ctx.fset.Position(node.End()).Offset
	if start < 0 || end > len(ctx.src) || start >= end {
		return
	}
	mutatedSrc := append([]byte{}, ctx.src...)
	segment := string(ctx.src[start:end])
	next, err := replaceFirst(segment, original, mutated)
	if err != nil {
		return
	}
	mutatedSrc = append(mutatedSrc[:start], append([]byte(next), mutatedSrc[end:]...)...)
	diff := unifiedDiff(ctx.filename, string(ctx.src), string(mutatedSrc))
	fp := fingerprint(ctx.filename, strconv.Itoa(pos.Line), strconv.Itoa(start), strconv.Itoa(end), operator, original, mutated, diff)
	id := fmt.Sprintf("%s:%d:%s:%s", ctx.filename, pos.Line, operator, fp[:12])
	*ctx.mutants = append(*ctx.mutants, Mutant{
		ID:               id,
		Package:          ctx.pkg,
		File:             ctx.filename,
		Line:             pos.Line,
		Function:         ctx.fn,
		Operator:         operator,
		Original:         original,
		Mutated:          mutated,
		StartOffset:      start,
		EndOffset:        end,
		Diff:             diff,
		Fingerprint:      fp,
		Hint:             hint(operator),
		Description:      description(ctx.fn, operator, original, mutated),
		EquivalentRisk:   equivalentRisk(operator),
		Recommendation:   recommendation(operator),
		CompileErrorRisk: compileErrorRisk(operator),
	})
}

func operatorEnabled(operator, profile string) bool {
	switch operator {
	case opConditionalsNegation, opConditionalsBoundary, opArithmeticBasic:
		return true
	case opLogical, opBooleanLiterals, opStringEmptyLiterals:
		return profile == ProfileConservative || profile == ProfileDefault || profile == ProfileAggressive
	case opNilChecks, opErrorReturns, opNumericLiterals, opReturnBoolLiterals, opAssignmentArithmetic, opIncDec:
		return profile == ProfileDefault || profile == ProfileAggressive
	case opLiterals, opReturns, opLoopControl, opSliceMapLenBoundary:
		return profile == ProfileAggressive
	default:
		return false
	}
}

func equivalentRisk(operator string) string {
	switch operator {
	case opArithmeticBasic, opBooleanLiterals:
		return "low"
	case opConditionalsNegation, opLogical, opStringEmptyLiterals, opNumericLiterals, opReturnBoolLiterals, opAssignmentArithmetic, opIncDec, opSliceMapLenBoundary:
		return "medium"
	case opConditionalsBoundary, opNilChecks, opErrorReturns, opLiterals, opReturns, opLoopControl:
		return "high"
	default:
		return "unknown"
	}
}

func recommendation(operator string) string {
	switch operator {
	case opArithmeticBasic, opConditionalsNegation, opConditionalsBoundary:
		return "fast-ci"
	case opLogical, opBooleanLiterals, opStringEmptyLiterals:
		return "conservative"
	case opNilChecks, opErrorReturns, opNumericLiterals, opReturnBoolLiterals, opAssignmentArithmetic, opIncDec:
		return "default"
	case opLiterals, opReturns, opLoopControl:
		return "aggressive"
	default:
		return "review"
	}
}

func compileErrorRisk(operator string) string {
	switch operator {
	case opConditionalsNegation, opConditionalsBoundary, opLogical, opBooleanLiterals, opNilChecks, opStringEmptyLiterals, opNumericLiterals, opReturnBoolLiterals, opIncDec, opSliceMapLenBoundary:
		return "low"
	case opArithmeticBasic, opErrorReturns, opLiterals, opReturns, opLoopControl:
		return "medium"
	default:
		return "unknown"
	}
}

func ignored(line int, operator string, ignores []inlineIgnore) bool {
	for _, ignore := range ignores {
		if ignore.line == line && operatorMatchesIgnore(operator, ignore.operator) {
			return true
		}
	}
	return false
}

func operatorMatchesIgnore(operator, ignored string) bool {
	if ignored == "*" || ignored == operator {
		return true
	}
	if ignored == "conditionals" && strings.HasPrefix(operator, "conditionals-") {
		return true
	}
	return false
}

func replaceFirst(s, old, new string) (string, error) {
	idx := strings.Index(s, old)
	if idx < 0 {
		return "", errors.New("original token not found")
	}
	return s[:idx] + new + s[idx+len(old):], nil
}

func unifiedDiff(filename, original, mutated string) string {
	return fmt.Sprintf("--- %s\n+++ %s\n@@\n-%s\n+%s\n", filename, filename, strings.TrimRight(original, "\n"), strings.TrimRight(mutated, "\n"))
}

func fingerprint(parts ...string) string {
	hash := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(hash[:])
}

func hint(operator string) string {
	switch operator {
	case opConditionalsNegation, opConditionalsBoundary, opNilChecks:
		return "Add assertions for the opposite branch or boundary condition."
	case opLogical:
		return "Add a test where only one side of the boolean expression changes the outcome."
	case opBooleanLiterals, opReturnBoolLiterals:
		return "Assert both boolean outcomes instead of only executing the path."
	case opArithmeticBasic:
		return "Add assertions for the computed numeric result and edge cases."
	case opStringEmptyLiterals:
		return "Add an assertion for non-empty text or exact message behavior."
	case opNumericLiterals:
		return "Add an assertion for numeric boundaries and configured constants."
	case opSliceMapLenBoundary:
		return "Add tests for empty and single-element collection boundaries."
	case opAssignmentArithmetic:
		return "Add assertions for cumulative updates and arithmetic assignment effects."
	case opIncDec:
		return "Add assertions for counter direction and loop iteration counts."
	case opLoopControl:
		return "Add tests for loop boundary counts and off-by-one behavior."
	default:
		return "Add an assertion that observes the changed behavior."
	}
}

func description(fn, operator, original, mutated string) string {
	where := "expression"
	if fn != "" {
		where = "function " + fn
	}
	return fmt.Sprintf("%s mutation in %s: changed %s to %s.", operator, where, original, mutated)
}

func FormatNode(fset *token.FileSet, node ast.Node) string {
	var b strings.Builder
	_ = printer.Fprint(&b, fset, node)
	return b.String()
}
