# Gremlins-Focused Comparison Pool

Tracking issue: https://gitea.cervbox.synology.me/CervoSoft/cervo-mutant/issues/13

Date: 2026-05-28

Reusable tool-specific lessons from this campaign are maintained in
[External tool findings ledger](tool-findings.md).

This campaign narrows external comparison to:

```text
CervoMutant vs Gremlins
```

`gomu` and `go-mutesting` remain historical findings only. They repeatedly
failed to produce stable, comparable metrics under Windows and WSL2 resource
limits, so they are no longer useful as primary benchmarks for CervoMutant.

## Current Anchors

The useful prior cross-tool anchors are:

| Repository | Target | Size bucket | Why it stays |
| --- | --- | --- | --- |
| `cobra` | `./doc` | small | Existing apples-to-apples CLI benchmark; CervoMutant is faster, Gremlins has stronger score. |
| `grpc-go` | `./metadata` | medium | CervoMutant completed cleanly in WSL2 and beat Gremlins on score. |
| `hugo` | `./helpers` | large | Stress target where Gremlins currently has better effective coverage/stability. |

`pflag` is kept in the pool but is not a primary anchor because previous runs
often exited without useful metrics for one or both tools.

## Size Buckets

The buckets are pragmatic, not prestige-based:

- `small`: library-sized targets expected to run locally without setup and give
  useful metrics inside a short CI budget.
- `medium`: framework or infrastructure libraries with broader dependency graphs
  or more package/test variance, but still practical as bounded CI targets.
- `large`: apps, platforms, or expensive framework targets that test scheduler,
  timeout, memory, coverage, and partial-report behavior.

## Small Pool: First Run

These are the first 10 to run. They intentionally cover CLI, logging, parsing,
numeric, validation, and utility domains while staying small enough for a local
Gremlins-vs-CervoMutant comparison.

| # | Repo | Target | Lane | Domain | Reason |
| ---: | --- | --- | --- | --- | --- |
| 1 | `cobra` | `./doc` | tuning | cli | Existing anchor; validates continuity with prior results. |
| 2 | `pflag` | `./...` | validation | cli | Cobra-adjacent but smaller; useful to detect CLI overfitting. |
| 3 | `logrus` | `./...` | tuning | logging | Mature small logging library with branch/string behavior. |
| 4 | `uuid` | `./...` | tuning | utility | Small correctness-focused package. |
| 5 | `decimal` | `./...` | tuning | numeric | Numeric/operator-sensitive code; previous timeout makes it a useful stress-small case. |
| 6 | `gjson` | `./...` | validation | parser | Fast parser/string-heavy library with useful survivor signal. |
| 7 | `sjson` | `./...` | validation | parser | JSON update logic, separate from `gjson` read path. |
| 8 | `jsonparser` | `./...` | validation | parser | Independent parser implementation. |
| 9 | `burntsushi-toml` | `./...` | holdout | parser | Independent TOML parser holdout. |
| 10 | `urfave-cli` | `./...` | validation | cli | CLI library independent from Cobra/pflag. |

Run command:

```powershell
.\scripts\compare-tools-pool.ps1 `
  -Names cobra,pflag,logrus,uuid,decimal,gjson,sjson,jsonparser,burntsushi-toml,urfave-cli `
  -Tools cervomut,gremlins `
  -OutputRoot "$env:TEMP\cervomut-gremlins-small-10" `
  -TimeoutSeconds 600 `
  -Workers 2 `
  -Resume `
  -MaxProcessTreeMemoryMB 6144 `
  -GoMemoryLimit 3GiB `
  -GoMaxProcs 2 `
  -GoFlags "-p=2"
```

Artifacts:

```text
C:\Users\c___h\AppData\Local\Temp\cervomut-gremlins-small-10
```

## Medium Pool

These should run after small-pool parser/stability issues are resolved.

