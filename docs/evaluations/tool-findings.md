# External Tool Findings Ledger

Tracking issue: https://github.com/cervantesh/cervo-mutants/issues/13

Purpose: keep reusable findings from external mutation testing tools in one
place so CervoMutant can learn from each tool without copying blindly or
repeating avoidable benchmarking mistakes.

Each tool section should answer:

- how to invoke the tool correctly;
- where the tool gives strong signal;
- where its metrics can mislead;
- operational failure modes;
- concrete CervoMutant improvements derived from the finding;
- comparison harness rules needed for fair future runs.

All future tool runs must follow the
[mutation tool comparison protocol](tool-comparison-protocol.md). Do not
summarize speed or score as a fairness claim unless the compared rows share the
same `effective_target` and `target_mode`.

## Gremlins

Primary references in this repo:

- [Cobra mutation tool comparison](2026-05-26-cobra-mutation-tool-comparison.md)
- [Gremlins-focused comparison pool](2026-05-28-gremlins-focused-pool.md)

### Correct Invocation

Gremlins should be treated as a package-root mutation tool, not as a direct
`go test ./...` equivalent.

For package-level comparisons:

```bash
gremlins unleash . \
  --workers 2 \
  --timeout-coefficient 4 \
  --threshold-efficacy 0 \
  --threshold-mcover 0 \
  --output gremlins.json
```

When the CervoMutant manifest target is `./...`, the comparison harness should
store both values:

```text
target=./...
effective_target=.
```

This is mandatory because Gremlins and CervoMutant do not have identical target
semantics. Without `effective_target`, later readers can accidentally treat the
results as apples-to-apples when they are not.

### Strong Signals To Learn From

Gremlins is useful as a reference for:

- concise package-level execution;
- coverage-first mutation semantics;
- separate `test_efficacy` and mutation coverage concepts;
- explicit `not covered` mutants;
- simple threshold controls;
- clear operator families for fast Go feedback.

CervoMutant should keep these concepts visible in CI and agent-oriented reports,
while preserving CervoMutant's stronger baseline, quarantine, JSON schema,
partial reporting, and governance model.

### Metric Risks

Gremlins' raw score can be misleading when the effective denominator is tiny.

Example from the 2026-05-29 corrected WSL2 run:

| Repo | Total effective | Killed | Survived | Not covered | Timed out | Score |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `gjson` | 3 | 3 | 0 | 37 | 1244 | 100.00 |
| `decimal` | 9 | 9 | 0 | 93 | 691 | 100.00 |
| `jsonparser` | 8 | 7 | 1 | 848 | 593 | 87.50 |

The score is technically true over `killed + survived`, but it does not tell the
whole story when timed-out and not-covered mutations dominate the run. CervoMutant
reports must therefore include a top-level denominator health section:

```text
generated
covered/effective
killed
survived
not_covered
timed_out
compile_error
score denominator
```

The report should flag high-risk denominator shapes, for example:

- effective mutants are below a configured minimum;
- timed-out mutations exceed effective mutants;
- not-covered mutations exceed effective mutants;
- score is high but denominator health is poor.

### Operational Findings

Observed Gremlins behavior:

- Passing `./...` can produce `No results to report`, no JSON, or misleadingly
  empty outputs even when the repository has tests.
- On Windows/OneDrive, earlier runs produced panic/no-report cases after
  coverage collection.
- In WSL2 with package-root invocation, Gremlins produced useful JSON in 9/10
  small-pool repos.
- `urfave-cli` still timed out at 600 seconds under the corrected WSL2 run.
- `--timeout-coefficient` materially affects whether results are useful; it
  should be explicit in every comparison run.

### Harness Rules

The comparison harness must classify Gremlins outcomes with actionable statuses:

- `ok`
- `no_report`
- `no_results`
- `panic`
- `all_timed_out`
- `not_covered_only`
- `timeout`
- `watchdog_kill`

For every Gremlins row, store:

- original manifest target;
- effective target;
- timeout coefficient;
- worker count;
- timeout;
- whether cgroup/process-tree limits were active;
- report path;
- log path.

Do not treat `exit 0` as success unless a report exists or the log clearly says
`No results to report`. A successful process with no JSON is a tool outcome that
needs classification, not an empty metric row.

### CervoMutant Improvements Derived From Gremlins

The Gremlins findings imply these CervoMutant improvements:

- Add a denominator health block to JSON, summary, and HTML reports.
- Preserve partial checkpoint metrics on timeout or watchdog termination.
- Add explicit `effective_target` to external comparison results.
- Add a `gremlins-package-root` comparison strategy.
- Add score warnings when `timed_out` or `not_covered` dwarf effective mutants.
- Keep `test_efficacy` and mutation coverage separate in user-facing reports.
- Improve budget-aware scheduling by using operator/repo timeout risk.
- Add deterministic sampling for high-timeout subjects before full campaigns.

Implementation status:

- Implemented denominator health in CervoMutant run summaries and reports.
- Implemented `effective_mutants`, `score_denominator`, and Gremlins-style
  `test_efficacy` over `killed + survived`.
- Implemented external comparison status classification fields for normalized
  tool results.
- Implemented Gremlins target normalization helpers for `gremlins-package-root`
  comparisons.
- Implemented `effective_target` and `not_comparable` metadata for comparison
  output.
- Implemented study-level `comparability` metadata with `apples_to_apples`,
  `manifest_equivalent`, `effective_targets`, `target_modes`, and warnings.
- Implemented CervoMutant target metadata in `cervomut compare` so CervoMutant
  and external tools can be normalized together instead of normalizing only
  Gremlins.
- Updated the pool comparison harness with `CompareTargetMode package-root` so
  CervoMutant, Gremlins, gomu, and go-mutesting can all receive the same
  package-root effective target when the study is meant to be fair.
- Added agent-facing comparison protocol instructions to `AGENTS.md`.
- Implemented budget scheduling tie-breaks that prioritize lower timeout-risk
  operators within the same recommendation class.
- Re-ran the small 10-repo pool in WSL2 after these changes. Denominator health
  correctly flagged high-score Gremlins rows with poor effective denominators:
  `pflag`, `decimal`, and `jsonparser`.
- Implemented atomic partial mutation report writes after the re-run exposed a
  zero-byte CervoMutant checkpoint on `gjson` under watchdog termination.

Remaining implementation gap:

- Cap or segment very large partial reports; `jsonparser` produced a useful but
  large checkpoint around 75 MB.

### What Not To Copy

Do not copy Gremlins behavior where it hurts CI/agent reliability:

- do not hide denominator risk behind a single high score;
- do not silently return unusable metrics as success;
- do not require users to infer target semantics from command syntax;
- do not lose partial work when a long run times out;
- do not make coverage-only execution the only useful mode.

## gomu

Status: historical findings only for the current Gremlins-focused comparison
cycle. Keep adding concrete gomu findings here if it becomes useful again as a
reference.

Current known themes to preserve:

- avoid deriving filesystem names from raw absolute paths;
- distinguish invalid, not viable, compile-error, and survived mutants;
- learn from history/incremental execution ideas, but require deterministic
  cache keys and bounded resource behavior.

## go-mutesting

Status: historical findings only for the current Gremlins-focused comparison
cycle. Keep adding concrete go-mutesting findings here if it becomes useful
again as a reference.

Current known themes to preserve:

- broad operator catalog is useful as an operator-breadth reference;
- agent-oriented reports are useful;
- avoid implicit Unix tool dependencies;
- avoid writing reports into target checkouts without a predictable artifact
  contract.
