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
https://github.com/cervantesh/cervo-mutants/issues/1

Current documentation refresh:  
https://github.com/cervantesh/cervo-mutants/issues/33

Project site:
https://cervantesh.github.io/cervo-mutants/

Contribution guide:
[CONTRIBUTING.md](CONTRIBUTING.md)

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
- Semantic triage v1 for non-progress loop timeouts, permission-mode platform sensitivity, and equivalence-risk survivor groups.
- Semantic actionability heuristics extracted into `pkg/triage` for shared use across engine, runner, and reporting.
- Engine orchestration split across dedicated files for execution, checkpointing, history, summary, slicing, and mutation scheduling.
- CLI orchestration split across command-family files for run/eval, compare/pool, baseline/report, and shared entrypoint dispatch.
- Experimental daemon/worker JSON-lines mode for internal protocol exploration.
- External-tool comparison normalization for apples-to-apples studies.
- Benchmark corpus harness for runtime, peak-memory, and mutation-throughput regression tracking.
- Pool campaign manifests that orchestrate smoke, compare, and benchmark jobs under one resumable summary.

## Install

From a Go workspace:

```powershell
go install github.com/cervantesh/cervo-mutants/cmd/cervomut@latest
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

On Windows, `temp-workdir` runs now harden themselves automatically:

- they cap effective workers to `2`;
- they inject conservative `go test` settings such as `GOFLAGS=-p=1`;
- they prefer a temp root outside risky OneDrive-backed `TEMP` locations when
  possible;
- they expose the selected temp root and warnings in report metadata.

Important files:

| File | Purpose |
| --- | --- |
| `.cervomut/reports/mutation-report.json` | Stable JSON report for CI and agents. |
| `.cervomut/reports/summary.txt` | Human-readable run summary. |
| `.cervomut/reports/junit.xml` | CI test report format. |
| `.cervomut/reports/index.html` | Filterable survivor review workbench with raw-report fallback, diff browsing, and client-side triage filters. |
| `.cervomut/reports/mutation-report.sarif` | GitHub code-scanning friendly mutation findings. |
| `.cervomut/reports/github-summary.md` | Compact GitHub step summary markdown for PR/Actions views, including owner routing when configured. |
| `.cervomut/reports/history-dashboard.json` | Structured multi-run history snapshot for raw score, actionable score, survivor aging, timeout movement, and operator yield. |
| `.cervomut/reports/history-dashboard.html` | Human-readable historical dashboard built from persisted run history. |
| `.cervomut/reports/test-recommendations.md` | Actionable next-test queue derived from nearby tests, coverage source, operator family, survivor history, and optional owner routing. |
| `.cervomut/reports/governance-review.md` | Human-readable suppression/quarantine review pack with generated templates and expiry guidance. |
| `.cervomut/reports/governance-review.json` | Auditable structured export for suppression/quarantine review workflows. |
| `.cervomut/reports/survivors-actionable.txt` | Optional actionable-only survivor review view. |
| `.cervomut/reports/semantic-triage-ledger.json` | Auditable skip/quarantine suggestions for known noisy patterns. |
| `.cervomut/reports/partial-mutation-report.json` | Checkpoint report for timeout/interrupted runs. |
| `.cervomut/reports/partial-summary.json` | Small checkpoint summary. |
| `.cervomut/history.json` | Historical survivor/cache signal. |
| `.cervomut/baseline.json` | Accepted baseline for regression gates. |
| `.cervomut/baseline.candidate.json` | Optional accepted baseline candidate waiting for promotion. |
| `.cervomut/quarantine.json` | Temporary audited quarantine entries. |

The public JSON, JUnit, and HTML report formats are treated as compatibility
surfaces. Golden fixtures in `pkg/report/testdata/` lock those outputs so
schema or rendering drift fails tests before release.

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

For a more review-oriented flow, accept a candidate and then promote it
intentionally:

```powershell
cervomut baseline diff
cervomut baseline accept
cervomut baseline promote
```

The default `ci.fail_under` is `0`. CI should usually fail on:

- baseline regression;
- new survivors when the baseline policy says so;
- expired quarantine entries;
- timeouts or compile errors if configured.

For GitHub Actions, the first-party integration path is
[docs/github-action.md](docs/github-action.md).

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
cervomut run ./... --policy ci-fast --report summary,json,junit,sarif,github-summary
```

More detail: [docs/policy-presets.md](docs/policy-presets.md).

