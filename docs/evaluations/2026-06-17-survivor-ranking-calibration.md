# 2026-06-17 Survivor Ranking Calibration Review

Tracking issue: https://github.com/cervantesh/CervoMutants/issues/13

This note records the manual survivor review requested by issue #13. The goal
was to check whether current survivor ranking defaults match human judgment
across Cobra, pflag, one first-party application core target, and one
first-party shared library target.

First-party repository names are intentionally anonymized in this public
document. The underlying runs were executed locally against the same
first-party targets used in earlier calibration notes.

## Run Setup

Environment:

- OS: Windows amd64
- Go: `go1.25.6`
- isolation: `overlay`
- workers: `2`
- policy: `ci-balanced`
- sample: deterministic
- max mutants per target: `20`
- budget per target: `3m`

Command shape:

```powershell
cervomut run <target> --policy ci-balanced --max-mutants 20 --sample deterministic --workers 2 --budget 3m --out <out>
```

Targets:

- Cobra: `./doc`
- pflag: `./...`
- first-party application core: `./core`
- first-party shared library: `./...`

## Sample Results

All four runs had healthy denominators: every generated mutant was covered and
executed, with no timeout or compile-error rows in this sample.

| Target | Generated | Killed | Survived | Score |
| --- | ---: | ---: | ---: | ---: |
| Cobra `./doc` | 20 | 12 | 8 | 60.00% |
| pflag `./...` | 20 | 17 | 3 | 85.00% |
| First-party application core `./core` | 20 | 18 | 2 | 90.00% |
| First-party shared library `./...` | 20 | 6 | 14 | 30.00% |

## Reviewed Survivors

Reviewed sample size: `9` survivors.

Classification summary:

- useful: `6`
- equivalent-risk or redundant: `3`
- invalid: `0`
- unclear: `0`

| Target | Mutant | Classification | Review note | Report context sufficient? |
| --- | --- | --- | --- | --- |
| Cobra | `doc/man_docs.go:67` `!= -> ==` in `GenManTreeFromOpts` | useful | Missing test for blank `CommandSeparator`; mutation changes generated filename separator behavior. | yes |
| Cobra | `doc/man_docs.go:165` `> -> >=` in `manPrintFlags` | useful | Missing assertion for flags without shorthand; mutation can incorrectly force shorthand formatting. | yes |
| pflag | `bool_slice.go:30` `!= -> ==` in `Set` | useful | Missing negative-path test around CSV parser error handling. | partial |
| pflag | `errors.go:107` `== -> !=` in `Error` | useful | Missing exact formatting assertion for `InvalidValueError` when shorthand is present or deprecated. | partial |
| pflag | `flag.go:1033` `> -> >=` in `stripUnknownFlagValue` | equivalent-risk | Changes `nil` versus empty-slice behavior for the single-argument branch; likely no externally visible effect. | yes |
| First-party application core | `core/errors.go:155` `== -> !=` in `Errors.Error` | useful | Missing exact single-item collection assertion; current tests only check substring presence. | yes |
| First-party application core | `core/errors.go:225` `< -> <=` in `SortStable` | equivalent-risk | Stable-sort comparator boundary survivor is likely redundant/noisy for equal keys, not a first-priority test target. | yes |
| First-party shared library | `retry.go:31` `< -> <=` in `Delay` | equivalent-risk | No observable behavior change for `attempt == 0`; clamp branch already leaves zero unchanged. | yes |
| First-party shared library | `retry.go:80` `== -> !=` in `ClassifyHTTPStatus` | useful | Missing assertion that a non-empty response body stays in the formatted message. | yes |

Additional shared-library observation:

- The same run produced a large cluster of additional `conditionals-boundary`
  survivors at `retry.go:35`, `39`, `43`, `47`, `49`, and `53`.
- Those survivors are highly similar review work and should not dominate the
  top of the queue ahead of clearly fault-revealing negation survivors.

## Conclusions

1. Current report context is good enough for manual triage.
   File, function, diff, and nearby-test hints were enough to classify all
   reviewed survivors. The main weakness is breadth, not absence: on larger
   packages like pflag, the nearby-test list is too broad to point directly to
   the exact test to edit.

2. `conditionals-boundary` was too optimistic at `equivalent_risk=medium`.
   In this review, boundary survivors were overrepresented in the
   equivalent-risk/redundant subset, and a repeated boundary cluster in the
   shared-library sample ranked above more clearly actionable negation
   survivors.

3. Default suppression posture should remain unchanged.
   This sample does not justify enabling any new default `action: suppress`
   rules. The current default posture of report-only or lower-priority audit
   rules remains the safer choice.

## Ranking Change Applied

Based on this review, the default equivalent-risk classification for
`conditionals-boundary` is raised from `medium` to `high`.

Reason:

- it better matches the observed review yield in this manual sample;
- it lowers the priority of comparator-boundary survivors without hiding them;
- it keeps the default suppression posture conservative because high-risk
  defaults still audit and rank lower instead of auto-suppressing.

## Follow-Up

This review closes the manual-calibration requirement from issue #13, but it
also exposes the next refinement step:

- semantic grouping or semantic triage for repeated boundary survivors would
  further reduce duplicate review work without changing raw mutation results.
