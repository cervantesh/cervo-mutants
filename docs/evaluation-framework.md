# CervoMutant Evaluation Framework

Tracking issue: https://gitea.cervbox.synology.me/CervoSoft/cervo-mutant/issues/5

This framework evaluates mutation testing tools for CervoSoft and decides
whether `cervo-mutant` should be the default for CervoClaw and related Go
libraries. It combines published mutation-testing tool comparison work with
engineering criteria needed for CI, large Go modules, and agent-driven test
improvement.

The decisive question is:

> Does the tool produce actionable survivors, with low noise, in acceptable
> time, and do those survivors lead to tests that detect real or realistic
> faults?

## Research Basis

Use these studies as the methodological baseline:

- **How Do Java Mutation Tools Differ?**  
  CACM tool-comparison framework. Use its five dimensions: tool version,
  deployment, mutation process, user-centric features, and mutation operators.
- **How effective are mutation testing tools?**  
  Empirical Software Engineering study using real faults and manual mutant
  analysis. Use it to judge fault-revealing effectiveness instead of relying on
  mutation score alone.
- **On the use of commit-relevant mutants**  
  Empirical Software Engineering study on commit-scoped mutation. Use it to
  judge CI and pull-request usefulness.
- **Selecting fault revealing mutants**  
  Empirical Software Engineering study on reducing mutant sets while preserving
  fault-revealing power. Use it for mutator profiles, sampling, and selection.
- **Mutation Testing Advances: An Analysis and Survey**  
  Use as a threats-to-validity checklist for equivalent mutants, flaky tests,
  sampling bias, operator bias, and over-reliance on mutation score.

These studies are technology-independent at the evaluation level. They should
not be copied as Java/JVM implementation details. Translate their concepts into
Go terms: classes become packages/files/functions, JUnit tests become `go test`
commands, bytecode mutants become AST/source patches, and build modules become
Go modules.

## Scorecard

Score each tool out of 100. Keep per-layer scores visible; do not hide a weak
layer behind a strong total score.

| Layer | Weight |
| --- | ---: |
| Tool capability | 20 |
| Fault-revealing effectiveness | 25 |
| CI and commit relevance | 15 |
| Actionability and agent utility | 15 |
| Cost and scalability | 10 |
| Noise and equivalent-mutant burden | 10 |
| Validity controls | 5 |

### Tool Capability - 20

Based on the CACM five-dimension framework.

| Criterion | Points | Evidence |
| --- | ---: | --- |
| Versioning and reproducibility | 3 | Version, config, schema, and toolchain are recorded in reports. |
| Deployment and CI setup | 4 | Install command, CI command, exit codes, and no hidden services required. |
| Mutation process | 5 | Clear discovery, mutation, isolation, selection, execution, and classification flow. |
| User-centric features | 4 | `affected`, `show`, `explain`, `report survivors`, HTML, JSON, JUnit. |
| Mutation operators | 4 | Operators are documented by profile, risk, node type, and example diff. |

### Fault-Revealing Effectiveness - 25

Based on real-fault and manual-analysis studies.

| Criterion | Points | Evidence |
| --- | ---: | --- |
| Real or realistic fault detection | 8 | Tests added from survivors detect historical or seeded faults. |
| Survivor-to-test yield | 5 | Percentage of survivors that lead to useful new assertions. |
| Oracle-strength improvement | 4 | New tests fail on the mutant and pass on the original code. |
| Manual mutant analysis | 4 | Sampled mutants classified as useful, redundant, equivalent, or invalid. |
| Complementarity | 4 | Tool finds useful gaps missed by coverage, existing tests, or other mutation tools. |

### CI And Commit Relevance - 15

Based on commit-relevant mutant research.

| Criterion | Points | Evidence |
| --- | ---: | --- |
| Changed-scope correctness | 4 | `affected --scope changed` matches changed modules/packages/files. |
| Cost reduction versus full run | 4 | PR run time and mutant count are materially lower than full mutation. |
| Signal preservation | 3 | Commit-scoped runs still identify relevant survivors. |
| Feedback latency | 2 | PR feedback arrives within the chosen CI budget. |
| False-negative control | 2 | Periodic full runs or sampling detect what changed-scope may miss. |

### Actionability And Agent Utility - 15

This layer is CervoSoft-specific because `cervo-mutant` is intended for both
humans and coding agents.

