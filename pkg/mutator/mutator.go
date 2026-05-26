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
	ProfileConservative = "conservative"
	ProfileDefault      = "default"
	ProfileAggressive   = "aggressive"
)

type Definition struct {
	Name                 string   `json:"name"`
	Profile              string   `json:"profile"`
	Risk                 string   `json:"risk"`
	EquivalentMutantRisk string   `json:"equivalent_mutant_risk"`
	ASTNodes             []string `json:"ast_nodes"`
	Example              string   `json:"example"`
}

type Mutant struct {
	ID          string `json:"id"`
	Module      string `json:"module"`
	Package     string `json:"package"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	Function    string `json:"function"`
	Operator    string `json:"operator"`
	Original    string `json:"original"`
	Mutated     string `json:"mutated"`
	StartOffset int    `json:"start_offset"`
	EndOffset   int    `json:"end_offset"`
	Diff        string `json:"unified_diff"`
	Fingerprint string `json:"fingerprint"`
	Hint        string `json:"hint"`
	Description string `json:"description"`
}

func Definitions() []Definition {
	return []Definition{
		{Name: "conditionals", Profile: ProfileConservative, Risk: "low", EquivalentMutantRisk: "medium", ASTNodes: []string{"ast.BinaryExpr"}, Example: "a == b -> a != b"},
		{Name: "logical", Profile: ProfileConservative, Risk: "low", EquivalentMutantRisk: "medium", ASTNodes: []string{"ast.BinaryExpr"}, Example: "a && b -> a || b"},
		{Name: "boolean-literals", Profile: ProfileConservative, Risk: "low", EquivalentMutantRisk: "low", ASTNodes: []string{"ast.Ident"}, Example: "true -> false"},
		{Name: "nil-checks", Profile: ProfileConservative, Risk: "low", EquivalentMutantRisk: "medium", ASTNodes: []string{"ast.BinaryExpr"}, Example: "err == nil -> err != nil"},
		{Name: "arithmetic-basic", Profile: ProfileConservative, Risk: "medium", EquivalentMutantRisk: "medium", ASTNodes: []string{"ast.BinaryExpr"}, Example: "a + b -> a - b"},
		{Name: "error-returns", Profile: ProfileDefault, Risk: "medium", EquivalentMutantRisk: "high", ASTNodes: []string{"ast.IfStmt"}, Example: "err == nil -> err != nil"},
		{Name: "literals", Profile: ProfileAggressive, Risk: "high", EquivalentMutantRisk: "high", ASTNodes: []string{"ast.BasicLit"}, Example: "1 -> 0"},
		{Name: "returns", Profile: ProfileAggressive, Risk: "high", EquivalentMutantRisk: "high", ASTNodes: []string{"ast.ReturnStmt"}, Example: "return true -> return false"},
		{Name: "loop-control", Profile: ProfileAggressive, Risk: "high", EquivalentMutantRisk: "high", ASTNodes: []string{"ast.ForStmt"}, Example: "< -> <="},
	}
}

type inlineIgnore struct {
	line     int
	operator string
	reason   string
}

func ValidateInlineIgnores(filename string, src []byte, requireReason bool) ([]inlineIgnore, error) {
	var ignores []inlineIgnore
	lines := strings.Split(string(src), "\n")
	for i, line := range lines {
		idx := strings.Index(line, "cervomut:ignore")
		if idx < 0 {
			continue
		}
		rest := strings.TrimSpace(line[idx+len("cervomut:ignore"):])
		fields := strings.Fields(rest)
		operator := "*"
		if len(fields) > 0 && !strings.HasPrefix(fields[0], "reason=") {
			operator = fields[0]
		}
		reason := ""
		if reasonIdx := strings.Index(rest, "reason="); reasonIdx >= 0 {
			raw := strings.TrimSpace(rest[reasonIdx+len("reason="):])
			if parsed, err := strconv.Unquote(raw); err == nil {
				reason = parsed
			} else {
				reason = strings.Trim(raw, `"`)
			}
		}
		if requireReason && reason == "" {
			return nil, fmt.Errorf("%s:%d inline ignore requires reason", filename, i+1)
		}
		ignores = append(ignores, inlineIgnore{line: i + 2, operator: operator, reason: reason})
	}
	return ignores, nil
}

func Generate(pkg, filename string, src []byte, profile string) ([]Mutant, error) {
	if profile == "" {
		profile = ProfileConservative
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
	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.FuncDecl:
			prev := fn
			fn = n.Name.Name
			ast.Inspect(n.Body, func(child ast.Node) bool {
				collectNode(&mutants, fset, pkg, filename, src, fn, child, profile, ignores)
				return true
			})
			fn = prev
			return false
		default:
			collectNode(&mutants, fset, pkg, filename, src, fn, node, profile, ignores)
		}
		return true
	})
	return mutants, nil
}

