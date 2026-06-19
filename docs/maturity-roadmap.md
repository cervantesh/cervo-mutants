# 24-Month Architecture-Led Maturity Roadmap

Tracking issue: #179

This roadmap replaces the earlier 18-month roadmap as the planning source of
truth. It expands the maturity horizon to 24 months while preserving the
already-seeded Phase 1-4 issue backbone as historical groundwork rather than
reopening it.

## Starting Point

- Public CI on `main` is green.
- Public releases, GitHub Pages, example repos, compatibility docs, and the
  first-party GitHub Action are already part of the published surface.
- The next transition is not more surface area for its own sake; it is better
  signal quality, less orchestration complexity, stronger public contracts,
  more trustworthy adoption flows, and clearer ecosystem boundaries.

## Governing Order

1. Improve signal quality and review actionability.
2. Reduce orchestration complexity and internal concentration.
3. Harden public contracts and extension seams.
4. Make adoption, operations, and multi-repo workflows trustworthy.
5. Scale ecosystem reach only after the core product is externally dependable.

## Maturity Targets

| Horizon | Functional | Code | Operational | Overall | Tracker |
| --- | ---: | ---: | ---: | ---: | --- |
| `0-4` months | `4.0 / 5` | `3.6 / 5` | `3.0 / 5` | `3.6 / 5` | historical backbone: #64 |
| `4-8` months | `4.2 / 5` | `3.9 / 5` | `3.4 / 5` | `3.9 / 5` | historical backbone: #65 |
| `8-12` months | `4.3 / 5` | `4.0 / 5` | `3.7 / 5` | `4.0 / 5` | historical backbone: #66 |
| `12-16` months | `4.4 / 5` | `4.1 / 5` | `4.0 / 5` | `4.1 / 5` | historical backbone: #67 |
| `16-20` months | `4.5 / 5` | `4.2 / 5` | `4.2 / 5` | `4.3 / 5` | #180 |
| `20-24` months | `4.7 / 5` | `4.4 / 5` | `4.4 / 5` | `4.5 / 5` | #181 |

## Foundation Carried Forward

The earlier roadmap tracker #63 and its phase epics #64-#67 are closed and now
serve as the historical backbone for the first four phases of this maturity
path. The v2 roadmap keeps those phases visible for continuity, but does not
reseed them as new tracker work.

### Phase 1: Signal Core And Complexity Reduction

Historical epic: #64  
Historical initiative set: #68, #69, #70, #71, #72, #73, #74

- ship semantic triage v1 as a first-class product surface
- deliver `non-progress loop` timeout classification, GOOS-aware survivor
  ranking, and equivalence-risk grouping
- add `--actionable-only` survivor review and reporting flows without hiding
  raw data
- generate a semantic triage ledger for reviewed-skip, quarantine, and
  repeated-review savings
- extract triage into a dedicated subsystem instead of growing heuristics inside
  orchestration
- split engine and CLI orchestration into smaller responsibility layers
- raise weak-subsystem coverage, especially orchestration and pool/campaign
  logic

Phase 1 architecture outcome:

- `pkg/triage` is the semantic decision layer
- `pkg/engine` becomes orchestration glue rather than a heuristic sink
- survivor output distinguishes raw result from actionable interpretation

Phase 1 exit gates:

- `pkg/engine >= 90%`
- `pkg/mutator >= 93%`
- `pkg/pool >= 80%`
- no single core orchestration file remains a practical review bottleneck
- public docs and example flows reflect shipped triage behavior

### Phase 2: Report Platform And Stable Output Contracts

Historical epic: #65  
Historical initiative set: #75, #76, #77, #78, #79, #80, #81

- turn the HTML report into a review workbench with filters for actionability,
  operator, semantic group, owner, and history state
- freeze JSON, JUnit, HTML, and summary outputs with golden fixtures and
  compatibility tests
- add an additive `actionable` report block without weakening the raw result
  model
- expand baseline workflows into a complete lifecycle: `diff`, `accept`, and
  `promote`
- add GitHub-native output surfaces such as SARIF and compact PR/check
  summaries
- create a representative benchmark corpus with pinned runtime and memory
  expectations
- add fuzz and property tests around parsing, fingerprinting, normalization,
  and schema stability

Phase 2 architecture outcome:

- report generation is treated as a versioned public platform
- baseline and actionability workflows are explicit product surfaces with
  compatibility expectations

Phase 2 exit gates:

- all public outputs have compatibility fixtures
- report schema remains additive
- performance regressions are visible before release
- GitHub CI can consume outputs without custom internal glue

### Phase 3: Guidance, Governance, And Review Productivity

Historical epic: #66  
Historical initiative set: #82, #83, #84, #85, #86, #87, #88

- deliver a test recommendation engine v1 using nearby tests, coverage gaps,
  operator type, and survivor history
- improve suppression and quarantine UX with generated templates, reviewer
  metadata, expiration, and audit export
- add ownership routing so survivors and reports can map to package or team
  ownership
- promote history and trend reporting into an explicit dashboard or export path
- strengthen branch, release, and time-window comparison flows as supported
  usage patterns
- add governance around repeated semantic groups so review-once behavior becomes
  durable

Phase 3 architecture outcome:

- review work shifts from raw survivor inspection to governed mutation review
- recommendation, ownership, and history become connected subsystems rather
  than separate reports

Phase 3 exit gates:

- at least one real adoption path is documented end to end
- suppression and quarantine flows are auditable without internal context
- actionable review throughput is measurably better than raw survivor review

### Phase 4: Extensibility And Multi-Repo Orchestration

Historical epic: #67  
Historical initiative set: #89, #90, #91, #92, #93, #94, #95

