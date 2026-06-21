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

type operatorSpec struct {
	Name                 string
	Risk                 string
	EquivalentMutantRisk string
	CompileErrorRisk     string
	ASTNodes             []string
	Example              string
	Reason               string
	Hint                 string
	Recommendation       string
	EnabledProfiles      []string
	DefinitionProfiles   []string
}

var operatorCatalog = []operatorSpec{
	{
		Name:                 opConditionalsNegation,
		Risk:                 "low",
		EquivalentMutantRisk: "medium",
		CompileErrorRisk:     "low",
		ASTNodes:             []string{astBinaryExpr},
		Example:              "a == b -> a != b",
		Reason:               "Fast branch behavior signal.",
		Hint:                 "Add assertions for the opposite branch or boundary condition.",
		Recommendation:       "fast-ci",
		EnabledProfiles:      []string{ProfileGremlinsCompatible, ProfileConservativeFast, ProfileConservative, ProfileDefault, ProfileAggressive},
		DefinitionProfiles:   []string{ProfileGremlinsCompatible, ProfileConservativeFast},
	},
	{
		Name:                 opConditionalsBoundary,
		Risk:                 "low",
		EquivalentMutantRisk: "high",
		CompileErrorRisk:     "low",
		ASTNodes:             []string{astBinaryExpr},
		Example:              "a < b -> a <= b",
		Reason:               "Fast boundary-condition signal.",
		Hint:                 "Add assertions for the opposite branch or boundary condition.",
		Recommendation:       "fast-ci",
		EnabledProfiles:      []string{ProfileGremlinsCompatible, ProfileConservativeFast, ProfileConservative, ProfileDefault, ProfileAggressive},
		DefinitionProfiles:   []string{ProfileGremlinsCompatible, ProfileConservativeFast},
	},
	{
		Name:                 opArithmeticBasic,
		Risk:                 "medium",
		EquivalentMutantRisk: "low",
		CompileErrorRisk:     "medium",
		ASTNodes:             []string{astBinaryExpr},
		Example:              "a + b -> a - b",
		Reason:               "Numeric result signal for fast CI.",
		Hint:                 "Add assertions for the computed numeric result and edge cases.",
		Recommendation:       "fast-ci",
		EnabledProfiles:      []string{ProfileGremlinsCompatible, ProfileConservativeFast, ProfileConservative, ProfileDefault, ProfileAggressive},
		DefinitionProfiles:   []string{ProfileGremlinsCompatible, ProfileConservativeFast},
	},
	{
		Name:                 opLogical,
		Risk:                 "low",
		EquivalentMutantRisk: "medium",
		CompileErrorRisk:     "low",
		ASTNodes:             []string{astBinaryExpr},
		Example:              "a && b -> a || b",
		Reason:               "Captures missing boolean combination assertions.",
		Hint:                 "Add a test where only one side of the boolean expression changes the outcome.",
		Recommendation:       "conservative",
		EnabledProfiles:      []string{ProfileConservative, ProfileDefault, ProfileAggressive},
		DefinitionProfiles:   []string{ProfileConservative},
	},
	{
		Name:                 opBooleanLiterals,
		Risk:                 "low",
		EquivalentMutantRisk: "low",
		CompileErrorRisk:     "low",
		ASTNodes:             []string{"ast.Ident"},
		Example:              "true -> false",
		Reason:               "Simple branch outcome signal.",
		Hint:                 "Assert both boolean outcomes instead of only executing the path.",
		Recommendation:       "conservative",
		EnabledProfiles:      []string{ProfileConservative, ProfileDefault, ProfileAggressive},
		DefinitionProfiles:   []string{ProfileConservative},
	},
	{
		Name:                 opStringEmptyLiterals,
		Risk:                 "medium",
		EquivalentMutantRisk: "medium",
		CompileErrorRisk:     "low",
		ASTNodes:             []string{astBasicLit},
		Example:              `"error" -> ""`,
		Reason:               "Controlled string behavior signal with low compile risk.",
		Hint:                 "Add an assertion for non-empty text or exact message behavior.",
		Recommendation:       "conservative",
		EnabledProfiles:      []string{ProfileConservative, ProfileDefault, ProfileAggressive},
		DefinitionProfiles:   []string{ProfileConservative},
	},
	{
		Name:                 opNilChecks,
		Risk:                 "medium",
		EquivalentMutantRisk: "high",
		CompileErrorRisk:     "low",
		ASTNodes:             []string{astBinaryExpr},
		Example:              "err == nil -> err != nil",
		Reason:               "Important Go error-path signal but high equivalence risk.",
		Hint:                 "Add assertions for the opposite branch or boundary condition.",
		Recommendation:       "default",
		EnabledProfiles:      []string{ProfileDefault, ProfileAggressive},
		DefinitionProfiles:   []string{ProfileDefault},
	},
	{
		Name:                 opErrorReturns,
		Risk:                 "medium",
		EquivalentMutantRisk: "high",
		CompileErrorRisk:     "medium",
		ASTNodes:             []string{"ast.IfStmt"},
		Example:              "err == nil -> err != nil",
		Reason:               "Controlled error-path mutation for nightly/default runs.",
		Hint:                 "Add an assertion that observes the changed behavior.",
		Recommendation:       "default",
		EnabledProfiles:      []string{ProfileDefault, ProfileAggressive},
		DefinitionProfiles:   []string{ProfileDefault},
	},
	{
		Name:                 opNumericLiterals,
		Risk:                 "medium",
		EquivalentMutantRisk: "medium",
		CompileErrorRisk:     "low",
		ASTNodes:             []string{astBasicLit},
		Example:              "2 -> 1",
		Reason:               "Controlled numeric literal signal for default runs.",
		Hint:                 "Add an assertion for numeric boundaries and configured constants.",
		Recommendation:       "default",
		EnabledProfiles:      []string{ProfileDefault, ProfileAggressive},
		DefinitionProfiles:   []string{ProfileDefault},
	},
	{
		Name:                 opReturnBoolLiterals,
		Risk:                 "medium",
		EquivalentMutantRisk: "medium",
		CompileErrorRisk:     "low",
		ASTNodes:             []string{"ast.ReturnStmt"},
		Example:              "return true -> return false",
		Reason:               "Return behavior signal without broad return rewrites.",
		Hint:                 "Assert both boolean outcomes instead of only executing the path.",
		Recommendation:       "default",
		EnabledProfiles:      []string{ProfileDefault, ProfileAggressive},
		DefinitionProfiles:   []string{ProfileDefault},
	},
	{
		Name:                 opAssignmentArithmetic,
		Risk:                 "medium",
		EquivalentMutantRisk: "medium",
		CompileErrorRisk:     "medium",
		ASTNodes:             []string{"ast.AssignStmt"},
		Example:              "x += n -> x -= n",
		Reason:               "Assignment-update signal from Go-heavy operator catalogs.",
		Hint:                 "Add assertions for cumulative updates and arithmetic assignment effects.",
		Recommendation:       "default",
		EnabledProfiles:      []string{ProfileDefault, ProfileAggressive},
		DefinitionProfiles:   []string{ProfileDefault},
	},
	{
		Name:                 opIncDec,
		Risk:                 "medium",
		EquivalentMutantRisk: "medium",
		CompileErrorRisk:     "low",
		ASTNodes:             []string{"ast.IncDecStmt"},
		Example:              "i++ -> i--",
		Reason:               "Loop and counter update signal without broad loop-control mutation.",
		Hint:                 "Add assertions for counter direction and loop iteration counts.",
		Recommendation:       "default",
		EnabledProfiles:      []string{ProfileDefault, ProfileAggressive},
		DefinitionProfiles:   []string{ProfileDefault},
	},
	{
		Name:                 opLiterals,
		Risk:                 "high",
		EquivalentMutantRisk: "high",
		CompileErrorRisk:     "medium",
		ASTNodes:             []string{astBasicLit},
		Example:              "1 -> 0",
		Reason:               "Broad campaign-only literal exploration.",
		Hint:                 "Add an assertion that observes the changed behavior.",
		Recommendation:       "aggressive",
		EnabledProfiles:      []string{ProfileAggressive},
		DefinitionProfiles:   []string{ProfileAggressive},
	},
	{
		Name:                 opReturns,
		Risk:                 "high",
		EquivalentMutantRisk: "high",
		CompileErrorRisk:     "medium",
		ASTNodes:             []string{"ast.ReturnStmt"},
		Example:              "return true -> return false",
		Reason:               "Campaign-only return behavior exploration.",
		Hint:                 "Add an assertion that observes the changed behavior.",
		Recommendation:       "aggressive",
		EnabledProfiles:      []string{ProfileAggressive},
		DefinitionProfiles:   []string{ProfileAggressive},
	},
	{
		Name:                 opLoopControl,
		Risk:                 "high",
		EquivalentMutantRisk: "high",
		CompileErrorRisk:     "medium",
		ASTNodes:             []string{"ast.ForStmt"},
		Example:              "i < n -> i <= n",
		Reason:               "Campaign-only loop boundary exploration.",
		Hint:                 "Add tests for loop boundary counts and off-by-one behavior.",
		Recommendation:       "aggressive",
		EnabledProfiles:      []string{ProfileAggressive},
		DefinitionProfiles:   []string{ProfileAggressive},
	},
	{
		Name:                 opSliceMapLenBoundary,
		Risk:                 "high",
		EquivalentMutantRisk: "medium",
		CompileErrorRisk:     "low",
		ASTNodes:             []string{astBinaryExpr},
		Example:              "len(xs) > 0 -> len(xs) >= 0",
		Reason:               "Targets collection boundary assumptions.",
		Hint:                 "Add tests for empty and single-element collection boundaries.",
		Recommendation:       "review",
		EnabledProfiles:      []string{ProfileAggressive},
		DefinitionProfiles:   []string{ProfileAggressive},
	},
}

