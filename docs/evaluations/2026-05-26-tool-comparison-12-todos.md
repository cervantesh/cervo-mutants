# Tool Comparison 12 TODOs

Tracking issue: https://github.com/cervantesh/cervo-mutants/issues/13

This file intentionally records the paused 12-repository comparison against the
three external tools: Gremlins, gomu, and go-mutesting.

## Scope

Repositories:

```text
cobra, pflag, moby, hugo, prometheus, terraform, grpc-go, echo, logrus, validator, decimal, gjson
```

Tools:

```text
cervomut
gremlins
gomu
go-mutesting
```

Timeout:

```text
600 seconds per tool/repository
```

Output root:

```text
C:\Users\c___h\AppData\Local\Temp\cervomut-tool-comparison-12
```

## Current Partial Results

Completed before pausing:

| Repo | Tool | Exit | Seconds | Notes |
| --- | --- | ---: | ---: | --- |
| `cobra` | `cervomut` | 0 | 8.63 | Report parsed, but metrics need review because current parser saw killed/survived as zero for this cached run. |
| `cobra` | `gremlins` | 0 | 41.02 | Parsed. |
| `cobra` | `gomu` | 0 | 76.75 | Parsed. |
| `cobra` | `go-mutesting` | 0 | 104.08 | Parsed. |
| `pflag` | `cervomut` | 0 | 2.43 | Report parsed, but metrics need review. |
| `pflag` | `gremlins` | 2 | 1.26 | Failed; inspect log. |
| `pflag` | `gomu` | 1 | 0.09 | Failed; inspect log. |
| `pflag` | `go-mutesting` | 124 | 600.21 | Timed out. |
| `moby` | `cervomut` | 0 | 338.12 | Parsed. |
| `moby` | `gremlins` | 0 | 53.95 | Completed but parser did not extract metrics; inspect report/log shape. |
| `moby` | `gomu` | 124 | 600.21 | Timed out. |
| `moby` | `go-mutesting` | 124 | 600.36 | Timed out. |
| `hugo` | `cervomut` | 0 | 355.46 | Parsed. |

Partial summary file:

```text
C:\Users\c___h\AppData\Local\Temp\cervomut-tool-comparison-12\summary.json
```

## TODO

1. Fix `scripts/compare-tools-pool.ps1` CervoMutant parser so cached/fast reports
   do not show `killed=0` and `survived=0` when the report contains real results.
2. Fix Gremlins parser for repos where Gremlins exits 0 but metrics are null.
3. Re-run with `-Resume` after parser fixes:

   ```powershell
   .\scripts\compare-tools-pool.ps1 -TimeoutSeconds 600 -Workers 2 -Resume
   ```

4. If the full 12-repo comparison remains too long, split by repo groups:

   ```powershell
   .\scripts\compare-tools-pool.ps1 -Names cobra,pflag,moby,hugo -TimeoutSeconds 600 -Workers 2 -Resume
   .\scripts\compare-tools-pool.ps1 -Names prometheus,terraform,grpc-go,echo -TimeoutSeconds 600 -Workers 2 -Resume
   .\scripts\compare-tools-pool.ps1 -Names logrus,validator,decimal,gjson -TimeoutSeconds 600 -Workers 2 -Resume
   ```

5. Convert `summary.json` into a Markdown comparison table with:
   - completion rate;
   - timeouts;
   - killed/survived/not-covered/errors/not-viable;
   - score/test efficacy;
   - denominator caveats per tool.

6. Update issue #13 with the completed comparison.

## 2026-05-27 Resource-Bounded Retry Findings

Issue #13 later expanded the comparison from the paused 12-repository mixed run
into separated tool phases over 20 repositories, followed by one-at-a-time
retries for `hugo` and `grpc-go`.

The retry target was to recover metrics for the two non-CervoMutant reference
tools that failed to produce usable metrics for `hugo` and `grpc-go`:

```text
gomu
go-mutesting
```

The one-at-a-time retry used process-tree resource controls instead of passive
global memory waiting:

