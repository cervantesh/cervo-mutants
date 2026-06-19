# Adoption Guide: Fit, Limits, And Rollout

Tracking issues: #137, #165, #212, #256

This guide turns the current public evidence into a practical answer to three
questions:

1. Is CervoMutants a good fit for this repository right now?
2. What limits or tradeoffs should the team expect?
3. What is the safest rollout path into local development or CI?

The guidance below is grounded in the current public release, the maintained
example workspaces, the first reproducible external validation wave across five
public Go repositories, and the latest released-surface GitHub Action wave
against `v0.4.2`.

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
- the released GitHub Action surface `github-action@v0.4.2` completed a hosted
  public-wave sample with persisted actionable-yield evidence
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
- [docs/evaluations/2026-06-19-external-github-action-wave-v0.4.2.md](evaluations/2026-06-19-external-github-action-wave-v0.4.2.md)
- [docs/example-repos.md](example-repos.md)
- [docs/rollout-playbooks.md](rollout-playbooks.md)
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

## First Useful Signal Before Broader Rollout

A first run that completes is not automatically a first run that teaches you
anything useful.

The recent hosted adoption waves showed the difference clearly:

- the workflow can be operationally healthy while still producing weak
  denominator health
- low-signal first runs were usually a target-selection problem before they
  were a triage or engine problem
- retargeting to healthier package roots produced materially better review
  signal without changing the core product

For the first bounded run, read these fields before you judge the score:

- `effective mutants`
- `not covered`
- denominator-health warnings in `summary.txt` or `github-summary.md`
- whether there are any actionable review units at all

A useful first signal can still have two different shapes:

1. healthy review signal:
   denominator health is reasonable and the run gives you survivors or
   actionable review units worth discussing
2. healthy retargeting signal:
   the run completes, preserves artifacts, and clearly shows that the current
   target is too cold or too broad to judge product value yet

Those are both useful outcomes. What is not useful is a broken workflow with
missing artifacts, or a rollout that treats denominator-poor output as proof
that the repository is a bad fit without retargeting first.

If the first run lands near `effective=0`, or if `not covered` dominates the
report, do **not** immediately conclude that the repository is a bad fit or
that semantic triage needs tuning first.

Treat that as rollout feedback:

1. preserve the artifacts
2. narrow the target to a hotter package, submodule, or bounded shard
3. rerun before widening the lane or setting policy expectations

That is the safer interpretation supported by the current field evidence in:

- [docs/evaluations/2026-06-19-external-github-action-wave-v0.4.2.md](evaluations/2026-06-19-external-github-action-wave-v0.4.2.md)
- [docs/evaluations/2026-06-19-post-release-field-findings.md](evaluations/2026-06-19-post-release-field-findings.md)
- [docs/evaluations/2026-06-19-external-github-action-wave-candidate-retargeting.md](evaluations/2026-06-19-external-github-action-wave-candidate-retargeting.md)

The latest released hosted wave makes the interpretation concrete:

- `pflag-root` and `gjson-root` produced healthy denominator behavior with real
  actionable review units
- `logrus-root` produced zero actionable review units together with denominator
  warnings and `not covered` pressure

That is not evidence that the hosted path is broken. It is evidence that the
first bounded target can be too cold, and that retargeting should come before
broader policy judgments.

Healthy first useful reports can still have one more shape that needs explicit
review semantics:

- repeated equivalent-risk boundary survivors inside an otherwise healthy run

The released hosted adoption-feedback issues for `pflag`, `gjson`, and
`apimachinery` make that concrete, and `prometheus` adds the zero-action
counterexample:

| Shape | Example counts | Meaning | Next move |
| --- | --- | --- | --- |
| Healthy direct review lane | `generated=10 effective=10 survived=3 actionable_review_units=3` | Raw survivors and review workload match closely. | Start with `test-recommendations.md`. |
| Healthy grouped-review lane | `generated=10 effective=10 survived=3 actionable_review_units=2 semantic_group_review_units=1` | Repeated equivalent-risk boundary survivors were collapsed into one review-once family. | Review the semantic group once before adding multiple new tests. |
| Healthy no-action lane | `generated=10 covered=7 effective=7 survived=0 actionable_review_units=0 not_covered=3` | The slice produced useful denominator evidence but no current survivor work. | Keep the artifact, decide whether this slice is already good enough, and retarget only if the wider rollout still needs more review pressure. |

- raw survivors can be higher than the real review workload
- `semantic_group_review_units`, `test-recommendations.md`, and
  `governance-review.md` together often tell you that the right action is
  "review once" or `report-only`, not "treat every raw survivor as a separate
  test task"
- `actionable_review_units=0` is not automatically a failed rollout; if
  denominator health is still understandable, it can be a healthy bounded lane
  with no immediate follow-up work

When a bounded run is otherwise healthy, read those artifacts before you decide
that the lane is noisy.

## Recommended Rollout Path

Use this rollout order unless the repository already has a mature mutation lane:

1. Start with `cervomut doctor`, `cervomut init`, and a dry run.
2. Run the first bounded local pass on the same report directory you will use
   for the first baseline, usually `.cervomut/reports`.
3. Save a baseline before introducing harder policy expectations.
4. Add nightly or campaign-style runs only after the PR lane is understandable.
5. Introduce slicing, ownership routing, quarantine, and historical review when
   the basic lane is already trusted.

Practical starting commands:

```powershell
cervomut init
cervomut doctor
cervomut run ./... --dry-run
cervomut fast ./... --budget 10m --sample deterministic --out .cervomut/reports
cervomut baseline update
```

If that run produces poor denominator health, rerun on a narrower package or
subtree before you add CI or nightly depth.

For repository-specific starting points, use the maintained examples first
instead of inventing custom config on day one.

For decision-complete rollout paths by repository profile, continue with
[docs/rollout-playbooks.md](rollout-playbooks.md).

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
