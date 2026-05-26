# Signal-First Mutation Testing Framework

Tracking issue: https://gitea.cervbox.synology.me/CervoSoft/cervo-mutant/issues/10

This document captures the non-operator lessons CervoMutant should reuse from
Gremlins, recent mutation-testing research, and the Cobra comparison study.
Operator selection matters, but it is only one part of making mutation testing
useful in CI and useful for coding agents.

Every major design rule below lists its supporting references explicitly. The
goal is to make future implementation work auditable: if a behavior is adopted
because of research, the relevant source should be visible near the decision,
not only in a bibliography at the end.

The central product rule is:

> Optimize for actionable signal per minute of CI and per minute of human or
> agent attention, not for raw mutant volume or a single mutation score.

## Sources And What We Reuse

| Source | What it contributes | CervoMutant decision |
| --- | --- | --- |
| Gremlins documentation | Coverage-first execution, explicit `not covered`, separate test efficacy and mutant coverage, simple CI thresholds, concise output. | Keep coverage-gated execution and separate efficacy/coverage metrics as first-class concepts. |
| Practical mutation testing at scale / Google | Mutants are most valuable as actionable developer guidance. Large systems need selection, ranking, and limited surfaced work rather than all possible mutants. | Prioritize `affected`, baseline comparison, budgets, history, survivor ranking, and compact agent-ready reports. |
| Equivalent Mutants in the Wild, ISSTA 2024 | Many equivalent mutants arise from common patterns and can be suppressed with targeted analyses. | Add AST/pattern-based suppression rules with sampled false-suppression checks. |
| Mutation Coverage is Not Strongly Correlated with Mutation Coverage, AST 2024 | Mutation scores can vary sharply based on mutant generator and operator set. | Never compare scores without reporting profile, operators, not-covered counts, invalid counts, and equivalent-risk context. |
| Manually Created Equivalent Mutants, 2024 | Humans also create and misclassify equivalent mutants. | Do not rely on a single reviewer. Store evidence level and reviewer agreement for equivalence decisions. |
| Static detection of equivalent mutants, 2022 | Some equivalence/subsumption conditions can be reasoned about before execution. | Every mutator should eventually define `generate`, `suppress`, `subsumed_by`, and `risk` rules. |
| Ratio of Equivalent Mutants, JSS 2021 | Equivalent risk depends on both program structure and mutation operators. | Track equivalent-risk by repository, package, and operator rather than hard-coding one universal risk. |
| Commit-relevant mutants research | Mutants close to a change are more relevant for pull-request feedback than all mutants in a codebase. | Make changed-scope execution and periodic full-run backstops part of CI design. |
| PIT-style history and prioritization | History, timing, and previous outcomes reduce cost while preserving useful feedback. | Treat cache, timing history, and survivor history as core scheduling inputs. |
| Cobra comparison study | CervoMutant is fast with overlay/parallel execution, but Gremlins has cleaner coverage semantics and a less noisy default. | Keep the speed architecture, but improve coverage prefiltering, score decomposition, and default profile governance. |

## Pillars

### 1. Coverage-Gated Semantics

References: Gremlins features and limitations; Gremlins `unleash` metrics;
Google practical mutation testing at scale.

Coverage is not a quality score by itself, but execution is a prerequisite for a
test to kill a mutant. If tests never execute the mutated code, the result should
not be reported as a survivor.

CervoMutant should classify:

| Status | Meaning |
| --- | --- |
| `not_covered` | The mutated code was not executed by selected tests. |
| `survived` | The mutated code was executed and tests still passed. |
| `killed` | A selected test observed the behavioral change. |
| `compile_error` | The mutant did not produce a testable program. |
| `timed_out` | Execution exceeded the configured timeout. |
| `not_viable` | The mutant is syntactically valid as a patch idea but not viable for execution. |

Design implications:

- `coverage` mode should be stable enough for CI, not only an experimental
  optimization.
- When coverage data exists, package mode should still be able to mark obvious
  `not_covered` mutants.
- Mutation coverage and test efficacy must stay separate:

```text
mutation_coverage = (killed + survived) / (killed + survived + not_covered)
test_efficacy     = killed / (killed + survived)
```

### 2. Score Decomposition

References: Mutation Coverage is Not Strongly Correlated with Mutation Coverage,
AST 2024; Gremlins `unleash` metrics; The ratio of equivalent mutants, JSS
2021.

A single mutation score is too easy to misread. CervoMutant reports must always
show the components behind the score.

Required score context:

