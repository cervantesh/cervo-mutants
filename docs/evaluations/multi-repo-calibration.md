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