type Mutant struct {
	ID                  string   `json:"id"`
	Module              string   `json:"module"`
	Package             string   `json:"package"`
	File                string   `json:"file"`
	Line                int      `json:"line"`
	Function            string   `json:"function"`
	Operator            string   `json:"operator"`
	Original            string   `json:"original"`
	Mutated             string   `json:"mutated"`
	StartOffset         int      `json:"start_offset"`
	EndOffset           int      `json:"end_offset"`
	Diff                string   `json:"unified_diff"`
	Fingerprint         string   `json:"fingerprint"`
	Hint                string   `json:"hint"`
	Description         string   `json:"description"`
	EquivalentRisk      string   `json:"equivalent_risk"`
	Recommendation      string   `json:"recommendation"`
	CompileErrorRisk    string   `json:"compile_error_risk"`
	SemanticTags        []string `json:"semantic_tags,omitempty"`
	SemanticGroup       string   `json:"semantic_group,omitempty"`
	GroupLabel          string   `json:"group_label,omitempty"`
	GroupReason         string   `json:"group_reason,omitempty"`
	PlatformSensitive   bool     `json:"platform_sensitive,omitempty"`
	NonProgressRisk     string   `json:"non_progress_risk,omitempty"`
	SuggestedSkipReason string   `json:"suggested_skip_reason,omitempty"`
}