| # | Repo | Target | Lane | Domain | Reason |
| ---: | --- | --- | --- | --- | --- |
| 1 | `grpc-go` | `./metadata` | validation | networking | Existing anchor; strong CervoMutant result in WSL2. |
| 2 | `echo` | `./...` | validation | web | Web framework with prior CervoMutant signal. |
| 3 | `chi` | `./...` | tuning | web | Smaller router, good fast web target. |
| 4 | `gin` | `./...` | validation | web | Popular web framework; setup needs review. |
| 5 | `fiber` | `./...` | validation | web | Alternative web framework with different dependency shape. |
| 6 | `validator` | `./...` | tuning | validation | Tag/rule validation; prior not-covered result needs Gremlins comparison. |
| 7 | `testify` | `./assert` | tuning | testing | Assertion logic; survivors should be human-actionable. |
| 8 | `zap` | `./zapcore` | validation | logging | Performance-oriented logging core. |
| 9 | `go-toml` | `./...` | validation | parser | Config parser with many edge cases. |
| 10 | `go-redis` | `./internal/...` | validation | database | Protocol/client internals without live Redis target. |

## Large Pool

These are for scheduler, budget, memory, and partial-report validation. They
should be run one at a time or in WSL2/cgroup-protected mode when possible.

| # | Repo | Target | Lane | Domain | Reason |
| ---: | --- | --- | --- | --- | --- |
| 1 | `hugo` | `./helpers` | holdout | static-site | Existing large anchor; Gremlins currently wins effective coverage. |
| 2 | `moby` | `./pkg/...` | holdout | containers | Large production codebase; previous tools struggled. |
| 3 | `prometheus` | `./model/...` | holdout | observability | Metrics/parser-heavy production Go. |
| 4 | `terraform` | `./internal/addrs/...` | holdout | iac | Config/state semantics with bounded package target. |
| 5 | `gitea` | `./modules/...` | validation | devtools | Large Go app relevant to Cervo infrastructure. |
| 6 | `rclone` | `./fs/...` | holdout | storage | IO-heavy behavior and broad package graph. |
| 7 | `etcd` | `./client/v3/...` | holdout | distributed-systems | Distributed systems client code with concurrency. |
| 8 | `kubernetes` | `./pkg/scheduler/cache` | special | orchestration | Empirical-study-grade large repo; scoped target only. |
| 9 | `go` | `./src/cmd/compile/...` | special | language | Special layout; useful only after runner setup adapts. |
| 10 | `go-ethereum` | `./common/...` | special | blockchain | Large correctness-heavy codebase; scoped package target. |

## Evaluation Rules

- Compare only CervoMutant and Gremlins.
- Report denominators explicitly: generated, covered, executed, killed,
  survived, not-covered, timed-out, compile errors.
- Do not compare scores without noting not-covered and timed-out counts.
- Track wall time and seconds per executed mutant.
- Treat controlled watchdog exits as a useful CervoMutant/runner property, but
  not as a quality win unless partial metrics are preserved.
- Do not tune from holdout results until a candidate change is frozen.

## Small-Pool Result Table

First run completed on Windows/OneDrive.

Command:

```powershell
.\scripts\compare-tools-pool.ps1 `
  -Names cobra,pflag,logrus,uuid,decimal,gjson,sjson,jsonparser,burntsushi-toml,urfave-cli `
  -Tools cervomut,gremlins `
  -OutputRoot "$env:TEMP\cervomut-gremlins-small-10" `
  -TimeoutSeconds 600 `
  -Workers 2 `
  -Resume `
  -MaxProcessTreeMemoryMB 6144 `
  -GoMemoryLimit 3GiB `
  -GoMaxProcs 2 `
  -GoFlags "-p=2"
```

Artifacts:

```text
C:\Users\c___h\AppData\Local\Temp\cervomut-gremlins-small-10\summary.json
```