- mutator profile
- operator counts
- killed/survived/not-covered per operator
- compile errors and timeouts
- equivalent-risk estimates
- active quarantine and expired quarantine
- selected tests and selection reason
- cache hit/miss counts

Product rule:

> A score without generator context is not an adoption decision.

### 3. Actionability Before Volume

References: Practical mutation testing at scale / Google; An Empirical
Evaluation of Manually Created Equivalent Mutants, 2024; Cobra comparison study.

A survivor is valuable when it can guide a concrete test improvement. Large
reports full of weak or redundant survivors are worse than smaller reports with
clear next actions.

Each survivor should answer:

- What changed?
- Where is the changed behavior?
- Which tests were selected?
- Was the code covered?
- What nearby tests exist?
- What assertion would likely kill this mutant?
- Is this survivor new, old, quarantined, or already known?

Minimum JSON fields for actionability:

```json
{
  "mutant_id": "...",
  "status": "survived",
  "status_reason": "...",
  "selected_tests": ["go", "test", "./pkg"],
  "mutant": {
    "file": "...",
    "line": 42,
    "function": "...",
    "operator": "...",
    "description": "...",
    "nearby_tests": ["foo_test.go"],
    "unified_diff": "...",
    "hint": "..."
  }
}
```

### 4. CI Relevance And Scope Control

References: On the use of commit-relevant mutants, EMSE 2022; Practical
mutation testing at scale / Google; PIT-style history and prioritization.

CI should not run every possible mutant on every pull request. It should run the
mutants most relevant to the change, then use scheduled full runs to prevent
blind spots.

Recommended CI layers:

| Layer | Purpose | Example |
| --- | --- | --- |
| PR fast | Changed packages and changed files. | `cervomut run ./... --scope changed --since origin/main --budget 10m` |
| PR fallback | Package scope when diff mapping is uncertain. | `selection.mode: package` |
| Nightly | Broader sampling across repository. | deterministic sample plus baseline compare |
| Weekly/full | Full run, operator trend, quarantine review. | complete report and proposal issues |

CI should fail on:

- baseline regression
- new unquarantined survivors in changed scope
- expired quarantines
- unexpected compile-error or timeout rates above policy

CI should not fail on:

- old known survivors when baseline policy allows adoption
- `not_covered` in unrelated legacy code unless the policy explicitly gates it

### 5. History-Aware Scheduling

References: PIT-style history and prioritization; Practical mutation testing at
scale / Google; On the use of commit-relevant mutants, EMSE 2022.

History is not a shortcut to hide bad results. It is a way to spend time on the
mutants most likely to produce useful feedback.

Scheduling inputs:

- previous mutant status
- previous runtime
- cache validity
- changed files/packages
- operator historical value
- survivor age
- flake/timeout history
- quarantine expiry

Priority order for PR feedback:

1. New mutants in changed code.
2. Previously survived mutants touched by the change.
3. Operators with high historical useful-survivor yield.
4. Fast mutants that improve confidence inside the budget.
5. Long-running or noisy mutants deferred to scheduled runs.

### 6. Equivalent-Risk Governance

References: Equivalent Mutants in the Wild, ISSTA 2024; Static detection of
equivalent mutants, 2022; The ratio of equivalent mutants, JSS 2021; An
Empirical Evaluation of Manually Created Equivalent Mutants, 2024.

Equivalent mutants cannot be fully solved, but CervoMutant can make the burden
visible and reduce obvious noise.

Equivalence workflow:

| Evidence level | Meaning | Allowed action |
| --- | --- | --- |
| `suspected` | One reviewer or heuristic suspects equivalence. | Lower priority, keep visible. |
| `accepted` | Two reviewers agree or rule is reproducible. | Suppress or quarantine with issue. |
| `rule-suppressed` | AST/pattern rule suppresses it. | Sample false suppression periodically. |
| `disputed` | Reviewers disagree or assumptions are unclear. | Keep as survivor, require stronger test or issue. |

Suppression rules must be auditable:

- rule name
- operator
- AST pattern
- reason
- sample count
- false-suppression findings

Do not silently remove mutants from all reports. Suppressed counts should appear
in summary and JSON.

### 7. Operator Governance Beyond Operator Lists

References: Mutation Coverage is Not Strongly Correlated with Mutation Coverage,
AST 2024; The ratio of equivalent mutants, JSS 2021; Equivalent Mutants in the
Wild, ISSTA 2024; Cobra comparison study.

Operators are promoted or demoted based on measured utility, not taste.