func Definitions() []Definition {
	definitions := make([]Definition, 0, len(operatorCatalog))
	for _, profile := range []string{ProfileGremlinsCompatible, ProfileConservativeFast, ProfileConservative, ProfileDefault, ProfileAggressive} {
		for _, spec := range operatorCatalog {
			if !containsProfile(spec.DefinitionProfiles, profile) {
				continue
			}
			definitions = append(definitions, Definition{
				Name:                 spec.Name,
				Profile:              profile,
				Risk:                 spec.Risk,
				EquivalentMutantRisk: spec.EquivalentMutantRisk,
				CompileErrorRisk:     spec.CompileErrorRisk,
				ASTNodes:             append([]string{}, spec.ASTNodes...),
				Example:              spec.Example,
				Reason:               spec.Reason,
			})
		}
	}
	return definitions
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
	parents  map[ast.Node]ast.Node
}

type semanticMetadata struct {
	tags                []string
	group               string
	groupLabel          string
	groupReason         string
	platformSensitive   bool
	nonProgressRisk     string
	suggestedSkipReason string
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
	ctx := mutationContext{mutants: &mutants, fset: fset, pkg: pkg, filename: filename, src: src, profile: profile, ignores: ignores, parents: buildParentIndex(file)}
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
	meta := classifySemanticMutation(ctx, node, operator)
	mutatedSrc = append(mutatedSrc[:start], append([]byte(next), mutatedSrc[end:]...)...)
	diff := unifiedDiff(ctx.filename, string(ctx.src), string(mutatedSrc))
	fp := fingerprint(ctx.filename, strconv.Itoa(pos.Line), strconv.Itoa(start), strconv.Itoa(end), operator, original, mutated, diff)
	id := fmt.Sprintf("%s:%d:%s:%s", ctx.filename, pos.Line, operator, fp[:12])
	*ctx.mutants = append(*ctx.mutants, Mutant{
		ID:                  id,
		Package:             ctx.pkg,
		File:                ctx.filename,
		Line:                pos.Line,
		Function:            ctx.fn,
		Operator:            operator,
		Original:            original,
		Mutated:             mutated,
		StartOffset:         start,
		EndOffset:           end,
		Diff:                diff,
		Fingerprint:         fp,
		Hint:                hint(operator),
		Description:         description(ctx.fn, operator, original, mutated),
		EquivalentRisk:      equivalentRisk(operator),
		Recommendation:      recommendation(operator),
		CompileErrorRisk:    compileErrorRisk(operator),
		SemanticTags:        meta.tags,
		SemanticGroup:       meta.group,
		GroupLabel:          meta.groupLabel,
		GroupReason:         meta.groupReason,
		PlatformSensitive:   meta.platformSensitive,
		NonProgressRisk:     meta.nonProgressRisk,
		SuggestedSkipReason: meta.suggestedSkipReason,
	})
}

