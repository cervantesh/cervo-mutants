# Policy Presets

Tracking issue: https://github.com/cervantesh/cervo-mutants/issues/11

CervoMutant presets are adoption modes, not marketing labels. They control
operator breadth, coverage selection, isolation, report formats, and timeout
expectations.

| Preset | Intended use | Operators | Selection | Isolation | Reports |
| --- | --- | --- | --- | --- | --- |
| `ci-fast` | Pull-request smoke and changed-code feedback. | `conservative-fast` | coverage with prefilter | overlay | summary, JSON, JUnit |
| `ci-balanced` | Regular CI where the test suite cost is acceptable. | `conservative` | coverage with prefilter | overlay | summary, JSON, JUnit |
| `comparison-safe` | External tool and multi-repo apples-to-apples calibration. | `gremlins-compatible` | package | overlay | summary, JSON, JUnit |
| `nightly` | Scheduled deeper signal without campaign noise. | `default` | coverage with prefilter | overlay | summary, JSON, JUnit, HTML |
| `campaign` | Manual or scheduled deep mutation campaign. | `aggressive` | package | temp-workdir | summary, JSON, JUnit, HTML |

## Default Adoption Rule

Start with:

```powershell
cervomut fast ./... --scope changed --since origin/main --budget 10m
```

Use `ci-balanced` when the team accepts more runtime for broader conservative
operators:

```powershell
cervomut run ./... --policy ci-balanced --max-mutants 100 --sample deterministic
```

Use `comparison-safe` for external tool comparison and multi-repo calibration:

```powershell
cervomut run . --policy comparison-safe
```

This preset is intentionally bounded:

- `execution.workers` is capped at 2;
- `execution.budget` defaults to 10 minutes;
- `tests.timeout` defaults to 20 seconds per mutant;
- `limits.sample` is deterministic;
- `limits.max_mutants` defaults to 250 unless explicitly set;
- `partial-summary.json` and `partial-mutation-report.json` preserve observed
  denominators on timeout.

Use `nightly` for scheduled jobs and baseline comparison:

```powershell
cervomut run ./... --policy nightly --budget 30m --sample deterministic
```

Use `campaign` only when review time is available:

```powershell
cervomut run ./... --policy campaign --out .cervomut/campaign
```

## Governance Defaults

- `ci.fail_under` remains `0`; adoption is baseline-first.
- `baseline.fail_on_regression` and `baseline.fail_on_new_survivors` remain
  enabled.
- Quarantine requires owner, issue, reason, expiry, and renewal limits.
- Suppression is stricter than quarantine: explicit `action: suppress` requires
  `evidence: confirmed` and at least one reviewer.
- History is enabled by default at `.cervomut/history.json` so reports can
  identify new and long-standing survivors.

