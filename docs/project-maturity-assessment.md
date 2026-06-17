# Project Maturity Assessment

Tracking issue: https://github.com/cervantesh/cervo-mutants/issues/57

Assessment date: 2026-06-17

Follow-on roadmap: [docs/maturity-roadmap.md](maturity-roadmap.md)

This document records a dated maturity snapshot for CervoMutants across product,
code, and operations. It is intentionally time-bound: maturity can move quickly
as CI, releases, public adoption, and major features change.

## Scale

| Score | Meaning |
| --- | --- |
| `1` | concept or early prototype |
| `2` | working prototype with major gaps |
| `3` | usable beta |
| `4` | production-ready for broader external use |
| `5` | mature project with stable operations and adoption |

## Snapshot Summary

| Dimension | Score | Assessment |
| --- | ---: | --- |
| Functional / product maturity | `3.7 / 5` | strong beta |
| Code / engineering maturity | `3.3 / 5` | good internal structure with concentrated complexity |
| Operational maturity | `2.2 / 5` | weak for public confidence today |
| Overall maturity | `3.3 / 5` | usable beta, not yet broadly production-ready |

## Executive Summary

CervoMutants is no longer a prototype. It already has a meaningful product
surface:

- mutation generation and execution
- baseline and quarantine workflows
- JSON, JUnit, summary, and HTML reports
- survivor ranking and denominator-health reporting
- large-repo slicing and resume support
- external comparison normalization

The project is therefore functionally credible. The main weakness is not core
capability but operational confidence. As of 2026-06-17, local validation is
strong, but the public GitHub Actions `test` workflow is failing repeatedly
while `pages` is green. That gap is large enough to keep the project in beta.

## Evidence Snapshot

### Repository and release state

- Public GitHub repository created on 2026-06-17.
- Public releases present on 2026-06-17: `v0.1.0`, `v0.2.0`, `v0.3.0`.
- GitHub Pages is live at `https://cervantesh.github.io/cervo-mutants/`.
- Repository topics are present and the public project identity is coherent.

### Codebase shape

- `16` packages under `pkg/`
- `50` source files under `pkg/` and `cmd/`
- `22` test files under `pkg/` and `cmd/`
- about `12,449` lines of product code
- about `4,566` lines of test code
- about `4,121` lines of repo documentation in `docs/`, `README`, `site`, and workflows

### Local validation on 2026-06-17

- `go vet ./...` passed
- `go test ./...` passed
- fresh package coverage sample:
- `pkg/engine`: `88.7%`
  - `pkg/report`: `92.6%`
  - `pkg/runner`: `96.8%`
  - `pkg/mutator`: `90.5%`
  - `pkg/config`: `93.9%`
  - `pkg/pool`: `42.3%`

### Public CI state on 2026-06-17

- `.github/workflows/pages.yml`: green after merge
- `.github/workflows/test.yml`: repeated failures in GitHub Actions on `main`

## Functional Maturity

### Score: `3.7 / 5`

The product already solves a real problem, and the scope is clearer than most
young testing tools.

#### What is mature enough already

- The core value proposition is explicit in the public docs: mutation testing
  for CI, large Go modules, and AI-assisted test improvement.
- The CLI surface is broad enough to support real usage, not just demos:
  `run`, `fast`, `doctor`, `affected`, `eval`, `compare`, `baseline`, `report`,
  `show`, `explain`, `list-mutators`, and daemon/worker commands.
- Reporting is meaningfully structured. The tool distinguishes `survived`,
  `killed`, `timed_out`, `memory_killed`, `compile_error`, `not_covered`,
  `pending_budget`, `skipped_resource`, `quarantined`, and `cached`.
- The project has clear product doctrines instead of ad hoc features:
  baseline-first adoption, denominator health, history-aware signal, and
  apples-to-apples comparison rules.

#### Functional gaps keeping it in beta

- The daemon/worker surface exists, but the public positioning still describes
  it as contracts for future distributed execution rather than a mature
  distributed system.
- The higher-order semantic triage layer is not yet a first-class shipped
  feature. Survivor ranking exists, but semantic equivalence-risk grouping,
  actionable-only filtering, and richer triage heuristics are still gaps.
