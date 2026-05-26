# Cobra Mutation Tool Comparison Study

Issue: [#10](https://gitea.cervbox.synology.me/CervoSoft/cervo-mutant/issues/10)

Date: 2026-05-26

## Purpose

This is the first empirical comparison study for CervoMutant improvements. The goal is not to declare a universal winner. The goal is to identify concrete design gaps by running CervoMutant and three Go mutation-testing tools against the same popular Go project under the same local Windows/OneDrive environment.

The comparison follows the evaluation framework in [docs/evaluation-framework.md](../evaluation-framework.md): tool capability, operational reliability, CI relevance, actionability, cost, noise risk, and validity controls.

## Subject

Repository: `github.com/spf13/cobra`

Commit: `ad460ea8f249db69c943a365fb84f3a59042d54e`

Scope: `./doc`

Reason for selection:

- Popular Go module with real tests and real documentation-generation code.
- Small enough package scope to run multiple tools locally.
- Good fit for mutation operators around conditionals, logical operators, arithmetic, and error handling.

Baseline:

```text
go test ./doc
```

Result: pass.

## Environment

Host: Windows, workspace under OneDrive.

Study workspace:

```text
%TEMP%\cervomut-study-cobra
```

Important validity note: Windows path handling is part of this study because CervoMutant explicitly targets Windows/OneDrive as a supported development environment. Tools that fail on `C:\...` paths are marked as operational failures rather than excluded from the comparison.

## Tools

| Tool | Command family | Result type |
| --- | --- | --- |
| CervoMutant | `cervomut eval` | Completed |
| Gremlins | `gremlins unleash` | Completed |
| gomu | `gomu run` | Failed during mutation preparation |
| go-mutesting v2 | `go-mutesting` | Failed during mutation execution on Windows path handling |

## Commands

CervoMutant:

```text
cervomut eval ./doc --max-mutants 20 --budget 2m --out %TEMP%\cervomut-study-cobra\results\cervomut-eval-doc
```

Gremlins:

```text
gremlins unleash ./doc --workers 4 --threshold-efficacy 0 --threshold-mcover 0 --output %TEMP%\cervomut-study-cobra\results\gremlins-doc.json
```

gomu:

```text
gomu run ./doc --workers 4 --timeout 30 --threshold 0 --fail-on-gate=false --output json
```

go-mutesting v2:

```text
go-mutesting /noop /quiet /no-diffs /logger-summary-json /logger-agentic-json /coverage /per-test /exec-timeout:30 /workers:4 github.com/spf13/cobra/doc
```

Relative path retry:

```text
go-mutesting /noop /quiet /no-diffs /logger-summary-json /logger-agentic-json /coverage /per-test /exec-timeout:30 /workers:4 ./doc
```

## Normalized Results

### Windows Native

| Tool | Completed? | Mutants attempted | Killed | Survived/lived | Other statuses | Time | Notes |
| --- | ---: | ---: | ---: | ---: | --- | ---: | --- |
| CervoMutant | Yes | 20 | 13 | 7 | 0 timed out, 0 compile errors | 47.26s | Limited by `--max-mutants 20`; produced summary, JSON, JUnit, HTML, survivor data, and evaluation scorecard. |
| Gremlins | Yes | 87 | 58 | 29 | 5 not covered, 0 not viable | 32.12s | Strong local baseline for speed and concise mutation summary. |
| gomu | No | 390 discovered | 0 valid | 0 valid | 390 preparation errors | 1.59s | All mutants failed before execution due invalid temp directory name derived from Windows absolute paths. |
| go-mutesting v2 | No | Not completed | N/A | N/A | Panic or timeout | 15.74s before panic; retry timed out at 184s | Import-path run panicked on `C:` temp path; relative retry stalled while building per-test coverage map. |

Patched local validation:

| Tool | Patch | Completed? | Mutants attempted | Killed | Survived/escaped | Other statuses | Time | Notes |
| --- | --- | ---: | ---: | ---: | ---: | --- | ---: | --- |
| gomu v0.2.0 | [windows paths](../external-patches/gomu-v0.2.0-windows-paths.patch) | Yes | 390 | 64 | 99 survived | 176 errors, 51 not viable | 98s | The `mutant_C:` preparation failure is fixed. Remaining errors are mutation/compile outcomes from gomu's generated mutants, not temp path creation failures. |
| go-mutesting v2.6.13 | [windows paths](../external-patches/go-mutesting-v2.6.13-windows-paths.patch) | Yes | 361 | 170 | 191 escaped | 0 errored, 0 not covered, 0 skipped | 129s | The `C:` temp path panic and missing Unix `diff` executable failure are fixed for the non-coverage run. |

### WSL/Linux Baseline

The comparison was repeated in WSL Ubuntu 24.04 on the Linux filesystem under
`/tmp/cervomut-study-cobra-wsl`, using unpatched upstream tools and GNU `diff`.
This is the preferred baseline for comparing CervoMutant against the Go
ecosystem because it avoids Windows-only failures in tools that assume Unix
paths or Unix command-line utilities.

| Tool | Completed? | Mutants attempted | Killed | Survived/lived/escaped | Other statuses | Time | Notes |
| --- | ---: | ---: | ---: | ---: | --- | ---: | --- |
| CervoMutant | Yes | 20 | 13 | 7 survived | 0 not covered, 0 timed out, 0 compile errors | within 2m budget | Same bounded `eval` run as Windows; score 65%, test efficacy 65%, mutation coverage 100%. |
| Gremlins | Yes | 87 | 58 | 29 lived | 5 not covered, 0 not viable | 6.78s | Same outcome as Windows, much faster on WSL/Linux filesystem. |
| gomu v0.2.0 | Yes | 413 | 64 | 101 survived | 197 errors, 51 not viable | 21.82s | Unpatched upstream works in WSL; Windows native failure was path-name preparation, not a general inability to run. |
| go-mutesting v2.6.13 | Yes | 361 | 170 | 191 escaped | 0 errored, 0 not covered, 0 skipped | 20.8s | Unpatched upstream works in WSL with GNU `diff`; MSI 47.09%. |

Gremlins reported:

```text
Killed: 58, Lived: 29, Not covered: 5
Timed out: 0, Not viable: 0, Skipped: 0
Test efficacy: 66.67%
Mutator coverage: 94.57%
```

CervoMutant reported:

```text
Mutation score: 65.00%
Killed: 13
Survived: 7
Quarantined: 0
Timed out: 0
Compile errors: 0
```

gomu failure signature:

```text
Failed to prepare mutation: failed to create mutant directory:
mkdir ...\gomu_overlay_...\mutant_C:: The filename, directory name, or volume label syntax is incorrect.
```

go-mutesting v2 failure signature:

```text
panic: mkdir ...\go-mutesting-...\C:: The filename, directory name, or volume label syntax is incorrect.
```

## What Gremlins Gives Us As A Concrete Reference

Gremlins is the most useful direct Go reference from this run.

Capabilities to study:

- Fast package-level execution on a real project.
- Clear distinction between `KILLED`, `LIVED`, `NOT COVERED`, `NOT VIABLE`, and timeout-like outcomes.
- Thresholds split between mutation efficacy and mutator coverage.
- Compact JSON summary with file-level mutation entries.
- Practical default output for humans.

Implications for CervoMutant:

- Add first-class `not_covered` status or explicit coverage reason instead of folding every unexecuted mutant into skipped/survived categories.
- Separate "test efficacy" from "mutation coverage" in reports, because they answer different questions.
- Include mutator statistics in summary reports, not only in the detailed JSON.
- Provide a fast package-only mode that can run a comparable scope without a heavy evaluation wrapper.

## What CervoMutant Did Better In This Environment

CervoMutant completed successfully under the Windows/OneDrive path shape where gomu and go-mutesting failed.

It also produced richer CI and agent artifacts in one command:

- Stable JSON report.
- JSON schema.
- JUnit XML.
- HTML report.
- Survivor report.
- Evaluation scorecard.
- Explicit summary threshold section.

That is aligned with the goal of being the default AI-friendly mutation-testing tool.

## CervoMutant Gaps Exposed By The Study

1. CervoMutant is slower per attempted mutant than Gremlins in this run.

   CervoMutant ran 20 mutants in about 47 seconds, while Gremlins ran 87 in about 32 seconds. The runs are not strictly equivalent because CervoMutant used `eval`, report generation, isolation, and `--max-mutants 20`, but this still points to isolation/test-selection overhead as a priority.

2. The comparison is not yet apples-to-apples.

   CervoMutant needs a dedicated external comparison harness that records installed tool versions, exact command lines, host information, normalized status mappings, and raw artifact checksums.

3. CervoMutant reports do not yet expose Gremlins-style mutation coverage as a top-level concept.

   `not_covered` is important because a survived mutant and an unexecuted mutant demand different fixes.

4. The scorecard still has review-required dimensions that need structured reviewer workflow.

   Fault-revealing effectiveness, equivalent-mutant burden, and longitudinal relevance cannot be inferred from one automated run. The report should make those review tasks explicit.

5. There is no direct "run like Gremlins" mode yet.

   `cervomut run ./doc --report summary,json --selection package --max-mutants 0` should be able to serve as the simple benchmark path without invoking the broader evaluation workflow.

## Improvement Backlog Candidates

These should become proposal issues before implementation:

- Add `not_covered` as a first-class mutation status or reason in JSON schema v1. Implemented in issue #10 follow-up work.
- Add top-level mutation coverage and test efficacy metrics, distinct from mutation score. Implemented in issue #10 follow-up work.
- Add mutator statistics to summary, JSON, and HTML reports. Implemented in issue #10 follow-up work.
- Add an external-tool comparison harness that can ingest Gremlins, gomu, and go-mutesting reports/logs.
- Optimize temp-workdir and package-mode execution to close the performance gap with Gremlins.
- Add a benchmark profile for popular Go repos with pinned commits, package scopes, and repeatable commands.
- Add Windows path regression fixtures for isolation, overlay, and worker temp paths.

## Issue #10 Follow-Up Implementation

After the initial study, CervoMutant added Gremlins-inspired reporting primitives:

- `not_covered` mutation status for coverage-guided selection when the coverage profile does not execute the mutated file.
- `test_efficacy` and `mutation_coverage` as separate summary and evaluation metrics.
- `mutator_statistics` in JSON plus visible mutator statistics in summary and HTML reports.

The same Cobra `./doc` sample now reports these fields in `summary.txt`, `mutation-report.json`, and `evaluation.json`.

## Windows Path Hardening Follow-Up

The gomu and go-mutesting failures were converted into executable CervoMutant safeguards:

- isolated workdirs now use a sanitized module token plus hash instead of any raw absolute path fragment;
- isolated workdirs include a `.cervomut-workdir` marker, and cleanup refuses to delete unmarked paths;
- mutant patch target paths are resolved through a containment check, so a malformed worker job cannot escape the copied module;
- engine-generated mutant IDs are module-relative and slash-normalized instead of embedding raw absolute `C:\...` paths;
- regression tests cover Windows-invalid filename characters, OneDrive-style paths with spaces, worker containment, and source-tree preservation.

Local patches were also produced for the external tools so the comparison can be rerun on Windows:

- [gomu v0.2.0 Windows path patch](../external-patches/gomu-v0.2.0-windows-paths.patch)
- [go-mutesting v2.6.13 Windows path patch](../external-patches/go-mutesting-v2.6.13-windows-paths.patch)

## Current Assessment

For this Windows/OneDrive study, CervoMutant is operationally more robust than gomu and go-mutesting, but Gremlins is currently the strongest direct Go implementation reference for speed and concise mutation semantics.

The next improvement cycle should focus on Gremlins-inspired execution/reporting concepts while preserving CervoMutant's stronger CI, schema, baseline, and agent-oriented artifacts.
