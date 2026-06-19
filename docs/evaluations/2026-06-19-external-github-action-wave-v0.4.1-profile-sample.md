# Released GitHub Action Validation Wave: v0.4.1 Profile Sample

Tracking issue: [#264](https://github.com/cervantesh/cervo-mutants/issues/264)

Date: 2026-06-19

This document records the next released-surface GitHub Action wave after the
bounded `pflag/logrus/gjson` sample. The goal here was not to re-prove that the
hosted path works. That was already established. The goal was narrower:

1. keep the released `github-action@v0.4.1` surface under test
2. add broader repository profiles beyond the small-library controls
3. measure whether those broader profiles add effective review signal or only
   increase denominator-poor results

Committed aggregate artifact:

- [2026-06-19-external-github-action-wave-v0.4.1-profile-sample-summary.json](2026-06-19-external-github-action-wave-v0.4.1-profile-sample-summary.json)

## Inputs

Workflow and manifest under test:

- [.github/workflows/external-action-wave.yml](../../.github/workflows/external-action-wave.yml)
- [external-github-action-wave-v0.4.1-profile-sample.json](external-github-action-wave-v0.4.1-profile-sample.json)

Verification run:

- successful run: `27842880305`
- successful run URL:
  `https://github.com/cervantesh/cervo-mutants/actions/runs/27842880305`
- branch carrying the manifest: `codex/264-broader-v041-validation-wave`
- released surface under test: `github-action@v0.4.1`

## Candidate Mix

This wave intentionally mixed known smaller-repo controls with broader-profile
targets:

| Repo | Target | Profile | Ref style |
| --- | --- | --- | --- |
| `spf13/pflag` | `./...` | `small-library` | release tag |
| `sirupsen/logrus` | `./...` | `medium-library` | release tag |
| `tidwall/gjson` | `./...` | `validation-library` | release tag |
| `prometheus/prometheus` | `./model/labels` | `medium-service-scoped` | pinned commit |
| `kubernetes/apimachinery` | `./pkg/util/sets` | `large-multipackage-scoped` | pinned commit |

The broader-profile repos were pinned by commit SHA so the validation sample
would stay reproducible even without depending on an external "latest tag"
interpretation.

## Result

Aggregate result from run `27842880305`:

- selected repos: `5`
- reports captured: `5`
- missing reports: `0`
- generated mutants: `40`
- effective mutants: `24`
- killed: `18`
- survived: `6`
- not covered: `16`
- timed out: `0`
- compile errors: `0`
- actionable review units: `5`
- semantic group review units: `2`
- recommendation entries: `6`
- recommendation review units: `5`
- collapsed recommendation duplicates: `1`
- ledger entries: `3`
- governance suggestions: `9`
- repos with denominator warnings: `2`

Per-repo result:

| Repo | Profile | Generated | Effective | Killed | Survived | Not covered | Actionable review units | Denominator health |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| `pflag-root` | `small-library` | `10` | `10` | `7` | `3` | `0` | `3` | `healthy` |
| `gjson-root` | `validation-library` | `10` | `10` | `7` | `3` | `0` | `2` | `healthy` |
| `logrus-root` | `medium-library` | `10` | `4` | `4` | `0` | `6` | `0` | `warning: not_covered_exceeds_effective, high_score_poor_denominator_health` |
| `prometheus-labels` | `medium-service-scoped` | `10` | `0` | `0` | `0` | `10` | `0` | `warning: no_effective_mutants` |
| `apimachinery-sets` | `large-multipackage-scoped` | `0` | `0` | `0` | `0` | `0` | `0` | `healthy=false` |

## Profile Interpretation

### Healthy control profiles

The small-library and validation-library controls behaved exactly as hoped:

- `pflag-root` stayed denominator-healthy and produced `3` review units
- `gjson-root` stayed denominator-healthy and produced `2` review units after
  semantic collapse

That confirms the released `v0.4.1` hosted path is still solid for the same
bounded smaller-repo shapes validated earlier.

### Partial-signal medium library

`logrus-root` remained the "mixed" case:

- some effective mutants were still present
- denominator warnings remained
- no new actionable review units appeared

This repo still works as a useful weak-signal comparison point, but it does
not improve broader-profile confidence on its own.

### Broader-profile under-signal

The two new broader-profile targets added scope, but not new effective review
yield:

- `prometheus-labels` generated `10` mutants, but all `10` were `not_covered`
- `apimachinery-sets` generated `0` mutants under this hosted released-surface
  lane

That means the wider sample increased selected repositories and aggregate
`not_covered`, but it did **not** increase:

- effective mutants
- survivors
- actionable review units
- recommendation-bearing review units

In practice, the additional profile breadth did not yet widen released-surface
review signal. It widened the evidence about where current bounded hosted
defaults still underperform.

## Comparison To The Earlier Released v0.4.1 Wave

This broader sample should not be read only as "more repos succeeded."

Compared with the earlier released `v0.4.1` wave across `pflag`, `logrus`, and
`gjson`, the wider sample kept the same effective review signal from the
smaller-repo controls while the new broader-profile repos contributed no new
effective mutation work.

That is the most important result of `#264`:

- the released hosted path stays stable
- the current bounded defaults do **not** yet generalize across broader
  repository profiles just because the workflow succeeds

## Findings

1. The current released hosted path is operationally stable across a broader
   profile sample. All five jobs completed and all reports were captured.
2. Broader repository profiles are still target-sensitive under bounded hosted
   defaults. The additional service-repo-scoped and large-multipackage-scoped
   targets did not add effective review signal in this wave.
3. The smaller-repo control sample remains the source of real released-surface
   actionable yield. The broader wave did not disprove that signal; it showed
   that the same defaults do not yet transfer cleanly to the added profiles.
4. Release claims should remain bounded. This wave supports saying the hosted
   `v0.4.1` path is credible across mixed profiles, but it does **not** support
   saying current bounded defaults are broadly calibrated for service-repo or
   large multi-package targets.
5. The next useful follow-up is profile-specific target calibration, not more
   generic confidence language. That follow-up is now tracked in
   [#265](https://github.com/cervantesh/cervo-mutants/issues/265).

## Threats To Validity

1. The broader-profile repos were still run through tightly bounded `5m`,
   `ci-balanced`, `10`-mutant lanes, so this is a rollout-shape study more than
   a deep mutation study.
2. The service and large-multipackage profiles each used one scoped package
   target, not whole-repo mutation breadth.
3. The external target refs for `prometheus` and `apimachinery` were pinned
   commits rather than named release tags, which improves reproducibility but
   makes the sample less "marketing clean" than the tag-based library controls.
4. This wave shows where broader profiles under-signal; it does not yet prove
   whether better package selection, different bounded defaults, or a separate
   manifest strategy is the best correction.

## Conclusion

Issue `#264` now has the evidence it needed:

- the released `github-action@v0.4.1` hosted path was exercised across a
  broader mix of repository profiles
- the smaller library profiles kept producing real actionable yield
- the broader-profile additions did not produce new effective review signal
  under the same bounded defaults
- the next justified step is targeted calibration for service-repo-scoped and
  large-multipackage profiles, not broader release claims
