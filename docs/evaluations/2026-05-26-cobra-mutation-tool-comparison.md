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

## Library-By-Library Lessons

This section analyzes the non-CervoMutant tools one by one. The point is not to
copy their implementation. The point is to identify product and engineering
choices that CervoMutant should either avoid or deliberately adopt.

### Gremlins

Negative points to avoid:

- Gremlins is strong for a fast package-level run, but its report contract is
  comparatively narrow for CI and AI agents. CervoMutant should not stop at a
  compact mutation summary; it needs stable schema, survivor context, selected
  tests, threshold decisions, baseline comparison, and machine-readable reasons.
- The human output is useful, but the product surface is mostly "run and read
  result". CervoMutant should avoid making follow-up work manual by default.
  Survivors need enough context to directly generate or review tests.
- Gremlins exposes useful efficacy and coverage metrics, but does not solve the
  governance problem around baselines, quarantine, expiry, and regression policy.
  CervoMutant should keep quality gates baseline-first and auditable.

Positive points to adopt:

- Keep a fast path for package-level mutation testing. In the WSL run Gremlins
  completed 87 mutants in 6.78 seconds, which is the direct speed target for
  CervoMutant package mode.
- Preserve distinct statuses for `killed`, `lived`, `not covered`, `not viable`,
  timeout, and skipped. This avoids confusing unexecuted mutants with weak tests.
- Keep separate metrics for test efficacy and mutator coverage. CervoMutant has
  already adopted `test_efficacy` and `mutation_coverage`; these should remain
  first-class and visible in summary, JSON, and HTML reports.
- Keep per-mutator statistics visible. They make weak operator families obvious
  and help tune mutator profiles.
- Keep the default terminal summary compact. CervoMutant can keep richer
  artifacts, but the default console path should stay scannable.

Application to CervoMutant:

- Add a "fast package benchmark" profile that minimizes report overhead and
  runs with `selection.mode=package`.
- Track Gremlins-style status mapping in the external comparison harness.
- Treat Gremlins as the speed/clarity reference, not as the governance/reporting
  reference.

### gomu

Negative points to avoid:

- Do not derive filesystem names from raw mutant IDs or absolute paths. gomu's
  Windows failure came from a temp directory name containing `C:\...`.
- Do not let very broad mutator generation inflate the denominator with many
  invalid, not viable, or compile-error mutants. In WSL, gomu found 413 mutants
  but reported 197 errors and 51 not viable. That creates review noise and makes
  score interpretation harder.
- Do not mix preparation failures with mutation outcomes. A path-preparation
  error should be reported as an operational/tool failure, not as a survived or
  killed mutant.
- Do not make the main report hard to compare across tools. gomu's JSON is useful
  but needs normalization before it can be evaluated beside Gremlins,
  go-mutesting, and CervoMutant.

Positive points to adopt:

- Overlay-based execution is worth studying. It avoids modifying the source tree
  and can reduce copy overhead when implemented with safe path handling.
- Incremental history is valuable for large projects. gomu's history-oriented
  design reinforces CervoMutant's decision to make cache/history a first-class
  feature.
- gomu has broad Go-specific mutation ideas, including error handling and return
  mutations. CervoMutant should borrow the categories carefully, but keep them
  behind profiles with explicit equivalent-mutant risk.
- The CLI has CI concepts such as thresholds and fail-on-gate behavior. CervoMutant
  should keep similar CI ergonomics but combine them with baseline-first adoption.

Application to CervoMutant:

- Keep all filesystem names hash/token based and add regression tests for Windows
  drive letters, spaces, and OneDrive-style paths.
- Explore overlay isolation as an optional execution backend, but only after
  containment checks and cleanup markers are part of the design.
- Keep aggressive mutators separate from conservative/default profiles, and
  report invalid/not viable rates by mutator.
- Add external-result normalization for gomu so future studies can distinguish
  valid mutants from errors and not-viable outcomes.

### go-mutesting v2

Negative points to avoid:

- Do not assume Unix tools exist. go-mutesting required an external `diff`
  executable and failed on Windows native. CervoMutant should use Go-native
  libraries or internal implementations for required operations.
- Do not build temp paths by concatenating a temp directory with a user/source
  path. go-mutesting's Windows failure came from creating paths under
  `tmpDir/.../C:`.
- Do not let advanced modes block basic usefulness. The first Windows retry with
  `coverage` and `per-test` stalled before producing comparable results.
  Advanced selection should have timeouts, progress, and graceful fallback.
- Do not make report filenames unpredictable or hard to collect. In WSL,
  go-mutesting wrote `report.json` in the target checkout while the command
  flag suggested summary JSON behavior. CervoMutant should keep all report
  outputs under the configured output directory.

Positive points to adopt:

- go-mutesting has the richest mutator set among the Go tools observed in this
  study. It is the best reference for long-term operator breadth.
- Its per-mutator breakdown is useful and should remain a visible CervoMutant
  report section.
- Its agentic JSON direction is aligned with CervoMutant's AI-first goal:
  survived mutants should include stable IDs, diffs, context, nearby tests,
  descriptions, and hints.
- The `noop` preflight is valuable. Running the clean suite before mutation
  prevents meaningless mutation results.
- The `coverage` and `per-test` modes are strategically important for large
  projects, even if they need careful fallback behavior.
