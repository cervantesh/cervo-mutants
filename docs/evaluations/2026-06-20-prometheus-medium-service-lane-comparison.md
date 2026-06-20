# Prometheus Medium-Service Hosted Control Lane Comparison

Tracking issue: [#329](https://github.com/cervantesh/cervo-mutants/issues/329)

Date: 2026-06-20

This note evaluates whether the current medium-service hosted control lane for
Prometheus should stay on `./model/labels` or move to a hotter bounded target.

The question came from the deeper released-surface calibration wave in
[2026-06-19-external-github-action-wave-v0.4.2-signal-calibration.md](2026-06-19-external-github-action-wave-v0.4.2-signal-calibration.md),
where the current Prometheus lane stayed operationally healthy but converted
extra depth mostly into `not_covered` rows rather than new review-bearing
survivors.

## Goal

Make an explicit decision on the current medium-service hosted control lane:

1. keep `./model/labels`
2. retarget to a hotter bounded package
3. reshape the lane only if the hotter package also stays reproducible on the
   released GitHub Action path

The important constraint is not only local signal. The lane must remain
credible on GitHub-hosted runners under the public `github-action@v0.4.2`
surface.

## Inputs

Released hosted comparison manifest:

- [external-github-action-wave-prometheus-medium-service-comparison.json](external-github-action-wave-prometheus-medium-service-comparison.json)

Hosted verification run:

- run id: `27854147547`
- run URL:
  `https://github.com/cervantesh/cervo-mutants/actions/runs/27854147547`
- branch: `codex/329-medium-service-lane`
- install path under test: `github-action@v0.4.2`

Committed hosted summary artifact:

- [2026-06-20-prometheus-medium-service-lane-comparison-summary.json](2026-06-20-prometheus-medium-service-lane-comparison-summary.json)

Local scouting also screened plausible alternatives on the same pinned
Prometheus commit `505095b64b43dd76baf08839e1800a8d473c97e0` before the hosted
comparison.

## Local Scout

Local bounded scout shape:

- policy: `ci-balanced`
- budget: `3m`
- max mutants: `8`
- workers: `2`
- sample: `deterministic`
- `coverage_prefilter=true`

Shortlisted results:

| Target | Generated | Effective | Killed | Survived | Not covered | Lane shape | Notes |
| --- | ---: | ---: | ---: | ---: | ---: | --- | --- |
| `./model/labels` | `8` | `5` | `5` | `0` | `3` | healthy no-action lane | current hosted lane shape reproduced locally |
| `./model/relabel` | `8` | `8` | `8` | `0` | `0` | healthy no-action lane | hotter than `labels`, but still no review-bearing survivors |
| `./rules` | `8` | `8` | `7` | `1` | `0` | direct review lane | plausible hotter replacement |
| `./storage/remote` | `8` | `8` | `6` | `2` | `0` | direct review lane | strongest local review yield of the screened set |

Local screening therefore did find hotter bounded candidates. But local signal
alone was not enough to replace the hosted control lane. The next step had to
be a GitHub-hosted comparison.

## Hosted Comparison Result

The hosted comparison used the same released action surface against three jobs:

| Job | Target | Budget | Max mutants | coverage_prefilter |
| --- | --- | --- | ---: | --- |
| `prometheus-labels` | `./model/labels` | `5m` | `8` | `true` |
| `prometheus-rules` | `./rules` | `5m` | `8` | `true` |
| `prometheus-storage-remote` | `./storage/remote` | `5m` | `8` | `true` |

Hosted result summary:

| Target | Generated | Effective | Killed | Survived | Not covered | Failure kind | Decision signal |
| --- | ---: | ---: | ---: | ---: | ---: | --- | --- |
| `./model/labels` | `8` | `5` | `5` | `0` | `3` | none | operationally healthy |
| `./rules` | `0` | `0` | `0` | `0` | `0` | `runner_error` | baseline test path timed out before mutation |
| `./storage/remote` | `0` | `0` | `0` | `0` | `0` | `runner_error` | baseline test path timed out before mutation |

Two important facts follow from that:

1. The alternatives were not merely colder on survivor yield. They failed to
   reach mutation at all on GitHub-hosted runners in this bounded lane.
2. The current `./model/labels` target remained the only candidate that was
   both:
   - denominator-healthy enough to preserve usable rollout evidence
   - reproducible under the released hosted execution path

## Failure Shape Of The Alternatives

Both hotter alternatives failed in the same way:

- `runner_error: baseline tests failed before mutation`
- underlying runner detail: `go test -coverprofile ... ./...`
- status reason: `test command timed out`
- the visible output was dominated by dependency downloads rather than mutation
  execution

That matters because it makes the current decision narrower and cleaner.

This is **not** evidence that:

- `./rules` is a bad package
- `./storage/remote` is a bad package
- the local scout was misleading about review-bearing signal

It **is** evidence that, under the current released hosted control-lane shape,
those packages are less reproducible than `./model/labels`.

## Decision

Keep `prometheus/prometheus ./model/labels` as the current medium-service
hosted control lane.

Reason:

- it remains operationally healthy on GitHub-hosted runners
- it preserves useful denominator evidence without workflow failure
- both hotter local alternatives regressed from "review-bearing local scout"
  into "baseline timeout before mutation" on the actual hosted surface we are
  trying to represent

## What Changed In The Understanding

Before this comparison, the open question was:

> Is `./model/labels` too cold, and should it be replaced by a hotter package?

After the hosted comparison, the narrower answer is:

> `./model/labels` is still the best current hosted control lane for this repo,
> because the hotter local alternatives do not yet survive the same bounded
> hosted execution path.

That is a meaningful improvement in calibration knowledge even though the lane
did not change.

## Practical Consequence

For future hosted medium-service control waves on the released action surface:

- keep `prometheus/prometheus ./model/labels` as the bounded reproducible lane
- treat `./rules` and `./storage/remote` as promising local scout targets, not
  as current hosted replacements
- only reopen retargeting if the hosted baseline path for those hotter
  packages becomes reproducible enough to reach mutation reliably

## Conclusion

Issue `#329` now has an explicit evidence-backed decision:

- local scouting found hotter review-bearing packages
- the released hosted comparison proved those hotter packages are not yet good
  replacements for the current control lane
- the correct decision for now is to keep `./model/labels` rather than
  overfit to stronger local-only survivor yield