| Criterion | Points | Evidence |
| --- | ---: | --- |
| Stable JSON contract | 3 | `schema_version`, documented enums, additive-only v1 fields, golden tests. |
| Survivor context | 3 | File, line, function, package, operator, diff, selected tests, output, reason. |
| `show` and `explain` usefulness | 3 | A survivor can be turned into a concrete test task without extra discovery. |
| Determinism | 2 | Same seed, config, and inputs produce stable mutant IDs and ordering. |
| Low-context reports | 2 | Reports are compact enough for agents while retaining necessary evidence. |
| Grouping and prioritization | 2 | Survivors can be sorted by package, operator, changed code, or prior history. |

### Cost And Scalability - 10

| Criterion | Points | Evidence |
| --- | ---: | --- |
| Time budget support | 2 | `--budget` stops work predictably and reports skipped mutants. |
| Incremental cache value | 2 | Cache hit rate and time saved are reported and reproducible. |
| Test selection value | 2 | Package/coverage selection reduces test work without losing signal. |
| Workspace isolation cost | 2 | Temp-workdir overhead is measured, especially on Windows/OneDrive. |
| Large-project threshold behavior | 2 | Behavior is measured at 100+ packages, 5+ minute tests, or 1,000+ mutants. |

### Noise And Equivalent-Mutant Burden - 10

| Criterion | Points | Evidence |
| --- | ---: | --- |
| Equivalent rate | 3 | Manual sample estimates equivalent survivors by mutator/profile. |
| Triage time | 2 | Median time to classify or act on a survivor. |
| Redundancy | 2 | Mutants killed by the same test or producing same signal are measured. |
| Quarantine discipline | 2 | Quarantine requires reason, owner, issue, expiry, and does not inflate score. |
| Repeat offenders | 1 | Noisy operators are identified and moved to stricter profiles or disabled. |

### Validity Controls - 5

Use this layer to keep evaluation claims defensible.

| Criterion | Points | Evidence |
| --- | ---: | --- |
| Baseline stability | 1 | Baseline tests pass repeatedly before mutation. |
| Flaky-test control | 1 | Timeouts/flaky failures are rerun or classified separately. |
| Sampling validity | 1 | Sampling method and seed are recorded. |
| Operator bias control | 1 | Results are reported per mutator profile/operator. |
| Threats documented | 1 | Known limitations are recorded with their likely impact. |

## Required Metrics

Collect these metrics for every evaluation run:

- mutation score and effective score excluding active quarantine
- total, killed, survived, timed out, compile error, skipped, ignored,
  quarantined, and cached mutants
- runtime, budget usage, cache hits, cache misses, and test-selection mode
- number of survivors converted into tests
- number of survivors classified as equivalent, redundant, invalid, or useful
- cost per killed mutant
- cost per actionable survivor
- cost per real or realistic fault revealed
- quarantine active count, expired count, and growth over time
- changed-scope mutant count versus full-run mutant count

Do not use global mutation score as the primary success criterion. It is a
supporting metric, not the decision rule.

## Empirical Protocol

Use two evaluation levels.

### Level 1: Tool Comparison

Compare `cervo-mutant` against available Go mutation tools such as Gremlins,
go-mutesting variants, and gomu when they can run on the same target.

For each tool:

1. Record version, command, config, Go version, OS, CPU count, and commit SHA.
2. Run baseline tests repeatedly until stable.
3. Run mutation on the same packages with comparable timeout and budget.
4. Export raw reports and normalize results into the scorecard.
5. Manually classify a representative survivor sample.
6. Record setup friction, failures, unsupported cases, and workspace artifacts.

### Level 2: CervoClaw Adoption Study

Use CervoClaw and at least one CervoSoft library as target projects.

For each target:

1. Run `cervomut affected ./... --scope changed --since origin/main`.
2. Run `cervomut run ./... --scope changed --since origin/main --budget <N>`.
3. Run a full or sampled baseline mutation run.
4. Select survivors from changed-scope and full/sampled runs.
5. Ask a human or agent to write tests from survivor reports.
6. Verify each new test fails against the mutant and passes against original
   code.
7. Check whether those tests detect historical bugs, seeded realistic bugs, or
   realistic contract violations.
8. Record time spent, false starts, equivalent mutants, and test value.

## Acceptance Guidance

`cervo-mutant` can be considered the default only if:

- it scores at least 80 overall,
- it scores at least 18/25 on fault-revealing effectiveness,
- it scores at least 11/15 on actionability and agent utility,
- it does not create untracked artifacts in target workspaces when `--out` is
  used,
- quarantine does not grow without issue-backed cleanup,
- and survivors regularly lead to useful tests rather than mostly equivalent or
  redundant findings.

Reject or defer default adoption if the tool is fast but not actionable, has a
high equivalent-mutant burden, produces unstable CI results, or improves
mutation score without improving real-fault detection.

