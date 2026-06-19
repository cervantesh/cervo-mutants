# Changelog

All notable changes to CervoMutants should be recorded here.

The format is intentionally simple:

- keep newest versions first
- add a `docs/upgrade-notes/<version>.md` file before tagging a release
- keep entries grounded in merged behavior, not roadmap intent

## [Unreleased]

- No unreleased entries yet.

## [v0.4.2] - 2026-06-19

### Changed

- Hosted external-wave result and summary generation now run through tested Go
  helpers in `cmd/actionhelper` instead of large inline workflow scripts.
- The hosted external-wave workflow now runs those helper commands from the
  checked-out source module, fixing the post-`#278` working-directory
  regression on GitHub-hosted runners.

### Documentation

- Published a dated current-main hosted-wave verification note and committed
  aggregate summary artifact for the post-hardening rerun after the helper-path
  repair.

### Verification

- `go run ./cmd/releasehelper verify-compat`
- `go run ./cmd/releasehelper notes --version v0.4.2 --out .tmp/release-notes-v0.4.2.md`

## [v0.4.1] - 2026-06-19

### Changed

- Hosted adoption-wave summaries now distinguish raw recommendation entries
  from collapsed recommendation review units, and raw governance suggestion
  totals from per-status governance counts, which makes released GitHub Action
  evidence easier to interpret without hiding the raw totals.
- The published first-party GitHub Action now uses `actions/setup-go@v6`,
  removing the stale Node 20 deprecation path from the released action surface.
- Windows compatibility validation is more deterministic:
  pre-canceled pool commands no longer start real processes, and the serial
  runner budget/quarantine coverage no longer depends on real-clock races.

### Documentation

- Published dated follow-up notes for released adoption-wave recommendation
  review units and governance suggestion status counts.

### Verification

- `go test ./...`
- `go run ./cmd/releasehelper verify-compat`
- `go run ./cmd/releasehelper notes --version v0.4.1 --out .tmp/release-notes-v0.4.1.md`

## [v0.4.0] - 2026-06-19

### Added

- Semantic triage v1 with `pkg/triage`, actionable scoring, semantic grouping,
  actionable-only survivor views, and triage-aware report fields.
- GitHub-native review surfaces including SARIF, GitHub summary output, and
  first-party GitHub Action support for the standard CI path.
- Historical dashboards, recommendation surfaces, ownership routing, and
  baseline review lifecycle expansion for more durable survivor governance.
- Pool campaign orchestration, benchmark-corpus support, and committed example
  workspaces for small, medium, and large-repo adoption paths.

### Changed

- The public support story now includes an explicit compatibility matrix,
  additive compatibility policy, and daemon/worker narrowing as an
  experimental-only surface until protocol versioning exists.
- Release automation now verifies release notes, archive contents, checksums,
  upgrade handoff, `go install`, and archive-install report flows before
  publication.
- Public CI and release guidance now reflect the first hardened cross-platform
  validation lane on GitHub Actions.

### Documentation

- Added adoption guidance, rollout playbooks, maintainer operations guidance,
  adoption feedback intake, hosted-layer scope notes, and commercialization
  guardrails for the OSS core.
- Published the 24-month roadmap v2 and refreshed the project maturity
  assessment against the current repo-head evidence.

### Verification

- `go test ./...`
- `go run ./cmd/releasehelper notes --version v0.4.0 --out .tmp/release-notes-v0.4.0.md`
- `go run ./cmd/releasehelper verify-compat`

## [v0.3.0] - 2026-06-17

### Changed

- Hardened Windows-native mutation execution with safer temp-root handling and
  more conservative worker defaults.
- Added runtime warning metadata so reports expose selected temp-root and
  Windows execution caveats.

### Added

- Large-repo CI slicing mode with deterministic sharding and per-run bounds.
- Survivor-ranking calibration follow-up that reduces noisy comparator-boundary
  competition in review queues.

### Verification

- `go test ./...`

## [v0.2.0] - 2026-06-17

### Added

- First-class resource statuses in reports, including `memory_killed`,
  `skipped_resource`, and `pending_budget`.
- Structured failure artifacts for `run` and `eval`, including correlation IDs,
  `failure-debug.json`, and panic recovery.
- `stopped_reason`, `last_completed_mutant`, and per-mutant
  `memory_peak_bytes` in report outputs.

### Verification

- `go test ./...`

## [v0.1.0] - 2026-06-17

### Added

- Initial GitHub release for CervoMutants.
- Apache License 2.0, `NOTICE`, and trademark guidance.

### Changed

- Aligned repository naming, module path, and documentation with the
  `CervoMutants` public project name.

### Verification

- `go test ./...`
