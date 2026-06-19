# 2026-06-19 Hosted Wave Candidate Retargeting

Tracking issue: [#227](https://github.com/cervantesh/cervo-mutants/issues/227)

Date: 2026-06-19

This note records the follow-up that changed the **default hosted adoption-wave
candidate set** after the first tuned hosted manifest still produced weak
denominator health.

The point of this pass was not to change workflow plumbing or summary shape.
Those were already addressed in [#224](https://github.com/cervantesh/cervo-mutants/issues/224)
and [#223](https://github.com/cervantesh/cervo-mutants/issues/223).

The point here was narrower:

1. replace hosted candidates that repeatedly collapsed to `no_effective_mutants`
2. prefer public targets that had already shown healthy denominator behavior in
   the 2026-06-17 external validation wave
3. verify that the default hosted manifest becomes materially healthier on real
   GitHub-hosted runners

Committed aggregate artifact:

- [2026-06-19-external-github-action-wave-candidate-retargeting-summary.json](2026-06-19-external-github-action-wave-candidate-retargeting-summary.json)

## Inputs

Workflow and manifest under test:

- [.github/workflows/external-action-wave.yml](../../.github/workflows/external-action-wave.yml)
- [external-github-action-wave-default-candidates.json](external-github-action-wave-default-candidates.json)

Exact verification run:

- run id: `27833924857`
- run URL:
  `https://github.com/cervantesh/cervo-mutants/actions/runs/27833924857`
- branch: `codex/227-hosted-wave-candidates`
- workflow result: `success`

## Candidate Change

Previous default manifest from `#224`:

- `cobra-doc`
- `logrus-root`
- `grpc-metadata`

New default manifest for `#227`:

- `pflag-root`
- `logrus-root`
- `gjson-root`

Why these replacements were chosen:

- `pflag` and `gjson` had already shown `not_covered=0` in the
  2026-06-17 external validation wave
- both had non-zero survivors in that earlier public wave, so they were better
  bets for hosted denominator health **and** future actionable review yield
- `logrus-root` stayed because it was already the only candidate in the prior
  hosted wave that produced any effective mutants

## Result

### Before vs After

The aggregate comparison here is about **default candidate-set quality**, not
about the same repository triplet under identical targets. That distinction is
important.

| Hosted wave | Default candidates | Covered | Effective | Killed | Survived | Not covered | Warning repos | Actionable review units |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| tuned-but-poor set `27833007931` | `cobra-doc`, `logrus-root`, `grpc-metadata` | `4` | `4` | `4` | `0` | `26` | `3` | `0` |
| retargeted set `27833924857` | `pflag-root`, `logrus-root`, `gjson-root` | `24` | `24` | `18` | `6` | `6` | `1` | `5` |

### Per-Repo Result

| Repo | Target | Covered | Effective | Killed | Survived | Not covered | Healthy |
| --- | --- | ---: | ---: | ---: | ---: | ---: | --- |
| `pflag-root` | `./...` | `10` | `10` | `7` | `3` | `0` | `true` |
| `logrus-root` | `./...` | `4` | `4` | `4` | `0` | `6` | `false` |
| `gjson-root` | `./...` | `10` | `10` | `7` | `3` | `0` | `true` |

### Triage Yield Now Visible

Because `#223` already landed before this pass, the same aggregate summary also
preserves the new triage block:

- actionable review units: `5`
- actionable survivors: `6`
- semantic groups formed: `2`
- recommendation entries: `6`
- ledger entries: `3`
- repos with actionable review units: `2`

That means this candidate retargeting did not only improve denominator health.
It also produced materially better hosted review signal.

## Findings

1. The previous default hosted set was indeed the bottleneck. Replacing
   `cobra-doc` and `grpc-metadata` produced a large denominator-health jump
   without changing workflow machinery.
2. The new default set is materially better for hosted calibration work:
   - `covered`: `4 -> 24`
   - `effective`: `4 -> 24`
   - `not_covered`: `26 -> 6`
   - `warning_repos`: `3 -> 1`
3. The hosted wave now produces actionable review signal instead of only raw
   operational evidence:
   - `6` survivors
   - `5` actionable review units
   - `6` recommendation entries
4. `logrus-root` still carries denominator warnings, so the default set is not
   perfect. But the issue goal was to make it **materially better**, and that
   threshold is clearly met by the verification run.

## Conclusion

`#227` is a successful retargeting pass.

The default hosted manifest is now much healthier for future external
calibration and public GitHub Action demonstrations:

- two of the three default candidates are fully healthy on denominator metrics
- the aggregate hosted wave now produces real reviewable survivor output
- the remaining weak candidate is isolated instead of dominating the whole wave

This is strong enough to merge as the new default hosted candidate set and use
as the evidence base for the broader field-findings report in `#213`.
