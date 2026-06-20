# External-Action-Wave Helper Compatibility With `v0.4.2`

Tracking issue: [#332](https://github.com/cervantesh/cervo-mutants/issues/332)

Date: 2026-06-20

This note records the regression fix for `external-action-wave` when the
current workflow logic needs newer helper metadata than the action version
under test provides.

## Problem

Before this fix, the workflow checked out one repo copy and used it for both:

- the GitHub Action surface under test
- the helper commands that build per-repo and aggregate wave summaries

That coupling broke as soon as the workflow started passing a new helper flag
that did not exist in the older action ref being tested.

Observed failure:

- run: `27854409001`
- failing step: `Extract wave result`
- concrete error: `flag provided but not defined: -prewarm-modules`

The failing manifest pinned:

- `action_ref: v0.4.2`
- `install_path: github-action@v0.4.2`

So the workflow had a newer shell path, but the checked-out helper code came
from the older `v0.4.2` action source and could not parse the new argument.

## Fix

The workflow now separates the two roles explicitly:

- `action-source`: checked out at the manifest-resolved `action_ref`
- `workflow-source`: checked out at the current workflow commit

After the change:

- `Resolve Go version` still runs from `action-source`
- `Run CervoMutants` still uses `./action-source`
- `build-wave-result`, `render-wave-result-markdown`, and
  `build-wave-summary` run from `workflow-source`

That preserves the tested action surface while allowing the reporting helpers
to evolve with the current workflow.

## Validation

Validation used the older released manifest again:

- [external-github-action-wave-prometheus-medium-service-comparison.json](external-github-action-wave-prometheus-medium-service-comparison.json)

Hosted verification run:

- run: `27854873531`
- URL: `https://github.com/cervantesh/cervo-mutants/actions/runs/27854873531`
- branch: `codex/332-wave-helper-decoupling`

Committed summary artifact:

- [2026-06-20-external-action-wave-v0.4.2-helper-compatibility-summary.json](2026-06-20-external-action-wave-v0.4.2-helper-compatibility-summary.json)

## Result

The important proof is not a different Prometheus mutation outcome. The
important proof is that the workflow completed and produced a valid aggregate
summary while still reporting the tested released surface accurately:

- `action_ref = v0.4.2`
- `install_path = github-action@v0.4.2`
- `reports = 3`
- `missing_reports = 0`
- no summary-stage helper failure

The per-target mutation outcome stayed consistent with the earlier released
wave shape:

- `./model/labels` remained healthy
- `./rules` and `./storage/remote` still timed out in baseline testing before
  mutation

That is acceptable here, because `#332` is a compatibility fix, not a signal
or performance claim.

## Conclusion

`#332` is validated when this evidence lands:

- newer workflow helper metadata no longer breaks waves that intentionally pin
  an older released `action_ref`
- the aggregate summary still reflects the released action surface actually
  under test
- the fix is behaviorally narrow and does not change the underlying mutation
  outcome of the older released wave
