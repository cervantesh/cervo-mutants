# 2026-06-19 Hosted Wave Recheck After Actionhelper Extraction

Tracking issue: [#280](https://github.com/cervantesh/cervo-mutants/issues/280)

Date: 2026-06-19

This note records the first hosted external-wave recheck against current
`main` after the hosted hardening sequence through [#278](https://github.com/cervantesh/cervo-mutants/issues/278).

The goals were specific:

1. verify that `.github/workflows/external-action-wave.yml` still runs end to
   end after moving wave-result and wave-summary generation into
   `cmd/actionhelper`
2. pin one clean hosted manifest to current repo-head with fresh
   `tracking_issue`, `action_ref`, and `install_path` metadata
3. persist committed current-main evidence for denominator health, actionable
   review units, recommendations, semantic grouping, ledgers, and governance
   suggestions

Committed aggregate artifact:

- [2026-06-19-external-github-action-wave-post-hardening-main-summary.json](2026-06-19-external-github-action-wave-post-hardening-main-summary.json)

## Inputs

Workflow and manifest under test:

- [.github/workflows/external-action-wave.yml](../../.github/workflows/external-action-wave.yml)
- [external-github-action-wave-post-hardening-main.json](external-github-action-wave-post-hardening-main.json)

Verification runs:

- initial failing run: `27847103082`
- initial failing run URL:
  `https://github.com/cervantesh/cervo-mutants/actions/runs/27847103082`
- successful rerun: `27847223361`
- successful rerun URL:
  `https://github.com/cervantesh/cervo-mutants/actions/runs/27847223361`
- branch carrying the manifest and workflow repair:
  `codex/280-post-hardening-hosted-wave`
- action surface under test:
  `github-action@aa5c10255137df7a3dfa1480f088fd999ebdccfd`

## What Changed

This pass started as a manifest-only recheck, but the first attempt exposed a
shallow hosted-workflow regression immediately:

- the helper commands introduced by `#278` were invoked as
  `go run ./cervomut-source/cmd/actionhelper ...`
- those invocations ran from the workspace root on GitHub-hosted runners
- Go therefore resolved `GOMOD=/dev/null` and failed with:
  `go: go.mod file not found in current directory or any parent directory`

Issue `#280` repaired that workflow path by running helper commands from the
checked-out `cervomut-source` directory while still emitting `wave-result.json`
and `wave-summary.json` to the workspace root for artifact upload.

The successful rerun on the same manifest then completed without further hosted
plumbing changes.

## Result

Aggregate result from successful run `27847223361`:

- selected repos: `3`
- reports captured: `3`
- missing reports: `0`
- generated mutants: `30`
- covered mutants: `24`
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

Per-repo result:

| Repo | Job | Report | Generated | Effective | Killed | Survived | Not covered | Score | Actionable | Review units | Recommendations | Ledger | Denominator health |
| --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| `gjson-root` | `success` | `full` | `10` | `10` | `7` | `3` | `0` | `70` | `77.77777777777779` | `2` | `3` | `2` | `healthy` |
| `logrus-root` | `success` | `full` | `10` | `4` | `4` | `0` | `6` | `100` | `100` | `0` | `0` | `0` | `warning: not_covered_exceeds_effective, high_score_poor_denominator_health` |
| `pflag-root` | `success` | `full` | `10` | `10` | `7` | `3` | `0` | `70` | `70` | `3` | `3` | `1` | `healthy` |

## Findings

1. The hosted wave path did regress during the actionhelper extraction, but the
   break was shallow and fully repaired in the same issue. The failure happened
   before mutation execution and did not reflect a mutation-engine defect.
2. Current repo-head preserves the hosted review yield already seen in the
   released-wave evidence. The successful rerun reproduced the same bounded
   aggregate signal as the `v0.4.1` released wave:
   - `18` killed
   - `6` survived
   - `6` not covered
   - `5` actionable review units
   - `6` recommendation entries
   - `3` ledger entries
3. The remaining weak repo is still `logrus-root`, and it is weak for
   denominator-health reasons rather than triage collapse. It again shows
   `effective=4`, `not_covered=6`, and zero actionable review units.
4. The committed evidence is cleaner than earlier ad hoc reruns because the
   manifest now identifies:
   - `tracking_issue: "#280"`
   - `action_ref: "aa5c10255137df7a3dfa1480f088fd999ebdccfd"`
   - `install_path: "github-action@aa5c10255137df7a3dfa1480f088fd999ebdccfd"`

## Conclusion

`#280` now provides the post-hardening evidence it was meant to create:

- a dedicated manifest for current `main`
- a repaired hosted workflow after the `#278` extraction
- a successful external run with committed aggregate evidence
- no sign that the actionhelper extraction changed survivor or triage semantics
  on this bounded public sample

This is strong enough to merge as the current-main hosted verification record
and use as durable evidence for future field-calibration and hosted-action
maintenance work.