```text
MaxProcessTreeMemoryMB: 6144
GOMEMLIMIT: 3GiB
GOMAXPROCS: 1
GOFLAGS: -p=1
workers: 1
timeout: 1800s per repo/tool
```

Results:

| Repo | Tool | Exit | Seconds | Outcome |
| --- | --- | ---: | ---: | --- |
| `hugo` | `gomu` | 124 | 1802.67 | Timed out without usable metrics. |
| `grpc-go` | `gomu` | 126 | 75.81 | Killed by process-tree watchdog at ~6953MB working set / ~6971MB private. |
| `hugo` | `go-mutesting` | 124 | 1802.12 | Timed out without usable metrics. |
| `grpc-go` | `go-mutesting` | 126 | 30.51 | Killed by process-tree watchdog at ~8464MB working set / ~8801MB private. |

Finding:

- Both reference tools can fail to degrade gracefully under resource limits on
  larger Go targets. Even with one repository, one package target, one worker,
  `GOMAXPROCS=1`, `GOFLAGS=-p=1`, and `GOMEMLIMIT=3GiB`, the `grpc-go` runs
  exceeded the 6GB process-tree limit through tool and `go test` child-process
  activity.
- `GOMEMLIMIT` is useful but insufficient as a hard memory boundary. It does not
  bound the whole process tree, compiler/linker subprocesses, or all native
  allocations. A CI-safe mutation tool needs an explicit process-tree watchdog.
- Timeout-only failure is not enough. A useful tool should write partial,
  comparable metrics before budget exhaustion or watchdog termination. These
  retries produced controlled exits but no additional metrics, which limits
  their value for large-project CI comparison.

CervoMutant design implications:

- Keep process-tree memory accounting in the comparison runner and move the same
  concept into CervoMutant's own execution model where possible.
- Prefer incremental result checkpoints after each mutant, not only at the end
  of a package/tool run.
- Treat timeout, memory-watchdog, and skipped-for-resources as first-class
  statuses in JSON reports.
- Budget-aware scheduling should stop before resource exhaustion and still
  report `attempted`, `killed`, `survived`, `pending`, and `stopped_reason`.
- Large-project CI profiles should support smaller package slices, maximum
  mutants per package, and early partial summaries so a failed run is still
  diagnostically useful.

## 2026-05-28 Local WSL2 Same-Limit Retry

To avoid cloud cost and remove Windows-native path/process overhead from the
comparison, the `hugo` and `grpc-go` retry was repeated inside local
`Ubuntu-24.04` WSL2 on the Linux filesystem under `/tmp`.

The WSL2 retry used a hard `systemd-run --user --scope` cgroup per case:

```text
MemoryMax: 6G
MemorySwapMax: 1024M
CPUQuota: 100%
TasksMax: 384
GOMEMLIMIT: 3GiB
GOMAXPROCS: 1
GOFLAGS: -p=1
workers: 1
outer timeout: 1900s per repo/tool
```

Artifacts:

```text
/tmp/cervomut-wsl-results/local-retry-startprocess-20260528-010616
/tmp/cervomut-wsl-results/cervomut-same-limits-20260528-020929
```

Baseline package tests passed before mutation:

| Repo | Package | Exit | Peak RSS |
| --- | --- | ---: | ---: |
| `hugo` | `./helpers` | 0 | 441472 KB |
| `grpc-go` | `./metadata` | 0 | 99328 KB |

Same-limit mutation results:

| Repo | Tool | Exit | Wall time | Peak RSS | Outcome |
| --- | --- | ---: | ---: | ---: | --- |
| `hugo` | `gomu` | 124 | ~31m 36s | not reported | Timed out after processing part of file 3/8; no final metrics. |
| `grpc-go` | `gomu` | 143 | ~2m 13s | not reported | Terminated inside the bounded scope after starting `metadata.go` with 123 mutants; no final metrics. |
| `hugo` | `go-mutesting` | 2 | 0.62s | 37372 KB | Panicked immediately in `go/types` through old `golang.org/x/tools` package loading on Go 1.25. |
| `grpc-go` | `go-mutesting` | 2 | 0.38s | 21760 KB | Same immediate `go/types` panic on Go 1.25. |
| `hugo` | `cervomut` | 0 | 14m 58.59s | 452736 KB | Completed with JSON/summary/survivor artifacts. |
| `grpc-go` | `cervomut` | 0 | 10.38s | 100096 KB | Completed with JSON/summary/survivor artifacts. |

