# Maintainer Handoff

Status checked: 2026-10-23

Tracking issue: [#386](https://github.com/cervantesh/cervo-mutants/issues/386)

CervoMutants is **paused, not abandoned or archived**. The repository remains
open for issue-backed contributions. There is no promised response time,
release cadence, or active feature schedule during this pause. Do not interpret
the absence of open feature issues as a claim that the roadmap is complete.

## Published Baseline

- Latest release at this check: [v0.4.5](https://github.com/cervantesh/cervo-mutants/releases/tag/v0.4.5), published 2026-08-07 with Windows, Linux, and macOS archives, `release-manifest.json`, and `SHA256SUMS`.
- The last recorded `main` [test workflow](https://github.com/cervantesh/cervo-mutants/actions/runs/31170820014), [workflow lint](https://github.com/cervantesh/cervo-mutants/actions/runs/31170818529), and [Pages build](https://github.com/cervantesh/cervo-mutants/actions/runs/31170817705) all passed on 2026-08-07. These are dated results, not ongoing health checks.
- The baseline/CI gate, mutator signal fixes, and report contract hardening landed before v0.4.5: [#369](https://github.com/cervantesh/cervo-mutants/pull/369), [#376](https://github.com/cervantesh/cervo-mutants/pull/376), [#377](https://github.com/cervantesh/cervo-mutants/pull/377), and [#380](https://github.com/cervantesh/cervo-mutants/pull/380).
- As of this check, there were no open issues or pull requests before this handoff issue was created. The earlier `codex:spark` issues #353-#364 are closed; do not reimplement them from the historical roadmap.

The [release evidence trail](release-evidence-trail.md) records what each tag
actually proved. The [maturity assessment](project-maturity-assessment.md)
retains a historical scorecard, not a fresh October product score.

## Safe Restart

1. Start from a clean checkout of `main`, fetch current GitHub state, and read [AGENTS.md](../AGENTS.md) and [CONTRIBUTING.md](../CONTRIBUTING.md). Do not build on an old local WIP branch without reviewing its diff against `main`.
2. Use the documented Go `1.25.x` toolchain and [OS matrix](go-toolchain-compatibility.md). Newer Go versions are not yet in the supported matrix.
3. Run `go vet ./...`, `go test ./...`, and `go run ./cmd/releasehelper verify-compat`. Check current CI, not only the dated links above.
4. For a behavior change, verify it with a bounded run from [quickstart](quickstart.md) or the [example repositories](example-repos.md). For report changes, preserve additive schema v1 and update golden/compatibility fixtures.
5. Search for an existing issue, or open a focused one **before implementation**. Keep scope, decisions, tests, and deviations on the issue; link the PR. Do not promise a release until its gates pass.

If a previous run has no final report, inspect
`partial-mutation-report.json` before classifying it as `no_report`. External
tool comparisons must follow the
[apples-to-apples protocol](evaluations/tool-comparison-protocol.md).

## Good Next Contributions

These are **candidates, not assigned work or reopened epics**. Confirm the gap
on current `main`, then open a narrowly scoped issue.

1. Re-run a bounded released-action validation wave on v0.4.5 or newer and measure recommendation usefulness against actual survivor review decisions. The newest published tag has a candidates manifest but no new released hosted-wave write-up in the [evidence trail](release-evidence-trail.md).
2. Improve one adoption or governance friction point backed by a real report bundle and [feedback intake](feedback-intake.md); avoid speculative changes to ranking heuristics.
3. Revalidate install paths, action behavior, and the Go/OS support matrix before expanding any compatibility claim.
4. Evaluate campaign reproducibility or extension seams with a single testable contract, using the existing [pool campaign](evaluations/pool-campaigns.md) and [extensibility](extensibility.md) docs.

The broader strategic options remain in the [maturity roadmap](maturity-roadmap.md).
Daemon/worker is still [experimental](daemon-worker.md); do not present it as
supported distributed execution without a versioned protocol and operational
evidence.

## Where To Find Things

- New adopter: [quickstart](quickstart.md), [adoption guide](adoption-guide.md), [rollout playbooks](rollout-playbooks.md).
- Contributor: [technical contribution guide](contributing-technical.md), [compatibility policy](compatibility-policy.md), [architecture decisions](adr/README.md).
- Maintainer: [operations pack](maintainer-operations-pack.md), [release process](releasing.md), [release evidence](release-evidence-trail.md).
- Troubleshooting: attach the evidence bundle described in the [operations pack](maintainer-operations-pack.md); never infer broad compatibility from a single local run.