- Public adoption evidence is minimal. On 2026-06-17 the repository had no
  stars, forks, or watchers, so functional maturity is currently demonstrated
  mostly by internal implementation depth rather than market validation.

## Code And Engineering Maturity

### Score: `3.3 / 5`

The codebase is stronger than its public age suggests. It has credible domain
modeling, real tests, bounded dependencies, and a reasonable package split. The
main weakness is concentration of complexity in a few files.

### What is strong

- Domain modeling is explicit and useful. The result schema in
  `pkg/engine/types.go` captures statuses, scoring, denominator health,
  quarantine, history, slice metadata, and failure artifacts.
- The report layer is intentionally multi-surface rather than accidental:
  `Summary`, `Survivors`, `HTML`, and `JUnit` all exist in `pkg/report/report.go`.
- The engine is test-heavy. `pkg/engine/engine_test.go` is large and covers dry
  runs, reporting, checkpoint behavior, resume, parallelism, and structured
  failure scenarios.
- Dependencies are intentionally light. The module file depends almost entirely
  on the standard library plus `gopkg.in/yaml.v3`.
- Fresh local validation shows that most core packages have high statement
  coverage.

### What is weak

- Complexity is concentrated in a few files:
  - `pkg/engine/engine.go`: about `2,282` lines
  - `cmd/cervomut/main.go`: about `1,131` lines
- `pkg/pool` coverage is materially lower than the rest of the system at
  `42.3%`, even though it orchestrates comparison-study automation.
- There is evidence of repo-hygiene drift:
  - `cervomut.exe` is committed at repo root
  - the tracked `coverage.out` still reflects an old pre-rename module path
- The repository was rewritten into a single public snapshot earlier on
  2026-06-17, so the current public history is intentionally shallow. That is
  acceptable for publishing, but it lowers maintainability evidence for outside
  contributors reviewing project evolution.

## Operational Maturity

### Score: `2.2 / 5`

This is currently the weakest part of the project.

### What is working

- GitHub Pages is live and deploys successfully.
- The repository has a public license, trademark notice, releases, homepage,
  and topics.
- The repository follows issue-gated work and keeps change scope auditable.

### What is not mature enough

- The main public CI test workflow is not currently reliable in GitHub Actions.
  That makes the public repository look less trustworthy than the local state.
- Validation confidence is therefore asymmetric:
  - strong locally
  - weak in public automation
- Release cadence exists, but it is extremely compressed in time. Multiple
  public releases on the same day are useful for bootstrapping, but they are not
  yet evidence of stable release operations.

## Strengths

- Strong product thesis and differentiation
- Clear mutation-testing semantics instead of simple score chasing
- Good report model for CI and machine consumers
- High local test coverage across critical packages
- Real docs and evaluation artifacts, not just code
- Lightweight dependency surface

## Main Risks

- Public CI failure undermines confidence faster than documentation can repair it
- Central orchestration files are large enough to slow safe iteration
- Low coverage in comparison-pool code leaves one operational slice under-tested
- Repo artifacts such as committed binaries and stale coverage files weaken
  hygiene expectations
- Public adoption is still too early to validate external usability assumptions

## Priority Actions

### Immediate

1. Fix GitHub Actions `test.yml` until `main` is consistently green.
2. Remove committed generated artifacts from the repo and harden `.gitignore`.
3. Regenerate or stop tracking stale coverage artifacts after module-path changes.

### Near term

1. Split `pkg/engine/engine.go` into smaller orchestration units.
2. Split `cmd/cervomut/main.go` by command family.
3. Raise `pkg/pool` coverage materially.

### Product maturity next

1. Implement semantic triage features that reduce equivalent-mutant noise.
2. Add stronger actionable grouping and weighting to survivor review.
3. Turn daemon/worker from contract surface into a clearly supported execution mode,
   or narrow the public claim.

## Current Maturity Classification

As of 2026-06-17, CervoMutants should be described as:

> a usable beta mutation-testing toolkit with strong functional depth, good
> local code quality, and weak public operational maturity.

That is a respectable stage. The next transition is not "invent more features"
but "make the existing system externally trustworthy."