- It has useful CI ideas: minimum MSI gates, changed-line filtering, and
  baseline-style survivor handling.

Application to CervoMutant:

- Use go-mutesting as the operator-breadth and agent-report reference.
- Keep all command dependencies internal or explicitly checked by `doctor`.
- Make `doctor` detect external Unix-tool assumptions when running external
  comparison studies.
- Build coverage/per-test selection with bounded setup time, progress reporting,
  and fallback to package/all selection when coverage mapping fails.
- Add report fields for nearby tests, natural-language mutation description, and
  concrete test-writing hints.

## Cross-Tool Design Rules For CervoMutant

Rules to avoid repeating negative patterns:

- Never use raw absolute paths, mutant IDs, or package names as filesystem names.
  Use relative paths for identity and hashes/tokens for files.
- Never rely on external Unix commands for core behavior.
- Never treat operational failures as mutation outcomes.
- Never expose an advanced mode without timeout, fallback, and reportable reason.
- Never make reports appear outside the configured output directory.
- Never optimize for raw mutant count without tracking invalid, equivalent,
  not viable, compile-error, and not-covered rates.

Practices to adopt:

- Gremlins-style speed and concise summaries.
- Gremlins-style distinction between efficacy and coverage.
- gomu-style overlay/isolation ideas, but with strict path containment.
- gomu-style incremental history, but with deterministic cache keys.
- go-mutesting-style broad operator catalog, but split by conservative/default/
  aggressive profiles.
- go-mutesting-style AI/agent report richness.
- CI-friendly preflight, thresholds, changed-scope execution, baseline compare,
  and report formats.

Actions for CervoMutant:

1. Add a benchmark/compare harness that runs Gremlins, gomu, go-mutesting, and
   CervoMutant under WSL/Linux and normalizes their result schemas.
2. Optimize CervoMutant package mode against Gremlins' speed target.
3. Add a Go-native overlay execution backend prototype with the path-hardening
   rules already implemented in `pkg/isolate`.
4. Expand mutator profiles using go-mutesting and gomu as references, but track
   invalid/not-viable/equivalent risk per operator.
5. Improve AI actionability by adding nearby tests, mutation descriptions, and
   stronger hints to survivor reports.

Issue #10 did not limit implementation to the highest-priority items. The
follow-up work addressed every action at least to an executable first version:

- `cervomut compare` and `pkg/extcompare` normalize CervoMutant, Gremlins, gomu,
  and go-mutesting outputs into one schema for repeatable studies.
- Package mode remains the default selection path, and the comparison harness
  gives future benchmark runs a stable place to capture speed deltas.
- `execution.isolation: overlay` adds a Go-native overlay backend that runs tests
  from the source module without copying the whole worktree.
- Aggressive mutator profile now includes literal and return mutations, while
  conservative stays limited to lower-risk operators.
- JSON schema v1 gained additive `description` and `nearby_tests` fields so AI
  agents and reviewers can move from survivor to candidate test quickly.

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

These backlog candidates were either implemented in issue #10 or left as
tracked candidates for deeper follow-up:

- Add `not_covered` as a first-class mutation status or reason in JSON schema v1. Implemented in issue #10 follow-up work.
- Add top-level mutation coverage and test efficacy metrics, distinct from mutation score. Implemented in issue #10 follow-up work.
- Add mutator statistics to summary, JSON, and HTML reports. Implemented in issue #10 follow-up work.
- Add an external-tool comparison harness that can ingest Gremlins, gomu, and go-mutesting reports/logs. Implemented in issue #10 follow-up work.
- Optimize temp-workdir and package-mode execution to close the performance gap with Gremlins.
- Add a benchmark profile for popular Go repos with pinned commits, package scopes, and repeatable commands.
- Add Windows path regression fixtures for isolation, overlay, and worker temp paths. Implemented for temp-workdir and overlay in issue #10 follow-up work.

## Issue #10 Follow-Up Implementation

After the initial study, CervoMutant added Gremlins-inspired reporting primitives:

- `not_covered` mutation status for coverage-guided selection when the coverage profile does not execute the mutated file.
- `test_efficacy` and `mutation_coverage` as separate summary and evaluation metrics.
- `mutator_statistics` in JSON plus visible mutator statistics in summary and HTML reports.

The same Cobra `./doc` sample now reports these fields in `summary.txt`, `mutation-report.json`, and `evaluation.json`.

## Full Lesson Coverage Follow-Up

The one-by-one tool analysis produced five concrete improvements, all now
represented in code:

- CLI study UX: global `help`/`--help` now exits successfully, and `report` plus
  `show` accept `--out` so artifact directories from evaluations can be inspected
  without changing config files.
- External comparison: `pkg/extcompare` parses the observed CervoMutant,
  Gremlins, gomu, and go-mutesting report shapes and writes a normalized
  `schema_version: "1"` study file through `cervomut compare`.
- Report actionability: mutants carry natural-language `description` and
  package-local `nearby_tests` in JSON reports.
- Mutator breadth: aggressive profile includes literal and return mutations,
  keeping noisy operators out of conservative mode.
- Isolation strategy: `execution.isolation: overlay` uses Go's `-overlay` flag
  for mutation runs that should avoid full module copies while preserving source
  tree cleanliness.

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
