# GitHub Action

Tracking issue: #159

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
  uses: cervantesh/cervo-mutants@main
  with:
    policy: ci-fast
    budget: 5m
    report: summary,json,junit,github-summary
    out: .cervomut/pr
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

The action also exposes `report-dir` as an output so follow-up artifact steps
can reference the chosen output path consistently.

## Versioning Note

The examples currently use `@main` because this action lands after the existing
public tags. Once a newer release includes the action, pin production workflows
to that release tag instead of following `main`.
