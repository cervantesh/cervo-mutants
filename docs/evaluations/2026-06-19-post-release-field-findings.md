# 2026-06-19 Post-Release Field Findings

Tracking issue: [#213](https://github.com/cervantesh/cervo-mutants/issues/213)

Date: 2026-06-19

This report aggregates the post-release field findings that emerged from the
first public validation wave and the first hosted GitHub Action adoption waves.

The purpose of this note is not to replay every experiment. The purpose is to
turn repeated field friction into one durable product-facing record that can be
cited by future maturity assessments, release notes, and follow-up planning.

## Evidence Base

Primary committed evidence used in this report:

- [2026-06-17-external-validation-wave.md](2026-06-17-external-validation-wave.md)
- [2026-06-19-external-github-action-wave-2.md](2026-06-19-external-github-action-wave-2.md)
- [2026-06-19-field-calibration-from-adoption-waves.md](2026-06-19-field-calibration-from-adoption-waves.md)
- [2026-06-19-external-github-action-wave-default-tuning.md](2026-06-19-external-github-action-wave-default-tuning.md)
- [2026-06-19-external-github-action-wave-triage-yield.md](2026-06-19-external-github-action-wave-triage-yield.md)
- [2026-06-19-external-github-action-wave-candidate-retargeting.md](2026-06-19-external-github-action-wave-candidate-retargeting.md)

Issue trail covered by those findings:

- [#210](https://github.com/cervantesh/cervo-mutants/issues/210)
- [#211](https://github.com/cervantesh/cervo-mutants/issues/211)
- [#220](https://github.com/cervantesh/cervo-mutants/issues/220)
- [#223](https://github.com/cervantesh/cervo-mutants/issues/223)
- [#224](https://github.com/cervantesh/cervo-mutants/issues/224)
- [#227](https://github.com/cervantesh/cervo-mutants/issues/227)

## Executive Summary

The post-release field picture is sharper now:

1. The first-party GitHub Action path is real and operationally trustworthy on
   GitHub-hosted runners.
2. The biggest early hosted-wave problem was **default candidate and target
   quality**, not a fundamental inability to get useful signal from public Go
   repositories.
3. The second big problem was **evidence shape**, not only runtime behavior:
   the original hosted summaries did not preserve enough triage-specific yield
   for later review.
4. The triage and recommendation surfaces were conservative rather than noisy.
   Early under-signal came from denominator-poor inputs, not from obvious
   false-positive recommendation behavior.
5. The main remaining gap is now public expectation management and rollout
   guidance, not missing core product machinery.

## Findings By Category

| Category | Evidence | Classification | Outcome |
| --- | --- | --- | --- |
| Hosted workflow hygiene | Hosted wave 2 succeeded, but emitted Node 20 artifact-action deprecation warnings. | Operational product gap | Fixed in [#220](https://github.com/cervantesh/cervo-mutants/issues/220). |
| Hosted default settings too weak | Wave `27831153375` produced `0` killed, `0` survived, `10` not covered. | Functional product-default gap | Improved in [#224](https://github.com/cervantesh/cervo-mutants/issues/224). |
| Hosted candidate mix too weak | Tuned wave `27833007931` still had `covered=4`, `effective=4`, `not_covered=26`, `warning_repos=3`. | Functional product-default gap | Fixed materially in [#227](https://github.com/cervantesh/cervo-mutants/issues/227). |
| Adoption-wave summaries too thin for calibration | Early summaries did not preserve recommendation, ledger, or governance-yield counts directly. | Code/report-contract gap | Fixed in [#223](https://github.com/cervantesh/cervo-mutants/issues/223). |
| Hosted triage/recommendation usefulness underpowered | Early hosted waves yielded `actionable_score=0` and no review units. | Evidence gap, not immediate heuristic bug | Reframed and partially resolved by better inputs in [#211](https://github.com/cervantesh/cervo-mutants/issues/211) and [#227](https://github.com/cervantesh/cervo-mutants/issues/227). |
| Public expectations still too easy to overstate | Hosted path now works, but signal quality is still target-sensitive and bounded by candidate choice. | Documentation and expectation gap | Remaining follow-up under [#212](https://github.com/cervantesh/cervo-mutants/issues/212). |

## Repeated Friction Themes

### 1. Public-repo signal quality was a defaults problem before it was a core-engine problem

The strongest repeated finding is that the initial hosted friction was not
"public repositories are too noisy" and not "semantic triage is broken."

The evidence points to a narrower bottleneck:

- local external validation wave on 2026-06-17:
  - `50` mutants executed
  - `13` survivors
  - `0` not covered
- first hosted GitHub Action wave on 2026-06-19:
  - `0` killed
  - `0` survived
  - `10` not covered
- tuned hosted default wave:
  - `effective=4`
  - `not_covered=26`
- retargeted hosted default wave:
  - `effective=24`
  - `survived=6`
  - `not_covered=6`
  - `actionable_review_units=5`

That progression matters. It shows the product could already produce useful
signal on public repositories, but the first hosted default set was a poor
choice for that path.

### 2. Evidence contracts mattered almost as much as runtime behavior

Another repeated friction point was not execution failure, but missing durable
context in the committed summaries.

Before [#223](https://github.com/cervantesh/cervo-mutants/issues/223), later
reviewers could not answer basic questions from the committed hosted summary
alone:

- were recommendations generated?
- was the ledger empty?
- were governance suggestions present?

After `#223`, the hosted summary preserves additive triage fields directly.
That moved calibration review from ephemeral artifact inspection into committed
repo evidence.

### 3. The recommendation and triage layers behaved conservatively, not noisily

The early hosted waves did **not** show evidence that actionable scoring or
recommendation logic was inventing noisy advice.

What they showed instead:

- denominator-poor inputs produced zero actionable review units
- recommendation entries stayed empty when there was no actionable work
- governance and reporting surfaces still degraded safely

After the candidate retargeting pass in `#227`, the hosted wave produced:

- `6` survivors
- `5` actionable review units
- `6` recommendation entries
- `3` ledger entries

That is important because it narrows the interpretation:

- the product did not need heuristic panic-tuning first
- it needed healthier hosted inputs and better persisted evidence

### 4. Operational trust issues were real, but shallow

The operational problems found in the first hosted wave were meaningful, but
they were shallow infrastructure debt rather than structural failures:

- artifact-action deprecation warnings
- initial workflow bootstrap awkwardness before the workflow existed on default
  branch

Those did not invalidate the hosted path. They were cleanup issues that needed
to be removed so the hosted wave could become a cleaner public trust surface.

## What The Field Evidence Does Not Support

The evidence collected so far does **not** support these stronger claims:

- that any arbitrary public Go target will immediately produce high-signal
  hosted mutation results under bounded defaults
- that recommendation quality is already broadly calibrated across public-repo
  hosted runs
- that semantic grouping usefulness has been fully validated on diverse hosted
  samples
- that poor early hosted signal implied a deep engine flaw

Those claims would overstate what the field data proves today.

## Resolved Versus Remaining Gaps

### Resolved or materially improved

- Hosted GitHub Action execution on GitHub-hosted runners
- Node 20 artifact-action warning cleanup
- Hosted default settings stronger than the original smoke defaults
- Hosted default candidate set materially healthier on denominator metrics
- Hosted summary artifacts preserving triage and governance yield directly

### Remaining

- Public docs should better frame hosted-wave expectations, especially around
  bounded defaults, target sensitivity, and what a first useful signal really
  looks like on external repositories

That remaining work is narrower than the earlier product gaps and belongs under
[#212](https://github.com/cervantesh/cervo-mutants/issues/212).

## Practical Interpretation

The maturity implication is straightforward:

- The project no longer has to defend whether hosted GitHub Action execution is
  real. It is real.
- The project no longer has to treat denominator-poor hosted evidence as an
  ambiguous warning. The concrete causes were isolated and improved.
- The next product-facing communication work should be honest about limits:
  hosted waves are now credible, but still bounded and candidate-sensitive.

That is a better position than either of the two weaker narratives:

- "everything is already solved"
- "public hosted validation failed"

Neither is true. The stronger truth is that post-release field feedback
surfaced a series of concrete, fixable gaps, and most of those gaps were
already corrected inside the same release-era evidence cycle.

## Follow-Up

- Remaining open follow-up:
  - [#212](https://github.com/cervantesh/cervo-mutants/issues/212) Tighten
    rollout defaults and docs from repeated field friction
- Parent tracking:
  - [#206](https://github.com/cervantesh/cervo-mutants/issues/206) Epic:
    Post-roadmap external adoption and signal calibration
