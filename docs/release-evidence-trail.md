# Release Evidence Trail

Tracking issue: [#293](https://github.com/cervantesh/cervo-mutants/issues/293)

This document keeps a lightweight, auditable trail of the public evidence that
backs each post-hardening release.

It is intentionally narrower than a full release retrospective. The goal is to
make these questions easy to answer from one place:

- which public tag was published when
- which public assets and upgrade notes shipped with it
- which public evidence supports the install and compatibility claims
- which hosted-wave evidence exists for that specific release
- how the release-to-release confidence story is changing over time

## Common Evidence Contract For Every Release

Every row in this trail assumes the same public release gates:

- the tag was published through the public
  [release workflow](../.github/workflows/release.yml)
- publication was blocked on:
  - `verify-compat`
  - `verify-install` from `go install`
  - `verify-release`
  - `verify-install` from the Linux release archive
- the published release includes:
  - `release-manifest.json`
  - `SHA256SUMS`
  - cross-platform archives
  - a matching upgrade note under [docs/upgrade-notes/](upgrade-notes/)

That means the trail does **not** need a private checklist per tag. The release
workflow already enforces the install and compatibility gates. This document
records the public artifacts and release-specific validation notes that make
that enforcement auditable across multiple cycles.

## Post-Hardening Release Trail

| Release | Published UTC | Public release artifacts | Install and compatibility evidence | Hosted-wave evidence | What this cycle added |
| --- | --- | --- | --- | --- | --- |
| [`v0.4.0`](https://github.com/cervantesh/cervo-mutants/releases/tag/v0.4.0) | `2026-06-19T13:34:12Z` | [release-manifest.json](https://github.com/cervantesh/cervo-mutants/releases/download/v0.4.0/release-manifest.json), [SHA256SUMS](https://github.com/cervantesh/cervo-mutants/releases/download/v0.4.0/SHA256SUMS), [upgrade note](upgrade-notes/v0.4.0.md) | published under the shared release workflow gates in [releasing.md](releasing.md) and [.github/workflows/release.yml](../.github/workflows/release.yml) | [released hosted wave](evaluations/2026-06-19-external-github-action-wave-v0.4.0.md), [released triage-yield note](evaluations/2026-06-19-released-adoption-triage-yield.md) | first post-hardening public tag with explicit released-surface hosted evidence and committed released-artifact triage analysis |
| [`v0.4.1`](https://github.com/cervantesh/cervo-mutants/releases/tag/v0.4.1) | `2026-06-19T17:18:06Z` | [release-manifest.json](https://github.com/cervantesh/cervo-mutants/releases/download/v0.4.1/release-manifest.json), [SHA256SUMS](https://github.com/cervantesh/cervo-mutants/releases/download/v0.4.1/SHA256SUMS), [upgrade note](upgrade-notes/v0.4.1.md) | same shared release workflow gates; the public tag repeated the install/archive/compatibility publication contract rather than introducing a new one-off path | [released hosted wave](evaluations/2026-06-19-external-github-action-wave-v0.4.1.md), [broader-profile sample](evaluations/2026-06-19-external-github-action-wave-v0.4.1-profile-sample.md) | proved the release process repeated cleanly and extended public evidence beyond the initial released control sample |
| [`v0.4.2`](https://github.com/cervantesh/cervo-mutants/releases/tag/v0.4.2) | `2026-06-19T20:49:17Z` | [release-manifest.json](https://github.com/cervantesh/cervo-mutants/releases/download/v0.4.2/release-manifest.json), [SHA256SUMS](https://github.com/cervantesh/cervo-mutants/releases/download/v0.4.2/SHA256SUMS), [upgrade note](upgrade-notes/v0.4.2.md) | same shared release workflow gates; the release-era follow-up later added explicit release-alignment checks for the live hosted-wave and install guidance surfaces | [released hosted wave](evaluations/2026-06-19-external-github-action-wave-v0.4.2.md), [broader-profile sample](evaluations/2026-06-19-external-github-action-wave-v0.4.2-profile-sample.md), [structured adoption-feedback wave](evaluations/2026-06-19-released-adoption-feedback-wave.md) | repeated the release cycle again, refreshed broader-profile released evidence, and converted released-surface rollout evidence into structured adoption-feedback issues |

## Comparison View

The release-confidence trend is intentionally simple:

| Release | Public asset set complete | Matching upgrade note | Released hosted-wave note | Broader-profile released sample | Structured adoption-feedback issues |
| --- | --- | --- | --- | --- | --- |
| `v0.4.0` | yes | yes | yes | no | no |
| `v0.4.1` | yes | yes | yes | yes | no |
| `v0.4.2` | yes | yes | yes | yes | yes |

That progression matters more than any one tag in isolation:

- `v0.4.0` established the first credible released-surface evidence line
- `v0.4.1` showed the release process repeated and broadened the profile mix
- `v0.4.2` repeated the cycle again and added structured adoption-feedback
  capture on top of the released hosted evidence

This is the evidence trail behind the maturity claim that release operations are
no longer a one-off rescue effort, even though the repo still needs more
release cycles over time.

## How To Extend The Trail

After each public release:

1. add the new row with:
   - release URL
   - `release-manifest.json`
   - `SHA256SUMS`
   - matching upgrade note
   - release-specific hosted-wave or validation notes
2. update the comparison table
3. update [releasing.md](releasing.md) if the public release contract itself
   changed
4. update [project-maturity-assessment.md](project-maturity-assessment.md) only
   if the new release changes the maturity claim boundary, not merely because a
   new row exists