- promote the comparison harness into a first-class product workflow
- harden `pkg/pool` into a reliable campaign layer for comparison studies,
  benchmark runs, and adoption waves
- expose stable extension seams for custom mutators, suppressors, rankers, and
  external report adapters
- define versioned extension contracts so integrations do not depend on
  unstable internals
- add deterministic campaign manifests, resume safety, and artifact lineage
  expectations across pool runs
- separate supported extension APIs from non-contract internal helpers

Phase 4 architecture outcome:

- CervoMutants is extensible on purpose
- multi-repo and comparative execution is reproducible rather than
  clever-but-fragile

Phase 4 exit gates:

- extension points are documented, tested, and evolution-safe
- campaign resume and reuse behavior is deterministic
- comparison workflows are auditable enough to cite in evaluations and releases

## Phase 5: Supported Platform Surfaces And Operational Confidence

Milestone: `Phase 5 - Supported Platform and Operational Confidence`  
Epic: #180  
Initiatives: #182, #183, #184, #185, #186, #187, #188

- mature the GitHub Action into a stable integration surface with compatibility
  tests across tags, branches, and local-source install paths
- enforce the Go and OS compatibility matrix in CI and release policy, not only
  in docs and `doctor`
- automate releases with verified artifacts, checksums, changelog discipline,
  upgrade notes, and rollback guidance
- add upgrade compatibility tests from the previous supported release line into
  current `main`
- make install, release, and report compatibility part of the release gate
- decide whether daemon/worker advances to protocol v1 or remains experimental
- if daemon/worker advances, require protocol versioning, leases, retries,
  durable coordination, and compatibility coverage before public support claims

Phase 5 architecture outcome:

- operational trust becomes systemic and repeatable
- compatibility policy stops being descriptive only and becomes enforced
  behavior

Phase 5 exit gates:

- supported install paths are continuously validated
- release automation is reproducible
- compatibility guarantees are tested, documented, and tied to versioning rules
- daemon/worker status is explicit and defensible

## Phase 6: Ecosystem Reach, External Validation, And Commercial Readiness

Milestone: `Phase 6 - Ecosystem Reach and External Validation`  
Epic: #181  
Initiatives: #189, #190, #191, #192, #193, #194

- run structured external validation with multiple public or semi-public repos
  representing different sizes and CI shapes
- add adoption analytics and durable feedback intake to capture onboarding
  friction, noisy operators, and policy pain
- publish reference playbooks for small library, medium service, and large
  monorepo rollout strategies
- build a maintainers' operations pack for external adopters: upgrade
  checklist, support matrix, known limits, and issue templates tied to report
  artifacts
- evaluate whether a separate optional hosted or reporting layer is justified
  by real adoption, but keep it outside the OSS core contract unless demand is
  proven
- if commercialization is explored, keep it adjacent to the OSS core: support,
  enterprise policies, dashboards, campaign hosting, or governance packs rather
  than a forked mutation engine surface

Phase 6 architecture outcome:

- the project graduates from strong beta to an externally adoptable platform
  with clear support boundaries
- any future commercial layer remains downstream of the stable open product

Phase 6 exit gates:

- external validation findings materially influence the next maturity
  assessment
- at least three distinct adoption profiles have durable playbooks and evidence
- the OSS core remains coherent even if a commercial adjunct track is later
  added

## Public Interface Evolution

Expected public-surface evolution over the full roadmap:

- Reports:
  - `semantic_tags`
  - `semantic_group`
  - `group_label`
  - `group_reason`
  - `actionable_score`
  - `suggested_skip_reason`
  - an additive `actionable` summary block
- CLI:
  - `--actionable-only`
  - richer baseline lifecycle commands
  - GitHub-native output modes
  - explicit history and dashboard export flows
- GitHub Action:
  - blank-version installs prefer the action source over ambiguous ref strings
  - branch-safe and tag-safe install behavior remains part of the supported
    contract
- Internal architecture:
  - a dedicated `pkg/triage`
  - smaller orchestration units in engine and CLI
  - stable extension points for mutators, suppression evaluators, rankers, and
    normalization hooks
- Protocol:
  - daemon/worker remains outside supported compatibility until a versioned
    protocol contract exists

Schema evolution stays additive unless there is a deliberate `v2` decision.

## Test And Acceptance Plan

Core scenarios that must stay covered across the roadmap:

- semantic triage heuristics correctly classify non-progress loops,
  platform-sensitive survivors, and equivalence-risk groups
- survivor ranking preserves raw evidence while adding actionable
  interpretation
- JSON, JUnit, summary, and HTML outputs remain backward-compatible within the
  documented policy
- GitHub Action installation works for tags, commit SHAs, and slash-qualified
  branch refs when `cervomut-version` is blank
- baseline `diff`, `accept`, and `promote` flows produce auditable and
  deterministic state transitions
- benchmark corpus catches runtime and memory regressions before release
- example repos stay green and useful as adoption references
- multi-repo campaigns preserve deterministic targeting, resume safety, and
  artifact lineage
- extension seams reject unsupported shapes cleanly and remain stable across
  minor releases
- upgrade tests validate that the previous supported release can hand data and
  workflows forward without silent breakage

## Execution Rules

- Each roadmap item must remain tied to its own issue or epic before
  implementation starts.
- No phase is complete if `main` is red or public docs are stale.
- Compatibility claims must not outrun the evidence currently enforced in CI
  and release gates.
- Distributed execution is downstream of signal quality, review UX, stable
  contracts, and explicit support decisions.
- Hosted or commercial adjacencies must remain optional and downstream of the
  OSS core.