CervoMutant outputs under the same limits:

| Repo | Generated | Covered | Executed | Killed | Survived | Not covered | Timed out | Compile errors | Mutation score |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `hugo` `./helpers` | 105 | 77 | 77 | 37 | 12 | 28 | 28 | 0 | 48.05% |
| `grpc-go` `./metadata` | 18 | 18 | 18 | 16 | 2 | 0 | 0 | 0 | 88.89% |

Additional findings:

- WSL2 with cgroup scopes is a viable local, no-cost way to run hostile mutation
  experiments without exhausting the Windows host. The Windows-native retries
  needed a process-tree watchdog; WSL2 can enforce the cap at the kernel/cgroup
  level.
- `go-mutesting` is not a useful Go 1.25 reference without a compatibility pass.
  Its failure here is not mutation performance; it crashes before mutation due
  to stale package-loading dependencies. A follow-up comparison can test it with
  older Go toolchains, but for modern CervoSoft defaults this is a real adoption
  risk.
- `gomu` degraded better in WSL2 than in Windows-native retries because the
  machine stayed healthy, but it still failed to return comparable final metrics
  for the two target packages.
- CervoMutant finished both same-limit runs and produced machine-readable
  reports, but it was too quiet during the long `hugo` run. It should emit
  progress and durable checkpoints while running, not only final artifacts.

New CervoMutant design implications:

- Add first-class WSL/Linux cgroup resource-limit guidance to `doctor` and CI
  documentation for local large-repo experiments.
- Add periodic progress output and a partial JSON checkpoint file during `run`.
- Preserve a final report even when the outer watchdog kills the run by writing
  checkpointed results incrementally.
- Record `go_version`, `toolchain`, `GOFLAGS`, `GOMAXPROCS`, `GOMEMLIMIT`, and
  cgroup/memory-limit metadata in JSON reports so comparisons remain auditable.
- Add a compatibility finding category for external tools that panic before
  mutation starts; do not mix those failures with timeout or mutation-quality
  outcomes.

## 2026-05-28 Follow-Up Implementation Notes

The first hardening batch from these findings landed in `f1f7962`:

- `progress.jsonl` is written during `cervomut run`.
- `partial-mutation-report.json` is updated incrementally after completed
  mutants.
- Reports include environment metadata: OS/arch, Go version, temp/workdir,
  isolation, workers, timeout, selected Go env, WSL/cgroup hints, and Windows
  OneDrive detection.
- `doctor` reports Windows/WSL diagnostics without failing on advisory
  warnings.
- CLI panic recovery reports an `internal_error` instead of exposing only a raw
  panic.

The second hardening batch landed after that:

- `cervomut run --resume` reuses `partial-mutation-report.json` explicitly.
  Resume is opt-in to avoid accidentally trusting stale checkpoints.
- Cached/resumed results preserve `previous_status`, so mutation score and
  executed/covered counters remain meaningful.
- Progress events include `eta` and `active_mutant`.
- Mutant results include `failure_kind` for structured operational taxonomy
  without expanding the primary v1 status enum.
- `cervomut run` and `cervomut eval` accept `--max-process-memory-mb`.
- Windows builds use a Job Object best-effort process limit when
  `--max-process-memory-mb` is set.
- `cervomut init` now includes `execution.resume` and
  `execution.resources`.

Verification:

```text
go test ./...
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build ./cmd/cervomut
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build ./cmd/cervomut
GOOS=windows GOARCH=amd64 go build ./cmd/cervomut
```

Remaining limitations:

- Windows Job Objects are now wired for memory/process limits, but this is still
  best-effort and needs larger real-project validation.
- Resume currently trusts mutant IDs in the partial report. It should later add
  stronger checkpoint compatibility checks using config/toolchain/source
  fingerprints.
- macOS remains compile-tested only; no real runtime validation has been done.

