# Follow-Up Ledger

Tracking issues: #138, #166

This ledger keeps repeated field findings and comparison findings tied to
explicit follow-up work instead of leaving them buried in one-off study notes.

It is intentionally lightweight: the goal is to make recurring evidence visible
and link it to the right repo-tracked next step.

## Field And Adoption Findings

| Finding source | Repeated finding | Priority | Status | Tracked work |
| --- | --- | --- | --- | --- |
| [2026-06-17 external validation wave](2026-06-17-external-validation-wave.md) | Public-repo validation is real, but direct maintainer feedback is still missing. | medium | implemented | [#136](https://github.com/cervantesh/cervo-mutants/issues/136), [docs/feedback-intake.md](../feedback-intake.md) |
| [Adoption feedback intake](../feedback-intake.md) and [adoption analytics](../adoption-analytics.md) | External rollout feedback needs structured dimensions and release-loop review so repeated blockers can be aggregated instead of rediscovered. | medium | implemented | [#190](https://github.com/cervantesh/cervo-mutants/issues/190), [docs/adoption-analytics.md](../adoption-analytics.md), [adoption-feedback issue template](../../.github/ISSUE_TEMPLATE/adoption-feedback.yml) |
| [2026-06-19 released adoption-feedback response audit](2026-06-19-released-adoption-feedback-response-audit.md) | Structured internal adoption-feedback issues do not by themselves produce direct maintainer engagement; the project still needs a repeatable outward-response capture loop before claiming external adoption depth. | medium | open | [#308](https://github.com/cervantesh/cervo-mutants/issues/308), [2026-06-19-released-adoption-feedback-response-audit.md](2026-06-19-released-adoption-feedback-response-audit.md) |
| [Adoption guide](../adoption-guide.md) | Distributed execution must stay explicitly narrowed until the daemon/worker path becomes a supported surface. | high | implemented | [#147](https://github.com/cervantesh/cervo-mutants/issues/147), [#148](https://github.com/cervantesh/cervo-mutants/issues/148), [#149](https://github.com/cervantesh/cervo-mutants/issues/149), [#150](https://github.com/cervantesh/cervo-mutants/issues/150), [docs/daemon-worker.md](../daemon-worker.md) |
| [2026-06-19 released adoption feedback wave](2026-06-19-released-adoption-feedback-wave.md) and [#294](https://github.com/cervantesh/cervo-mutants/issues/294), [#295](https://github.com/cervantesh/cervo-mutants/issues/295), [#297](https://github.com/cervantesh/cervo-mutants/issues/297) | Healthy released GitHub Action runs still repeatedly surface equivalent-risk boundary survivors; the right default is review-once/report-only guidance, not treating every raw survivor as independent follow-up work. | medium | implemented | [#292](https://github.com/cervantesh/cervo-mutants/issues/292), [docs/adoption-guide.md](../adoption-guide.md), [2026-06-19-released-adoption-feedback-wave.md](2026-06-19-released-adoption-feedback-wave.md) |
| [2026-06-17 survivor ranking calibration](2026-06-17-survivor-ranking-calibration.md) | Recommendation quality needs measured yield on representative repositories, not only heuristic confidence. | medium | implemented | [#158](https://github.com/cervantesh/cervo-mutants/issues/158), [2026-06-17-survivor-ranking-calibration.md](2026-06-17-survivor-ranking-calibration.md) |
| [History data contract](../history-data.md) | Historical outputs need a repeatable branch and release comparison workflow without pretending there is a shared-store merge model. | medium | implemented | [#152](https://github.com/cervantesh/cervo-mutants/issues/152), [docs/branch-release-comparisons.md](../branch-release-comparisons.md) |
| [2026-06-19 post-release field findings](2026-06-19-post-release-field-findings.md) and the hosted released-surface wave notes | Hosted rollout expectations were easy to overstate until the docs separated healthy review signal, healthy retargeting signal, and true workflow failure. | high | implemented | [#212](https://github.com/cervantesh/cervo-mutants/issues/212), [#256](https://github.com/cervantesh/cervo-mutants/issues/256), [docs/github-action.md](../github-action.md), [docs/adoption-guide.md](../adoption-guide.md), [docs/rollout-playbooks.md](../rollout-playbooks.md) |

## Comparative And Benchmark Findings

| Finding source | Repeated finding | Priority | Status | Tracked work |
| --- | --- | --- | --- | --- |
| [tool comparison protocol](tool-comparison-protocol.md) and [comparison harness](comparison-harness.md) | Public performance claims need explicit apples-to-apples target semantics. | high | implemented | [#163](https://github.com/cervantesh/cervo-mutants/issues/163), [benchmark-corpus.md](benchmark-corpus.md), [comparison-harness.md](comparison-harness.md) |
| [example repos](../example-repos.md) and [adoption guide](../adoption-guide.md) | Public examples must stay current enough to be trusted as living references. | medium | implemented | [#164](https://github.com/cervantesh/cervo-mutants/issues/164), [internal/examples/examples_test.go](../../internal/examples/examples_test.go) |
| [tool findings ledger](tool-findings.md) | Some comparison gaps are now intentionally documented as deferred rather than hidden, such as very large partial-report segmentation. | low | explicitly narrowed | [#166](https://github.com/cervantesh/cervo-mutants/issues/166), [tool-findings.md](tool-findings.md) |

## Operating Rule

When a finding repeats across at least two studies, waves, or adoption-feedback
issues, it should be moved into this ledger with:

- the evidence source
- a priority
- a status
- the linked doc, issue, or explicit narrowing decision

That keeps future release or roadmap reviews anchored to tracked evidence
instead of memory.
