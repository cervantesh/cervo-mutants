# Quickstart

Tracking issues: #161, #212, #300

This is the shortest supported path from first install to a useful mutation
report.

If you are new to CervoMutants, do not start by tuning every flag. Start with
one bounded run that proves the repo is compatible, then establish a baseline,
then move into PR or nightly workflows.

## Local 5-Minute Path

From a Go workspace with `go` and `git` available:

```powershell
go install github.com/cervantesh/cervo-mutants/cmd/cervomut@latest
cervomut init
cervomut doctor
cervomut run ./... --dry-run --out .cervomut/preview
cervomut fast ./... --budget 5m --sample deterministic --out .cervomut/reports
```

What you should get:

- `.cervomut/reports/summary.txt`
- `.cervomut/reports/mutation-report.json`
- `.cervomut/reports/junit.xml`
- `.cervomut/reports/github-summary.md`

If that works, establish the first baseline:

```powershell
cervomut baseline update
```

That is the default safe adoption point. Do not jump to a raw score gate first.

A healthy first useful report can look like any of these:

- `generated=10 effective=10 survived=3 actionable_review_units=3`
  Healthy review lane. Start with the ranked survivors and nearby-test hints.
- `generated=10 effective=10 survived=3 actionable_review_units=2 semantic_group_review_units=1`
  Healthy grouped-review lane. Repeated equivalent-risk boundary survivors were
  already collapsed into one review-once group.
- `generated=10 covered=7 effective=7 survived=0 actionable_review_units=0 not_covered=3`
  Healthy no-action lane. The slice produced usable denominator evidence but no
  immediate survivor follow-up work.

## GitHub Actions 5-Minute Path

Use the first-party action for the common CI lane:

```yaml
name: cervomut

on:
  pull_request:
  workflow_dispatch:

jobs:
  mutation:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: cervantesh/cervo-mutants@main
        with:
          policy: ci-fast
          budget: 5m
          report: summary,json,junit,github-summary
          out: .cervomut/pr
```

That gives you:

- a bounded PR mutation lane
- a GitHub step summary
- CI-friendly JSON and JUnit artifacts
- a path that matches the maintained example workflows

Full action details live in [docs/github-action.md](github-action.md).

## Which Example To Copy

Copy the closest maintained example before inventing custom config:

- [examples/small-library](../examples/small-library) for the lowest-friction
  PR lane
- [examples/medium-service](../examples/medium-service) for PR plus nightly
  lanes
- [examples/large-repo-ci](../examples/large-repo-ci) for deterministic shards
  and bounded density

The public map for those examples is [docs/example-repos.md](example-repos.md).

## First Things To Read In The Report

Do not judge the run by the score alone.

Read these first:

1. `summary.txt`
2. the denominator counts in `mutation-report.json`
3. `actionable_review_units` and `semantic_group_review_units`
4. `github-summary.md` or the GitHub step summary in CI
5. survivor details only after the run itself looks healthy

Check denominator health before you worry about the score:

- did the run produce any `effective mutants`?
- is `not covered` dominating the report?
- are there denominator warnings in `summary.txt`?

The first useful question is:

> Did the run produce a bounded, reviewable signal?

Not:

> Is the score already high enough to gate the repo?

If `survived` is higher than `actionable_review_units`, inspect
`test-recommendations.md` and `governance-review.md` before assuming the lane
is noisy. That pattern usually means semantic grouping already compressed a
repeated equivalent-risk family into one review-once task.

If `survived=0` and `actionable_review_units=0` but denominator health is still
good, that is a useful first result too. It means the bounded slice already
looks clean enough for this pass, not that the workflow failed.

## Common First-Run Problems

If the first run is not useful, check these in order:

1. `cervomut doctor`
   This catches missing tools and common Windows/OneDrive path risks.
2. Baseline test stability
   If `go test ./...` is already flaky, mutation output will not become clearer
   than the baseline.
3. Target size
   If `./...` is too broad, or the first run yields near-zero `effective
   mutants` with mostly `not covered` rows, switch to a hotter package target
   or one of the example shard patterns before widening again.
4. Budget
   Keep the first run bounded. A fast useful run is better than a huge noisy
   run.

## Next Step After Quickstart

Once the first bounded run works:

1. save the baseline
2. add the PR lane in CI
3. introduce nightly depth only after the PR lane is understandable

For rollout fit and tradeoffs, continue with
[docs/adoption-guide.md](adoption-guide.md).
