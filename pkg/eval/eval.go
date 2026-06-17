package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cervantesh/cervo-mutants/pkg/engine"
)

const (
	DecisionCandidateDefault = "candidate_default"
	DecisionNeedsReview      = "needs_review"
	DecisionNeedsTuning      = "needs_mutator_tuning"
	DecisionNightlyOnly      = "nightly_only"
	DecisionNotDefault       = "not_default"

	EvidenceMeasured       = "measured"
	EvidenceRequiresReview = "requires_review"
	EvidenceReviewed       = "reviewed"
	EvidenceReplicated     = "replicated"
	EvidenceLongitudinal   = "longitudinal"
)

type BuildRequest struct {
	Tool       string
	Target     string
	Commit     string
	Command    []string
	Framework  string
	Run        engine.RunResult
	ManualMode bool
}

type Evaluation struct {
	SchemaVersion        string           `json:"schema_version"`
	Framework            string           `json:"framework"`
	GeneratedAt          time.Time        `json:"generated_at"`
	Tool                 string           `json:"tool"`
	Target               string           `json:"target"`
	Commit               string           `json:"commit"`
	Command              []string         `json:"command"`
	Decision             string           `json:"decision"`
	Scorecard            Scorecard        `json:"scorecard"`
	Metrics              Metrics          `json:"metrics"`
	RequiredManualReview []string         `json:"required_manual_review"`
	RawReport            engine.RunResult `json:"raw_report"`
}

type Scorecard struct {
	Total           int       `json:"total"`
	ToolCapability  Criterion `json:"tool_capability"`
	FaultRevealing  Criterion `json:"fault_revealing_effectiveness"`
	CIRelevance     Criterion `json:"ci_and_commit_relevance"`
	Actionability   Criterion `json:"actionability_and_agent_utility"`
	CostScalability Criterion `json:"cost_and_scalability"`
	Noise           Criterion `json:"noise_and_equivalent_mutant_burden"`
	Longitudinal    Criterion `json:"longitudinal_and_evolution_relevance"`
	Validity        Criterion `json:"validity_controls"`
}

type Criterion struct {
	Score    int    `json:"score"`
	Max      int    `json:"max"`
	Evidence string `json:"evidence"`
	Reason   string `json:"reason"`
}

type Metrics struct {
	MutationScore                  float64 `json:"mutation_score"`
	TotalMutants                   int     `json:"total_mutants"`
	Killed                         int     `json:"killed"`
	Survived                       int     `json:"survived"`
	NotCovered                     int     `json:"not_covered"`
	TimedOut                       int     `json:"timed_out"`
	CompileError                   int     `json:"compile_error"`
	Skipped                        int     `json:"skipped"`
	Quarantined                    int     `json:"quarantined"`
	Cached                         int     `json:"cached"`
	TestEfficacy                   float64 `json:"test_efficacy"`
	MutationCoverage               float64 `json:"mutation_coverage"`
	CacheHitRate                   float64 `json:"cache_hit_rate"`
	ActionableSurvivorsReviewed    int     `json:"actionable_survivors_reviewed"`
	EquivalentReviewed             int     `json:"equivalent_reviewed"`
	UniqueActionableSurvivors      int     `json:"unique_actionable_survivors"`
	LongStandingSurvivorRate       float64 `json:"long_standing_survivor_rate"`
	EquivalentSuppressionPrecision float64 `json:"equivalent_suppression_precision"`
}