func buildParentIndex(root ast.Node) map[ast.Node]ast.Node {
	parents := map[ast.Node]ast.Node{}
	stack := []ast.Node{}
	ast.Inspect(root, func(node ast.Node) bool {
		if node == nil {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			return false
		}
		if len(stack) > 0 {
			parents[node] = stack[len(stack)-1]
		}
		stack = append(stack, node)
		return true
	})
	return parents
}

func classifySemanticMutation(ctx mutationContext, node ast.Node, operator string) semanticMetadata {
	meta := semanticMetadata{}
	if risk := nonProgressLoopRisk(ctx.parents, node, operator); risk != "" {
		meta.nonProgressRisk = risk
		meta.tags = appendUnique(meta.tags, "non-progress-loop-risk")
		meta.suggestedSkipReason = "reviewed-skip or quarantine if timeout confirms a non-progress loop"
	}
	if permissionModeMutation(ctx.parents, node, operator) {
		meta.platformSensitive = true
		meta.tags = appendUnique(meta.tags, "platform-sensitive")
		if meta.suggestedSkipReason == "" {
			meta.suggestedSkipReason = "review on the target OS before treating this permission-mode mutant as actionable"
		}
	}
	group, label, reason := semanticGroup(ctx, node, operator)
	if group != "" {
		meta.group = group
		meta.groupLabel = label
		meta.groupReason = reason
		meta.tags = appendUnique(meta.tags, "equivalence-risk-group")
		if label != "" {
			meta.tags = appendUnique(meta.tags, normalizeTag(label))
		}
		if meta.suggestedSkipReason == "" {
			meta.suggestedSkipReason = "review once for this semantic group before treating each survivor independently"
		}
	}
	if len(meta.tags) == 0 {
		return semanticMetadata{}
	}
	return meta
}

