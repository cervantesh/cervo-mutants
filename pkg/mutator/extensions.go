package mutator

import (
	"strconv"
	"strings"
)

type Generator interface {
	Generate(pkg, filename string, src []byte, profile string) ([]Mutant, error)
}

type GeneratorFunc func(pkg, filename string, src []byte, profile string) ([]Mutant, error)

func (f GeneratorFunc) Generate(pkg, filename string, src []byte, profile string) ([]Mutant, error) {
	return f(pkg, filename, src, profile)
}

func DefaultGenerator() Generator {
	return GeneratorFunc(Generate)
}

func ChainGenerators(generators ...Generator) Generator {
	return GeneratorFunc(func(pkg, filename string, src []byte, profile string) ([]Mutant, error) {
		combined := make([]Mutant, 0)
		seen := map[string]bool{}
		for _, generator := range generators {
			if generator == nil {
				continue
			}
			mutants, err := generator.Generate(pkg, filename, src, profile)
			if err != nil {
				return nil, err
			}
			for _, mutant := range mutants {
				key := generatorMutantKey(mutant)
				if seen[key] {
					continue
				}
				seen[key] = true
				combined = append(combined, mutant)
			}
		}
		return combined, nil
	})
}

func generatorMutantKey(mutant Mutant) string {
	if hasCanonicalMutantKey(mutant) {
		return strings.Join([]string{
			mutant.File,
			strconv.Itoa(mutant.Line),
			mutant.Operator,
			mutant.Original,
			mutant.Mutated,
			strconv.Itoa(mutant.StartOffset),
			strconv.Itoa(mutant.EndOffset),
		}, "\x00")
	}
	if value := strings.TrimSpace(mutant.ID); value != "" {
		return "id:" + value
	}
	return ""
}

func hasCanonicalMutantKey(mutant Mutant) bool {
	return strings.TrimSpace(mutant.File) != "" ||
		mutant.Line != 0 ||
		strings.TrimSpace(mutant.Operator) != "" ||
		strings.TrimSpace(mutant.Original) != "" ||
		strings.TrimSpace(mutant.Mutated) != "" ||
		mutant.StartOffset != 0 ||
		mutant.EndOffset != 0
}
