package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/cervantesh/cervo-mutants/pkg/config"
	"github.com/cervantesh/cervo-mutants/pkg/discover"
	"github.com/cervantesh/cervo-mutants/pkg/mutator"
)

func (e *Engine) generateMutants(discovered discover.Result) ([]Mutant, error) {
	var mutants []Mutant
	for _, file := range discovered.Files {
		if file.IsTest {
			continue
		}
		data, err := os.ReadFile(file.Path)
		if err != nil {
			return nil, err
		}
		generated, err := e.mutantGenerator.Generate(file.Package, file.Path, data, e.cfg.Mutators.Profile)
		if err != nil {
			return nil, err
		}
		for i := range generated {
			generated[i].Module = file.ModuleDir
			generated[i].Package = file.Package
			id, fingerprint := e.stableMutantIdentity(generated[i])
			mutants = append(mutants, Mutant{
				ID:                  id,
				Module:              generated[i].Module,
				Package:             generated[i].Package,
				File:                generated[i].File,
				Line:                generated[i].Line,
				Function:            generated[i].Function,
				Operator:            generated[i].Operator,
				Original:            generated[i].Original,
				Mutated:             generated[i].Mutated,
				StartOffset:         generated[i].StartOffset,
				EndOffset:           generated[i].EndOffset,
				Diff:                generated[i].Diff,
				Fingerprint:         fingerprint,
				Hint:                generated[i].Hint,
				Description:         generated[i].Description,
				NearbyTests:         nearbyTests(file.ModuleDir, file.Path),
				EquivalentRisk:      generated[i].EquivalentRisk,
				Recommendation:      generated[i].Recommendation,
				CompileErrorRisk:    generated[i].CompileErrorRisk,
				SemanticTags:        append([]string{}, generated[i].SemanticTags...),
				SemanticGroup:       generated[i].SemanticGroup,
				GroupLabel:          generated[i].GroupLabel,
				GroupReason:         generated[i].GroupReason,
				PlatformSensitive:   generated[i].PlatformSensitive,
				NonProgressRisk:     generated[i].NonProgressRisk,
				Ownership:           e.ownershipRoute(generated[i].Package, generated[i].File),
				SuggestedSkipReason: generated[i].SuggestedSkipReason,
				SuppressionAudit:    e.suppressionAudit(generated[i]),
			})
		}
	}
	return mutants, nil
}

func (e *Engine) scheduleMutants(mutants []Mutant) {
	sort.SliceStable(mutants, func(i, j int) bool {
		if e.cfg.Execution.Budget > 0 {
			left := recommendationPriority(mutants[i].Recommendation)
			right := recommendationPriority(mutants[j].Recommendation)
			if left != right {
				return left < right
			}
			left = timeoutRiskPriority(mutants[i])
			right = timeoutRiskPriority(mutants[j])
			if left != right {
				return left < right
			}
			left = platformSensitivityPriority(mutants[i])
			right = platformSensitivityPriority(mutants[j])
			if left != right {
				return left < right
			}
		}
		return mutants[i].ID < mutants[j].ID
	})
}

func (e *Engine) suppressionAudit(mutant mutator.Mutant) []SuppressionAudit {
	if e.suppressionEvaluator == nil {
		return nil
	}
	return e.suppressionEvaluator.Evaluate(mutant)
}

func suppressionRuleMatches(rule config.SuppressionRule, mutant mutator.Mutant) bool {
	return optionalMatch(rule.Operator, mutant.Operator) &&
		optionalMatch(rule.EquivalentRisk, mutant.EquivalentRisk) &&
		(rule.File == "" || suppressionFileMatches(rule.File, mutant.File)) &&
		optionalMatch(rule.Original, mutant.Original) &&
		optionalMatch(rule.Mutated, mutant.Mutated)
}

func optionalMatch(want, got string) bool {
	return want == "" || want == got
}

func suppressionAuditFromRule(rule config.SuppressionRule) SuppressionAudit {
	evidenceLevel := "suspected"
	if rule.Evidence != "" {
		evidenceLevel = rule.Evidence
	}
	if rule.Action == "suppress" {
		evidenceLevel = "rule-suppressed"
	}
	return SuppressionAudit{Name: rule.Name, Action: rule.Action, Reason: rule.Reason, EvidenceLevel: evidenceLevel, ReviewerCount: rule.Reviewers}
}

func suppressionFileMatches(pattern, file string) bool {
	file = filepath.ToSlash(file)
	pattern = filepath.ToSlash(pattern)
	if ok, err := filepath.Match(pattern, file); err == nil && ok {
		return true
	}
	return file == pattern || strings.HasSuffix(file, "/"+pattern)
}

func nearbyTests(moduleDir, sourceFile string) []string {
	dir := filepath.Dir(sourceFile)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var tests []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		rel, err := filepath.Rel(moduleDir, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			rel = path
		}
		tests = append(tests, filepath.ToSlash(rel))
	}
	sort.Strings(tests)
	return tests
}

func (e *Engine) stableMutantIdentity(mutant mutator.Mutant) (string, string) {
	rel, err := filepath.Rel(mutant.Module, mutant.File)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		rel = filepath.Base(mutant.File)
	}
	rel = filepath.ToSlash(rel)
	fp := digestBytes([]byte(strings.Join([]string{
		rel,
		strconv.Itoa(mutant.Line),
		strconv.Itoa(mutant.StartOffset),
		strconv.Itoa(mutant.EndOffset),
		mutant.Operator,
		mutant.Original,
		mutant.Mutated,
		mutant.Diff,
	}, "\x00")))
	return fmt.Sprintf("%s:%d:%s:%s", rel, mutant.Line, mutant.Operator, fp[:12]), fp
}
