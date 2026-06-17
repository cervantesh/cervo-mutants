package engine

import (
	"runtime"
	"strconv"
	"strings"

	"github.com/cervantesh/cervo-mutants/pkg/config"
	"github.com/cervantesh/cervo-mutants/pkg/mutator"
	"github.com/cervantesh/cervo-mutants/pkg/triage"
)

type EngineOption func(*Engine)

type SuppressionEvaluator interface {
	Evaluate(mutant mutator.Mutant) []SuppressionAudit
}

type SuppressionEvaluatorFunc func(mutant mutator.Mutant) []SuppressionAudit

func (f SuppressionEvaluatorFunc) Evaluate(mutant mutator.Mutant) []SuppressionAudit {
	if f == nil {
		return nil
	}
	return f(mutant)
}

type SurvivorRanking struct {
	MutantID            string
	SurvivorRank        int
	RankScore           float64
	RankReason          string
	Actionability       string
	SuggestedTestScope  string
	TestRecommendation  *TestRecommendation
	SuggestedSkipReason string
	SemanticGroupSize   int
	NearestTests        []string
}

type SurvivorRanker interface {
	Rank(goos string, results []MutantResult) []SurvivorRanking
}

type SurvivorRankerFunc func(goos string, results []MutantResult) []SurvivorRanking

func (f SurvivorRankerFunc) Rank(goos string, results []MutantResult) []SurvivorRanking {
	if f == nil {
		return nil
	}
	return f(goos, results)
}

func WithMutantGenerator(generator mutator.Generator) EngineOption {
	return func(e *Engine) {
		if generator != nil {
			e.mutantGenerator = generator
		}
	}
}

func WithSuppressionEvaluator(evaluator SuppressionEvaluator) EngineOption {
	return func(e *Engine) {
		if evaluator != nil {
			e.suppressionEvaluator = evaluator
		}
	}
}

func WithSurvivorRanker(ranker SurvivorRanker) EngineOption {
	return func(e *Engine) {
		if ranker != nil {
			e.survivorRanker = ranker
		}
	}
}

func DefaultSuppressionEvaluator(cfg config.Config) SuppressionEvaluator {
	if !cfg.Suppression.Enabled {
		return SuppressionEvaluatorFunc(func(mutator.Mutant) []SuppressionAudit { return nil })
	}
	rules := append([]config.SuppressionRule{}, cfg.Suppression.Rules...)
	return SuppressionEvaluatorFunc(func(mutant mutator.Mutant) []SuppressionAudit {
		audits := make([]SuppressionAudit, 0, len(rules))
		for _, rule := range rules {
			if !suppressionRuleMatches(rule, mutant) {
				continue
			}
			audits = append(audits, suppressionAuditFromRule(rule))
		}
		return audits
	})
}

func ChainSuppressionEvaluators(evaluators ...SuppressionEvaluator) SuppressionEvaluator {
	return SuppressionEvaluatorFunc(func(mutant mutator.Mutant) []SuppressionAudit {
		combined := make([]SuppressionAudit, 0)
		seen := map[string]bool{}
		for _, evaluator := range evaluators {
			if evaluator == nil {
				continue
			}
			for _, audit := range evaluator.Evaluate(mutant) {
				key := suppressionAuditKey(audit)
				if seen[key] {
					continue
				}
				seen[key] = true
				combined = append(combined, audit)
			}
		}
		return combined
	})
}

func DefaultSurvivorRanker() SurvivorRanker {
	return SurvivorRankerFunc(defaultSurvivorRankings)
}

func suppressionAuditKey(audit SuppressionAudit) string {
	return strings.Join([]string{
		audit.Name,
		audit.Action,
		audit.Reason,
		audit.EvidenceLevel,
		strconv.Itoa(audit.ReviewerCount),
	}, "\x00")
}

func defaultSurvivorRankings(goos string, results []MutantResult) []SurvivorRanking {
	ranked := triage.RankSurvivors(goos, triageResults(results))
	out := make([]SurvivorRanking, 0, len(ranked))
	for _, survivor := range ranked {
		var recommendation *TestRecommendation
		if survivor.TestRecommendation != nil {
			recommendation = &TestRecommendation{
				Priority:            survivor.TestRecommendation.Priority,
				Strategy:            survivor.TestRecommendation.Strategy,
				Summary:             survivor.TestRecommendation.Summary,
				CandidateTests:      append([]string{}, survivor.TestRecommendation.CandidateTests...),
				SuggestedAssertions: append([]string{}, survivor.TestRecommendation.SuggestedAssertions...),
				Rationale:           append([]string{}, survivor.TestRecommendation.Rationale...),
			}
		}
		out = append(out, SurvivorRanking{
			MutantID:            survivor.MutantID,
			SurvivorRank:        survivor.SurvivorRank,
			RankScore:           survivor.RankScore,
			RankReason:          survivor.RankReason,
			Actionability:       survivor.Actionability,
			SuggestedTestScope:  survivor.SuggestedTestScope,
			TestRecommendation:  recommendation,
			SuggestedSkipReason: survivor.SuggestedSkip,
			SemanticGroupSize:   survivor.SemanticGroupSize,
			NearestTests:        append([]string{}, survivor.NearestTests...),
		})
	}
	return out
}

func applySurvivorRankings(results []MutantResult, rankings []SurvivorRanking) {
	byID := make(map[string]SurvivorRanking, len(rankings))
	for _, ranking := range rankings {
		byID[ranking.MutantID] = ranking
	}
	for i := range results {
		ranking, ok := byID[results[i].MutantID]
		if !ok {
			continue
		}
		results[i].SurvivorRank = ranking.SurvivorRank
		results[i].RankScore = ranking.RankScore
		results[i].RankReason = ranking.RankReason
		results[i].Actionability = ranking.Actionability
		results[i].SuggestedTestScope = ranking.SuggestedTestScope
		if ranking.TestRecommendation != nil {
			results[i].TestRecommendation = &TestRecommendation{
				Priority:            ranking.TestRecommendation.Priority,
				Strategy:            ranking.TestRecommendation.Strategy,
				Summary:             ranking.TestRecommendation.Summary,
				CandidateTests:      append([]string{}, ranking.TestRecommendation.CandidateTests...),
				SuggestedAssertions: append([]string{}, ranking.TestRecommendation.SuggestedAssertions...),
				Rationale:           append([]string{}, ranking.TestRecommendation.Rationale...),
			}
		}
		results[i].SuggestedSkipReason = ranking.SuggestedSkipReason
		results[i].SemanticGroupSize = ranking.SemanticGroupSize
		results[i].NearestTests = append([]string{}, ranking.NearestTests...)
	}
}

func (e *Engine) applySurvivorRanking(results []MutantResult) {
	ranker := e.survivorRanker
	if ranker == nil {
		ranker = DefaultSurvivorRanker()
	}
	applySurvivorRankings(results, ranker.Rank(runtime.GOOS, results))
}
