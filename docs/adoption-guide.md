# Adoption Guide: Fit, Limits, And Rollout

Tracking issues: #137, #165

This guide turns the current public evidence into a practical answer to three
questions:

1. Is CervoMutants a good fit for this repository right now?
2. What limits or tradeoffs should the team expect?
3. What is the safest rollout path into local development or CI?

The guidance below is grounded in the current public release, the maintained
example workspaces, and the first reproducible external validation wave across
five public Go repositories.

## Current Best Fit

CervoMutants is a strong fit today when all of these are true:

- the repository is Go-first and already has a stable `go test` baseline
- the team wants reviewable survivor signal, not only a raw score
- the team can adopt baseline-first instead of failing immediately on a
  threshold
- CI has enough budget for bounded mutation runs on pull requests or on a
  nightly lane

The public examples map cleanly to that starting point:

- [`examples/small-library`](../examples/small-library): small libraries or
  compact modules that want a single PR mutation lane
- [`examples/medium-service`](../examples/medium-service): multi-package
  services that need richer review output and a nightly lane
- [`examples/large-repo-ci`](../examples/large-repo-ci): larger repositories
  that need deterministic slicing and bounded shard density before going wider

## Good Fit Patterns

These are the use cases where the current product is strongest:

- small to medium Go repositories with healthy and deterministic tests
- teams that want actionable survivor review with JSON, HTML, JUnit, SARIF, and
  GitHub summary outputs
- repositories that need baseline, quarantine, and history-aware governance
- evaluation and comparison work where denominator health and explicit target
  semantics matter

## Weak Fit Or Deliberate Non-Goals

CervoMutants is not yet the strongest choice when any of these are the primary
need:

- immediate score-only enforcement on an existing repository with no baseline
- very large monorepos that require mature distributed orchestration on day one
- heavily flaky or nondeterministic test suites where mutation noise is already
  dominated by baseline test instability
- workflows that expect test generation instead of ranked review guidance
- repositories whose only acceptable experience is one-click zero-tuning
  adoption on every OS shape

The daemon/worker mode remains explicitly experimental. It should not be sold
internally as a stable distributed execution platform yet.

## What Public Evidence Actually Proves

The current public evidence is strong enough to support bounded adoption claims,
but not every possible trust claim.

What is proven:

- the current release works as a local and CI mutation runner
- bounded external validation succeeded across five public Go repositories on a
  Windows host without repo-specific patches
- example workspaces exist for small, medium, and large-repo rollout patterns
- baseline-first governance, actionable review, and compatibility surfaces are
  publicly documented

What is not yet proven:

- broad maintainer adoption by external upstream teams
- deep campaign behavior across many large monorepos
- stable distributed execution as a supported public surface
- identical ergonomics across every Windows, Linux, and macOS environment

See:

- [docs/evaluations/2026-06-17-external-validation-wave.md](evaluations/2026-06-17-external-validation-wave.md)
- [docs/example-repos.md](example-repos.md)
- [docs/feedback-intake.md](feedback-intake.md)
- [docs/go-toolchain-compatibility.md](go-toolchain-compatibility.md)
- [docs/daemon-worker.md](daemon-worker.md)

## Known Limits From Current Field Evidence

The external validation wave intentionally proved a narrow but meaningful claim:
the tool can run bounded mutation successfully on real public repositories
outside this repo. It did not try to prove every operational scenario.

The main known limits today are:

- public validation so far is bounded, not deep-campaign saturation
- the public wave used one Windows host, so it complements but does not replace
  the support matrix
- runtime variance between repositories is real; deeper mutation breadth still
  needs conscious budgeting
- semantic triage reduces noisy survivors, but it does not eliminate the need
  for human review
- distributed execution is still an experimental path, not a default
  recommendation

When a real rollout surfaces those limits, capture it through the
[`Adoption feedback`](../.github/ISSUE_TEMPLATE/adoption-feedback.yml) issue
template so the evidence becomes tracked product feedback instead of a private
note.

## Recommended Rollout Path

Use this rollout order unless the repository already has a mature mutation lane:

1. Start with `cervomut doctor`, `cervomut init`, and a dry run.
2. Run a bounded local or PR-style pass with `ci-fast` or `ci-balanced`.
3. Save a baseline before introducing harder policy expectations.
4. Add nightly or campaign-style runs only after the PR lane is understandable.
5. Introduce slicing, ownership routing, quarantine, and historical review when
   the basic lane is already trusted.

Practical starting commands:

```powershell
cervomut init
cervomut doctor
cervomut run ./... --dry-run
cervomut fast ./... --budget 10m --sample deterministic
cervomut baseline update
```

For repository-specific starting points, use the maintained examples first
instead of inventing custom config on day one.

## Key Tradeoffs

The current product is intentionally opinionated. These tradeoffs are part of
the design, not accidental rough edges:

- actionable signal over raw mutant volume
- baseline-first adoption over immediate fail-under score gating
- explicit denominator health over one-number reporting
- reviewable governance and auditability over silent suppression
- bounded PR lanes plus deeper nightlies over trying to run everything at once

Those tradeoffs are usually good for teams that want durable mutation workflows.
They are a weaker fit for teams that only want a single headline percentage with
no review process around it.

## Decision Rule

Adopt CervoMutants now if your team wants a baseline-first, review-oriented Go
mutation workflow and can budget bounded runs in CI.

Wait or narrow the rollout if your repository depends on fully supported
distributed execution, cannot keep `go test` deterministic, or only values raw
score enforcement without survivor review.
