# Go Repository Pool For Anti-Fitting Calibration

Tracking issue: https://gitea.cervbox.synology.me/CervoSoft/cervo-mutant/issues/13

This pool exists to keep CervoMutant from overfitting to Cobra. Cobra remains a
useful tuning target, but it must not be the arbiter for operator promotion,
ranking weights, coverage behavior, or performance claims.

The machine-readable manifest is
[go-repo-pool-40.json](go-repo-pool-40.json).

## Source Rationale

The pool prioritizes repositories that are either:

- mentioned by Go empirical-study or benchmark material, such as `golang/go`,
  `kubernetes/kubernetes`, `moby/moby`, `gin-gonic/gin`, Hugo, and Kubernetes
  benchmark packages;
- already relevant to Go mutation tools and comparison studies, such as Cobra,
  pflag, and other CLI libraries;
- important Go ecosystem libraries with fast, local test targets and distinct
  code shapes: web, parser, logging, numeric, database, messaging, storage,
  distributed systems, and validation.

## Lanes

| Lane | Meaning | Rule |
| --- | --- | --- |
| `tuning` | Allowed to guide implementation choices. | Use sparingly and document every decision. |
| `validation` | Used to check whether tuning generalizes. | May block promotion if it regresses. |
| `holdout` | Blind validation. | Do not use for tuning before freezing a candidate change. |
| `special` | Important but expensive or setup-sensitive. | Run only scoped smoke or curated packages. |

## First Execution Plan

1. Clone all 40 candidates with shallow clones.
2. Run `go test <target>` for each target.
3. Run `cervomut run <target> --dry-run --policy ci-fast --max-mutants 10`.
4. Mark repos as runnable, needs-scope, needs-setup, or unsuitable.
5. Run `ci-balanced --max-mutants 100` only on the first 12-16 runnable repos
   with domain diversity.
6. Keep holdout results separate until operator/ranking changes are frozen.

Runner:

```powershell
.\scripts\calibration-smoke.ps1 -Limit 40
```

Optional bounded mutation run for selected repos:

```powershell
.\scripts\calibration-smoke.ps1 -Limit 12 -RunMutation -MaxMutants 25 -Workers 2
```

## Initial 40-Repos

| # | Repo | Lane | Domain | Target |
| ---: | --- | --- | --- | --- |
| 1 | `spf13/cobra` | tuning | cli | `./doc` |
| 2 | `spf13/pflag` | validation | cli | `./...` |
| 3 | `golang/go` | special | language | `./src/cmd/compile/...` |
| 4 | `kubernetes/kubernetes` | special | orchestration | `./pkg/scheduler/cache` |
| 5 | `moby/moby` | holdout | containers | `./pkg/...` |
| 6 | `gohugoio/hugo` | holdout | static-site | `./helpers` |
| 7 | `gin-gonic/gin` | validation | web | `./...` |
| 8 | `etcd-io/etcd` | holdout | distributed-systems | `./client/v3/...` |
| 9 | `prometheus/prometheus` | holdout | observability | `./model/...` |
| 10 | `go-gitea/gitea` | validation | devtools | `./modules/...` |
| 11 | `rclone/rclone` | holdout | storage | `./fs/...` |
| 12 | `ethereum/go-ethereum` | special | blockchain | `./common/...` |
| 13 | `hashicorp/terraform` | holdout | iac | `./internal/addrs/...` |
| 14 | `hashicorp/consul` | validation | distributed-systems | `./api/...` |
| 15 | `hashicorp/vault` | special | security | `./sdk/...` |
| 16 | `grpc/grpc-go` | validation | networking | `./metadata` |
| 17 | `go-chi/chi` | tuning | web | `./...` |
| 18 | `labstack/echo` | validation | web | `./...` |
| 19 | `beego/beego` | holdout | web | `./core/...` |
| 20 | `gofiber/fiber` | validation | web | `./...` |
| 21 | `stretchr/testify` | tuning | testing | `./assert` |
| 22 | `uber-go/zap` | validation | logging | `./zapcore` |
| 23 | `zeromicro/go-zero` | holdout | microservices | `./core/...` |
| 24 | `sirupsen/logrus` | tuning | logging | `./...` |
| 25 | `go-playground/validator` | tuning | validation | `./...` |
| 26 | `google/uuid` | tuning | utility | `./...` |
| 27 | `shopspring/decimal` | tuning | numeric | `./...` |
| 28 | `tidwall/gjson` | validation | parser | `./...` |
| 29 | `tidwall/sjson` | validation | parser | `./...` |
| 30 | `buger/jsonparser` | validation | parser | `./...` |
| 31 | `segmentio/encoding` | holdout | encoding | `./...` |
| 32 | `pelletier/go-toml` | validation | parser | `./...` |
| 33 | `BurntSushi/toml` | holdout | parser | `./...` |
| 34 | `go-yaml/yaml` | special | parser | `./...` |
| 35 | `urfave/cli` | validation | cli | `./...` |
| 36 | `alecthomas/kingpin` | holdout | cli | `./...` |
| 37 | `rs/zerolog` | validation | logging | `./...` |
| 38 | `jackc/pgx` | special | database | `./pgtype` |
| 39 | `redis/go-redis` | validation | database | `./internal/...` |
| 40 | `nats-io/nats.go` | holdout | messaging | `./...` |

