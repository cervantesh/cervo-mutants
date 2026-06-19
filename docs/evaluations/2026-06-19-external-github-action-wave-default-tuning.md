# 2026-06-19 Hosted External Wave Default Tuning

Tracking issue: [#224](https://github.com/cervantesh/cervo-mutants/issues/224)

Date: 2026-06-19

This note records the first tuning pass on the hosted
`external-action-wave.yml` defaults after the initial hosted wave under-produced
effective mutants.

The goal here was narrower than "solve hosted calibration completely". The goal
was to verify three things:

1. the hosted wave can carry a better default manifest than the original
   smoke-style `ci-fast` settings
2. those tuned defaults increase effective-mutant yield on at least part of the
   sample
3. the aggregate summary artifact preserves the manifest metadata needed for
   later review

Committed aggregate artifact:

- [2026-06-19-external-github-action-wave-default-tuning-summary.json](2026-06-19-external-github-action-wave-default-tuning-summary.json)

## Inputs

Workflow and manifest under test:

- [.github/workflows/external-action-wave.yml](../../.github/workflows/external-action-wave.yml)
- [external-github-action-wave-default-candidates.json](external-github-action-wave-default-candidates.json)

Exact verification run:

- run id: `27833007931`
- run URL:
  `https://github.com/cervantesh/cervo-mutants/actions/runs/27833007931`
- branch: `codex/224-hosted-wave-yield`
- workflow result: `success`

## Changes Under Test

Compared with the initial hosted wave in
[2026-06-19-external-github-action-wave-2.md](2026-06-19-external-github-action-wave-2.md),
this tuning pass changed:

- policy from `ci-fast` to `ci-balanced`
- `max_mutants` from `5` to `10`
- `workers` from `1` to `2`
- sampling to `deterministic`
- the `grpc` target from `./status` to `./metadata`

It also fixed a workflow wiring bug where the `summarize` job tried to read
`plan` outputs without declaring `plan` in `needs`, which left
`tracking_issue` and `manifest_path` empty in the generated `wave-summary.json`
artifact.

## Result

### Before vs After

| Wave | Policy | Max mutants | Workers | Generated | Effective | Killed | Not covered | Metadata in summary |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| initial hosted wave `27831153375` | `ci-fast` | `5` | `1` | `10` | `0` | `0` | `10` | `tracking_issue`/`manifest_path` missing |
| tuned hosted wave `27833007931` | `ci-balanced` | `10` | `2` | `30` | `4` | `4` | `26` | populated |

### Per-Repo Result

| Repo | Target | Generated | Effective | Killed | Not covered | Warning shape |
| --- | --- | ---: | ---: | ---: | ---: | --- |
| `cobra-doc` | `./doc` | `10` | `0` | `0` | `10` | `no_effective_mutants` |
| `grpc-metadata` | `./metadata` | `10` | `0` | `0` | `10` | `no_effective_mutants` |
| `logrus-root` | `./...` | `10` | `4` | `4` | `6` | `not_covered_exceeds_effective`, `high_score_poor_denominator_health` |

## Findings

1. The tuned hosted defaults are materially better than the original
   smoke-style settings. The hosted wave moved from `0` effective mutants and
   `0` kills to `4` effective mutants and `4` kills.
2. The workflow metadata bug is fixed. The generated summary now preserves:
   - `tracking_issue: "#224"`
   - `manifest_path: "docs/evaluations/external-github-action-wave-default-candidates.json"`
3. The current hosted candidate mix is still denominator-poor. Two of the
   three repositories still produced only `not_covered` mutants, and all three
   repos still emitted denominator-health warnings.
4. That means the next tuning step should focus on target and candidate
   selection, not on more workflow plumbing and not yet on semantic-ranking
   weight changes.

## Conclusion

This pass is worth merging even though it does not "solve" hosted calibration:

- the default hosted manifest now produces some real effective-mutant signal
- the workflow summary now carries the metadata needed for later review
- the remaining gap is clearer and narrower: better hosted targets, not missing
  workflow evidence

That is a concrete improvement in both signal yield and auditability, and it
gives the next hosted-wave iteration a cleaner starting point.
