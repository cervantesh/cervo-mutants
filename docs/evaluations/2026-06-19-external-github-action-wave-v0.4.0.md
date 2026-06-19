# Released GitHub Action Validation Wave: v0.4.0

Tracking issue: [#239](https://github.com/cervantesh/cervo-mutants/issues/239)

Date: 2026-06-19

This document records the first external adoption wave that intentionally used a
released GitHub Action surface instead of the current branch source for the
action implementation.

The goal was specific:

1. prove that the hosted wave can bootstrap `cervomut-source` from the released
   `v0.4.0` tag
2. persist `install_path` and `action_ref` into per-repo and aggregate wave
   artifacts
3. leave issue `#239` with committed evidence instead of relying only on
   ephemeral GitHub Actions artifacts

Committed aggregate artifact:

- [2026-06-19-external-github-action-wave-v0.4.0-summary.json](2026-06-19-external-github-action-wave-v0.4.0-summary.json)

## Inputs

Workflow and manifest under test:

- [.github/workflows/external-action-wave.yml](../../.github/workflows/external-action-wave.yml)
- [external-github-action-wave-v0.4.0-candidates.json](external-github-action-wave-v0.4.0-candidates.json)

Verification runs:

- initial failing run: `27836052376`
- fixed successful run: `27836288517`
- successful run URL:
  `https://github.com/cervantesh/cervo-mutants/actions/runs/27836288517`
- branch carrying the workflow fix: `codex/239-released-wave`
- released surface under test: `github-action@v0.4.0`

## What Changed

This wave required two small but important workflow changes:

1. the manifest `action_ref` became operational instead of decorative by
   feeding the checkout of `cervomut-source`
2. the per-repo extraction step started persisting `install_path` and
   `action_ref` into `wave-result.json` and the aggregate `wave-summary.json`

The first attempt exposed a wiring defect in the report-present `jq` branch:
the JSON body wrote `install_path` and `action_ref`, but the command forgot the
matching `--arg` declarations. That broke artifact extraction with exit code
`3` even though the mutation run itself succeeded. The rerun fixed only that
plumbing bug.

## Released Surface Evidence

The successful run did not execute the branch HEAD as the action under test.
The workflow log shows the matrix jobs checking out the released tag for
`cervomut-source`:

- `ref: v0.4.0`
- `git fetch ... +refs/tags/v0.4.0*:refs/tags/v0.4.0*`
- `git checkout --progress --force refs/tags/v0.4.0`
- `HEAD is now at 84f2460 Prepare v0.4.0 release notes and upgrade guide (#207) (#214)`

The same run also preserved the released-surface identity in generated
artifacts:

- `action_ref: "v0.4.0"`
- `install_path: "github-action@v0.4.0"`

That closes the gap where the manifest already carried `action_ref` but the
hosted wave workflow still ignored it.

## Repositories

| Repo | Target | Profile | Policy | Budget | Max mutants |
| --- | --- | --- | --- | --- | ---: |
| `spf13/pflag` | `./...` | small library | `ci-balanced` | `5m` | `10` |
| `sirupsen/logrus` | `./...` | medium library | `ci-balanced` | `5m` | `10` |
| `tidwall/gjson` | `./...` | validation library | `ci-balanced` | `5m` | `10` |

All jobs ran on GitHub-hosted `ubuntu-latest` runners. The branch supplied the
workflow plumbing, but the mutation action itself was bootstrapped from the
released `v0.4.0` tag.

## Result

Aggregate result from run `27836288517`:

- selected repos: `3`
- reports captured: `3`
- missing reports: `0`
- generated mutants: `30`
- effective mutants: `24`
- killed: `18`
- survived: `6`
- not covered: `6`
- timed out: `0`
- compile errors: `0`
- actionable review units: `5`
- semantic group review units: `2`
- governance suggestions: `8`
- repos with denominator warnings: `1`

Per-repo result:

| Repo | Job | Report | Generated | Effective | Killed | Survived | Not covered | Score | Actionable | Denominator health |
| --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| `gjson-root` | `success` | `full` | `10` | `10` | `7` | `3` | `0` | `70` | `77.77777777777779` | `healthy` |
| `logrus-root` | `success` | `full` | `10` | `4` | `4` | `0` | `6` | `100` | `100` | `warning: not_covered_exceeds_effective, high_score_poor_denominator_health` |
| `pflag-root` | `success` | `full` | `10` | `10` | `7` | `3` | `0` | `70` | `70` | `healthy` |

## Findings

1. The released GitHub Action surface is now validated by direct hosted
   evidence. This wave did not just run "the current branch"; it ran
   `github-action@v0.4.0` and kept that identity in the resulting artifacts.
2. The workflow auditability gap is closed for this path. Both per-repo
   `wave-result.json` files and the aggregate summary carry `install_path` and
   `action_ref`.
3. The failure on run `27836052376` was workflow plumbing only, not product
   execution failure. `Run CervoMutants` already succeeded there; the rerun
   proves the defect was confined to artifact extraction.
4. Signal quality on this bounded released-surface wave is credible enough to
   reuse downstream. Two repositories produced full denominator-healthy reports,
   and the third produced real kills plus a concrete denominator warning rather
   than empty evidence.
5. The remaining adoption question is no longer "can the released action run?"
   but "how much review yield do these released-surface artifacts produce?"
   That follow-up belongs under
   [#237](https://github.com/cervantesh/cervo-mutants/issues/237).

## Threats To Validity

1. This validates the released hosted action path more strongly than it
   validates long-running or high-budget mutation campaigns.
2. The wave is intentionally bounded to `10` mutants per repository with a `5m`
   budget, so it is representative of CI-oriented adoption, not deep exhaustive
   analysis.
3. All runs used Linux GitHub-hosted runners. This does not replace the broader
   compatibility matrix.
4. The branch still carried workflow-only fixes that were not part of
   `v0.4.0`. What is validated here is the released action surface plus the
   workflow's ability to target that release correctly.

## Conclusion

Issue `#239` now has the evidence it needed:

- the hosted wave can target a released action tag directly
- the released-surface identity survives into committed artifacts
- the first failed run has a narrow, diagnosed, fixed root cause
- the next useful step is yield analysis from these released-surface artifacts,
  not more debate about whether the release path works at all