| Repo | Tool | Exit | Seconds | Total | Killed | Survived | Not covered | Timed out | Errors | Score | Notes |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| `cobra` | `cervomut` | 0 | 28.49 | 69 | 51 | 18 | 0 | 0 | 0 | 73.91 | Complete metrics. |
| `cobra` | `gremlins` | 0 | 49.59 | 85 | 56 | 29 | 5 | 2 | 0 | 65.88 | Complete metrics. |
| `pflag` | `cervomut` | 0 | 223.33 | 214 | 185 | 27 | 0 | 2 | 0 | 86.45 | Complete metrics. |
| `pflag` | `gremlins` | 0 | 8.20 |  |  |  |  |  |  |  | Panic after coverage; no JSON report. |
| `logrus` | `cervomut` | 0 | 206.62 | 103 | 73 | 29 | 0 | 1 | 0 | 70.87 | Complete metrics. |
| `logrus` | `gremlins` | 0 | 9.64 |  |  |  |  |  |  |  | Panic after coverage; no JSON report. |
| `uuid` | `cervomut` | 0 | 182.87 | 89 | 70 | 16 | 0 | 3 | 0 | 78.65 | Complete metrics. |
| `uuid` | `gremlins` | 0 | 7.97 |  |  |  |  |  |  |  | Panic after coverage; no JSON report. |
| `decimal` | `cervomut` | 124 | 611.43 |  |  |  |  |  |  |  | Timeout before final metrics. |
| `decimal` | `gremlins` | 0 | 6.99 |  |  |  |  |  |  |  | Panic after coverage; no JSON report. |
| `gjson` | `cervomut` | 124 | 608.83 |  |  |  |  |  |  |  | Timeout before final metrics. |
| `gjson` | `gremlins` | 0 | 6.36 |  |  |  |  |  |  |  | Panic after coverage; no JSON report. |
| `sjson` | `cervomut` | 126 | 47.16 |  |  |  |  |  |  |  | Process-tree watchdog kill. |
| `sjson` | `gremlins` | 0 | 6.64 |  |  |  |  |  |  |  | Panic after coverage; no JSON report. |
| `jsonparser` | `cervomut` | 0 | 512.67 | 874 | 827 | 40 | 0 | 7 | 0 | 94.62 | Complete metrics, but near 10-minute budget. |
| `jsonparser` | `gremlins` | 0 | 6.82 |  |  |  |  |  |  |  | Panic after coverage; no JSON report. |
| `burntsushi-toml` | `cervomut` | 124 | 605.28 |  |  |  |  |  |  |  | Timeout before final metrics. |
| `burntsushi-toml` | `gremlins` | 0 | 7.80 |  |  |  |  |  |  |  | Panic after coverage; no JSON report. |
| `urfave-cli` | `cervomut` | 124 | 606.75 |  |  |  |  |  |  |  | Timeout before final metrics. |
| `urfave-cli` | `gremlins` | 0 | 7.52 |  |  |  |  |  |  |  | Panic after coverage; no JSON report. |

## Small-Pool Findings

1. `cobra` is the only small-pool repo where both tools produced complete
   comparable metrics in this Windows run.
2. Gremlins panicked after coverage collection on 9/10 small-pool targets. The
   process returned exit 0 in the harness, so the comparison runner must classify
   `panic:` in the log as an execution failure even when the process exit code is
   misleading.
3. CervoMutant produced complete metrics on 5/10 targets: `cobra`, `pflag`,
   `logrus`, `uuid`, and `jsonparser`.
4. CervoMutant timed out on 4/10 targets: `decimal`, `gjson`,
   `burntsushi-toml`, and `urfave-cli`.
5. CervoMutant watchdog-killed `sjson` quickly. This is safer than exhausting the
   host, but the result still needs partial metrics to be useful.
6. Full mutation is too expensive for this "small" pool. `jsonparser` generated
   874 mutants and took 512.67s. The next comparison should be bounded by
   deterministic sample or budget on both tools where possible.
7. CervoMutant has a reporting advantage when it completes: the JSON captures
   denominator counts and timeout counts consistently. The weakness is that
   timeout/watchdog exits currently lose partial per-mutant metrics.

## Immediate Improvements From This Run

- Update the comparison harness to detect Gremlins panics in logs and mark them
  as `panic` or `tool_error`, not successful runs with null metrics.
- Add CervoMutant checkpoint/partial-summary preservation for timeout and
  watchdog exits.
- Add a bounded comparison mode for CervoMutant vs Gremlins:
  - deterministic sample when the tool supports it;
  - otherwise fixed budget and explicit "not comparable denominator" flag.
