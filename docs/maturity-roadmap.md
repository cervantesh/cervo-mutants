# Maturity Roadmap And Pause State

Status checked: 2026-10-23

Historical 24-month roadmap tracker: [#179](https://github.com/cervantesh/cervo-mutants/issues/179)

Handoff issue: [#386](https://github.com/cervantesh/cervo-mutants/issues/386)

This is a **directional roadmap**, not an active delivery schedule. The project
is in a maintainer pause. Contributors can propose focused, issue-backed work;
no phase, target date, review time, or release cadence is promised. Start with
the [maintainer handoff](maintainer-handoff.md) for the supported baseline and
safe restart procedure.

## What Is Already Shipped

- The first four roadmap phases (#64-#67) are closed historical trackers. Their work established semantic triage, actionable review, report workbench and contracts, baseline workflows, recommendation and governance surfaces, extensibility seams, and campaign tooling.
- Phase 5 (#180) and Phase 6 (#181) are also closed as tracker epics. Closure records the tracked initiatives; it does **not** prove broad external adoption, a supported distributed protocol, or every aspirational maturity target.
- The Phase 2 report-contract follow-up (#352-#364), baseline/CI gate (#368), and mutator fixes (#374-#375) landed in [v0.4.5](https://github.com/cervantesh/cervo-mutants/releases/tag/v0.4.5). Do not treat these as open work.
- The last recorded `main` test, workflow-lint, and Pages runs passed on 2026-08-07. See the dated links in the [handoff](maintainer-handoff.md); recheck CI when work resumes.

There was no open implementation backlog at the October status check before
the documentation handoff issue was opened. A closed issue is not a standing
assignment. Search current issues and code before proposing work.

## Candidate Next Work

These lanes are ordered by evidence dependency, not by calendar. Each concrete
change needs its own issue and acceptance criteria before implementation.

### 1. Released-Surface Validation

Run a bounded campaign using a published Action/release ref, capture target
semantics and denominator health, and record whether report recommendations
led to useful test changes. The [release evidence trail](release-evidence-trail.md)
notes no new released hosted-wave write-up for v0.4.5.

Exit evidence: reproducible report bundles, documented target and environment,
and concrete review outcomes rather than score-only claims.

### 2. Review Quality From Field Feedback

Calibrate recommendation yield, semantic grouping, suppression/quarantine UX,
ownership routing, and historical review only where real adopters show friction.
Use the [feedback intake](feedback-intake.md) and
[operations pack](maintainer-operations-pack.md) to keep that work auditable.

Exit evidence: a narrow finding, before/after review effort, stable raw report
data, and tests for any changed interpretation.

### 3. Reproducibility And Supported Contracts

Strengthen campaign manifest/resume lineage, extension boundaries, Action
install paths, and upgrade compatibility where current tests leave a real gap.
Keep schema v1 additive and the Go/OS matrix evidence-driven. Daemon/worker
remains experimental until a separately funded, versioned protocol is proven.

Exit evidence: deterministic fixtures, compatibility checks on supported
platforms, and no undocumented change to public artifacts or flags.

### 4. Broader External Adoption

Only after the earlier lanes produce reliable evidence, validate distinct
repository shapes and update rollout playbooks. Optional hosted or commercial
adjacencies must remain downstream of the Apache-2.0 OSS core.

Exit evidence: independent adoption profiles, reproducible report artifacts,
documented support boundaries, and feedback that changes the product plan.

## Invariants For Future Maintainers

- Open or link a GitHub issue before implementing a change, per [AGENTS.md](../AGENTS.md).
- Keep raw results and actionable interpretation distinct; `--actionable-only` is a review view, not a silent suppression rule.
- Preserve report/schema compatibility and run golden fixtures for public output changes.
- Compare mutation tools only with explicit manifest/effective targets and target modes; otherwise mark the run diagnostic.
- Keep compatibility and maturity claims bounded by CI, release, and real-adopter evidence.
- Do not graduate daemon/worker from experimental status by documentation alone.

The previous 24-month numeric targets were aspirations, not measured completion
criteria or a current schedule. The dated [maturity assessment](project-maturity-assessment.md)
retains the historical scorecard without claiming a fresh October score.