## Reference Examples

Three maintained example workspaces now live under [examples/](examples/README.md):

- `small-library`: first adoption path for compact repos
- `medium-service`: richer review outputs for multi-package services
- `large-repo-ci`: slicing and matrix CI patterns for bounded large-repo rollout

The public guide is [docs/example-repos.md](docs/example-repos.md).
For repo fit, rollout guidance, limits, and tradeoffs, see
[docs/adoption-guide.md](docs/adoption-guide.md).

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
| `cervomut pool benchmark --corpus FILE` | Run the pinned benchmark corpus and fail on threshold regressions. |
| `cervomut pool campaign --file FILE` | Orchestrate multi-repo smoke, compare, and benchmark jobs from one manifest. |
| `cervomut baseline update` | Save current report as baseline. |
| `cervomut baseline compare` | Compare current report to baseline. |
| `cervomut baseline diff` | Show a review-oriented diff between the baseline and the current report or accepted candidate. |
| `cervomut baseline accept` | Save the current report as an accepted baseline candidate. |
| `cervomut baseline promote` | Promote the accepted candidate into the official baseline file. |
| `cervomut report summary --out DIR` | Print report summary. |
| `cervomut report survivors --out DIR` | Print ranked surviving mutants. |
| `cervomut report survivors --out DIR --actionable-only` | Print only the actionable survivor review set, with equivalent/platform-sensitive duplicates collapsed. |
| `cervomut report recommendations --out DIR` | Print the recommended next tests for actionable survivors. |
| `cervomut report governance --out DIR` | Print the generated suppression/quarantine review pack. |
| `cervomut report history --out DIR` | Print the historical trend summary derived from persisted run history. |
| `cervomut report sarif --out DIR` | Print GitHub code-scanning SARIF for the stored report. |
| `cervomut report github-summary --out DIR` | Print compact GitHub summary markdown for the stored report. |
| `cervomut report open` | Open the HTML survivor review workbench. |
| `cervomut show MUTANT_ID --out DIR` | Show a mutant diff/context. |
| `cervomut explain MUTANT_ID --format text\|json` | Explain what a survivor means. |
| `cervomut list-mutators` | List operators and risk metadata. |
| `cervomut daemon --experimental` | Experimental JSON-lines daemon mode. |
| `cervomut worker --experimental` | Experimental JSON-lines worker mode. |

`daemon` and `worker` are intentionally not treated as a stable compatibility
surface yet. They require explicit opt-in and are documented in
[docs/daemon-worker.md](docs/daemon-worker.md).

Common run flags:

```powershell
cervomut run ./... `
  --policy ci-fast `
  --budget 10m `
  --actionable-only `
  --max-mutants 100 `
  --slice-by package `
  --shard 1/4 `
  --max-files-per-run 20 `
  --max-mutants-per-package 10 `
  --sample deterministic `
  --workers 2 `
  --isolation overlay `
  --temp-root C:\Users\me\AppData\Local\CervoMutants\tmp `
  --test-timeout 20s `
  --coverage-prefilter `
  --resume `
  --max-process-memory-mb 6144 `
  --out .cervomut/reports
```

`--actionable-only` is a review view, not a suppression mode. Raw JSON, HTML,
and the normal `survivors.txt` remain complete; the actionable projection is
printed to stdout for `run`, available in `report survivors --actionable-only`,
and written to `.cervomut/reports/survivors-actionable.txt`.

`baseline update` remains the direct path for replacing the baseline in one
step. `baseline accept` plus `baseline promote` is the safer review flow when
you want a candidate file and an explicit promotion action.

`mutation-report.json` also carries an additive `summary.actionable` block so
consumers can read actionable score and triage-weighted survivor counts without
changing the meaning of the existing raw score fields.

When `sarif` is included in `reports.formats` or `--report`, CervoMutants writes
`mutation-report.sarif` for GitHub code scanning. When `github-summary` is
included, it writes `github-summary.md` and, if `GITHUB_STEP_SUMMARY` is set,
also publishes the same markdown directly into the GitHub Actions step summary.

`semantic-triage-ledger.json` is a companion artifact. It groups equivalent-risk
survivors, flags Windows-only permission-mode noise, and suggests quarantine
review for confirmed non-progress loop timeouts without mutating the raw report.

## Configuration

`cervomut init` writes `cervomut.yaml`. The defaults are conservative:

```yaml
version: 1
policy: ""
scope:
  mode: all
  since: origin/main
  include: ["./..."]
  slice_by: ""
  shard_index: 0
  shard_count: 0
