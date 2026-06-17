# 18-Month Maturity Roadmap

Tracking issue: #63

This roadmap turns the dated maturity snapshot into an executable backlog.
It focuses on raising CervoMutants functional and code maturity while only
including operational work that directly unlocks those two dimensions.

## Starting Point

- Public CI on `main` is green.
- The product is already a strong beta for local and CI mutation testing.
- The next transition is not more surface area for its own sake; it is better
  signal quality, less orchestration complexity, stronger public contracts, and
  credible external adoption.

## Score Targets

| Phase | Timeline | Functional target | Code target | Tracker |
| --- | --- | ---: | ---: | --- |
| Phase 1 | 0-3 months | `3.9 / 5` | `3.6 / 5` | #64 |
| Phase 2 | 3-6 months | `4.1 / 5` | `3.8 / 5` | #65 |
| Phase 3 | 6-12 months | `4.3 / 5` | `4.0 / 5` | #66 |
| Phase 4 | 12-18 months | `4.5 / 5` | `4.2+ / 5` | #67 |

Guiding order:

1. Improve signal quality and review UX.
2. Reduce complexity concentration.
3. Harden public contracts and extensibility.
4. Scale adoption, comparison workflows, and distributed execution only after
   the earlier layers are stable.

## Phase 1: Signal Quality And Complexity Reduction

Milestone: `Phase 1 - Signal and Complexity`

- #68 Implement semantic triage v1 with non-progress loops, GOOS-aware ranking,
  and equivalence grouping.
- #69 Add actionable-only survivor views to `report` and `run`.
- #70 Generate a semantic triage ledger with reviewed-skip and quarantine
  suggestions.
- #71 Extract triage heuristics into a dedicated `pkg/triage` package.
- #72 Split `pkg/engine` orchestration into smaller units.
- #73 Split `cmd/cervomut` CLI orchestration by command family.
- #74 Build a reusable fixture harness and raise coverage in weak subsystems.

Phase 1 exit gates:

- `pkg/engine >= 90%`
- `pkg/mutator >= 93%`
- `pkg/pool >= 80%`
- `main` stays green in CI
- public docs reflect shipped behavior

## Phase 2: Public UX And Stable Contracts

Milestone: `Phase 2 - Public UX and Stable Contracts`

- #75 Turn the HTML report into a filterable survivor review workbench.
- #76 Freeze report contracts with golden tests for JSON, JUnit, and HTML.
- #77 Add an `actionable` block to the report schema without replacing raw
  score.
- #78 Expand baseline workflows with `diff`, `accept`, and `promote` actions.
- #79 Add GitHub-native output such as SARIF or compact check summaries.
- #80 Create a representative benchmark corpus and regression thresholds.
- #81 Add fuzz and property tests for parsers, fingerprints, and normalization.

Phase 2 exit gates:

- all public outputs have golden or compatibility fixtures
- report schema remains additive
- benchmark regressions are visible before release

## Phase 3: Adoption And Extensibility

Milestone: `Phase 3 - Adoption and Extensibility`

- #82 Publish and maintain three example repos for small, medium, and
  large-repo adoption.
- #83 Promote the comparison harness into a first-class product workflow.
- #84 Deliver a v1 test recommendation engine using coverage, operator, and
  survivor history.
- #85 Improve suppression and quarantine UX with generated templates and audit
  export.
- #86 Add ownership routing for packages and survivors.
- #87 Expose stable internal extension points for mutators, suppression, and
  ranking.
- #88 Add historical dashboards for raw score, actionable score, and survivor
  aging.

Phase 3 exit gates:

- at least one real adoption flow is documented end to end
- recommendation and governance UX are usable without internal context
- extension points are explicit enough to evolve without large forks

## Phase 4: Platform And External Scale

Milestone: `Phase 4 - Platform and Scale`

- #89 Harden `pkg/pool` into a multi-repo orchestration layer for campaigns and
  benchmarks.
- #90 Decide whether daemon/worker becomes supported or stays explicitly
  experimental.
- #91 Publish an official Go-version and OS compatibility matrix with automated
  validation.
- #92 Automate releases with changelog, verified artifacts, and upgrade notes.
- #93 Define formal compatibility policy for CLI, report schema, and daemon
  protocol.
- #94 Publish ADRs and technical contribution guides.
- #95 Run an external validation wave with 3-5 public or semi-public adopters.

Phase 4 exit gates:

- compatibility policy is public and enforced
- release automation is reproducible
- daemon/worker status is explicit, not implied
- external validation informs the next maturity assessment

## Public Interface Evolution

Expected additions over the roadmap:

- Reports:
  - `semantic_tags`
  - `semantic_group`
  - `group_label`
  - `actionable_score`
  - `suggested_skip_reason`
  - an additive `actionable` summary block
- CLI:
  - `--actionable-only`
  - richer baseline review commands
  - GitHub-native outputs such as SARIF or check summaries
- Internal architecture:
  - a dedicated `pkg/triage`
  - smaller orchestration units in engine and CLI
- Protocol:
  - daemon/worker must be explicitly versioned before it is treated as a
    supported surface

## Execution Rules

- Each roadmap item must remain tied to its own issue or epic before
  implementation starts.
- No phase is complete if `main` is red or public docs are stale.
- Schema evolution stays additive unless there is a deliberate `v2` decision.
- Distributed execution is downstream of signal quality, review UX, and stable
  contracts.