- Re-run the small pool in WSL2 to distinguish Windows-specific Gremlins panic
  behavior from general Gremlins behavior.

## Small-Pool WSL2/Cgroup Run

Second run completed in Ubuntu-24.04 under WSL2 with per-case cgroup limits:

```text
systemd-run --user --quiet --wait --collect \
  -p MemoryMax=6G \
  -p MemorySwapMax=1G \
  -p CPUQuota=200%
```

Runtime environment:

```text
PATH=/tmp/cervomut-wsl-tools/go/bin:/usr/bin:/bin:/tmp/cervomut-wsl-bin
GOMEMLIMIT=3GiB
GOMAXPROCS=2
GOFLAGS=-p=2
timeout=600s
```

Artifacts:

```text
/tmp/cervomut-wsl-results/gremlins-small-10-cgroup-20260528-122948/summary.json
```

| Repo | Tool | Exit | Seconds | Total | Killed | Survived | Not covered | Timed out | Errors | Score | Notes |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| `cobra` | `cervomut` | 0 | 12 | 69 | 51 | 18 | 0 | 0 | 0 | 73.91 | Complete metrics. |
| `cobra` | `gremlins` | 0 | 23 | 0 | 0 | 0 | 5 | 87 | 0 | 0.00 | JSON was written, but the denominator is not comparable: Gremlins reported all observed mutations as timed out or not covered. |
| `pflag` | `cervomut` | 0 | 119 | 214 | 185 | 27 | 0 | 2 | 0 | 86.45 | Complete metrics. |
| `pflag` | `gremlins` | 0 | 1 |  |  |  |  |  |  |  | No stdout/stderr and no JSON report. |
| `logrus` | `cervomut` | 0 | 123 | 103 | 73 | 29 | 0 | 1 | 0 | 70.87 | Complete metrics. |
| `logrus` | `gremlins` | 0 | 3 |  |  |  |  |  |  |  | No stdout/stderr and no JSON report. |
| `uuid` | `cervomut` | 0 | 113 | 89 | 70 | 16 | 0 | 3 | 0 | 78.65 | Complete metrics. |
| `uuid` | `gremlins` | 0 | 2 |  |  |  |  |  |  |  | No stdout/stderr and no JSON report. |
| `decimal` | `cervomut` | 124 | 600 |  |  |  |  |  |  |  | Timeout before final metrics. |
| `decimal` | `gremlins` | 0 | 6 |  |  |  |  |  |  |  | No stdout/stderr and no JSON report. |
| `gjson` | `cervomut` | 124 | 599 |  |  |  |  |  |  |  | Timeout before final metrics. |
| `gjson` | `gremlins` | 0 | 11 |  |  |  |  |  |  |  | No stdout/stderr and no JSON report. |
| `sjson` | `cervomut` | 1 | 61 |  |  |  |  |  |  |  | Execution failure; logs were empty, so the runner needs better failure diagnostics. |
| `sjson` | `gremlins` | 0 | 2 |  |  |  |  |  |  |  | No stdout/stderr and no JSON report. |
| `jsonparser` | `cervomut` | 124 | 600 |  |  |  |  |  |  |  | Timeout before final metrics. |
| `jsonparser` | `gremlins` | 0 | 1 |  |  |  |  |  |  |  | No stdout/stderr and no JSON report. |
| `burntsushi-toml` | `cervomut` | 0 | 518 | 585 | 458 | 125 | 0 | 2 | 0 | 78.29 | Complete metrics near the 10-minute budget. |
| `burntsushi-toml` | `gremlins` | 0 | 3 |  |  |  |  |  |  |  | No stdout/stderr and no JSON report. |
| `urfave-cli` | `cervomut` | 124 | 600 |  |  |  |  |  |  |  | Timeout before final metrics. |
| `urfave-cli` | `gremlins` | 0 | 4 |  |  |  |  |  |  |  | No stdout/stderr and no JSON report. |

## WSL2 Findings

1. The cgroup wrapper worked: the run finished without host memory exhaustion,
   and each case was bounded by Linux cgroups rather than a Windows process-tree
   watchdog.
