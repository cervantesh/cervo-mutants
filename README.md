# cervo-mutant

`cervo-mutant` is a Go mutation testing library and CLI for CervoSoft projects.
It provides the `cervomut` binary plus importable packages for discovery,
mutation, execution, reporting, baseline comparison, and daemon-ready job
contracts.

Tracking issue: https://gitea.cervbox.synology.me/CervoSoft/cervo-mutant/issues/1

## Quick Start

```powershell
go install gitea.cervbox.synology.me/CervoSoft/cervo-mutant/cmd/cervomut@latest
cervomut init
cervomut doctor
cervomut run ./... --budget 10m --sample deterministic
cervomut run ./... --policy ci-fast --budget 10m
```

## Commands

- `cervomut init`
- `cervomut doctor`
- `cervomut affected ./...`
- `cervomut run ./...`
- `cervomut run ./... --dry-run`
- `cervomut run ./... --policy ci-fast --coverage-prefilter`
- `cervomut run ./... --workers 16 --isolation overlay`
- `cervomut eval ./... --out .cervomut/evaluation`
- `cervomut compare --cervomut .cervomut/reports/mutation-report.json --gremlins gremlins.json --gomu gomu.txt --go-mutesting go-mutesting.txt`
- `cervomut baseline update`
- `cervomut baseline compare`
- `cervomut report summary --out .cervomut/reports`
- `cervomut report survivors --out .cervomut/reports`
- `cervomut report open`
- `cervomut show <mutant-id> --out .cervomut/reports`
- `cervomut explain <mutant-id> --format text|json`
- `cervomut list-mutators`
- `cervomut daemon`
- `cervomut worker`

## Defaults

The default policy is baseline-first adoption:

- `ci.fail_under` is `0`.
- baseline regression and new survivors are tracked.
- quarantine entries require owner, issue, reason, and expiry.
- reports include summary, JSON schema v1, JUnit XML, and HTML.
- execution uses temp workdirs so source workspaces are not mutated.
- `execution.isolation: overlay` is available for Go-native overlay mutation
  runs that avoid copying the full module.
- coverage mode records a Go coverage profile during baseline and uses it to
  pick package-scoped test commands when the mutated file is covered.
- package mode can use `selection.prefilter: true` to report obvious
  `not_covered` mutants from the baseline coverage profile before spending
  mutation budget.
- budgeted runs schedule fast/recommended operators before broader campaign
  operators.
- mutator profiles are tiered: `gremlins-compatible`, `conservative-fast`,
  `conservative`, `default`, and `aggressive`.
- policy presets are available: `ci-fast`, `ci-balanced`, `nightly`, and
  `campaign`.
- JSON reports include mutation descriptions, nearby package tests, selected
  test commands, status reasons, diffs, hints, survivor ranks, suppression
  audit hits, and per-mutator statistics.
- worker mode applies jobs in isolated temp workdirs and returns the same JSON
  result schema as local execution.
- cache fingerprints include the mutant patch, source file, relevant tests,
  Go version, effective config, test command, and module files.

## Development

```powershell
go test ./...
go run ./cmd/cervomut list-mutators
```

## Evaluation

Use [docs/evaluation-framework.md](docs/evaluation-framework.md) to compare
mutation testing tools and decide whether `cervo-mutant` should be the default
for CervoClaw and CervoSoft projects.

Use [docs/signal-first-mutation-testing.md](docs/signal-first-mutation-testing.md)
for the product framework behind CervoMutant defaults: coverage semantics,
score decomposition, CI relevance, history-aware scheduling, equivalence
governance, and agent actionability.

Use [docs/evaluations/multi-repo-calibration.md](docs/evaluations/multi-repo-calibration.md)
to run the Cobra, CervoClaw, CervoRetry, and external Go repository calibration
before promoting operators or changing policy defaults.

Use [docs/evaluation-template.md](docs/evaluation-template.md) for manual
reviews, or run:

```powershell
cervomut eval ./... --out .cervomut/evaluation
```