func Build(req BuildRequest) Evaluation {
	metrics := metricsFromRun(req.Run)
	scorecard := score(metrics, req.ManualMode)
	review := []string{
		"classify a representative survivor sample",
		"measure survivor-to-test yield",
		"confirm equivalent-mutant classifications with accepted evidence",
		"check whether new tests reveal historical or realistic faults",
		"compare against at least one other Go mutation tool when available",
		"track long-standing survivors across releases before default adoption",
	}
	decision := decide(scorecard, req.ManualMode)
	return Evaluation{
		SchemaVersion:        "1",
		Framework:            defaultString(req.Framework, "cervosoft"),
		GeneratedAt:          time.Now().UTC(),
		Tool:                 defaultString(req.Tool, "cervo-mutant"),
		Target:               req.Target,
		Commit:               req.Commit,
		Command:              req.Command,
		Decision:             decision,
		Scorecard:            scorecard,
		Metrics:              metrics,
		RequiredManualReview: review,
		RawReport:            req.Run,
	}
}

func Write(dir string, evaluation Evaluation) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(evaluation, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "evaluation.json"), data, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "evaluation.md"), []byte(Markdown(evaluation)), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "evaluation.schema.json"), []byte(Schema()), 0o644)
}

func Markdown(e Evaluation) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# CervoMutant Evaluation\n\n")
	fmt.Fprintf(&b, "- Tool: `%s`\n- Target: `%s`\n- Commit: `%s`\n- Decision: `%s`\n- Score: `%d/100`\n\n", e.Tool, e.Target, e.Commit, e.Decision, e.Scorecard.Total)
	fmt.Fprintf(&b, "## Scorecard\n\n")
	fmt.Fprintf(&b, "| Layer | Score | Evidence |\n| --- | ---: | --- |\n")
	rows := []struct {
		name string
		c    Criterion
	}{
		{"Tool capability", e.Scorecard.ToolCapability},
		{"Fault-revealing effectiveness", e.Scorecard.FaultRevealing},
		{"CI and commit relevance", e.Scorecard.CIRelevance},
		{"Actionability and agent utility", e.Scorecard.Actionability},
		{"Cost and scalability", e.Scorecard.CostScalability},
		{"Noise and equivalent-mutant burden", e.Scorecard.Noise},
		{"Longitudinal and evolution relevance", e.Scorecard.Longitudinal},
		{"Validity controls", e.Scorecard.Validity},
	}
	for _, row := range rows {
		fmt.Fprintf(&b, "| %s | %d/%d | %s |\n", row.name, row.c.Score, row.c.Max, row.c.Evidence)
	}
	fmt.Fprintf(&b, "\n## Metrics\n\n")
	fmt.Fprintf(&b, "- Mutation score: %.2f\n- Test efficacy: %.2f\n- Mutation coverage: %.2f\n- Total mutants: %d\n- Killed: %d\n- Survived: %d\n- Not covered: %d\n- Cached: %d\n- Quarantined: %d\n\n", e.Metrics.MutationScore, e.Metrics.TestEfficacy, e.Metrics.MutationCoverage, e.Metrics.TotalMutants, e.Metrics.Killed, e.Metrics.Survived, e.Metrics.NotCovered, e.Metrics.Cached, e.Metrics.Quarantined)
	fmt.Fprintf(&b, "## Required Manual Review\n\n")
	for _, item := range e.RequiredManualReview {
		fmt.Fprintf(&b, "- [ ] %s\n", item)
	}
	return b.String()
}

func Schema() string {
	return `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://github.com/cervantesh/cervo-mutants/schemas/evaluation.schema.json",
  "title": "CervoMutant Evaluation",
  "type": "object",
  "required": ["schema_version", "framework", "tool", "target", "decision", "scorecard", "metrics", "required_manual_review"],
  "properties": {
    "schema_version": {"const": "1"},
    "framework": {"type": "string"},
    "tool": {"type": "string"},
    "target": {"type": "string"},
    "commit": {"type": "string"},
    "decision": {"enum": ["candidate_default", "needs_review", "needs_mutator_tuning", "nightly_only", "not_default"]},
    "scorecard": {"type": "object"},
    "metrics": {"type": "object"},
    "required_manual_review": {"type": "array", "items": {"type": "string"}}
  }
}
`
}

