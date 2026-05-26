# Multi-Repo Calibration Protocol

Tracking issue: https://gitea.cervbox.synology.me/CervoSoft/cervo-mutant/issues/10

This calibration exists so CervoMutant can become better than a single-repo
benchmark. Cobra proved the speed architecture is competitive, but policy
defaults should be calibrated across different Go code shapes.

## Target Repositories

| Repository | Why it matters | Minimum calibration |
| --- | --- | --- |
| Cobra | Popular CLI library, already used for tool comparison. | `./doc` quick run plus broader package sampling. |
| CervoClaw | Primary adoption target. | `ci-fast` budgeted PR-style run and `nightly` sampled run. |
| CervoRetry | CervoSoft reusable library target. | Full conservative run if small; otherwise `ci-balanced`. |
| Popular Go repo | External validity check outside CervoSoft. Prefer `stretchr/testify` or `gin-gonic/gin`. | `ci-fast` and `gremlins-compatible` sampled comparison. |

## Required Commands

Use fresh report directories per repo and policy:

```powershell
cervomut run ./... --policy ci-fast --budget 10m --sample deterministic --out .cervomut/calibration/ci-fast
cervomut run ./... --policy ci-balanced --max-mutants 200 --sample deterministic --out .cervomut/calibration/ci-balanced
cervomut run ./... --profile gremlins-compatible --workers 2 --isolation overlay --max-mutants 100 --sample deterministic --out .cervomut/calibration/gremlins-compatible
```

## Metrics To Compare

- wall-clock duration
- generated mutants
- covered mutants
- executed mutants
- killed, survived, `not_covered`, timeout, compile-error
- mutation score, test efficacy, mutation coverage
- survivor rank distribution
- equivalent-risk distribution
- suppression audit hits
- per-operator useful-survivor yield

## Promotion Rules

An operator can move closer to default only if it performs across at least three
repositories:

- low compile-error and timeout rate
- survivors are ranked as actionable by nearby-test and diff context
- equivalent-risk audit does not dominate the survivor set
- runtime remains acceptable under `ci-fast` or `ci-balanced`

An operator should move away from fast CI if it repeatedly produces:

- mostly `not_covered`
- high equivalent-risk survivors without useful hints
- high duplicate signal against another cheaper operator
- long-running packages that consume budget before faster high-signal mutants

## Current Implementation Support

CervoMutant now supports the calibration hooks needed for this study:

- `--policy ci-fast|ci-balanced|nightly|campaign`
- `selection.prefilter` for coverage prefiltering
- `not_covered` in package mode when the baseline coverage profile exists
- budget-aware scheduling by operator recommendation
- ranked survivors in JSON and `report survivors`
- auditable suppression rule hits in JSON

## 2026-05-26 Calibration Sample

Environment:

- OS: Windows under OneDrive paths.
- Binary: local `cervomut` built from issue #10 branch.
- Scope: 20 deterministic mutants per repository and policy.
- Workers: 4.
- Budget: 3 minutes.
- Output root: `C:\Users\c___h\AppData\Local\Temp\cervomut-multirepo-calibration-20260526-r2`.

Commands:

```powershell
cervomut run <target> --policy ci-fast --max-mutants 20 --sample deterministic --workers 4 --budget 3m --out <out>
cervomut run <target> --profile gremlins-compatible --isolation overlay --max-mutants 20 --sample deterministic --workers 4 --budget 3m --out <out>
```

Results:

| Repository | Target | Policy/profile | Seconds | Generated | Covered | Executed | Killed | Survived | Not covered | Score | Mutation coverage |
| --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Cobra | `./doc` | `ci-fast` | 6.21 | 20 | 20 | 20 | 13 | 7 | 0 | 65.00% | 100.00% |
| Cobra | `./doc` | `gremlins-compatible` | 6.31 | 20 | 20 | 20 | 13 | 7 | 0 | 65.00% | 100.00% |
| CervoClaw CervoCore | `./...` | `ci-fast` | 11.51 | 20 | 20 | 20 | 7 | 13 | 0 | 35.00% | 100.00% |
| CervoClaw CervoCore | `./...` | `gremlins-compatible` | 10.16 | 20 | 20 | 20 | 7 | 13 | 0 | 35.00% | 100.00% |
| CervoRetry | `./...` | `ci-fast` | 10.41 | 20 | 20 | 20 | 5 | 15 | 0 | 25.00% | 100.00% |
| CervoRetry | `./...` | `gremlins-compatible` | 5.73 | 20 | 20 | 20 | 5 | 15 | 0 | 25.00% | 100.00% |
| pflag | `./...` | `ci-fast` | 8.42 | 20 | 20 | 20 | 17 | 3 | 0 | 85.00% | 100.00% |
| pflag | `./...` | `gremlins-compatible` | 8.31 | 20 | 20 | 20 | 17 | 3 | 0 | 85.00% | 100.00% |

Findings:

- `ci-fast` is stable across the sampled repositories and does not exceed the
  3-minute budget in these small runs.
- `gremlins-compatible` and `ci-fast` intentionally produce the same operator
  set today, so scores match. Runtime variance is mostly baseline coverage and
  filesystem noise.
- CervoRetry exposed a Windows junction discovery bug: the repository path under
  `CervoSoft` is a junction to `CervoClaw\cervo-retry`, and Go's `WalkDir`
  reports a junction root as non-directory unless the path is normalized with a
  trailing separator. Discovery now handles this case.
- CervoCore and CervoRetry have low sampled scores. That is a test-suite signal,
  not a tool failure: all sampled mutants were covered and executed.
- pflag is a useful external control because it is small, popular, and produced
  a high kill rate under the same budget.

Next calibration step:

- Expand each target to 100 deterministic mutants.
- Add one `nightly` sample for CervoCore and CervoRetry.
- Compare survivor rankings against manual review to tune the rank formula.
- Add a deliberate uncovered fixture to verify `not_covered` reporting in
  package mode with `selection.prefilter: true`.

## 2026-05-26 Issue #11 Expansion

The 100-mutant `ci-balanced` expansion requested by issue #11 is recorded in
[2026-05-26-issue11-signal-followups.md](2026-05-26-issue11-signal-followups.md).

Summary:

| Repository | Generated | Killed | Survived | Not covered | Score | Mutation coverage | Wall time |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Cobra `./doc` | 100 | 46 | 39 | 15 | 54.12% | 85.00% | 31.75s |
| CervoCore `./...` | 100 | 12 | 28 | 60 | 30.00% | 40.00% | 50.46s |
| CervoRetry `./...` | 52 | 18 | 10 | 24 | 64.29% | 53.85% | 16.09s |
| pflag `./...` | 100 | 60 | 14 | 26 | 81.08% | 74.00% | 66.30s |

This run confirms that `ci-balanced` is operationally stable across the first
four calibration targets. The main remaining calibration need is qualitative:
review survivor rankings and suppression candidates against human judgement.

## 2026-05-26 Anti-Fitting Pool

Issue #13 added a 40-repository Go calibration pool and a smoke runner:

- [go-repo-pool-40.md](go-repo-pool-40.md)
- [go-repo-pool-40.json](go-repo-pool-40.json)
- [2026-05-26-go-pool-40-smoke.md](2026-05-26-go-pool-40-smoke.md)

Smoke result: all 40 repositories cloned, 23 passed the initial baseline target
within 30 seconds, and 36 passed CervoMutant dry-run discovery. A selected
12-repository mutation sample completed successfully in 11/12 repos; `decimal`
timed out and needs a narrower target before fast calibration.