2. WSL2 removed the Windows Gremlins panic mode, but did not produce comparable
   Gremlins metrics on this pool. Gremlins exited 0 on 9/10 targets while
   writing no JSON report and no diagnostic output.
3. Cobra is no longer apples-to-apples in this WSL2 run. Gremlins wrote JSON,
   but reported `mutants_total=0`, `mutants_not_covered=5`, and
   `mutants_timed_out=87`, while CervoMutant executed and scored 69 mutants.
4. CervoMutant ran substantially faster in WSL2 than in Windows/OneDrive for
   the complete-metric targets:
   - `cobra`: 28.49s -> 12s
   - `pflag`: 223.33s -> 119s
   - `logrus`: 206.62s -> 123s
   - `uuid`: 182.87s -> 113s
5. CervoMutant still needs better bounded-run behavior. Full mutation timed out
   on `decimal`, `gjson`, `jsonparser`, and `urfave-cli`; those runs should
   preserve partial metrics instead of ending with empty denominators.
6. `sjson` exposed a separate diagnostic gap: exit 1 with empty stdout, stderr,
   and time output is not actionable enough for CI or for an AI agent.
7. The next useful comparison should not use raw full mutation as the default.
   It should use deterministic sample, budget-aware scheduling, and partial
   checkpointing so both tools can be compared on observed work rather than
   only on final reports.

## Gremlins Actionable Re-Run

Follow-up run completed on 2026-05-29 to separate Gremlins behavior from a bad
comparison setup. The earlier WSL2 run passed `./...` to Gremlins for most
repos. This re-run used:

```text
effective target: . when manifest target was ./...
--timeout-coefficient 4
--workers 2
timeout=600s
MemoryMax=6G
MemorySwapMax=1G
CPUQuota=200%
GOMEMLIMIT=3GiB
GOMAXPROCS=2
GOFLAGS=-p=2
```

Artifacts:

```text
/tmp/cervomut-wsl-results/gremlins-actionable-small-10-20260529-132757/summary.json
```

| Repo | Manifest target | Effective Gremlins target | Exit | Seconds | Total | Killed | Survived | Not covered | Timed out | Score | Status | Notes |
| --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- | --- |
| `cobra` | `./doc` | `./doc` | 0 | 24 | 87 | 58 | 29 | 5 | 0 | 66.67 | `ok` | Matches the known package-level anchor. |
| `pflag` | `./...` | `.` | 0 | 257 | 36 | 36 | 0 | 18 | 335 | 100.00 | `ok` | Useful report, but many timed-out mutations are outside the score denominator. |
| `logrus` | `./...` | `.` | 0 | 99 | 119 | 85 | 34 | 31 | 0 | 71.43 | `ok` | Clean comparable Gremlins signal. |
| `uuid` | `./...` | `.` | 0 | 41 | 54 | 54 | 0 | 26 | 43 | 100.00 | `ok` | Score is inflated if timed-out mutations are ignored. |
| `decimal` | `./...` | `.` | 0 | 317 | 9 | 9 | 0 | 93 | 691 | 100.00 | `ok` | Very small effective denominator relative to timeout/not-covered count. |
| `gjson` | `./...` | `.` | 0 | 506 | 3 | 3 | 0 | 37 | 1244 | 100.00 | `ok` | Extreme timeout pressure; raw score is not enough. |
| `sjson` | `./...` | `.` | 0 | 48 | 34 | 34 | 0 | 33 | 134 | 100.00 | `ok` | Useful operator coverage, but timeout count must be first-class. |
| `jsonparser` | `./...` | `.` | 0 | 239 | 8 | 7 | 1 | 848 | 593 | 87.50 | `ok` | Very large not-covered pool; coverage semantics dominate. |
| `burntsushi-toml` | `./...` | `.` | 0 | 422 | 548 | 483 | 65 | 218 | 4 | 88.14 | `ok` | Best broad parser signal in this run. |
| `urfave-cli` | `./...` | `.` | 124 | 608 |  |  |  |  |  |  | `timeout` | Timed out before final JSON. |

## Actionable Conclusions

