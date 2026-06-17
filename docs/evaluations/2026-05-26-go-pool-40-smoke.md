# Go Pool 40 Smoke Run

Tracking issue: https://github.com/cervantesh/cervo-mutants/issues/13

Date: 2026-05-26

This is the first anti-fitting calibration run after defining the 40-repository
pool. The purpose is not to tune CervoMutant yet. The purpose is to classify
which repositories are runnable, which need better scopes, and which can support
bounded mutation runs.

## Commands

Runner:

```powershell
.\scripts\calibration-smoke.ps1 -Limit 40 -MaxMutants 10 -Workers 2 -CloneTimeoutSeconds 60 -TestTimeoutSeconds 30 -DryRunTimeoutSeconds 60
```

Selected mutation sample:

```powershell
.\scripts\calibration-smoke.ps1 -Names cobra,pflag,moby,hugo,prometheus,terraform,grpc-go,echo,logrus,validator,decimal,gjson -RunMutation -MaxMutants 25 -Workers 2 -CloneTimeoutSeconds 60 -TestTimeoutSeconds 60 -DryRunTimeoutSeconds 60 -MutationTimeoutSeconds 180
```

Artifacts:

```text
C:\Users\c___h\AppData\Local\Temp\cervomut-go-pool-40\summary.json
C:\Users\c___h\AppData\Local\Temp\cervomut-go-pool-40\selected-12-mutation-summary.json
```

## 40-Repo Smoke Summary

| Metric | Count |
| --- | ---: |
| Repositories in manifest | 40 |
| Shallow clone succeeded | 40 |
| Baseline `go test <target>` passed within timeout | 23 |
| Baseline timed out at 30s | 4 |
| `cervomut run --dry-run --policy ci-fast --max-mutants 10` passed | 36 |
| Dry-run failed | 4 |

Repos needing scope/setup adjustment:

| Repo | Baseline exit | Dry-run exit | Interpretation |
| --- | ---: | ---: | --- |
| `golang/go` | 1 | 2 | Special repo layout; current target is not a normal module invocation. |
| `kubernetes/kubernetes` | 1 | 2 | Special repo layout / workspace setup needed. |
| `gin-gonic/gin` | 1 | 0 | CervoMutant discovery works; baseline test scope needs dependency/setup review. |
| `etcd-io/etcd` | 1 | 0 | Discovery works; target likely needs narrowed package or env. |
| `go-gitea/gitea` | 124 | 0 | Discovery works; baseline target too broad for 30s. |
| `rclone/rclone` | 124 | 0 | Discovery works; baseline target too broad for 30s. |
| `ethereum/go-ethereum` | 0 | 2 | Baseline passes; dry-run target needs module/discovery adjustment. |
| `hashicorp/consul` | 1 | 0 | Discovery works; baseline setup needed. |
| `hashicorp/vault` | 1 | 0 | Discovery works; baseline setup needed. |
| `beego/beego` | 1 | 0 | Discovery works; baseline setup needed. |
| `gofiber/fiber` | 124 | 0 | Discovery works; baseline target too broad for 30s. |
| `stretchr/testify` | 1 | 0 | Discovery works; selected target or dependency version needs review. |
| `uber-go/zap` | 1 | 0 | Discovery works; selected target needs baseline review. |
| `zeromicro/go-zero` | 124 | 0 | Discovery works; baseline target too broad for 30s. |
| `segmentio/encoding` | 1 | 2 | Needs target adjustment before mutation. |
| `go-yaml/yaml` | 1 | 0 | Discovery works; baseline setup/version review needed. |
| `jackc/pgx` | 1 | 0 | Discovery works; target likely needs package/env adjustment. |
| `nats-io/nats.go` | 1 | 0 | Discovery works; baseline setup needed. |

## Selected 12 Mutation Sample

Bounded mutation used `ci-balanced`, 25 mutants, 2 workers, and 180s timeout per
repo.

| Repo | Lane | Domain | Mutants | Killed | Survived | Not covered | Score | Mutation seconds | Exit |
| --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `cobra` | tuning | cli | 25 | 10 | 14 | 1 | 41.67% | 18.30 | 0 |
| `pflag` | validation | cli | 25 | 12 | 5 | 8 | 70.59% | 18.62 | 0 |
| `moby` | holdout | containers | 25 | 0 | 1 | 24 | 0.00% | 9.24 | 0 |
| `hugo` | holdout | static-site | 25 | 19 | 2 | 4 | 90.48% | 64.49 | 0 |
| `prometheus` | holdout | observability | 25 | 5 | 0 | 20 | 100.00% | 9.54 | 0 |
| `terraform` | holdout | iac | 25 | 11 | 3 | 11 | 78.57% | 18.98 | 0 |
| `grpc-go` | validation | networking | 25 | 14 | 1 | 10 | 93.33% | 19.53 | 0 |
| `echo` | validation | web | 25 | 18 | 2 | 5 | 90.00% | 25.65 | 0 |
| `logrus` | tuning | logging | 25 | 10 | 4 | 10 | 66.67% | 64.30 | 0 |
| `validator` | tuning | validation | 25 | 0 | 0 | 25 | 0.00% | 11.08 | 0 |
| `decimal` | tuning | numeric | - | - | - | - | - | 180.14 | 124 |
| `gjson` | validation | parser | 25 | 20 | 5 | 0 | 80.00% | 109.58 | 0 |

## Conclusions

- The pool is usable: all 40 repos cloned, and 36/40 support CervoMutant dry-run
  with the initial targets.
- Cobra is not representative: the first 12 mutation sample already shows very
  different coverage and score shapes across CLI, containers, static-site,
  observability, IaC, networking, web, logging, validation, numeric, and parser
  domains.
- `not_covered` dominates some repos (`moby`, `validator`, `prometheus`), which
  means coverage baseline and package scope matter more than raw operator tuning
  for those repos.
- `decimal` timed out under the first mutation cap. It should not be used for
  fast calibration until its target or timeout policy is narrowed.
- Several large repos are discovery-viable but baseline-expensive. They should
  stay in special/holdout lanes with curated subpackages.

## Next Selection

Recommended 12 for the first 100-mutant calibration, after this smoke:

| Use | Repos |
| --- | --- |
| Tuning | `cobra`, `logrus`, `validator`, `uuid` |
| Validation | `pflag`, `grpc-go`, `echo`, `gjson` |
| Holdout | `hugo`, `terraform`, `prometheus`, `moby` |

Do not use holdout results to tune defaults until candidate rules are frozen.

