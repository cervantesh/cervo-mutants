# Released GitHub Action Validation Wave: v0.4.2

Tracking issue: [#284](https://github.com/cervantesh/cervo-mutants/issues/284)

Date: 2026-06-19

This document records the next external adoption wave that intentionally used
the current public GitHub Action release instead of the current branch source
for the action implementation.

The goals were specific:

1. prove that the hosted adoption wave still bootstraps from the released
   `v0.4.2` tag
2. persist committed released-surface evidence for denominator health,
   actionable review units, recommendations, semantic grouping, ledgers, and
   governance suggestions
3. determine whether the new public release changed review yield relative to
   the prior `v0.4.1` released wave

Committed aggregate artifact:

- [2026-06-19-external-github-action-wave-v0.4.2-summary.json](2026-06-19-external-github-action-wave-v0.4.2-summary.json)

## Inputs

Workflow and manifest under test:

- [.github/workflows/external-action-wave.yml](../../.github/workflows/external-action-wave.yml)
- [external-github-action-wave-v0.4.2-candidates.json](external-github-action-wave-v0.4.2-candidates.json)

Verification run:

- successful run: `27847908487`
- successful run URL:
  `https://github.com/cervantesh/cervo-mutants/actions/runs/27847908487`
- branch carrying the manifest:
  `codex/284-v042-released-wave`
- released surface under test: `github-action@v0.4.2`

## What Changed

This follow-up intentionally validates the newly published release after the
post-hardening fixes from [#280](https://github.com/cervantesh/cervo-mutants/issues/280) became part of the public tag.

The branch change for `#284` stayed intentionally small:

1. add a `v0.4.2` released-wave manifest
2. rerun the hosted validation wave directly against the released `v0.4.2`
   action tag
3. commit the aggregate summary and align the public docs that now point to
   `@v0.4.2`

That makes this run the first released-surface proof that the actionhelper
extraction and helper working-directory repair are both present in the current
public GitHub Action release.

## Released Surface Evidence

The successful run did not execute branch HEAD as the action under test. The
matrix jobs checked out the released tag for `cervomut-source`, and the
persisted artifacts carried the released identity:

- `action_ref: "v0.4.2"`
- `install_path: "github-action@v0.4.2"`

The committed aggregate summary also preserves the richer hosted fields now
expected from the post-hardening path, including:

- explicit Go version metadata per repo
- `coverage_prefilter` state
- `failed_reports`
- `failure_kinds`

That matters because `#284` is not only about "did the action run"; it is also
about proving that the currently documented release surface is the one the repo
has actually validated in hosted CI.

## Repositories

| Repo | Target | Profile | Policy | Budget | Max mutants |
| --- | --- | --- | --- | --- | ---: |
| `spf13/pflag` | `./...` | small library | `ci-balanced` | `5m` | `10` |
| `sirupsen/logrus` | `./...` | medium library | `ci-balanced` | `5m` | `10` |
| `tidwall/gjson` | `./...` | validation library | `ci-balanced` | `5m` | `10` |

All jobs ran on GitHub-hosted `ubuntu-latest` runners. The workflow branch only
supplied the updated manifest; the mutation action itself was bootstrapped from
the released `v0.4.2` tag.

## Result

Aggregate result from run `27847908487`:

- selected repos: `3`
- reports captured: `3`
- missing reports: `0`
- generated mutants: `30`
- covered mutants: `24`
- executed mutants: `24`
- effective mutants: `24`
- killed: `18`
- survived: `6`
- not covered: `6`
- timed out: `0`
- compile errors: `0`
- actionable review units: `5`
- semantic group review units: `2`
- semantic groups formed: `2`
- recommendation entries: `6`
- recommendation review units: `5`
- collapsed recommendation duplicates: `1`
- ledger entries: `3`
- governance suggestions: `8`
- repos with denominator warnings: `1`
- repos with reported failures: `0`

Per-repo result:

| Repo | Job | Report | Generated | Effective | Killed | Survived | Not covered | Score | Actionable | Review units | Recommendations | Rec review units | Ledger | Denominator health |
| --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| `gjson-root` | `success` | `full` | `10` | `10` | `7` | `3` | `0` | `70` | `77.77777777777779` | `2` | `3` | `2` | `2` | `healthy` |
| `logrus-root` | `success` | `full` | `10` | `4` | `4` | `0` | `6` | `100` | `100` | `0` | `0` | `0` | `0` | `warning: not_covered_exceeds_effective, high_score_poor_denominator_health` |
| `pflag-root` | `success` | `full` | `10` | `10` | `7` | `3` | `0` | `70` | `70` | `3` | `3` | `3` | `1` | `healthy` |

## Findings

1. The current public GitHub Action release is operationally stable on the same
   bounded public sample used for the earlier released waves. The `v0.4.2` run
   completed successfully across all three repositories with no missing
   reports.
2. The top-line review yield is unchanged relative to the prior `v0.4.1`
   released-wave evidence. This wave reproduced the same bounded aggregate
   signal:
   - `18` killed
   - `6` survived
   - `6` not covered
   - `5` actionable review units
   - `6` recommendation entries
   - `3` ledger entries
3. That parity matters because `v0.4.2` is the first public tag that includes
   the hosted actionhelper path and helper working-directory repair. On this
   sample, those release changes did not regress survivor or triage behavior.
4. The remaining weak repo is still `logrus-root`, and it is weak for
   denominator-health reasons rather than hosted-path instability or semantic
   triage collapse. It again shows `effective=4`, `not_covered=6`, and zero
   actionable review units.
5. The repo can now reference `v0.4.2` directly in its public adoption and
   GitHub Action guidance without leaning on stale released-surface evidence
   from `v0.4.1`.

## Threats To Validity

1. This validates the current public hosted action path more strongly than it
   validates large or long-running mutation campaigns.
2. The wave is intentionally bounded to `10` mutants per repository with a `5m`
   budget, so it remains representative of CI-oriented adoption rather than
   deep exhaustive analysis.
3. All runs used Linux GitHub-hosted runners. This does not replace the broader
   compatibility matrix.
4. The sample remains narrow and branch/boundary heavy. It is enough to support
   practical rollout guidance, but not broad claims about every operator family
   or heuristic class.

## Conclusion

Issue `#284` now has the evidence it needed:

- the hosted adoption wave runs cleanly against the current public release
  `github-action@v0.4.2`
- the committed released-surface summary preserves actionable-yield fields that
  humans can audit later
- the review yield remains real and stable on the bounded public sample
- the live docs can now point at the current release and its matching released-
  surface evidence instead of relying on the older `v0.4.1` wave
