# Comparison Harness Guide

This document describes the local mutation-tool comparison harness so a future
agent can run it without relying on thread memory.

Tracking issue: https://gitea.cervbox.synology.me/CervoSoft/cervo-mutant/issues/13

## Purpose

The harness compares CervoMutant with external Go mutation-testing tools while
preserving enough metadata to avoid invalid claims.

It is designed to answer two different questions:

1. Can each tool run the manifest target as written?
2. When normalized to the same effective target, how do the tools compare?

Do not mix those two questions. Use
[tool-comparison-protocol.md](tool-comparison-protocol.md) for the fairness
rules.

## Main Files

| File | Role |
| --- | --- |
| `docs/evaluations/go-repo-pool-40.json` | Repository manifest with name, URL, target, lane, domain, and reason. |
| `scripts/compare-tools-pool.ps1` | Main multi-tool runner for Windows/PowerShell. Supports memory guards, resume, target normalization, and per-tool parsing. |
| `scripts/calibration-smoke.ps1` | Lighter CervoMutant calibration smoke runner. Useful before expensive external comparisons. |
| `cmd/cervomut` `compare` | Normalizes existing tool reports into one JSON schema. |
| `pkg/extcompare` | Parser and comparability logic for CervoMutant, Gremlins, gomu, and go-mutesting reports. |
| `docs/evaluations/tool-findings.md` | Ledger of lessons learned from each external tool. |
| `docs/evaluations/2026-05-28-gremlins-focused-pool.md` | Current Gremlins-focused study and latest results. |

## Required Inputs

The main harness expects:

- a repository pool manifest;
- checked-out repositories under `WorkRoot`;
- binaries for each selected tool;
- an output directory;
- timeout and memory limits.

Default Windows paths are tuned for prior local experiments and may need to be
overridden:

```powershell
.\scripts\compare-tools-pool.ps1 `
  -Manifest docs/evaluations/go-repo-pool-40.json `
  -WorkRoot $env:TEMP/cervomut-go-pool-40 `
  -OutputRoot $env:TEMP/cervomut-tool-comparison `
  -CervoMutant $env:TEMP/cervomut-pool.exe `
  -Gremlins $env:TEMP/cervomut-study-cobra/tools/gremlins.exe
```

## Target Semantics

The harness records these fields per row:

- `target`: manifest target from the repo pool.
- `effective_target`: actual target passed to the tool.
- `target_mode`: `manifest` or `package-root`.
- `manifest_equivalent`: whether `target == effective_target`.
- `apples_to_apples_key`: `target_mode:effective_target`.

Use `CompareTargetMode package-root` for fair Gremlins comparisons when a
manifest target is `./...`:

```powershell
.\scripts\compare-tools-pool.ps1 `
  -Tools cervomut,gremlins `
  -Names cobra,pflag,logrus,uuid,decimal,gjson,sjson,jsonparser,burntsushi-toml,urfave-cli `
  -CompareTargetMode package-root `
  -GremlinsTargetMode package-root `
  -Workers 2 `
  -TimeoutSeconds 600 `
  -Resume
```

That makes CervoMutant and Gremlins both receive `.` instead of comparing
`cervomut run ./...` against `gremlins unleash .`.

For all-tool studies, keep the same `CompareTargetMode` for CervoMutant, gomu,
and go-mutesting. Normalizing only Gremlins creates a diagnostic run, not a fair
comparison.

## Tool Commands

The current harness builds command lines as follows.

| Tool | Command shape |
| --- | --- |
| CervoMutant | `cervomut run <effective_target> --policy comparison-safe --workers <n> --out <repoOut>/cervomut` |
| Gremlins | `gremlins unleash <effective_target> --workers <n> --threshold-efficacy 0 --threshold-mcover 0 --output <repoOut>/gremlins.json` |
| gomu | `gomu run <effective_target> --workers <n> --timeout <seconds> --threshold 0 --fail-on-gate=false --output json` |
| go-mutesting | `go-mutesting /noop /quiet /no-diffs /logger-summary-json /logger-agentic-json /exec-timeout:<seconds> /workers:<n> <effective_target>` |

