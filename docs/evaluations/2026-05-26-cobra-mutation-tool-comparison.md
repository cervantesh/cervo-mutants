# Cobra Mutation Tool Comparison Study

Issue: [#10](https://github.com/cervantesh/CervoMutants/issues/10)

Date: 2026-05-26

## Purpose

This is the first empirical comparison study for CervoMutants improvements. The goal is not to declare a universal winner. The goal is to identify concrete design gaps by running CervoMutants and three Go mutation-testing tools against the same popular Go project under the same local Windows/OneDrive environment.

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

Important validity note: Windows path handling is part of this study because CervoMutants explicitly targets Windows/OneDrive as a supported development environment. Tools that fail on `C:\...` paths are marked as operational failures rather than excluded from the comparison.

## Tools

| Tool | Command family | Result type |
| --- | --- | --- |
| CervoMutants | `cervomut eval` | Completed |
| Gremlins | `gremlins unleash` | Completed |
| gomu | `gomu run` | Failed during mutation preparation |
| go-mutesting v2 | `go-mutesting` | Failed during mutation execution on Windows path handling |

## Commands

CervoMutants:

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
| CervoMutants | Yes | 20 | 13 | 7 | 0 timed out, 0 compile errors | 47.26s | Limited by `--max-mutants 20`; produced summary, JSON, JUnit, HTML, survivor data, and evaluation scorecard. |
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
This is the preferred baseline for comparing CervoMutants against the Go
ecosystem because it avoids Windows-only failures in tools that assume Unix
paths or Unix command-line utilities.

| Tool | Completed? | Mutants attempted | Killed | Survived/lived/escaped | Other statuses | Time | Notes |
| --- | ---: | ---: | ---: | ---: | --- | ---: | --- |
| CervoMutants | Yes | 20 | 13 | 7 survived | 0 not covered, 0 timed out, 0 compile errors | within 2m budget | Same bounded `eval` run as Windows; score 65%, test efficacy 65%, mutation coverage 100%. |
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

CervoMutants reported:

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

This section analyzes the non-CervoMutants tools one by one. The point is not to
copy their implementation. The point is to identify product and engineering
choices that CervoMutants should either avoid or deliberately adopt.

### Gremlins

Negative points to avoid:

- Gremlins is strong for a fast package-level run, but its report contract is
  comparatively narrow for CI and AI agents. CervoMutants should not stop at a
  compact mutation summary; it needs stable schema, survivor context, selected
  tests, threshold decisions, baseline comparison, and machine-readable reasons.
- The human output is useful, but the product surface is mostly "run and read
  result". CervoMutants should avoid making follow-up work manual by default.
  Survivors need enough context to directly generate or review tests.
- Gremlins exposes useful efficacy and coverage metrics, but does not solve the
  governance problem around baselines, quarantine, expiry, and regression policy.
  CervoMutants should keep quality gates baseline-first and auditable.

Positive points to adopt:

- Keep a fast path for package-level mutation testing. In the WSL run Gremlins
  completed 87 mutants in 6.78 seconds, which is the direct speed target for
  CervoMutants package mode.
- Preserve distinct statuses for `killed`, `lived`, `not covered`, `not viable`,
  timeout, and skipped. This avoids confusing unexecuted mutants with weak tests.
- Keep separate metrics for test efficacy and mutator coverage. CervoMutants has
  already adopted `test_efficacy` and `mutation_coverage`; these should remain
  first-class and visible in summary, JSON, and HTML reports.
- Keep per-mutator statistics visible. They make weak operator families obvious
  and help tune mutator profiles.
- Keep the default terminal summary compact. CervoMutants can keep richer
  artifacts, but the default console path should stay scannable.

Application to CervoMutants:

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
  go-mutesting, and CervoMutants.

Positive points to adopt:

- Overlay-based execution is worth studying. It avoids modifying the source tree
  and can reduce copy overhead when implemented with safe path handling.
- Incremental history is valuable for large projects. gomu's history-oriented
  design reinforces CervoMutants' decision to make cache/history a first-class
  feature.
- gomu has broad Go-specific mutation ideas, including error handling and return
  mutations. CervoMutants should borrow the categories carefully, but keep them
  behind profiles with explicit equivalent-mutant risk.
- The CLI has CI concepts such as thresholds and fail-on-gate behavior. CervoMutants
  should keep similar CI ergonomics but combine them with baseline-first adoption.

Application to CervoMutants:

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
  executable and failed on Windows native. CervoMutants should use Go-native
  libraries or internal implementations for required operations.
- Do not build temp paths by concatenating a temp directory with a user/source
  path. go-mutesting's Windows failure came from creating paths under
  `tmpDir/.../C:`.
- Do not let advanced modes block basic usefulness. The first Windows retry with
  `coverage` and `per-test` stalled before producing comparable results.
  Advanced selection should have timeouts, progress, and graceful fallback.
- Do not make report filenames unpredictable or hard to collect. In WSL,
  go-mutesting wrote `report.json` in the target checkout while the command
  flag suggested summary JSON behavior. CervoMutants should keep all report
  outputs under the configured output directory.

Positive points to adopt:

- go-mutesting has the richest mutator set among the Go tools observed in this
  study. It is the best reference for long-term operator breadth.
- Its per-mutator breakdown is useful and should remain a visible CervoMutants
  report section.
- Its agentic JSON direction is aligned with CervoMutants' AI-first goal:
  survived mutants should include stable IDs, diffs, context, nearby tests,
  descriptions, and hints.
- The `noop` preflight is valuable. Running the clean suite before mutation
  prevents meaningless mutation results.
- The `coverage` and `per-test` modes are strategically important for large
  projects, even if they need careful fallback behavior.
- It has useful CI ideas: minimum MSI gates, changed-line filtering, and
  baseline-style survivor handling.

Application to CervoMutants:

- Use go-mutesting as the operator-breadth and agent-report reference.
- Keep all command dependencies internal or explicitly checked by `doctor`.
- Make `doctor` detect external Unix-tool assumptions when running external
  comparison studies.
- Build coverage/per-test selection with bounded setup time, progress reporting,
  and fallback to package/all selection when coverage mapping fails.
- Add report fields for nearby tests, natural-language mutation description, and
  concrete test-writing hints.

## Cross-Tool Design Rules For CervoMutants

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

Actions for CervoMutants:

1. Add a benchmark/compare harness that runs Gremlins, gomu, go-mutesting, and
   CervoMutants under WSL/Linux and normalizes their result schemas.
2. Optimize CervoMutants package mode against Gremlins' speed target.
3. Add a Go-native overlay execution backend prototype with the path-hardening
   rules already implemented in `pkg/isolate`.
4. Expand mutator profiles using go-mutesting and gomu as references, but track
   invalid/not-viable/equivalent risk per operator.
5. Improve AI actionability by adding nearby tests, mutation descriptions, and
   stronger hints to survivor reports.

Issue #10 did not limit implementation to the highest-priority items. The
follow-up work addressed every action at least to an executable first version:

- `cervomut compare` and `pkg/extcompare` normalize CervoMutants, Gremlins, gomu,
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

Implications for CervoMutants:

- Add first-class `not_covered` status or explicit coverage reason instead of folding every unexecuted mutant into skipped/survived categories.
- Separate "test efficacy" from "mutation coverage" in reports, because they answer different questions.
- Include mutator statistics in summary reports, not only in the detailed JSON.
- Provide a fast package-only mode that can run a comparable scope without a heavy evaluation wrapper.

## What CervoMutants Did Better In This Environment

CervoMutants completed successfully under the Windows/OneDrive path shape where gomu and go-mutesting failed.

It also produced richer CI and agent artifacts in one command:

- Stable JSON report.
- JSON schema.
- JUnit XML.
- HTML report.
- Survivor report.
- Evaluation scorecard.
- Explicit summary threshold section.

That is aligned with the goal of being the default AI-friendly mutation-testing tool.

## CervoMutants Gaps Exposed By The Study

1. CervoMutants is slower per attempted mutant than Gremlins in this run.

   CervoMutants ran 20 mutants in about 47 seconds, while Gremlins ran 87 in about 32 seconds. The runs are not strictly equivalent because CervoMutants used `eval`, report generation, isolation, and `--max-mutants 20`, but this still points to isolation/test-selection overhead as a priority.

2. The comparison is not yet apples-to-apples.

   CervoMutants needs a dedicated external comparison harness that records installed tool versions, exact command lines, host information, normalized status mappings, and raw artifact checksums.

3. CervoMutants reports do not yet expose Gremlins-style mutation coverage as a top-level concept.

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

After the initial study, CervoMutants added Gremlins-inspired reporting primitives:

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
- External comparison: `pkg/extcompare` parses the observed CervoMutants,
  Gremlins, gomu, and go-mutesting report shapes and writes a normalized
  `schema_version: "1"` study file through `cervomut compare`.
- Report actionability: mutants carry natural-language `description` and
  package-local `nearby_tests` in JSON reports.
- Mutator breadth: aggressive profile includes literal and return mutations,
  keeping noisy operators out of conservative mode.
- Operator governance: `conditionals` was split into `conditionals-negation`
  and `conditionals-boundary`; `conservative-fast` was added for PR/CI runs;
  `nil-checks` moved to `default` because the Cobra run showed a low kill rate
  and higher survivor burden for that operator family.
- Isolation strategy: `execution.isolation: overlay` uses Go's `-overlay` flag
  for mutation runs that should avoid full module copies while preserving source
  tree cleanliness.

The same Cobra `./doc` dry run now shows the operator-tiering effect:

| CervoMutants profile | Total mutants | Operator breakdown |
| --- | ---: | --- |
| `gremlins-compatible` | 69 | `arithmetic-basic=32`, `conditionals-boundary=12`, `conditionals-negation=25` |
| `conservative-fast` | 69 | `arithmetic-basic=32`, `conditionals-boundary=12`, `conditionals-negation=25` |
| `conservative` | 88 | `conservative-fast` plus `boolean-literals=4`, `logical=15` |
| `default` | 111 | `conservative` plus `nil-checks=23` |

Post-tiering execution against Gremlins:

| Tool | Profile | Workers | Mutants | Killed | Survived/Lived | Not covered | Timed out | Score | Time |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| CervoMutants | `conservative-fast` | 4 | 69 | 51 | 18 | 0 | 0 | 73.91% | 11.34s |
| CervoMutants | `conservative` | 4 | 88 | 59 | 29 | 0 | 0 | 67.05% | 15.27s |
| CervoMutants | `default` | 4 | 111 | 66 | 45 | 0 | 0 | 59.46% | 19.59s |
| Gremlins | default | 4 | 87 | 58 | 29 | 5 | 0 | 66.67% | 22.50s |
| CervoMutants | `conservative-fast` | 16 | 69 | 51 | 18 | 0 | 0 | 73.91% | 6.61s |
| CervoMutants | `conservative` | 16 | 88 | 59 | 29 | 0 | 0 | 67.05% | 9.45s |
| CervoMutants | `default` | 16 | 111 | 66 | 45 | 0 | 0 | 59.46% | 10.85s |
| Gremlins | default | 16 | 87 | 58 | 29 | 5 | 0 | 66.67% | 12.07s |

Interpretation: `conservative` is the closest CervoMutants comparison to the
Gremlins result shape: almost the same total mutants and survivor count, one
more killed mutant, higher score, and faster execution. `conservative-fast`
maximizes signal density and speed, but it also skips useful killed mutants that
Gremlins and CervoMutants `conservative` still exercise. `default` is broader and
finds more killed mutants, but brings back the survivor burden from `nil-checks`.

## Parallel Overlay Performance Follow-Up

The initial Windows run showed CervoMutants was operationally robust but too slow
against Gremlins. The target was tightened to: keep the same mutation score
surface, but make the CervoMutants Cobra `./doc` run complete in less than half
of Gremlins' measured time.

Command:

```text
cervomut eval ./doc --budget 6m --workers 16 --isolation overlay --out %TEMP%\cervomut-cobra-run-20260526-140115\results\cervomut-overlay-w16
```

Result:

| Tool/run | Mutants | Killed | Survived | Score | Time |
| --- | ---: | ---: | ---: | ---: | ---: |
| Gremlins | 87 | 58 | 29 | 66.67% test efficacy | 32.20s |
| CervoMutants serial temp-workdir | 99 | 58 | 41 | 58.59% | 154.48s |
| CervoMutants parallel overlay, 16 workers | 99 | 58 | 41 | 58.59% | 9.12s |

The optimized run is 71.7% faster than Gremlins on this host and 94.1% faster
than CervoMutants' original serial temp-workdir run. The mutation score did not
increase; the gain came from executing real mutant test jobs concurrently and
using Go overlay isolation to avoid full module copies.

## Apples-To-Apples Worker Comparison

The previous `9.12s` versus `32.20s` comparison was not worker-equivalent:
CervoMutants used 16 workers while Gremlins used 4. The study was rerun with
matching worker counts for all three external comparison tools.

Scope and controls:

- Subject: Cobra `./doc`
- Commit: `ad460ea8f249db69c943a365fb84f3a59042d54e`
- Baseline: `go test ./doc` passed
- CervoMutants mode: `--isolation overlay`
- gomu and go-mutesting: patched Windows binaries from this study
- Gremlins timeout coefficient was raised where needed to avoid false timeout
  classifications during high-worker runs.

Important interpretation limit: these tools do not run identical mutator sets.
The table compares each tool's real throughput and classification behavior at
the same worker count, not the same exact mutant list.

| Workers | Tool | Mutants | Killed | Survived/Escaped | Not covered | Errors | Not viable | Timed out | Score | Time |
| ---: | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 4 | CervoMutants overlay | 99 | 58 | 41 | 0 | 0 | 0 | 0 | 58.59% | 16.73s |
| 4 | Gremlins | 87 | 58 | 29 | 5 | 0 | 0 | 0 | 66.67% | 24.66s |
| 4 | gomu patched | 390 | 64 | 99 | 0 | 176 | 51 | 0 | 39.26% | 43.13s |
| 4 | go-mutesting patched | 361 | 170 | 191 | 0 | 0 | 0 | 0 | 47.09% | 61.52s |
| 16 | CervoMutants overlay | 99 | 58 | 41 | 0 | 0 | 0 | 0 | 58.59% | 9.53s |
| 16 | Gremlins | 87 | 58 | 29 | 5 | 0 | 0 | 0 | 66.67% | 13.78s |
| 16 | gomu patched | 390 | 64 | 99 | 0 | 176 | 51 | 0 | 39.26% | 28.81s |
| 16 | go-mutesting patched | 361 | 170 | 191 | 0 | 0 | 0 | 0 | 47.09% | 33.53s |

Findings:

- CervoMutants is the fastest tool in both equal-worker cuts.
- CervoMutants does not yet meet the stricter "half of Gremlins" target when
  Gremlins is given the same worker count.
- gomu remains noisy on this subject: 176 errors and 51 not viable mutants at
  both worker counts.
- go-mutesting has the broadest useful operator surface in this run, but remains
  slower than CervoMutants and Gremlins at both worker counts.
- Gremlins is still the strongest direct reference for concise package-level
  semantics, while CervoMutants now has the best measured throughput in this
  Windows/OneDrive apples-to-apples run.

## Windows Path Hardening Follow-Up

The gomu and go-mutesting failures were converted into executable CervoMutants safeguards:

- isolated workdirs now use a sanitized module token plus hash instead of any raw absolute path fragment;
- isolated workdirs include a `.cervomut-workdir` marker, and cleanup refuses to delete unmarked paths;
- mutant patch target paths are resolved through a containment check, so a malformed worker job cannot escape the copied module;
- engine-generated mutant IDs are module-relative and slash-normalized instead of embedding raw absolute `C:\...` paths;
- regression tests cover Windows-invalid filename characters, OneDrive-style paths with spaces, worker containment, and source-tree preservation.

Local patches were also produced for the external tools so the comparison can be rerun on Windows:

- [gomu v0.2.0 Windows path patch](../external-patches/gomu-v0.2.0-windows-paths.patch)
- [go-mutesting v2.6.13 Windows path patch](../external-patches/go-mutesting-v2.6.13-windows-paths.patch)

## Current Assessment

For this Windows/OneDrive study, CervoMutants is operationally more robust than gomu and go-mutesting, but Gremlins is currently the strongest direct Go implementation reference for speed and concise mutation semantics.

The next improvement cycle should focus on Gremlins-inspired execution/reporting concepts while preserving CervoMutants' stronger CI, schema, baseline, and agent-oriented artifacts.