tests:
  command: ["go", "test", "./..."]
  timeout: 30s
  baseline_required: true
mutators:
  profile: conservative
execution:
  workers: 4
  isolation: temp-workdir
  temp_root: ""
selection:
  mode: package
limits:
  max_mutants: 0
  max_mutants_per_package: 0
  max_files_per_run: 0
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
  actionable_only: false
ownership:
  default:
    owner: ""
    team: ""
    contact: ""
  rules: []
```

Supported selection modes:

- `all`: run the configured test command.
- `package`: rewrite `go test ./...` to the mutant package when possible.
- `coverage`: use a baseline Go coverage profile to avoid irrelevant tests and
  classify obvious `not_covered` mutants.

Supported isolation modes:

- `temp-workdir`: copy the module to a marked temporary workdir and mutate the
  copy. On Windows this mode uses a conservative worker cap and prefers a safe
  local temp root when the system `TEMP` is risky.
- `overlay`: use Go's `-overlay` support to avoid copying the module.

Optional ownership routing lets you map packages or files to the team that
should review actionable survivors first:

```yaml
ownership:
  default:
    team: platform
  rules:
    - name: cli
      file: cmd/**/*.go
      owner: cli-owner
      team: developer-experience
      contact: "@dx"
    - name: fs
      package: ./pkg/fs
      owner: fs-owner
      team: storage
```

When configured, ownership metadata is copied into each mutant and surfaced in
JSON, `report survivors`, the HTML workbench, test recommendations,
governance-review templates, SARIF properties, and GitHub step summaries.

## Large-Repo Slicing

Large repositories can cut one broad run into smaller, resumable slices without
hiding the raw report model.

Available controls:

- `--slice-by mutant|package|file|function|operator` to choose the deterministic
  shard grouping key
- `--shard INDEX/COUNT`
- `--max-files-per-run N`
- `--max-mutants-per-package N`

Examples:

```powershell
# PR lane: one package shard with a hard file cap
cervomut run ./... --policy ci-fast --slice-by package --shard 1/4 --max-files-per-run 20 --sample deterministic

# Nightly lane: broader shard with capped per-package density
cervomut run ./... --policy nightly --slice-by file --shard 3/12 --max-mutants-per-package 25 --sample deterministic
```

Reports keep the applied slice metadata under the top-level `slice` block so
later tooling can merge shard outputs without guessing how the run was cut.

Windows note:

- use `cervomut doctor` before large native runs;
- override the temp root with `execution.temp_root` or `--temp-root` when
  `TEMP` is inside OneDrive or the repo tree;
- prefer `overlay` for broader CI and `temp-workdir` for deeper local/manual
  campaigns.

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
- semantic-group penalties for repeated review-once families;
- non-progress loop risk classification for timeout-prone loop mutations;
- GOOS-aware platform sensitivity for permission-mode mutants;
- recommendation tier;
- nearby tests;
- exported function context;
- coverage source;
- history status;
- operator historical yield;
- suppression audit hits.

Actionable survivors also carry a structured `test_recommendation` block. v1
uses:

- nearby test files;
- coverage source tightness vs fallback/package-level selection;
- operator family;
- survivor history and age;
- target-OS sensitivity where relevant.

The goal is to answer "which test should I touch first and what kind of
assertion is missing?" without generating tests automatically.

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

`governance-review.md` and `governance-review.json` are the review companions
for this policy layer. They generate:

- quarantine candidate templates with suggested expiry windows and policy
  reminders;
- suppression rule templates with evidence and reviewer metadata;
- auditable exports suitable for issue comments, follow-up review, or external
  tooling.

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
- workflow study artifacts for review:
  `comparison-study.json` and `comparison-summary.md`.

Important rule:

> Do not compare `cervomut run ./...` with `gremlins unleash .` as if they were
> equivalent unless the harness records target normalization explicitly.

`cervomut pool compare` now writes three layers on purpose:

- `summary.json`: raw normalized row data for automation.
- `comparison-study.json`: repo-grouped study view with comparability labels.
- `comparison-summary.md`: quick human review summary for issue comments, PRs,
  and study notes.

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

Advanced integrators can extend the library without forking the core by
swapping or composing the built-in generator, suppression evaluator, and
survivor ranker:

```go
baseRanker := engine.DefaultSurvivorRanker()

