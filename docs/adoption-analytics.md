# Adoption Analytics And Feedback Loops

Tracking issue: #190

This document defines how external adoption evidence becomes durable product
signal instead of fading into issue threads, release notes, or maintainer
memory.

It complements the intake path in
[docs/feedback-intake.md](feedback-intake.md) and the maintainer operating
baseline in
[docs/maintainer-operations-pack.md](maintainer-operations-pack.md).

## Canonical Unit Of Evidence

The default unit of adoption evidence is one GitHub issue created with the
[`Adoption feedback`](../.github/ISSUE_TEMPLATE/adoption-feedback.yml)
template, backed by report artifacts and environment details.

Each issue should capture enough structure to answer:

- what kind of repository was being rolled out
- which adoption stage the team was in
- which install path and environment were used
- what the primary friction or blocker actually was
- whether the outcome points to docs, workflow, or product work

That gives maintainers something they can aggregate later without inventing a
parallel spreadsheet or private tracker.

## Structured Dimensions To Capture

Every adoption-feedback issue should preserve these dimensions explicitly:

- repository profile: compact library, medium service, large repo/monorepo, or
  another clearly named shape
- adoption stage: first dry run, first useful report, baseline setup, PR lane,
  nightly lane, history/governance, or comparison/benchmark work
- install path: `go install`, release archive, local source build, or GitHub
  Action
- environment: OS, Go version, local vs CI, and any important runtime limits
- rollout posture: policy, budget, workers, baseline/quarantine posture,
  actionable-only usage, and report surfaces
- primary blocker class: signal noise, review UX, runtime/resources,
  CI/install/platform, governance/history, or unsupported workflow
- observed outcome: docs gap, workflow friction, product defect, or evidence of
  a known limit

Those dimensions should come from the issue itself, not from later maintainer
guesswork.

## Derived Metrics That Matter

The goal is not vanity reporting. The goal is to make external adoption
friction measurable enough to prioritize product work.

The default metrics to derive from accumulated adoption-feedback issues are:

- time to first useful report: how long it takes from initial install to a run
  that the adopter considers reviewable
- first-run success rate: whether bounded dry run or first useful report worked
  without repo-specific surgery
- baseline progression rate: how often a team gets from first run to
  baseline-first governance
- primary blocker frequency: which blocker class repeats most often
- repeated noisy-signal patterns: which operator families, semantic groups, or
  report surfaces keep generating "review pain"
- docs vs product follow-up ratio: how much friction is solved by clearer
  guidance versus code changes
- unsupported-workflow rate: how often adopters are trying to use the product
  outside its public support boundary

These are release-planning metrics, not marketing metrics.

## Feedback Loops

### Per-Issue Loop

For every adoption-feedback issue:

1. confirm the artifact bundle and environment details
2. classify the repository profile, adoption stage, and primary blocker
3. decide whether the outcome is:
   - already explained by current docs
   - a documentation follow-up
   - a product or code follow-up
   - an unsupported workflow
4. link repeated findings into
   [docs/evaluations/follow-up-ledger.md](evaluations/follow-up-ledger.md)

### Release Loop

Before each release, review adoption-feedback issues opened or updated since
the previous release and summarize:

- repeated blocker classes
- repeated noisy operator or semantic-group complaints
- rollout steps that still create avoidable setup friction
- docs clarifications that should ship with the release
- product issues that should remain explicitly narrowed rather than silently
  implied as fixed

That review belongs in release preparation, not as an afterthought.

### Validation-Wave Loop

When running a public validation wave, compare the wave findings with the
historical adoption-feedback issues:

- did the same blockers repeat across multiple repository profiles?
- did the closest rollout playbook materially reduce friction?
- are the same unsupported expectations still appearing?
- are maintained examples still sufficient for the current release?

If the answer is yes, link the evidence into the follow-up ledger and open or
refresh the corresponding tracked work.

## Promotion Rules For Repeated Findings

Move a finding from a single issue into the explicit ledger when any of these
are true:

- the same blocker appears in at least two adoption-feedback issues
- the same docs gap appears across different repository profiles
- the same noisy operator or review surface keeps surfacing in real rollouts
- a known limit keeps getting interpreted as supported behavior

When promoted, record:

- the evidence source
- the repeated finding in one sentence
- the priority
- the current status
- the linked issue, doc, or narrowing decision

## What Not To Do

Do not treat one anecdote as broad adoption proof.

Do not collapse every complaint into "noise" without preserving the repository
shape, adoption stage, and artifact context.

Do not rely on maintainers to remember which complaints repeated last month.

Do not create a separate private analytics system before the public issue and
artifact path is working.

## Related Guides

- [docs/feedback-intake.md](feedback-intake.md)
- [docs/maintainer-operations-pack.md](maintainer-operations-pack.md)
- [docs/adoption-guide.md](adoption-guide.md)
- [docs/rollout-playbooks.md](rollout-playbooks.md)
- [docs/evaluations/follow-up-ledger.md](evaluations/follow-up-ledger.md)
