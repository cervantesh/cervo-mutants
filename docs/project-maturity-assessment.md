# Project Maturity Assessment

Historical scorecard: 2026-06-19

Evidence status checked: 2026-10-23

Handoff issue: [#386](https://github.com/cervantesh/cervo-mutants/issues/386)

This document separates an older qualitative score from later facts. The
numbers below were **not recalculated in October**. They are a historical
baseline, not a claim that paused code has been continuously validated. For a
restart checklist and current support boundary, see the
[maintainer handoff](maintainer-handoff.md).

## Historical Scorecard

| Dimension | Last assessed score | Interpretation at assessment |
| --- | ---: | --- |
| Functional/product | `4.0 / 5` | Strong public beta with a coherent local and CI workflow. |
| Code/engineering | `3.8 / 5` | Substantially reduced orchestration complexity; further field-driven work needed. |
| Operational | `3.7 / 5` | Public automation and release gates in place, but little long-term cadence evidence. |
| Overall | `3.9 / 5` | Credible public beta, not yet a broadly proven mature platform. |

Scale: `1` is concept/prototype; `3` is usable beta; `4` indicates readiness
for broader external use; `5` requires stable operations and adoption over
time. The table preserves the earlier judgment. A new score requires a fresh
assessment with current tests, release checks, and external-use evidence.

## Verified Later Evidence

- The latest release at this check is [v0.4.5](https://github.com/cervantesh/cervo-mutants/releases/tag/v0.4.5), published 2026-08-07 with multi-OS archives, checksums, and a release manifest.
- The final recorded `main` [test](https://github.com/cervantesh/cervo-mutants/actions/runs/31170820014), [workflow-lint](https://github.com/cervantesh/cervo-mutants/actions/runs/31170818529), and [Pages](https://github.com/cervantesh/cervo-mutants/actions/runs/31170817705) workflows passed on 2026-08-07. These are dated CI results, not proof of continuing maintenance.
- Baseline/CI gate policy, mutator signal fixes, and report-contract hardening were merged before v0.4.5. The related issue/PR links are in the [handoff](maintainer-handoff.md).
- The former Phase 2 hardening epic #352 and children #353-#364 are closed. There was no open implementation backlog at this check before the handoff issue was filed.
- The [release evidence trail](release-evidence-trail.md) records bounded hosted validation through earlier tags. It does not record a new released hosted-wave campaign on v0.4.5.
- The supported matrix remains Go `1.25.x` on Linux, Windows, and macOS, with different depth of validation by OS; see the [matrix](go-toolchain-compatibility.md).

The previously published assessment mixed a June date with August release
facts and called already closed work active. That inconsistency has been
removed; historical validation numbers should be read in their original
evaluation documents, not treated as October measurements.

## Strengths That Remain Relevant

- A clear product thesis: distinguish test coverage from mutation signal, keep denominator health visible, and review actionable survivors without losing raw results.
- A broad but documented CLI and reporting surface: baseline-first workflows, semantic triage, JSON schema v1, HTML, JUnit, SARIF, GitHub summary, history, governance, and the first-party Action.
- Explicit compatibility policy, golden report fixtures, release/install checks, and a maintained contribution path.
- Smaller orchestration entrypoints than the early beta: `pkg/engine/engine.go` is 169 lines and `cmd/cervomut/main.go` is 32 lines on the October checked `main`; complexity still exists across their subsystems.

## Limits And Unproven Claims

- Daemon/worker is experimental, not supported distributed execution.
- Public adoption and recommendation quality have bounded evidence, but not repeated independent maintainer use over a long period.
- The release trail is compressed; it does not prove a sustained release cadence during a maintainer pause.
- A green August CI run does not guarantee that a future toolchain, dependency, or external GitHub environment will work unchanged. Re-run the support and install gates when work resumes.
- A documentation-only handoff should not be interpreted as a new product release or fresh functional validation.

## Reassessment Gate

Before assigning a new maturity score, a future maintainer should:

1. Validate a clean `main` checkout on the documented Go/OS matrix and inspect the current CI and release gates.
2. Run a bounded released-surface adoption campaign, preserve report artifacts, and record target semantics and denominator health.
3. Measure real review outcomes: useful recommendations, suppressed/equivalent findings, time to actionable test changes, and repeated maintainer friction.
4. Compare current behavior to the [compatibility policy](compatibility-policy.md) and [roadmap](maturity-roadmap.md), then publish a dated assessment tied to evidence.

Until then, the honest classification is **strong public beta with a documented
pause and an issue-backed path for outside contributors**.