func collectNode(mutants *[]Mutant, fset *token.FileSet, pkg, filename string, src []byte, fn string, node ast.Node, profile string, ignores []inlineIgnore) {
	switch n := node.(type) {
	case *ast.BinaryExpr:
		addBinaryMutants(mutants, fset, pkg, filename, src, fn, n, profile, ignores)
	case *ast.Ident:
		if n.Name == "true" {
			addMutation(mutants, fset, pkg, filename, src, fn, n, "boolean-literals", "true", "false", profile, ignores)
		}
		if n.Name == "false" {
			addMutation(mutants, fset, pkg, filename, src, fn, n, "boolean-literals", "false", "true", profile, ignores)
		}
	case *ast.BasicLit:
		if n.Kind == token.INT && n.Value != "0" {
			addMutation(mutants, fset, pkg, filename, src, fn, n, "literals", n.Value, "0", profile, ignores)
		}
		if n.Kind == token.INT && n.Value == "0" {
			addMutation(mutants, fset, pkg, filename, src, fn, n, "literals", "0", "1", profile, ignores)
		}
		if n.Kind == token.STRING && n.Value != `""` {
			addMutation(mutants, fset, pkg, filename, src, fn, n, "literals", n.Value, `""`, profile, ignores)
		}
	case *ast.ReturnStmt:
		for _, result := range n.Results {
			ident, ok := result.(*ast.Ident)
			if !ok {
				continue
			}
			if ident.Name == "true" {
				addMutation(mutants, fset, pkg, filename, src, fn, ident, "returns", "true", "false", profile, ignores)
			}
			if ident.Name == "false" {
				addMutation(mutants, fset, pkg, filename, src, fn, ident, "returns", "false", "true", profile, ignores)
			}
		}
	}
}

func addBinaryMutants(mutants *[]Mutant, fset *token.FileSet, pkg, filename string, src []byte, fn string, expr *ast.BinaryExpr, profile string, ignores []inlineIgnore) {
	replacement := ""
	operator := ""
	switch expr.Op {
	case token.EQL:
		operator, replacement = "conditionals", "!="
	case token.NEQ:
		operator, replacement = "conditionals", "=="
	case token.LSS:
		operator, replacement = "conditionals", "<="
	case token.LEQ:
		operator, replacement = "conditionals", "<"
	case token.GTR:
		operator, replacement = "conditionals", ">="
	case token.GEQ:
		operator, replacement = "conditionals", ">"
	case token.LAND:
		operator, replacement = "logical", "||"
	case token.LOR:
		operator, replacement = "logical", "&&"
	case token.ADD:
		operator, replacement = "arithmetic-basic", "-"
	case token.SUB:
		operator, replacement = "arithmetic-basic", "+"
	case token.MUL:
		operator, replacement = "arithmetic-basic", "/"
	case token.QUO:
		operator, replacement = "arithmetic-basic", "*"
	}
	if operator == "" {
		return
	}
	if isNilCheck(expr) {
		operator = "nil-checks"
	}
	addMutation(mutants, fset, pkg, filename, src, fn, expr, operator, expr.Op.String(), replacement, profile, ignores)
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

func addMutation(mutants *[]Mutant, fset *token.FileSet, pkg, filename string, src []byte, fn string, node ast.Node, operator, original, mutated, profile string, ignores []inlineIgnore) {
	if !operatorEnabled(operator, profile) {
		return
	}
	pos := fset.Position(node.Pos())
	if ignored(pos.Line, operator, ignores) {
		return
	}
	start := fset.Position(node.Pos()).Offset
	end := fset.Position(node.End()).Offset
	if start < 0 || end > len(src) || start >= end {
		return
	}
	mutatedSrc := append([]byte{}, src...)
	segment := string(src[start:end])
	next, err := replaceFirst(segment, original, mutated)
	if err != nil {
		return
	}
	mutatedSrc = append(mutatedSrc[:start], append([]byte(next), mutatedSrc[end:]...)...)
	diff := unifiedDiff(filename, string(src), string(mutatedSrc))
	fp := fingerprint(filename, strconv.Itoa(pos.Line), strconv.Itoa(start), strconv.Itoa(end), operator, original, mutated, diff)
	id := fmt.Sprintf("%s:%d:%s:%s", filename, pos.Line, operator, fp[:12])
	*mutants = append(*mutants, Mutant{
		ID:          id,
		Package:     pkg,
		File:        filename,
		Line:        pos.Line,
		Function:    fn,
		Operator:    operator,
		Original:    original,
		Mutated:     mutated,
		StartOffset: start,
		EndOffset:   end,
		Diff:        diff,
		Fingerprint: fp,
		Hint:        hint(operator),
		Description: description(fn, operator, original, mutated),
	})
}

func operatorEnabled(operator, profile string) bool {
	switch operator {
	case "conditionals", "logical", "boolean-literals", "nil-checks", "arithmetic-basic":
		return true
	case "error-returns":
		return profile == ProfileDefault || profile == ProfileAggressive
	case "literals", "returns", "loop-control":
		return profile == ProfileAggressive
	default:
		return false
	}
}

func ignored(line int, operator string, ignores []inlineIgnore) bool {
	for _, ignore := range ignores {
		if ignore.line == line && (ignore.operator == "*" || ignore.operator == operator) {
			return true
		}
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
	case "conditionals", "nil-checks":
		return "Add assertions for the opposite branch or boundary condition."
	case "logical":
		return "Add a test where only one side of the boolean expression changes the outcome."
	case "boolean-literals":
		return "Assert both boolean outcomes instead of only executing the path."
	case "arithmetic-basic":
		return "Add assertions for the computed numeric result and edge cases."
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
