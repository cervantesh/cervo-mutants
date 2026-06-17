# CervoMutants

CervoMutants is a Go mutation testing toolkit built for CI, large Go modules,
and AI-assisted test improvement. It ships as:

- an importable Go library under `pkg`;
- a thin CLI, `cervomut`, under `cmd/cervomut`;
- stable JSON/JUnit/HTML reports for CI and agents;
- comparison tooling for Gremlins, gomu, and go-mutesting studies.

Mutation testing answers a different question than coverage:

> If this code is changed in a small realistic way, do the tests fail?

A surviving mutant is not automatically a bug. It is a signal that the current
tests either do not execute the behavior, do not assert the changed outcome, or
that the mutation is equivalent/noisy and should be governed explicitly.

Tracking issue for the original project:  
https://github.com/cervantesh/CervoMutants/issues/1

Current documentation refresh:  
https://github.com/cervantesh/CervoMutants/issues/33

## Current Status

CervoMutants is usable as a local/CI mutation runner and as an evaluation
platform. The project currently includes:

- AST/source rewrite mutators with stable mutant IDs.
- Temp-workdir and Go overlay isolation.
- Baseline-first adoption policy.
- Incremental cache and history files.
- Coverage/package/all test selection modes.
- Budgeted runs, deterministic sampling, and resume from partial reports.
- Strict quarantine and audited suppression rules.
- Summary, JSON schema v1, JUnit XML, and HTML reports.
- Survivor ranking and denominator-health reporting.
- Daemon/worker JSON-lines contracts for future distributed execution.
- External-tool comparison normalization for apples-to-apples studies.

## Install

From a Go workspace:

```powershell
go install github.com/cervantesh/CervoMutants/cmd/cervomut@latest
```

From this repository:

```powershell
go build ./cmd/cervomut
```

## First Run

Use `doctor` before the first run. It catches missing Go/Git tools and warns
about Windows/OneDrive paths that can make mutation runs slower or less stable.

```powershell
cervomut init
cervomut doctor
cervomut run ./... --dry-run
cervomut fast ./... --budget 10m --sample deterministic
```

The default output directory is `.cervomut/reports`.

Important files:

| File | Purpose |
| --- | --- |
| `.cervomut/reports/mutation-report.json` | Stable JSON report for CI and agents. |
| `.cervomut/reports/summary.txt` | Human-readable run summary. |
| `.cervomut/reports/junit.xml` | CI test report format. |
| `.cervomut/reports/index.html` | Human report for browsing survivors and diffs. |
| `.cervomut/reports/partial-mutation-report.json` | Checkpoint report for timeout/interrupted runs. |
| `.cervomut/reports/partial-summary.json` | Small checkpoint summary. |
| `.cervomut/history.json` | Historical survivor/cache signal. |
| `.cervomut/baseline.json` | Accepted baseline for regression gates. |
| `.cervomut/quarantine.json` | Temporary audited quarantine entries. |

## Recommended CI Flow

Start baseline-first. Do not fail existing projects on a raw score threshold on
day one.

```powershell
cervomut fast ./... --scope changed --since origin/main --budget 10m --sample deterministic
```

Then store or compare the baseline:

```powershell
cervomut baseline update
cervomut baseline compare
```

The default `ci.fail_under` is `0`. CI should usually fail on:

- baseline regression;
- new survivors when the baseline policy says so;
- expired quarantine entries;
- timeouts or compile errors if configured.

## Policy Presets

Policies are adoption modes. They set mutator breadth, selection behavior,
isolation, timeouts, and report formats.

| Policy | Use When | Mutators | Selection | Isolation | Reports |
| --- | --- | --- | --- | --- | --- |
| `ci-fast` | Pull requests and quick changed-code feedback. | `conservative-fast` | coverage + prefilter | overlay | summary, JSON, JUnit |
| `ci-balanced` | Regular CI with more runtime available. | `conservative` | coverage + prefilter | overlay | summary, JSON, JUnit |
| `comparison-safe` | Comparing against external tools. | `gremlins-compatible` | package | overlay | summary, JSON, JUnit |
| `nightly` | Scheduled deeper signal. | `default` | coverage + prefilter | overlay | summary, JSON, JUnit, HTML |
| `campaign` | Manual deep mutation campaign. | `aggressive` | package | temp-workdir | summary, JSON, JUnit, HTML |

Examples:

```powershell
cervomut run ./... --policy ci-balanced --max-mutants 100 --sample deterministic
cervomut run . --policy comparison-safe
cervomut run ./... --policy nightly --budget 30m
cervomut run ./... --policy campaign --out .cervomut/campaign
```

More detail: [docs/policy-presets.md](docs/policy-presets.md).

## CLI Reference