Per-operator metrics:

- kill rate
- useful-survivor yield
- equivalent-risk rate
- compile-error rate
- timeout rate
- not-covered rate
- median runtime
- duplicate/redundant signal rate
- percentage of survivors converted into tests

Promotion policy:

| Tier | Requirements |
| --- | --- |
| `conservative-fast` | High signal, low runtime, low equivalent risk, low compile error rate. |
| `conservative` | Good signal and acceptable noise for regular CI. |
| `default` | Valuable but broader; may require baseline adoption. |
| `aggressive` | Useful for deep campaigns, too noisy or expensive for default CI. |
| `experimental` | Insufficient evidence; must be opt-in. |

### 8. Validity Controls

References: Mutation Testing Advances: An Analysis and Survey; Mutation Coverage
is Not Strongly Correlated with Mutation Coverage, AST 2024; Cobra comparison
study.

Every benchmark or adoption decision should record threats to validity.

Required controls:

- exact repo commit
- exact commands
- tool versions or binary paths
- worker count
- timeout policy
- OS and filesystem shape
- baseline test result
- cache state
- coverage mode
- operator profile
- generated artifact paths
- known non-equivalence of mutator sets across tools

Claims must use precise language:

- "CervoMutant is faster in this Cobra package run" is valid.
- "CervoMutant has a better mutation score than tool X" is only meaningful when
  operator sets and coverage semantics are documented.
- "This operator belongs in default" requires evidence across more than one repo.

## Implementation Backlog

These are product improvements derived from the framework, separate from adding
more operators.

| Area | Improvement |
| --- | --- |
| Coverage | Make coverage prefilter available in fast CI and report `not_covered` even outside strict coverage mode when baseline coverage exists. |
| Reporting | Add score decomposition blocks to summary, HTML, and JSON: efficacy, mutation coverage, operator profile, equivalent-risk, and suppression counts. |
| Scheduling | Use history and timings to prioritize mutants inside `--budget`. |
| Baseline | Track survivor age, first seen commit, last seen commit, and whether the survivor is new in changed code. |
| Equivalence | Add auditable suppression rules and evidence levels. |
| CI | Add policy presets: `ci-fast`, `ci-balanced`, `nightly`, and `campaign`. |
| Evaluation | Keep apples-to-apples benchmark fixtures for Cobra and at least one CervoSoft module. |
| Agent UX | Add direct test-generation hints, nearest test files, and assertion suggestions in `explain`. |

## Near-Term Defaults To Evaluate

Do not change the default purely because Gremlins behaves differently. Evaluate
these defaults using the framework:

```yaml
selection:
  mode: coverage
reports:
  include_score_decomposition: true
execution:
  isolation: overlay
mutators:
  profile: conservative-fast
```

Candidate policy:

- `conservative-fast` for PR CI.
- `conservative` for local developer runs.
- `default` for nightly.
- `aggressive` for explicit campaigns and test-generation agents.

## References

- Gremlins features and limitations: https://gremlins.dev/0.2/
- Gremlins `unleash` command and metrics: https://gremlins.dev/next/usage/commands/unleash/
- Practical mutation testing at scale / Google: https://homes.cs.washington.edu/~rjust/publ/mutation_testing_practices_icse_2021.pdf
- Practical mutation testing at scale, IEEE TSE 2022 publication page: https://doi.org/10.1109/TSE.2021.3131293
- Equivalent Mutants in the Wild, ISSTA 2024: https://2024.issta.org/details/issta-2024-papers/53/Equivalent-Mutants-in-the-Wild-Identifying-and-Efficiently-Suppressing-Equivalent-Mu
- Mutation Coverage is Not Strongly Correlated with Mutation Coverage, AST 2024: https://researchwith.njit.edu/en/publications/mutation-coverage-is-not-strongly-correlated-with-mutation-covera/
- An Empirical Evaluation of Manually Created Equivalent Mutants, 2024: https://arxiv.org/abs/2404.09241
- Static detection of equivalent mutants, 2022: https://link.springer.com/article/10.1007/s10664-022-10149-y
- The ratio of equivalent mutants, JSS 2021: https://www.sciencedirect.com/science/article/abs/pii/S0164121221001369
- On the use of commit-relevant mutants, EMSE 2022: https://link.springer.com/article/10.1007/s10664-022-10138-1
- Mutation Testing Advances: An Analysis and Survey: https://doi.org/10.1016/bs.adcom.2018.03.015
- PIT mutation testing system: https://pitest.org/