The next checkpoint-hardening pass added a compatibility fingerprint to
`partial-mutation-report.json`:

- The fingerprint includes the selected policy/profile, selection mode,
  prefilter flag, isolation mode, test command, test timeout, Go version,
  `GOFLAGS`, scheduled mutant IDs, and Go source/test/module file digests.
- `--resume` now rejects missing or mismatched checkpoint fingerprints instead
  of silently mixing stale partial results.
- Checkpoint metadata is included in final JSON reports as well.

Windows Job Object validation was expanded with local smoke tests:

- A covered Windows-native run with `--max-process-memory-mb 768` completed and
  exercised child `go test` execution under the Job Object path.
- A deliberately tiny `--max-process-memory-mb 16` failed during baseline
  without hanging the host, confirming the low-limit path degrades early.
- A larger Windows-native Cobra smoke with `--max-process-memory-mb 1024`,
  `--max-mutants 10`, `--workers 1`, and `policy=ci-fast` completed with exit
  0 in about 94 seconds:

  ```text
  Generated mutants: 10
  Covered mutants: 8
  Executed mutants: 8
  Killed: 6
  Survived: 2
  Not covered: 2
  Timed out: 0
  Compile errors: 0
  Mutation score: 75.00%
  ```
- A CervoSoft module validation on `cervo-retry` with `GOWORK=off`,
  `--max-process-memory-mb 1024`, `--max-mutants 20`, `--workers 1`, and
  `policy=ci-fast` completed with exit 0 in about 17 seconds:

  ```text
  Generated mutants: 20
  Covered mutants: 12
  Executed mutants: 12
  Killed: 5
  Survived: 7
  Not covered: 8
  Timed out: 0
  Compile errors: 0
  Mutation score: 41.67%
  ```
- A heavier CervoClaw module validation on `CervoWorkers` required a longer
  timeout than `ci-fast` because its coverage baseline takes about 55 seconds.
  With `GOWORK=off`, `policy=campaign`, `--max-process-memory-mb 2048`,
  `--max-mutants 5`, and `--workers 1`, it completed with exit 0 in about
  194 seconds:

  ```text
  Generated mutants: 5
  Covered mutants: 5
  Executed mutants: 5
  Killed: 2
  Survived: 3
  Not covered: 0
  Timed out: 0
  Compile errors: 0
  Mutation score: 40.00%
  ```
- A broader Windows Job Object campaign then ran four modules:

  | Module | Policy | Memory cap | Max mutants | Result |
  | --- | --- | ---: | ---: | --- |
  | `cervo-retry` | `ci-fast` | 1024MB | 20 | exit 0; 20 generated; 12 executed; 5 killed; 7 survived; 0 timeout/compile errors |
  | `cervo-config` | `ci-fast` | 1024MB | 15 | exit 0; 15 generated/executed; 14 killed; 1 survived; 0 timeout/compile errors |
  | `cervo-httpkit` | `ci-fast` | 1024MB | 15 | exit 0; 15 generated; 7 executed; 3 killed; 4 survived; 0 timeout/compile errors |
  | `CervoWorkers` | `campaign` + `--test-timeout 180s` | 2048MB | 5 | exit 0; 5 generated/executed; 2 killed; 3 survived; 0 timeout/compile errors |

  The first `CervoWorkers` attempt under the default `campaign` 2-minute test
  timeout failed during baseline. Adding explicit `--test-timeout 180s` made the
  limit visible and repeatable, which supports keeping timeout override in the
  CLI for larger Windows-native runs.

Remaining limitations after this pass:

- Checkpoint fingerprints now hash Go source/test/module files, but they do not
  include every possible non-Go runtime fixture by default. `execution.
  checkpoint_includes` now covers common fixture directories such as
  `testdata/**` and `fixtures/**`; projects with other fixture layouts should
  extend that list.
- Windows Job Object validation now covers Cobra, three CervoSoft modules, and
  a heavier CervoWorkers slice. It is credible for controlled local runs, but a
  full production-grade claim still needs a longer campaign across all CervoClaw
  modules that have clean baselines on the selected Go version.
