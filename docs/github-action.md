# GitHub Action

Tracking issues: #159, #182

CervoMutants now ships a first-party GitHub Action for the common CI path:
install the tool, set up Go, and run a bounded `cervomut run` mutation lane.

The current public action lives at the repository root as `action.yml`.

## Scope

The action is intentionally focused on the common mutation-run workflow.

It handles:

- setting up the Go toolchain
- installing `cervomut`
- running `cervomut run` with bounded CI-friendly inputs

It does not try to wrap every CLI subcommand. For advanced workflows such as
`baseline promote`, `pool campaign`, or custom post-processing, keep using
normal shell steps around the CLI.

## Basic Pull Request Lane

```yaml
- uses: actions/checkout@v4
- name: Run CervoMutants
  id: cervomut
  uses: cervantesh/cervo-mutants@main
  with:
    policy: ci-fast
    budget: 5m
    report: summary,json,junit,github-summary
    out: .cervomut/pr
- name: Upload mutation artifacts
  if: always()
  uses: actions/upload-artifact@v4
  with:
    include-hidden-files: true
    name: cervomut-pr
    path: ${{ steps.cervomut.outputs.report-dir }}
```

## Coverage-Aware Service Lane

```yaml
- uses: actions/checkout@v4
- name: Run CervoMutants
  uses: cervantesh/cervo-mutants@main
  with:
    policy: ci-balanced
    budget: 10m
    coverage-prefilter: "true"
    report: summary,json,junit,github-summary
    out: .cervomut/pr
```

## Nightly Lane With HTML And SARIF

```yaml
- uses: actions/checkout@v4
- name: Run CervoMutants
  uses: cervantesh/cervo-mutants@main
  with:
    policy: nightly
    budget: 20m
    report: summary,json,junit,html,sarif,github-summary
    out: .cervomut/nightly
```

Optional SARIF upload step:

```yaml
- name: Upload SARIF
  if: always()
  uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: .cervomut/nightly/mutation-report.sarif
```

## Large-Repo Shard Lane

```yaml
strategy:
  matrix:
    shard: ["1/4", "2/4", "3/4", "4/4"]

steps:
  - uses: actions/checkout@v4
  - name: Run shard
    uses: cervantesh/cervo-mutants@main
    with:
      policy: ci-fast
      slice-by: package
      shard: ${{ matrix.shard }}
      max-files-per-run: "20"
      sample: deterministic
      out: .cervomut/pr-${{ matrix.shard }}
```

## Inputs

Common inputs:

- `go-version`
- `cervomut-version`
- `working-directory`
- `targets`
- `policy`
- `budget`
- `report`
- `out`
- `sample`
- `slice-by`
- `shard`
- `max-files-per-run`
- `max-mutants-per-package`
- `max-mutants`
- `workers`
- `coverage-prefilter`
- `actionable-only`

Leave `cervomut-version` blank for the default behavior: the action installs
the CLI from the same pinned ref as the action itself, or from the local action
source when you use a checkout path. Set `cervomut-version: latest` only as an
explicit override.

That resolution path is now hardened in two ways:

- ref normalization accepts standard GitHub tag refs such as
  `refs/tags/v0.3.0` and standard branch refs such as `refs/heads/main`
- blank-version installs prefer the checked-out local action source before
  falling back to a ref-derived `go install` target, which keeps slash-qualified
  branch names safe when the action is executed from its source checkout

The action also exposes `report-dir` as a resolved absolute output path so
follow-up artifact or SARIF steps can reference the chosen output directory
consistently even when `working-directory` is a subdirectory.

When that directory lives under `.cervomut/...`, set
`include-hidden-files: true` on `actions/upload-artifact@v4`; otherwise GitHub
skips hidden report folders and the upload completes without the expected JSON,
JUnit, HTML, or SARIF files.

## Validation Coverage

The repository now backs the action with both unit and workflow evidence:

- `cmd/actionhelper` tests cover version/ref resolution and absolute
  `report-dir` output behavior
- the main CI workflow runs the composite action through `uses: ./` to validate
  the local action-source install path and the emitted `report-dir` output

## Versioning Note

The examples currently use `@main` because this action lands after the existing
public tags. Once a newer release includes the action, pin production workflows
to that release tag instead of following `main`.