custom := engine.NewWithOptions(
    cfg,
    engine.WithMutantGenerator(mutator.ChainGenerators(
        mutator.DefaultGenerator(),
        myGenerator,
    )),
    engine.WithSuppressionEvaluator(engine.ChainSuppressionEvaluators(
        engine.DefaultSuppressionEvaluator(cfg),
        mySuppressionEvaluator,
    )),
    engine.WithSurvivorRanker(engine.SurvivorRankerFunc(func(goos string, results []engine.MutantResult) []engine.SurvivorRanking {
        ranked := baseRanker.Rank(goos, results)
        return ranked
    })),
)

result, err := custom.Run(ctx, engine.RunRequest{Targets: []string{"./..."}})
```

That seam is intentionally programmatic and additive. The CLI behavior stays
unchanged unless you build a custom binary around the library API.

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
| `pkg/daemon` | Experimental JSON-lines worker/daemon protocol code. |

## Development

```powershell
go test ./...
go test ./... -coverprofile=coverage.out
go vet ./...
staticcheck ./...
go run ./cmd/cervomut list-mutators
```

Focused fuzz/property targets now cover inline-ignore parsing, external report
normalization, and `GOFLAGS` normalization. Example entry points:

```powershell
go test ./pkg/mutator -run Test -fuzz=FuzzParseInlineIgnore -fuzztime=10s
go test ./pkg/extcompare -run Test -fuzz=FuzzNormalizeTarget -fuzztime=10s
go test ./pkg/engine -run Test -fuzz=FuzzNormalizeGoFlags -fuzztime=10s
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
- [docs/project-maturity-assessment.md](docs/project-maturity-assessment.md):
  dated assessment of product, code, and operational maturity.
- [docs/maturity-roadmap.md](docs/maturity-roadmap.md): 18-month execution
  roadmap for raising functional and code maturity.
- [docs/policy-presets.md](docs/policy-presets.md): policy preset behavior.
- [docs/sonar.md](docs/sonar.md): local and CI Sonar workflow.
- [docs/go-toolchain-compatibility.md](docs/go-toolchain-compatibility.md):
  official OS/Go support matrix and `doctor` checks.
- [docs/compatibility-policy.md](docs/compatibility-policy.md): formal
  compatibility contract for CLI, report schema, and experimental protocol
  surfaces.
- [docs/adr/README.md](docs/adr/README.md): architecture decision records for
  durable product and system choices.
- [docs/releasing.md](docs/releasing.md): tag-driven release workflow, assets,
  checksums, and upgrade-note requirements.
- [docs/contributing-technical.md](docs/contributing-technical.md): package
  boundaries, validation expectations, and safe change patterns for
  contributors.
- [docs/example-repos.md](docs/example-repos.md): maintained example workspaces
  for small, medium, and large-repo adoption patterns.
- [docs/adoption-guide.md](docs/adoption-guide.md): public fit, rollout,
  limitation, and tradeoff guidance grounded in current validation evidence.
- [docs/github-review-workflow.md](docs/github-review-workflow.md): the default
  PR and CI review model using the maintained GitHub workflow examples.
- [docs/history-data.md](docs/history-data.md): storage, export, retention, and
  audit contract for persisted mutation history and history dashboard outputs.
- [docs/github-action.md](docs/github-action.md): first-party GitHub Action for
  bounded `cervomut run` CI lanes.
- [docs/extensibility.md](docs/extensibility.md): programmatic extension seams
  for custom mutators, suppression evaluators, and survivor rankers.
- [docs/daemon-worker.md](docs/daemon-worker.md): experimental daemon/worker
  mode, current guarantees, and explicit non-guarantees.
- [docs/evaluations/multi-repo-calibration.md](docs/evaluations/multi-repo-calibration.md):
  multi-repo calibration plan.
- [docs/evaluations/tool-comparison-protocol.md](docs/evaluations/tool-comparison-protocol.md):
  apples-to-apples comparison rules.
- [docs/evaluations/comparison-harness.md](docs/evaluations/comparison-harness.md):
  runnable harness description.
- [docs/evaluations/benchmark-corpus.md](docs/evaluations/benchmark-corpus.md):
  pinned benchmark corpus and regression-threshold harness.
- [docs/evaluations/pool-campaigns.md](docs/evaluations/pool-campaigns.md):
  campaign manifests for orchestrating pool smoke, compare, and benchmark jobs.
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

