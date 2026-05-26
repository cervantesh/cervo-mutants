# Mutation Tool Evaluation

Use this template with `docs/evaluation-framework.md`.

## Run Metadata

| Field | Value |
| --- | --- |
| Evaluation date | |
| Evaluator | |
| Tool | |
| Tool version/commit | |
| Target repository | |
| Target commit | |
| Go version | |
| OS / CPU count | |
| Command | |
| Config path | |
| Report path | |

## Automated Metrics

| Metric | Value |
| --- | ---: |
| Mutation score | |
| Total mutants | |
| Killed | |
| Survived | |
| Timed out | |
| Compile errors | |
| Skipped | |
| Cached | |
| Quarantined | |
| Runtime | |
| Cache hit rate | |
| Changed-scope mutant count | |
| Full/sampled mutant count | |

## Scorecard

| Layer | Score | Evidence level | Notes |
| --- | ---: | --- | --- |
| Tool capability | /20 | | |
| Fault-revealing effectiveness | /25 | | |
| CI and commit relevance | /15 | | |
| Actionability and agent utility | /15 | | |
| Cost and scalability | /8 | | |
| Noise and equivalent-mutant burden | /10 | | |
| Longitudinal and evolution relevance | /4 | | |
| Validity controls | /3 | | |
| **Total** | **/100** | | |

## Survivor Review

| Mutant ID | Status | Classification | Evidence level | Action |
| --- | --- | --- | --- | --- |
| | survived | useful / equivalent / redundant / invalid | preliminary / accepted / disputed | |

## Test Improvement Review

| Survivor | New test | Kills mutant | Detects realistic fault | Notes |
| --- | --- | --- | --- | --- |
| | | yes / no | yes / no | |

## Decision

Decision:

- [ ] candidate default
- [ ] needs review
- [ ] needs mutator tuning
- [ ] nightly only
- [ ] not default

Rationale:

## Required Follow-Up

- [ ] Create issue for mutator/profile changes.
- [ ] Create issue for equivalent-mutant suppression or quarantine cleanup.
- [ ] Create issue for flaky-test or CI instability.
- [ ] Create issue for tests discovered from actionable survivors.