Gremlins can add `--timeout-coefficient <n>` through
`GremlinsTimeoutCoefficient`.

CervoMutant's `comparison-safe` policy is the default apples-to-apples lane. It
uses the Gremlins-compatible operator profile, overlay isolation, deterministic
sampling, a 10 minute run budget, a 20 second per-mutant timeout, and a default
250 mutant cap when no explicit cap is provided.

## Memory And Timeout Guards

`scripts/compare-tools-pool.ps1` includes several guards because previous gomu
and go-mutesting runs exhausted memory.

Important knobs:

- `TimeoutSeconds`: per tool/repo wall-clock limit.
- `MinFreeMemoryMB`: wait before starting if physical memory is too low.
- `MinFreeCommitMB`: wait before starting if commit headroom is too low.
- `KillBelowFreeMemoryMB`: kill a running process if free memory drops below
  this value.
- `KillBelowFreeCommitMB`: kill a running process if commit headroom drops
  below this value.
- `MaxProcessTreeMemoryMB`: kill if the process tree exceeds this private or
  working-set memory.
- `MemoryPollSeconds`: polling interval.
- `GoMemoryLimit`, `GoMaxProcs`, `GoFlags`: Go runtime controls applied to the
  child process.

For fragile local runs, prefer one or two workers and a process-tree memory cap.

## Status Classification

The harness uses these statuses:

- `ok`
- `timeout`
- `partial_timeout`
- `watchdog_kill`
- `partial_watchdog_kill`
- `skipped`
- `panic`
- `no_report`
- `no_results`
- `all_timed_out`
- `not_covered_only`

For CervoMutant, when `mutation-report.json` is absent, the harness now falls
back to `partial-mutation-report.json` and sets `partial_report_used=true`.
CervoMutant also writes `partial-summary.json`, which is the first artifact to
inspect when a full partial mutant list is large.

## Outputs

Each run writes:

- `summary.json` at `OutputRoot`;
- per-repo logs under `OutputRoot/<repo>/<tool>.log`;
- CervoMutant reports under `OutputRoot/<repo>/cervomut`;
- Gremlins JSON under `OutputRoot/<repo>/gremlins.json`;
- copied external reports for gomu and go-mutesting when available.

`summary.json` is the first file to inspect. It contains normalized metrics,
status, target semantics, logs, and whether a partial report was used.

## Normalized Report Parser

Use `cervomut compare` when you already have report files and want a stable
schema:

```powershell
cervomut compare `
  --cervomut path/to/mutation-report.json `
  --cervomut-target ./... `
  --cervomut-target-mode package-root `
  --gremlins path/to/gremlins.json `
  --gremlins-target ./... `
  --gremlins-target-mode package-root `
  --out path/to/tool-comparison.json
```

The output includes:

- `comparability.apples_to_apples`;
- `comparability.manifest_equivalent`;
- `comparability.effective_targets`;
- `comparability.target_modes`;
- `comparability.warnings`;
- normalized per-tool metrics and denominator health.

## Agent Checklist

Before launching a comparison:

1. Confirm the Gitea issue.
2. Choose `manifest` or `package-root` mode.
3. Use the same effective target mode for every tool in any fairness claim.
4. Set memory and timeout guards.
5. Record tool binary paths and versions when possible.
6. Use `-Resume` for long pools.

After the run:

1. Inspect `summary.json`.
2. Check `apples_to_apples_key` before comparing score or speed.
3. Inspect logs for `panic`, `No results to report`, timeout, or watchdog kill.
4. Prefer CervoMutant final reports, then partial reports.
5. Document denominator health warnings, not just score.
6. Update issue #13 and the findings ledger.

## Common Mistakes

- Comparing `cervomut run ./...` to `gremlins unleash .` as a fair result.
- Reporting high scores without timed-out or not-covered counts.
- Treating exit code 0 as success when no report exists.
- Dropping CervoMutant partial data after a timeout.
- Running gomu or go-mutesting without memory guards on Windows.