func metricsFromRun(run engine.RunResult) Metrics {
	totalCache := run.Cache.Hits + run.Cache.Misses
	cacheHitRate := 0.0
	if totalCache > 0 {
		cacheHitRate = float64(run.Cache.Hits) / float64(totalCache)
	}
	return Metrics{
		MutationScore:    run.Summary.Score,
		TotalMutants:     run.Summary.Total,
		Killed:           run.Summary.Killed,
		Survived:         run.Summary.Survived,
		NotCovered:       run.Summary.NotCovered,
		TimedOut:         run.Summary.TimedOut,
		CompileError:     run.Summary.CompileError,
		Skipped:          run.Summary.Skipped,
		Quarantined:      run.Summary.Quarantined,
		Cached:           run.Summary.Cached,
		TestEfficacy:     run.Summary.TestEfficacy,
		MutationCoverage: run.Summary.MutationCoverage,
		CacheHitRate:     cacheHitRate,
	}
}

func score(metrics Metrics, manualMode bool) Scorecard {
	s := Scorecard{
		ToolCapability:  Criterion{Score: 14, Max: 20, Evidence: EvidenceMeasured, Reason: "CLI, reports, schema, and mutation flow are measurable from the run."},
		FaultRevealing:  Criterion{Score: 0, Max: 25, Evidence: EvidenceRequiresReview, Reason: "Requires survivor-to-test and real-fault review."},
		CIRelevance:     Criterion{Score: ciScore(metrics), Max: 15, Evidence: EvidenceMeasured, Reason: "Estimated from run completion, cache, skips, and reportability."},
		Actionability:   Criterion{Score: actionabilityScore(metrics), Max: 15, Evidence: EvidenceMeasured, Reason: "Estimated from survivor context available in JSON report."},
		CostScalability: Criterion{Score: costScore(metrics), Max: 8, Evidence: EvidenceMeasured, Reason: "Estimated from cache/skips/time-budget observable fields."},
		Noise:           Criterion{Score: 0, Max: 10, Evidence: EvidenceRequiresReview, Reason: "Requires manual equivalent/redundancy classification."},
		Longitudinal:    Criterion{Score: 0, Max: 4, Evidence: EvidenceRequiresReview, Reason: "Requires cross-release history."},
		Validity:        Criterion{Score: 1, Max: 3, Evidence: EvidenceMeasured, Reason: "Baseline and deterministic configuration are partially measurable."},
	}
	if !manualMode {
		s.FaultRevealing.Score = 0
	}
	s.Total = s.ToolCapability.Score + s.FaultRevealing.Score + s.CIRelevance.Score + s.Actionability.Score + s.CostScalability.Score + s.Noise.Score + s.Longitudinal.Score + s.Validity.Score
	return s
}

func ciScore(metrics Metrics) int {
	score := 8
	if metrics.TotalMutants > 0 {
		score += 2
	}
	if metrics.Skipped == 0 {
		score += 2
	}
	if metrics.TimedOut == 0 && metrics.CompileError == 0 {
		score += 3
	}
	if score > 15 {
		return 15
	}
	return score
}

func actionabilityScore(metrics Metrics) int {
	score := 8
	if metrics.Survived > 0 {
		score += 4
	}
	if metrics.TotalMutants > 0 {
		score += 3
	}
	if score > 15 {
		return 15
	}
	return score
}

func costScore(metrics Metrics) int {
	score := 4
	if metrics.Cached > 0 || metrics.CacheHitRate > 0 {
		score += 2
	}
	if metrics.Skipped == 0 {
		score += 2
	}
	if score > 8 {
		return 8
	}
	return score
}

func decide(scorecard Scorecard, manualMode bool) string {
	if manualMode {
		return DecisionNeedsReview
	}
	if scorecard.Total >= 80 && scorecard.FaultRevealing.Score >= 18 && scorecard.Actionability.Score >= 11 {
		return DecisionCandidateDefault
	}
	if scorecard.Noise.Score < 5 {
		return DecisionNeedsTuning
	}
	return DecisionNeedsReview
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