| Command | Purpose |
| --- | --- |
| `cervomut init` | Write `cervomut.yaml` with defaults. |
| `cervomut doctor` | Check Go/Git/runtime environment and path risks. |
| `cervomut affected ./...` | Estimate affected modules/packages/files and mutant count. |
| `cervomut run ./...` | Run mutation testing. |
| `cervomut run ./... --dry-run` | Discover mutants without executing tests. |
| `cervomut fast ./...` | Shortcut for fast CI defaults. |
| `cervomut eval ./...` | Build a structured evaluation artifact. |
| `cervomut compare ...` | Normalize external tool reports. |
| `cervomut baseline update` | Save current report as baseline. |
| `cervomut baseline compare` | Compare current report to baseline. |
| `cervomut report summary --out DIR` | Print report summary. |
| `cervomut report survivors --out DIR` | Print ranked surviving mutants. |
| `cervomut report open` | Open the HTML report. |
| `cervomut show MUTANT_ID --out DIR` | Show a mutant diff/context. |
| `cervomut explain MUTANT_ID --format text\|json` | Explain what a survivor means. |
| `cervomut list-mutators` | List operators and risk metadata. |
| `cervomut daemon` | JSON-lines daemon mode. |
| `cervomut worker` | JSON-lines worker mode. |

Common run flags:

```powershell
cervomut run ./... `
  --policy ci-fast `
  --budget 10m `
  --max-mutants 100 `
  --sample deterministic `
  --workers 2 `
  --isolation overlay `
  --test-timeout 20s `
  --coverage-prefilter `
  --resume `
  --max-process-memory-mb 6144 `
  --out .cervomut/reports
```

## Configuration

`cervomut init` writes `cervomut.yaml`. The defaults are conservative:

```yaml
version: 1
policy: ""
scope:
  mode: all
  since: origin/main
  include: ["./..."]
tests:
  command: ["go", "test", "./..."]
  timeout: 30s
  baseline_required: true
mutators:
  profile: conservative
execution:
  isolation: temp-workdir
selection:
  mode: package
cache:
  enabled: true
  mode: incremental
baseline:
  enabled: true
  fail_on_regression: true
  fail_on_new_survivors: true
ci:
  fail_under: 0
quarantine:
  enabled: true
  fail_on_expired: true
reports:
  formats: [summary, json, junit, html]
```

Supported selection modes:

- `all`: run the configured test command.
- `package`: rewrite `go test ./...` to the mutant package when possible.
- `coverage`: use a baseline Go coverage profile to avoid irrelevant tests and
  classify obvious `not_covered` mutants.

Supported isolation modes:

- `temp-workdir`: copy the module to a marked temporary workdir and mutate the
  copy.
- `overlay`: use Go's `-overlay` support to avoid copying the module.

## Mutator Profiles

| Profile | Intended Use | Operators |
| --- | --- | --- |
| `gremlins-compatible` | Apples-to-apples comparison with Gremlins. | conditionals, boundary, basic arithmetic |
| `conservative-fast` | PR feedback. | conditionals, boundary, basic arithmetic |
| `conservative` | Default high-signal set. | conservative-fast + logical, boolean literals, controlled string emptying |
| `default` | Nightly/default deeper signal. | conservative + nil checks, numeric literals, return bools, assignments, inc/dec |
| `aggressive` | Manual campaign. | default + broad literals, returns, loop-control, len boundaries |

Every operator declares risk, equivalent-mutant risk, compile-error risk, AST
node kinds, example diff, and inclusion reason. Use:

```powershell
cervomut list-mutators
```

## How To Read Results

The primary score is not enough. Always inspect denominator health:

- generated mutants;
- covered mutants;
- executed mutants;
- effective mutants: `killed + survived`;
- score denominator;
- not covered;
- timed out;
- compile errors.

This is intentional. A raw score can look excellent while most mutants are not
covered or timed out. CervoMutants makes that visible at the top level of the
JSON and summary reports.

Survivors are ranked by actionability using:

- equivalent-risk metadata;
- recommendation tier;
- nearby tests;
- exported function context;
- coverage source;
- history status;
- operator historical yield;
- suppression audit hits.

## Quarantine And Suppression

Quarantine is temporary debt, not a way to inflate score.

Quarantine entries must include:

- mutant id;
- reason;
- owner;
- issue;
- expiry;
- renewal count within policy.

Expired quarantines can fail CI. Quarantined mutants remain visible in reports
and do not improve the primary score.

Suppression is stricter. A rule with `action: suppress` requires confirmed
evidence and at least one reviewer. Lower-confidence rules should use
`report-only`, `lower-priority`, or `quarantine-required`.

Inline ignores are allowed only with reasons:

```go
// cervomut:ignore conditionals reason="generated contract covers this branch"
return value == expected
```

## External Tool Comparisons

CervoMutants includes a comparison harness because mutation tools do not always
mean the same thing by target, coverage, timeout, and score.

The harness records:

- manifest target;
- effective target;
- target mode;
- killed/survived/not-covered/timed-out/errors;
- test efficacy;
- mutation coverage;
- denominator health;
- status classification such as `ok`, `no_report`, `no_results`,
  `all_timed_out`, `not_covered_only`, `timeout`, and `watchdog_kill`.

Important rule:

> Do not compare `cervomut run ./...` with `gremlins unleash .` as if they were
> equivalent unless the harness records target normalization explicitly.

Docs:

- [docs/evaluations/tool-comparison-protocol.md](docs/evaluations/tool-comparison-protocol.md)
- [docs/evaluations/comparison-harness.md](docs/evaluations/comparison-harness.md)
- [docs/evaluations/tool-findings.md](docs/evaluations/tool-findings.md)

## Windows, WSL, And Memory

Windows-native mutation testing can be slower and more memory-sensitive because
of process spawning, path handling, file copying, antivirus/OneDrive sync, and
external tool behavior. CervoMutants mitigates this with:

- conservative Windows worker defaults;
- `doctor` warnings for OneDrive, long paths, and network paths;
- overlay isolation;
- process-tree memory controls where supported;
- partial checkpoint reports;
- resume support.

For large local experiments, WSL2 on the Linux filesystem is usually more
stable than running under `C:\Users\...\OneDrive`.

Recommended guarded local pattern:

```powershell
cervomut run ./... `
  --policy comparison-safe `
  --workers 1 `
  --budget 10m `
  --max-process-memory-mb 6144 `
  --resume
```