1. Gremlins should be compared as a package-root tool, not as a `go test ./...`
   equivalent. Passing `./...` produced misleading no-report/no-results cases.
   CervoMutant can still support `./...`, but the comparison harness must
   normalize Gremlins targets differently.
2. CervoMutant should expose the same distinction Gremlins makes between:
   - effective executed mutants: killed + survived;
   - not covered mutants;
   - timed-out mutations;
   - test efficacy over killed + survived.
3. Gremlins' raw score can be misleading when timeout/not-covered counts dwarf
   the effective denominator. For example, `gjson` scored 100% over only 3
   effective mutants while recording 1,244 timed-out mutations. CervoMutant
   reports should make this impossible to miss with a top-level denominator
   health section.
4. CervoMutant's next comparison mode should have a `gremlins-package-root`
   strategy:
   - use package-root target normalization for external Gremlins runs;
   - carry original manifest target and effective external target separately;
   - mark `not_comparable` when target semantics differ.
5. CervoMutant needs partial checkpoint reports for timeout cases. Gremlins gave
   useful completed reports for 9/10 in this corrected run; CervoMutant's
   timeout runs in the earlier WSL2 comparison still lost denominators.
6. Budget-aware scheduling should account for operator/repo timeout risk.
   `gjson`, `decimal`, `jsonparser`, and `pflag` show high timeout counts under
   Gremlins; these are good targets for CervoMutant's timeout-risk ranking and
   deterministic sampling.
7. The comparison harness now needs to classify:
   - `ok`
   - `no_report`
   - `no_results`
   - `panic`
   - `all_timed_out`
   - `not_covered_only`
   - `timeout`
   - `watchdog_kill`

These statuses are more actionable than empty metric cells and should be used
in future multi-repo calibration summaries.

## Post-Implementation WSL Re-Run

Follow-up run completed on 2026-05-29 after implementing Gremlins-compatible
target metadata, denominator health, test efficacy, mutation coverage, and
budget-aware operator risk ordering in CervoMutant.

Artifacts:

```text
/tmp/cervomut-wsl-results/cervomut-gremlins-small-10-current-20260529-184401/summary.json
```

Execution guardrails:

```text
WSL Ubuntu-24.04
Go: /tmp/cervomut-wsl-tools/go/bin/go go1.25.6 linux/amd64
MemoryMax=6G
MemorySwapMax=1G
CPUQuota=200%
GOMEMLIMIT=3GiB
GOMAXPROCS=2
GOFLAGS=-p=2
timeout=600s per tool case
CervoMutant: cervomut run <target> --profile gremlins-compatible --isolation overlay --workers 2
Gremlins: gremlins unleash <effective_target> --workers 2 --timeout-coefficient 4
```

