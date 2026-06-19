# Project Maturity Assessment

Tracking issues: #208, #234

Assessment date: 2026-06-19

Follow-on roadmap: [docs/maturity-roadmap.md](maturity-roadmap.md)

This document records a dated maturity snapshot for CervoMutants across
product, code, and operations. It is intentionally time-bound: maturity can
move quickly as CI, releases, public adoption, and major features change.

This snapshot evaluates the current `main` branch state on 2026-06-19. The
latest public release is now `v0.4.2` from 2026-06-19, so the earlier
release-head lag plus the smaller hosted follow-ups are now also packaged in
the public release surface. The current `main` branch and latest public tag
are now effectively aligned for the supported install, CI, and reporting
surfaces described in this assessment.

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
| Functional / product maturity | `4.0 / 5` | strong public beta |
| Code / engineering maturity | `3.8 / 5` | well-structured with a few remaining hotspots |
| Operational maturity | `3.7 / 5` | credible public automation and aligned release surface, still young release practice |
| Overall maturity | `3.9 / 5` | credible public beta approaching broader external use |

## Executive Summary

CervoMutants is no longer just a usable beta with promising local validation.
On `main`, it now has:

- green public CI
- a first-party GitHub Action
- semantic triage and actionable-only review flows
- additive compatibility policy and report-schema discipline
- release automation with install, archive, and upgrade verification
- bounded external validation with committed artifacts and rollout guidance

The biggest remaining gap is no longer broken public automation, and it is no
longer a major lag between repository head and latest public release. The
bigger remaining gaps are still direct external maintainer adoption evidence,
release repetition over time, and deeper proof that the newer review surfaces
stay useful outside bounded validation waves.

That keeps the project just short of a clean `4 / 5` overall maturity
classification. The current state is best described as a credible public beta
whose latest public release now represents most of the hardened repository
surface.

## Evidence Snapshot

### Repository and release state

- Public GitHub repository created on 2026-06-17.
- Public releases present on 2026-06-19: `v0.1.0`, `v0.2.0`, `v0.3.0`,
  `v0.4.0`, `v0.4.1`, `v0.4.2`.
- Latest public release is `v0.4.2` from 2026-06-19 with multi-OS assets,
  `release-manifest.json`, and `SHA256SUMS`.
- GitHub Pages is live at `https://cervantesh.github.io/cervo-mutants/`.
- The seeded Phase 5 and Phase 6 roadmap backlog is closed.
- The immediate post-roadmap follow-up trio `#206`, `#207`, and `#208` is
  closed.

### Codebase shape

- `17` packages under `pkg/`
- `60` product source files under `pkg/` and `cmd/`
- `39` test files under `pkg/` and `cmd/`
- about `14,922` lines of product code
- about `9,654` lines of test code
- about `10,416` lines of repo documentation in `README`, `docs/`, `site/`, and
  workflow files

### Local validation on 2026-06-19

- `go vet ./...` passed
- `go test ./...` passed
- fresh package coverage sample:
  - `pkg/engine`: `90.0%`
  - `pkg/report`: `93.9%`
  - `pkg/runner`: `97.3%`
  - `pkg/mutator`: `93.6%`
  - `pkg/config`: `94.3%`
  - `pkg/pool`: `84.2%`

### Public automation state on 2026-06-19

- latest `test` workflow on `main`: green
- latest `workflow-lint` workflow on `main`: green
- latest `pages` workflow on `main`: green
- support matrix is documented for Linux, Windows, and macOS on Go `1.25.x`
- current release automation verifies install, archive, upgrade, and report
  compatibility before publication

### Repo-head versus release-head

The latest release is now effectively aligned with repository-head maturity for
the supported public surfaces.

`v0.4.2` now packages:

- Windows-native execution hardening
- deterministic large-repo slicing
- survivor-ranking calibration improvements
- semantic triage, actionable-only review, and additive actionable reporting
- first-party GitHub Action support
- released GitHub Action alignment on `actions/setup-go@v6`
- hosted wave result and summary generation through tested Go helpers
- repaired helper working-directory behavior for hosted external waves
- CI-enforced compatibility matrix behavior
- previous-release upgrade smoke
- install and archive verification gates
- deterministic Windows validation for serial-runner budget coverage and
  pre-canceled pool commands
- released follow-up guidance for adoption review-unit counts and governance
  status counts
- expanded adoption guidance, operations guidance, and external-validation
  framing

There is no material supported-surface gap called out in this snapshot between
the latest release and `main`. The remaining maturity limits are now about
release repetition, external adoption depth, and continued field calibration,
not about a missing public catch-up release.

## Functional Maturity

### Score: `4.0 / 5`

The product now looks like a strong public beta with a coherent workflow, not
just a capable engine surrounded by rough edges.

#### What is mature enough already

- The CLI surface is broad and purposeful: `run`, `fast`, `doctor`,
  `affected`, `eval`, `compare`, `pool`, `baseline`, `report`, `show`,
  `explain`, `list-mutators`, plus explicitly experimental daemon/worker
  commands.
- Semantic triage is now a first-class public surface. The repository exposes
  `pkg/triage`, `actionable_score`, semantic grouping, actionable-only survivor
  views, and triage-aware report fields.
- Reporting is clearly multi-surface and CI-aware:
  JSON, summary, HTML, JUnit, SARIF, GitHub summary, survivor review views, and
  history dashboards are all documented and exercised.
- The project has a coherent product doctrine:
  baseline-first rollout, denominator health, actionability over raw survivor
  volume, and apples-to-apples comparison semantics.
- The repository now includes a first-party GitHub Action and maintained
  rollout guidance for small libraries, medium services, and large repos.