## Library API

The CLI is intentionally thin. The importable engine API is the long-term
automation surface:

```go
result, err := engine.New(cfg).Run(ctx, engine.RunRequest{
    Targets: []string{"./..."},
})
```

Important packages:

| Package | Responsibility |
| --- | --- |
| `pkg/config` | Defaults, YAML loading, policy presets, validation. |
| `pkg/discover` | Modules, packages, files, generated/vendor exclusions. |
| `pkg/mutator` | AST mutators, stable IDs, inline ignores. |
| `pkg/isolate` | Temp-workdir copy and cleanup safety. |
| `pkg/engine` | Orchestration, scheduling, cache/history/baseline/report flow. |
| `pkg/runner` | Test execution and status classification. |
| `pkg/report` | Summary, JSON, JUnit, HTML. |
| `pkg/extcompare` | External tool normalization. |
| `pkg/daemon` | JSON-lines worker/daemon contracts. |

## Development

```powershell
go test ./...
go test ./... -coverprofile=coverage.out
go vet ./...
staticcheck ./...
go run ./cmd/cervomut list-mutators
```

Sonar docs: [docs/sonar.md](docs/sonar.md).

Latest local Sonar pass after issue #31:

| Metric | Value |
| --- | ---: |
| Coverage | 90.7% |
| Code smells | 0 |
| Duplicated lines density | 0.0% |
| Bugs | 0 |
| Vulnerabilities | 0 |

## Documentation Map

- [docs/evaluation-framework.md](docs/evaluation-framework.md): rigorous
  evaluation framework and research basis.
- [docs/signal-first-mutation-testing.md](docs/signal-first-mutation-testing.md):
  product framework behind CervoMutants defaults.
- [docs/policy-presets.md](docs/policy-presets.md): policy preset behavior.
- [docs/sonar.md](docs/sonar.md): local and CI Sonar workflow.
- [docs/go-toolchain-compatibility.md](docs/go-toolchain-compatibility.md):
  supported Go versions and `doctor` checks.
- [docs/evaluations/multi-repo-calibration.md](docs/evaluations/multi-repo-calibration.md):
  multi-repo calibration plan.
- [docs/evaluations/tool-comparison-protocol.md](docs/evaluations/tool-comparison-protocol.md):
  apples-to-apples comparison rules.
- [docs/evaluations/comparison-harness.md](docs/evaluations/comparison-harness.md):
  runnable harness description.
- [docs/evaluations/tool-findings.md](docs/evaluations/tool-findings.md):
  findings from Gremlins, gomu, and go-mutesting comparisons.

## Design Principles

- Prefer actionable survivors over raw mutant volume.
- Make denominator health impossible to miss.
- Keep CI adoption baseline-first.
- Keep quarantine temporary and auditable.
- Preserve partial data on timeout.
- Normalize external comparisons before drawing conclusions.
- Make JSON stable enough for CI and AI agents.

## License And Trademarks

The source code in this repository is licensed under Apache License 2.0. See
[LICENSE](LICENSE) and [NOTICE](NOTICE).

The CervoMutants name, logos, and project branding are reserved and are not
granted under the code license. See [TRADEMARKS.md](TRADEMARKS.md).