func semanticGroup(ctx mutationContext, node ast.Node, operator string) (string, string, string) {
	if operator == opConditionalsBoundary {
		if call, _, ok := enclosingCallForNode(ctx.parents, node); ok && isSortComparatorCall(ctx.parents, node, call) {
			line := ctx.fset.Position(call.Pos()).Line
			return fmt.Sprintf("sort-comparator-boundary:%s:%d", ctx.filename, line),
				"sort comparator boundary",
				"Boundary mutations inside sort comparator closures often collapse into one review decision."
		}
	}
	expr, ok := node.(*ast.BinaryExpr)
	if !ok {
		return "", "", ""
	}
	if operator == opSliceMapLenBoundary || (operator == opConditionalsBoundary && isLenComparison(expr)) || (operator == opLoopControl && isLenComparison(expr)) {
		return fmt.Sprintf("len-boundary:%s:%s", ctx.filename, normalizeExpr(FormatNode(ctx.fset, expr))),
			"len boundary",
			"Length-boundary mutations are often review-once families rather than independent actionable survivors."
	}
	return "", "", ""
}

func nonProgressLoopRisk(parents map[ast.Node]ast.Node, node ast.Node, operator string) string {
	forStmt := enclosingForStmt(parents, node)
	if forStmt == nil {
		return ""
	}
	switch n := node.(type) {
	case *ast.IncDecStmt:
		ident := identName(n.X)
		if ident == "" {
			return ""
		}
		condDirection, ok := loopConditionDirection(forStmt.Cond, ident)
		if !ok {
			return ""
		}
		originalDirection := directionForIncDec(n.Tok)
		mutatedDirection := oppositeDirection(originalDirection)
		if originalDirection == "" || mutatedDirection == "" {
			return ""
		}
		if originalDirection == condDirection && mutatedDirection != condDirection {
			return "high"
		}
	case *ast.AssignStmt:
		if operator != opAssignmentArithmetic || len(n.Lhs) == 0 {
			return ""
		}
		ident := identName(n.Lhs[0])
		if ident == "" {
			return ""
		}
		condDirection, ok := loopConditionDirection(forStmt.Cond, ident)
		if !ok {
			return ""
		}
		originalDirection := directionForAssign(n.Tok)
		mutatedDirection := oppositeDirection(originalDirection)
		if originalDirection == "" || mutatedDirection == "" {
			return ""
		}
		if originalDirection == condDirection && mutatedDirection != condDirection {
			return "high"
		}
	}
	return ""
}

func permissionModeMutation(parents map[ast.Node]ast.Node, node ast.Node, operator string) bool {
	if operator != opNumericLiterals && operator != opLiterals {
		return false
	}
	call, argIndex, ok := enclosingCallForNode(parents, node)
	if !ok {
		return false
	}
	switch selectorName(call.Fun) {
	case "os.Mkdir", "os.MkdirAll", "os.Chmod":
		return argIndex == 1
	case "os.OpenFile":
		return argIndex == 2
	default:
		return false
	}
}

func enclosingForStmt(parents map[ast.Node]ast.Node, node ast.Node) *ast.ForStmt {
	for current := node; current != nil; current = parents[current] {
		if forStmt, ok := current.(*ast.ForStmt); ok {
			return forStmt
		}
	}
	return nil
}

func loopConditionDirection(expr ast.Expr, ident string) (string, bool) {
	direction := ""
	found := false
	ast.Inspect(expr, func(node ast.Node) bool {
		if found {
			return false
		}
		binary, ok := node.(*ast.BinaryExpr)
		if !ok {
			return true
		}
		switch {
		case exprHasIdent(binary.X, ident):
			direction, found = directionFromComparison(binary.Op, true)
		case exprHasIdent(binary.Y, ident):
			direction, found = directionFromComparison(binary.Op, false)
		}
		return !found
	})
	return direction, found
}

func directionFromComparison(op token.Token, identOnLeft bool) (string, bool) {
	if identOnLeft {
		switch op {
		case token.LSS, token.LEQ:
			return "ascending", true
		case token.GTR, token.GEQ:
			return "descending", true
		}
		return "", false
	}
	switch op {
	case token.GTR, token.GEQ:
		return "ascending", true
	case token.LSS, token.LEQ:
		return "descending", true
	default:
		return "", false
	}
}

