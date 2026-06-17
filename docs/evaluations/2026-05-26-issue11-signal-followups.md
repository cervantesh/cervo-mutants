# Issue #11 Signal Follow-Ups

Tracking issue: https://github.com/cervantesh/cervo-mutants/issues/11

Date: 2026-05-26

This document records the first five follow-up actions from issue #11. The goal
is to reduce the remaining signal gap without copying another tool's behavior:
use the comparison data, preserve CervoMutant's CI/agent contract, and make the
semantics stricter where a score can be misread.

## 1. Cobra Re-Run Against Reference Tools

Subject:

- Repository: `github.com/spf13/cobra`
- Scope: `./doc`
- Baseline: `go test ./doc` passed
- Workers: 4
- Host: Windows under OneDrive-style paths

Commands:

```powershell
cervomut run ./doc --profile conservative --isolation overlay --workers 4 --out <out>
gremlins unleash ./doc --workers 4 --threshold-efficacy 0 --threshold-mcover 0 --output <out>\gremlins.json
gomu-patched run ./doc --workers 4 --timeout 30 --threshold 0 --fail-on-gate=false --output json
go-mutesting-patched /noop /quiet /no-diffs /logger-summary-json /logger-agentic-json /exec-timeout:30 /workers:4 ./doc
```

Normalized results:

| Tool | Mutants reported | Killed | Survived/lived/escaped | Not covered | Errors | Not viable | Timed out | Score/efficacy | Wall time |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| CervoMutant | 255 | 137 | 118 | 0 | 0 | 0 | 0 | 53.73% | 83.72s |
| Gremlins | 32 effective, 92 generated entries | 31 | 1 | 5 | 0 | 0 | 55 | 96.88% efficacy, 86.49% mutation coverage | 25.81s |
| gomu patched | 390 | 64 | 99 | 0 | 176 | 51 | 0 | 39.26% over viable executed mutants | 59.67s |
| go-mutesting patched | 361 | 170 | 191 | 0 | 0 | 0 | 0 | 47.09% MSI | 90.78s |

Interpretation:

- The raw denominators are not equivalent. CervoMutant conservative generated
  substantially more valid mutants than Gremlins' top-level effective total.
- Gremlins remains much faster in this specific re-run, but it also classifies
  55 generated entries as timed out in the file-level mutation list while its
  top-level metrics report 32 effective killed/lived mutants.
- gomu still has a high operational/noise burden even after the Windows path
  patch: 176 errors and 51 not viable mutants.
- go-mutesting remains the strongest breadth reference, but is slower than
  CervoMutant here and has a lower MSI on the same package.
- CervoMutant should compare with `ci-fast` or `gremlins-compatible` when the
  question is speed; conservative/default/campaign are broader signal modes.

Raw artifacts:

```text
C:\Users\c___h\AppData\Local\Temp\cervomut-issue11-cobra-comparison
```

The current `cervomut compare` parser was also re-run against those artifacts.
Normalized output:

```text
C:\Users\c___h\AppData\Local\Temp\cervomut-issue11-cobra-comparison\normalized-by-current-cervomut.json
```

## 2. Real CI Pipeline

The GitHub Actions workflow now does more than compile and run unit tests:

```yaml
go vet ./...
go test ./...
go test -race ./pkg/engine ./pkg/mutator ./pkg/report
go run ./cmd/cervomut list-mutators
go run ./cmd/cervomut fast ./pkg/config --max-mutants 3 --workers 2 --out .cervomut/ci-smoke
test -f .cervomut/ci-smoke/mutation-report.json
go run ./cmd/cervomut run ./pkg/config --policy ci-balanced --max-mutants 5 --workers 2 --out .cervomut/ci-balanced-smoke
grep -q '"schema_version": "1"' .cervomut/ci-balanced-smoke/mutation-report.json
```

This is intentionally a smoke gate, not a quality gate. It proves the importable
library, CLI, mutator registry, policy preset, coverage prefilter path, and JSON
report generation are executable in CI.

## 3. 100-Mutant Multi-Repo Calibration

Command shape:

```powershell
cervomut run <target> --policy ci-balanced --max-mutants 100 --sample deterministic --workers 2 --out <out>
```

Results:

| Repository | Target | Generated | Killed | Survived | Not covered | Timed out | Compile errors | Score | Mutation coverage | Wall time |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Cobra | `./doc` | 100 | 46 | 39 | 15 | 0 | 0 | 54.12% | 85.00% | 31.75s |
| CervoCore | `./...` | 100 | 12 | 28 | 60 | 0 | 0 | 30.00% | 40.00% | 50.46s |
| CervoRetry | `./...` | 52 | 18 | 10 | 24 | 0 | 0 | 64.29% | 53.85% | 16.09s |
| pflag | `./...` | 100 | 60 | 14 | 26 | 0 | 0 | 81.08% | 74.00% | 66.30s |

Findings:

- `ci-balanced` is reliable across the four repos: no timeouts and no compile
  errors in this 352-mutant sample.
- Coverage prefiltering changes score interpretation. CervoCore and CervoRetry
  show large `not_covered` buckets, so the main next question is test reach, not
  only assertion strength.