- External validation is no longer only internal theory. A committed bounded
  validation wave succeeded across five public Go repositories.

#### Functional gaps keeping it in beta

- Daemon/worker remains correctly experimental. It still lacks a versioned
  protocol, durable coordination, and retry semantics required for supported
  distributed execution.
- Public adoption evidence is real but still bounded. The repo proves first-run
  viability and rollout guidance, but it does not yet prove broad maintainer
  adoption or long-running production usage across multiple external teams.
- Recommendation and triage quality still need more field calibration from real
  adoption feedback, not only fixtures and internal review samples.
- The newest hosted and guidance refinements are now represented in the public
  release line, but they are still additive polish rather than a fundamental
  maturity jump by themselves.

## Code And Engineering Maturity

### Score: `3.8 / 5`

The engineering story is materially stronger than the previous assessment. The
repository now shows better separation of concerns, stronger coverage in weak
areas, and much less concentrated orchestration risk.

### What is strong

- Complexity concentration has dropped sharply:
  - `pkg/engine/engine.go`: about `894` lines
  - `cmd/cervomut/main.go`: about `89` lines
- Semantic actionability logic is extracted into `pkg/triage` instead of being
  buried inside orchestration code.
- Coverage in previously weak areas is materially better, especially
  `pkg/pool` at `84.2%`.
- The engine, report, runner, mutator, and config packages all sit at high
  statement coverage for a tool of this size.
- The report model remains one of the strongest parts of the system:
  raw score, actionable score, governance, history, recommendations, and
  semantic metadata are all represented explicitly in typed surfaces.
- Dependencies remain intentionally light and mostly standard-library based.
- The repo-hygiene drift called out in the previous assessment is no longer
  present in tracked files: `cervomut.exe` and `coverage.out` are not tracked.

### What is still weak

- `pkg/engine` is much smaller than before, but it is still one of the more
  complex parts of the system and remains a likely hotspot for future changes.
- Some maintainability evidence is still young in public:
  the repo history was squashed into a public snapshot, so outside contributors
  still have limited long-form history to study.
- The daemon/worker path is intentionally experimental, which is the correct
  product claim, but it also means one code surface stays outside the normal
  supported-contract expectations.
- The project still needs more evidence of outside contribution and maintenance
  over time, not just strong local development discipline.

## Operational Maturity

### Score: `3.7 / 5`

Operational maturity is no longer the obvious weak point. The project now has
credible public automation and compatibility discipline, but it is still early
in release repetition and external adoption proof.

### What is working

- Public GitHub Actions are green on `main` for `test`, `workflow-lint`, and
  `pages`.
- The compatibility matrix is not just documented; it is tied to automated
  validation and release expectations.
- The release workflow now verifies assets, checksums, install paths, upgrade
  handoff, and report compatibility before publication.
- GitHub Pages, release docs, upgrade notes, compatibility policy, and
  maintainer operations guidance now form a coherent public support story.
- The repository follows issue-gated work and keeps scope, decisions, and
  validation auditable through GitHub issues and pull requests.

### What is not mature enough yet

- The strongest operational hardening is now packaged in `v0.4.2`, but public
  release history is still extremely young and compressed. Six releases in
  three days are enough to bootstrap a project, not enough to prove a stable
  long-term release cadence.
- Small repo-head versus release-head drift can still reopen quickly if the
  next release cadence slips, even though the `v0.4.2` alignment materially
  reduced that gap.
- External adoption evidence is better than before, but still too early to
  claim broad field validation across multiple independent teams.

## Strengths

- Strong mutation-testing product thesis and clear differentiation
- Rich report model that keeps raw and actionable interpretations visible
- High test coverage across core packages
- Substantially improved package boundaries and reduced orchestration sprawl
- Public CI, compatibility, upgrade, and install gating are now real
- Real documentation and evaluation artifacts rather than only implementation

## Main Risks

- Release-state drift can reopen quickly if the repo keeps improving faster
  than tagged releases are cut
- Broad external adoption is still a hypothesis supported by bounded evidence,
  not yet a fully repeated field result
- Daemon/worker can still create expectation drift if users read it as a mature
  distribution surface
- Some of the next signal-calibration work now depends on field findings rather
  than only internal engineering effort

## Priority Actions

### Immediate

1. Keep release-head aligned with repo-head; the `v0.4.2` alignment should
   become the norm, not a one-off.
2. Refresh the public maturity narrative and install guidance whenever release
   state changes.
3. Start the next external adoption and signal-calibration workstream from real
   repository usage, not only internal fixtures.

### Near term

1. Measure recommendation yield and triage quality from the next adoption wave.
2. Tighten rollout defaults, docs, or low-signal guidance where repeated field
   friction appears.
3. Keep building evidence of stable release operations across more than one
   post-hardening release cycle.

### Product maturity next

1. Calibrate semantic triage and recommendations from field evidence.
2. Keep daemon/worker experimental unless real adopter demand justifies
   versioned protocol work.
3. Continue refining historical review and governance surfaces using actual CI
   and campaign usage.

## Current Maturity Classification

As of 2026-06-19, CervoMutants should be described as:

> a credible public beta mutation-testing toolkit with strong functional depth,
> materially improved engineering maturity, and operational confidence that is
> now represented in the latest public release, without a meaningful supported
> surface gap between repo-head and release-head.

That is a materially stronger position than the 2026-06-17 snapshot. The next
transition is not "make the repo stop looking broken," and it is no longer
"catch the public release up to the hardened repo head." The next transition is
"gather deeper external evidence, keep release discipline steady, and calibrate
the remaining judgment-heavy features from real usage."