func exprHasIdent(expr ast.Expr, ident string) bool {
	found := false
	ast.Inspect(expr, func(node ast.Node) bool {
		name, ok := node.(*ast.Ident)
		if ok && name.Name == ident {
			found = true
			return false
		}
		return true
	})
	return found
}

func directionForIncDec(tok token.Token) string {
	switch tok {
	case token.INC:
		return "ascending"
	case token.DEC:
		return "descending"
	default:
		return ""
	}
}

func directionForAssign(tok token.Token) string {
	switch tok {
	case token.ADD_ASSIGN:
		return "ascending"
	case token.SUB_ASSIGN:
		return "descending"
	default:
		return ""
	}
}

func oppositeDirection(direction string) string {
	switch direction {
	case "ascending":
		return "descending"
	case "descending":
		return "ascending"
	default:
		return ""
	}
}

func enclosingCallForNode(parents map[ast.Node]ast.Node, node ast.Node) (*ast.CallExpr, int, bool) {
	child := node
	for current := parents[node]; current != nil; current = parents[current] {
		call, ok := current.(*ast.CallExpr)
		if ok {
			for index, arg := range call.Args {
				if arg == child {
					return call, index, true
				}
			}
		}
		child = current
	}
	return nil, 0, false
}

func isSortComparatorCall(parents map[ast.Node]ast.Node, node ast.Node, call *ast.CallExpr) bool {
	if !isSortCall(call) {
		return false
	}
	child := node
	for current := parents[node]; current != nil; current = parents[current] {
		if current == call {
			break
		}
		if fn, ok := current.(*ast.FuncLit); ok {
			for _, arg := range call.Args {
				if arg == fn {
					return true
				}
			}
		}
		child = current
	}
	_ = child
	return false
}

func isSortCall(call *ast.CallExpr) bool {
	name := selectorName(call.Fun)
	return name == "sort.Slice" || name == "sort.SliceStable"
}

func selectorName(expr ast.Expr) string {
	switch node := expr.(type) {
	case *ast.SelectorExpr:
		if ident, ok := node.X.(*ast.Ident); ok {
			return ident.Name + "." + node.Sel.Name
		}
		return node.Sel.Name
	case *ast.Ident:
		return node.Name
	default:
		return ""
	}
}

func identName(expr ast.Expr) string {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return ""
	}
	return ident.Name
}

func normalizeExpr(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func appendUnique(tags []string, tag string) []string {
	if tag == "" {
		return tags
	}
	for _, existing := range tags {
		if existing == tag {
			return tags
		}
	}
	return append(tags, tag)
}

func normalizeTag(value string) string {
	replaced := strings.ToLower(strings.TrimSpace(value))
	replaced = strings.ReplaceAll(replaced, " ", "-")
	return replaced
}

func operatorEnabled(operator, profile string) bool {
	spec, ok := operatorSpecFor(operator)
	if !ok {
		return false
	}
	return containsProfile(spec.EnabledProfiles, profile)
}

func equivalentRisk(operator string) string {
	spec, ok := operatorSpecFor(operator)
	if !ok {
		return "unknown"
	}
	return spec.EquivalentMutantRisk
}

func recommendation(operator string) string {
	spec, ok := operatorSpecFor(operator)
	if !ok {
		return "review"
	}
	return spec.Recommendation
}

func compileErrorRisk(operator string) string {
	spec, ok := operatorSpecFor(operator)
	if !ok {
		return "unknown"
	}
	return spec.CompileErrorRisk
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
	spec, ok := operatorSpecFor(operator)
	if !ok {
		return "Add an assertion that observes the changed behavior."
	}
	return spec.Hint
}

func operatorSpecFor(operator string) (operatorSpec, bool) {
	for _, spec := range operatorCatalog {
		if spec.Name == operator {
			return spec, true
		}
	}
	return operatorSpec{}, false
}

func containsProfile(profiles []string, want string) bool {
	for _, profile := range profiles {
		if profile == want {
			return true
		}
	}
	return false
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
