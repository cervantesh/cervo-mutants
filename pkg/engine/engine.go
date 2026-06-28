package engine

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/cervantesh/cervo-mutants/pkg/config"
	"github.com/cervantesh/cervo-mutants/pkg/discover"
	"github.com/cervantesh/cervo-mutants/pkg/mutator"
)

type Engine struct {
	cfg                  config.Config
	mutantGenerator      mutator.Generator
	suppressionEvaluator SuppressionEvaluator
	survivorRanker       SurvivorRanker
	now                  func() time.Time
	deps                 engineDeps
}

func New(cfg config.Config) *Engine {
	return NewWithOptions(cfg)
}

func NewWithOptions(cfg config.Config, opts ...EngineOption) *Engine {
	e := &Engine{
		cfg:                  cfg,
		mutantGenerator:      mutator.DefaultGenerator(),
		suppressionEvaluator: DefaultSuppressionEvaluator(cfg),
		survivorRanker:       DefaultSurvivorRanker(),
		now:                  time.Now,
		deps:                 defaultEngineDeps(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}
	return e
}

func (e *Engine) clockNow() time.Time {
	if e != nil && e.now != nil {
		return e.now()
	}
	return time.Now()
}

func (e *Engine) elapsedSince(start time.Time) time.Duration {
	return e.clockNow().Sub(start)
}

func (e *Engine) Run(ctx context.Context, req RunRequest) (result RunResult, err error) {
	defer recoverEnginePanic("run", &err)

	targets := e.runTargets(req.Targets)
	session := e.newRunSession()
	mutants, err := e.deps.discoverMutants(e, targets)
	if err != nil {
		return RunResult{}, wrapStageError("discovery_error", err)
	}
	mutants, sliceMeta := e.applySlicing(mutants)
	session.sliceMeta = sliceMeta
	e.scheduleMutants(mutants)
	if e.cfg.Limits.MaxMutants > 0 && len(mutants) > e.cfg.Limits.MaxMutants {
		mutants = mutants[:e.cfg.Limits.MaxMutants]
	}
	session.setCheckpointScope(mutants)
	quarantined, expired, err := session.loadQuarantine()
	if err != nil {
		return RunResult{}, wrapStageError("environment_error", err)
	}
	result = RunResult{
		SchemaVersion: "1",
		Environment:   e.environment(len(mutants)),
		Slice:         sliceMeta,
		Checkpoint:    e.checkpoint(mutants, "final"),
		Thresholds:    configuredThresholds(e.cfg),
		Mutants:       []MutantResult{},
		Quarantine: QuarantineStats{
			Active:        len(quarantined),
			Expired:       expired,
			Path:          e.cfg.Quarantine.Path,
			ExpireAfter:   e.cfg.Quarantine.ExpireAfter.String(),
			RequireReason: e.cfg.Quarantine.RequireReason,
			RequireOwner:  e.cfg.Quarantine.RequireOwner,
			RequireIssue:  e.cfg.Quarantine.RequireIssue,
			FailOnExpired: e.cfg.Quarantine.FailOnExpired,
			MaxRenewals:   e.cfg.Quarantine.MaxRenewals,
		},
	}
	if req.DryRun {
		return e.dryRunResult(result, mutants), nil
	}
	baselineResult, err := e.deps.runBaseline(session, ctx, targets)
	if err != nil && e.cfg.Tests.BaselineRequired {
		return RunResult{}, wrapStageError("runner_error", err)
	}
	_ = baselineResult
	mutantResults, err := e.deps.runMutants(session, ctx, mutants, quarantined)
	if err != nil {
		return RunResult{}, wrapStageError("runner_error", err)
	}
	result.Mutants = mutantResults
	result.StoppedReason, result.LastCompletedMutant = runStopMetadata(result.Mutants)
	result.History = e.applyHistory(result.Mutants)
	e.applySurvivorRanking(result.Mutants)
	result.Summary = summarize(result.Mutants)
	result.Summary.NewSurvivors = result.History.NewSurvivors
	result.Summary.LongStandingSurvivors = result.History.LongStandingSurvivors
	if e.cfg.Baseline.Enabled {
		if prev, ok, err := session.loadBaseline(); err != nil {
			return RunResult{}, wrapStageError("environment_error", err)
		} else if ok {
			result.Baseline = compareBaseline(prev, result)
		} else {
			result.Baseline = BaselineComparison{Enabled: true, CurrentScore: result.Summary.Score}
		}
	}
	result.Gate = EvaluateGatePolicy(e.cfg, result)
	e.recordHistoryRun(&result)
	if err := e.deps.writeReports(e, result); err != nil {
		return RunResult{}, wrapStageError("environment_error", err)
	}
	return result, nil
}

func (e *Engine) Affected(ctx context.Context, req AffectedRequest) (result AffectedResult, err error) {
	defer recoverEnginePanic("affected", &err)
	discovered, err := discover.Discover(req.Targets)
	if err != nil {
		return AffectedResult{}, wrapStageError("discovery_error", err)
	}
	mutants, err := e.generateMutants(discovered)
	if err != nil {
		return AffectedResult{}, wrapStageError("discovery_error", err)
	}
	packages := map[string]bool{}
	files := map[string]bool{}
	for _, file := range discovered.Files {
		if file.IsTest {
			continue
		}
		packages[file.Package] = true
		files[file.Path] = true
	}
	result = AffectedResult{Modules: discovered.Modules, EstimatedMutants: len(mutants)}
	for pkg := range packages {
		result.Packages = append(result.Packages, pkg)
	}
	for file := range files {
		result.Files = append(result.Files, file)
	}
	return result, nil
}

func (e *Engine) Explain(ctx context.Context, req ExplainRequest) (result ExplainResult, err error) {
	defer recoverEnginePanic("explain", &err)
	if strings.TrimSpace(req.MutantID) == "" {
		return ExplainResult{}, errors.New("mutant id is required")
	}
	return ExplainResult{
		MutantID:    req.MutantID,
		Explanation: "This mutant changes program behavior. If tests still pass, the current suite executes the code without asserting the changed outcome.",
		Suggestion:  "Add an assertion that fails for the mutated expression and passes for the original behavior.",
	}, nil
}