| Repo | Tool | Effective target | Exit | Sec | Status | Total | Killed | Survived | Not covered | Timed out | Efficacy | Coverage | Denom health | Warnings |
| --- | --- | --- | ---: | ---: | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- | --- |
| `cobra` | `cervomut` | `./doc` | 0 | 15 | `ok` | 69 | 51 | 18 | 0 | 0 | 73.91 | 100 | true |  |
| `cobra` | `gremlins` | `./doc` | 0 | 28 | `ok` | 81 | 52 | 29 | 5 | 6 | 64.20 | 94.57 | true |  |
| `pflag` | `cervomut` | `./...` | 0 | 135 | `ok` | 214 | 185 | 27 | 0 | 2 | 87.26 | 100 | true |  |
| `pflag` | `gremlins` | `.` | 0 | 261 | `ok` | 29 | 28 | 1 | 18 | 342 | 96.55 | 95.37 | false | `timed_out_exceeds_effective`, `high_score_poor_denominator_health` |
| `logrus` | `cervomut` | `./...` | 0 | 65 | `ok` | 103 | 73 | 29 | 0 | 1 | 71.57 | 100 | true |  |
| `logrus` | `gremlins` | `.` | 0 | 159 | `ok` | 81 | 77 | 4 | 31 | 38 | 95.06 | 79.33 | true |  |
| `uuid` | `cervomut` | `./...` | 0 | 97 | `ok` | 89 | 70 | 16 | 0 | 3 | 81.40 | 100 | true |  |
| `uuid` | `gremlins` | `.` | 0 | 50 | `ok` | 49 | 49 | 0 | 26 | 48 | 100.00 | 78.86 | true |  |
| `decimal` | `cervomut` | `./...` | 124 | 600 | `timeout` |  |  |  |  |  |  |  |  | `no_report` |
| `decimal` | `gremlins` | `.` | 0 | 456 | `ok` | 29 | 29 | 0 | 93 | 671 | 100.00 | 88.27 | false | `timed_out_exceeds_effective`, `not_covered_exceeds_effective`, `high_score_poor_denominator_health` |
| `gjson` | `cervomut` | `./...` | 124 | 600 | `timeout` |  |  |  |  |  |  |  |  | `no_report` |
| `gjson` | `gremlins` | `.` | 124 | 615 | `timeout` |  |  |  |  |  |  |  |  | `no_report` |
| `sjson` | `cervomut` | `./...` | 1 | 60 | `no CervoMutant JSON report` |  |  |  |  |  |  |  |  | `no_report` |
| `sjson` | `gremlins` | `.` | 0 | 118 | `ok` | 100 | 100 | 0 | 33 | 68 | 100.00 | 83.58 | true |  |
| `jsonparser` | `cervomut` | `./...` | 124 | 600 | `timeout` |  |  |  |  |  |  |  |  | `no_report` |
| `jsonparser` | `gremlins` | `.` | 0 | 321 | `ok` | 17 | 16 | 1 | 848 | 584 | 94.12 | 41.48 | false | `timed_out_exceeds_effective`, `not_covered_exceeds_effective`, `high_score_poor_denominator_health` |
| `burntsushi-toml` | `cervomut` | `./...` | 0 | 374 | `ok` | 585 | 458 | 125 | 0 | 2 | 78.56 | 100 | true |  |
| `burntsushi-toml` | `gremlins` | `.` | 0 | 557 | `ok` | 547 | 482 | 65 | 218 | 5 | 88.12 | 71.69 | true |  |
| `urfave-cli` | `cervomut` | `./...` | 0 | 255 | `ok` | 317 | 159 | 44 | 0 | 0 | 78.33 | 100 | true |  |
| `urfave-cli` | `gremlins` | `.` | 124 | 623 | `timeout` |  |  |  |  |  |  |  |  | `no_report` |

Findings:

1. Denominator health now exposes misleading high scores. Gremlins reported
   high efficacy on `pflag`, `decimal`, and `jsonparser`, but each row is now
   visibly unhealthy because timeout or not-covered counts dwarf the effective
   killed/survived denominator.
2. CervoMutant completed 6/10 rows with healthy denominators and complete
   mutation coverage over its generated pool: `cobra`, `pflag`, `logrus`,
   `uuid`, `burntsushi-toml`, and `urfave-cli`.
3. CervoMutant timed out on `decimal`, `gjson`, and `jsonparser`; `sjson`
   exited without a final JSON report. These remain the best small-pool targets
   for bounded scheduling and diagnostic-report work.
4. Gremlins completed 8/10 rows under package-root normalization. It timed out
   on `gjson` and `urfave-cli`.
5. CervoMutant partial checkpoints existed for timeout/error cases, but report
   quality differed:
   - `decimal`: usable `partial-mutation-report.json` around 2.4 MB.
   - `jsonparser`: usable `partial-mutation-report.json` around 75 MB.
   - `sjson`: usable `partial-mutation-report.json` around 1.0 MB despite exit 1.
   - `gjson`: zero-byte `partial-mutation-report.json`, which exposed a bug in
     direct partial-report writes under process termination.

Implementation response:

- CervoMutant now writes partial mutation reports through an atomic
  temp-file-and-rename path so a watchdog kill during JSON serialization cannot
  replace the last valid checkpoint with an empty file.
- The comparison harness still needs to parse partial reports when final reports
  are absent, otherwise timeout rows lose denominators even when CervoMutant
  preserved useful checkpoint data.
