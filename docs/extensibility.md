# Extensibility

CervoMutants exposes programmatic extension seams for advanced users who want
to add custom mutation generation or tune review policy without forking the
entire core.

This is intentionally a library surface, not a plugin marketplace or a CLI
distribution protocol.

## Stability Goal

The built-in defaults remain the reference behavior:

- `engine.New(cfg)` still uses the stock generator, suppression evaluator, and
  survivor ranker.
- `engine.NewWithOptions(cfg, ...)` lets a custom binary or automation layer
  replace or compose those seams.
- report schema and CLI flags do not change when you use the defaults.

## Available Seams

### Mutant generation

Use `mutator.Generator` when you want to replace the built-in AST mutator set
or append extra generators around it.

Helpers:

- `mutator.DefaultGenerator()`
- `mutator.ChainGenerators(...)`
- `mutator.GeneratorFunc`

Example:

```go
customGenerator := mutator.GeneratorFunc(func(pkg, filename string, src []byte, profile string) ([]mutator.Mutant, error) {
    return []mutator.Mutant{
        {
            File:             filename,
            Line:             12,
            Operator:         "custom-op",
            Original:         "enabled",
            Mutated:          "disabled",
            StartOffset:      100,
            EndOffset:        107,
            Diff:             "--- file.go\n+++ file.go\n@@\n-enabled\n+disabled\n",
            Hint:             "Add a direct assertion for the custom behavior.",
            Description:      "Custom generator mutation.",
            EquivalentRisk:   "medium",
            Recommendation:   "custom",
            CompileErrorRisk: "low",
        },
    }, nil
})

engine.NewWithOptions(
    cfg,
    engine.WithMutantGenerator(mutator.ChainGenerators(
        mutator.DefaultGenerator(),
        customGenerator,
    )),
)
```

`ChainGenerators` deduplicates repeated mutants so wrappers can safely layer on
top of the built-in generator.

### Suppression evaluation

Use `engine.SuppressionEvaluator` when you want to add or replace suppression
audits before execution and reporting.

Helpers:

- `engine.DefaultSuppressionEvaluator(cfg)`
- `engine.ChainSuppressionEvaluators(...)`
- `engine.SuppressionEvaluatorFunc`

Example:

```go
customSuppression := engine.SuppressionEvaluatorFunc(func(mutant mutator.Mutant) []engine.SuppressionAudit {
    if mutant.Operator != "custom-op" {
        return nil
    }
    return []engine.SuppressionAudit{{
        Name:          "custom-op-review",
        Action:        "report-only",
        Reason:        "Custom operators stay visible until the review workflow matures.",
        EvidenceLevel: "heuristic",
    }}
})

engine.NewWithOptions(
    cfg,
    engine.WithSuppressionEvaluator(engine.ChainSuppressionEvaluators(
        engine.DefaultSuppressionEvaluator(cfg),
        customSuppression,
    )),
)
```

### Survivor ranking

Use `engine.SurvivorRanker` when you want to tune prioritization, actionability,
or next-test guidance after execution.

Helpers:

- `engine.DefaultSurvivorRanker()`
- `engine.SurvivorRankerFunc`

Example:

```go
baseRanker := engine.DefaultSurvivorRanker()

customRanker := engine.SurvivorRankerFunc(func(goos string, results []engine.MutantResult) []engine.SurvivorRanking {
    ranked := baseRanker.Rank(goos, results)
    for i := range ranked {
        if strings.Contains(ranked[i].MutantID, "critical") {
            ranked[i].RankScore += 50
            ranked[i].RankReason += " critical-package-boost"
        }
    }
    sort.SliceStable(ranked, func(i, j int) bool {
        return ranked[i].RankScore > ranked[j].RankScore
    })
    for i := range ranked {
        ranked[i].SurvivorRank = i + 1
    }
    return ranked
})

engine.NewWithOptions(cfg, engine.WithSurvivorRanker(customRanker))
```

## Guidance

- Prefer composition over replacement when you only need to add behavior.
- Keep custom operators additive and auditable in reports.
- Custom generators must emit patch-ready metadata such as `StartOffset`,
  `EndOffset`, and `Diff`, because execution still relies on deterministic file
  patching.
- Reuse the default ranker unless you have a concrete review policy that
  justifies diverging from the stock actionability model.
- If you distribute a custom binary, document which seams you overrode so
  report consumers understand why behavior differs from upstream defaults.