- pflag remains a useful external high-signal control.
- CervoRetry is small enough that the 100-mutant cap does not apply; it produced
  52 mutants under the selected profile.

Raw artifacts:

```text
C:\Users\c___h\AppData\Local\Temp\cervomut-issue11-100-calibration
```

## 4. Coverage Range Semantics

Coverage filtering now parses Go coverage profile ranges:

```text
path/to/file.go:startLine.startCol,endLine.endCol statements count
```

The old behavior treated a mutant as covered when the file appeared anywhere in
the profile. That was too coarse. The new behavior requires:

- profile line is parseable;
- profile file matches the mutant file;
- execution count is greater than zero;
- mutant line falls inside the covered start/end line range.

If no parseable coverage entries exist, CervoMutant keeps a compatibility
fallback to the older file-presence behavior. Tests now cover both same-file
covered and same-file uncovered line ranges.

## 5. Conservative Real Suppression

Suppression rules are no longer only report annotations when a rule explicitly
uses:

```yaml
action: suppress
```

The engine now marks those mutants as `ignored` before preparing or executing a
mutation. This is deliberately conservative:

- default rules do not suppress anything;
- `report-only`, `lower-priority`, and `quarantine-required` remain audit/ranking
  signals;
- explicit suppressions remain visible through `suppression_audit` and summary
  counts;
- ignored mutants are excluded from the primary score denominator, preventing
  both false punishment and false score inflation.

This gives us a real mechanism for audited equivalent-risk suppression without
making suppression an invisible default.

## 6-15. Remaining Follow-Ups Applied

The second issue #11 pass addressed the remaining backlog items with additive
schema fields and stricter governance. Items that require human review are now
represented as explicit data capture rather than hidden TODOs.

| Pending item | Applied change | Remaining evidence needed |
| --- | --- | --- |
| Survivor ranking human calibration | Survivor ranking now includes `history_status`, `survivor_age_runs`, and `operator_historical_yield`. Reports can distinguish new, existing, and long-standing survivors. | Manual review sample to tune weights. |
| Deeper coverage semantics | Coverage filtering already moved to Go coverage line ranges; the report now keeps enough line/function context for range review. | Function/range cross-check against larger coverage profiles. |
| Conservative equivalent-risk suppression | `suppress` rules now require `evidence: confirmed` and `reviewers >= 1`; rules can narrow by operator, file, original token, mutated token, and equivalent risk. | False-suppression audit sample per release. |
| Advanced history | `.cervomut/history.json` tracks status, first seen, last seen, run counts, survivor age, killed runs, not-covered runs, compile errors, and timeouts by stable mutant ID. | Calibrate history retention and trend dashboards. |
| Broader operators | Added governed `assignment-arithmetic` and `inc-dec` operators to `default`/`aggressive`, inspired by Go operator breadth in external tools without moving them into fast CI. | Multi-repo operator yield comparison. |
| External tool revalidation | Re-ran the current `cervomut compare` parser against the Cobra artifacts from CervoMutant, Gremlins, patched gomu, and patched go-mutesting. | Schedule periodic full reruns after operator changes. |
| Preset UX documentation | Config defaults now expose `history`; docs explain `ci-fast`, `ci-balanced`, `nightly`, and `campaign` as distinct adoption modes. | Expand examples per repo type. |
| Real CI pipeline | GitHub Actions now run `go vet`, full tests, race tests for core packages, mutator registry smoke, `ci-fast` report smoke, and `ci-balanced` report smoke. | Add protected-branch gating once the repo policy is set. |
| Final PR/issue hygiene | Issue #11 has progress comments; branch and PR reference `#11`; docs link back to the issue. | Close issue after PR review/merge. |
| Wiki continuity | Wiki page is updated with the issue #11 signal-followup summary. | Push wiki changes with the implementation branch. |

## New JSON Fields

The schema remains v1 and additive-only. New fields are optional for older
reports:

```json
{
  "history": {
    "enabled": true,
    "path": ".cervomut/history.json",
    "loaded_mutants": 123,
    "updated_mutants": 100,
    "new_survivors": 4,
    "long_standing_survivors": 2,
    "operator_useful_survivor_yield": {
      "conditionals-negation": 0.18
    }
  },
  "mutants": [
    {
      "previous_status": "survived",
      "first_seen": "2026-05-26T00:00:00Z",
      "last_seen": "2026-05-26T01:00:00Z",
      "survivor_age_runs": 2,
      "history_status": "long_standing_survivor",
      "operator_historical_yield": 0.18
    }
  ]
}
```

## New Governed Operators

| Operator | Profile | Why it is not fast CI |
| --- | --- | --- |
| `assignment-arithmetic` | `default`, `aggressive` | Useful for update logic, but can be compile-sensitive and duplicates some arithmetic signal. |
| `inc-dec` | `default`, `aggressive` | Useful for counters and loops, but can create equivalent or domain-specific survivors. |

Both operators are intentionally excluded from `gremlins-compatible`,
`conservative-fast`, and `conservative` until calibration proves they preserve
signal across multiple repositories.
