# Maintainer Operations Pack For External Adopters

Tracking issue: #192

This document is the maintainer-facing operations pack for handling real
external adoption.

It is meant for maintainers triaging rollout questions, runtime complaints,
artifact bundles, and upgrade concerns from teams using CervoMutants outside
this repository.

## What This Pack Covers

This pack gives maintainers a compact operating baseline for:

- support scope and support boundaries
- upgrade and release handling
- known limits to state explicitly
- the evidence bundle to request from adopters
- the issue templates and report artifacts that should anchor triage
- the adoption analytics loop used to aggregate repeated external friction

## Current Support Baseline

When responding to adopters, start from the public support matrix and current
fit guidance rather than ad hoc promises:

- support matrix: [docs/go-toolchain-compatibility.md](go-toolchain-compatibility.md)
- compatibility policy: [docs/compatibility-policy.md](compatibility-policy.md)
- adoption fit and limits: [docs/adoption-guide.md](adoption-guide.md)
- rollout playbooks by repo shape: [docs/rollout-playbooks.md](rollout-playbooks.md)

Operationally, that means:

- Linux `1.25.x` is the primary deep-validation lane
- Windows and macOS `1.25.x` are supported through compatibility smoke lanes
- daemon/worker is still experimental unless public docs say otherwise
- baseline-first rollout is the supported default posture

## Upgrade Checklist For Maintainers

When an adopter is upgrading or validating a release, use this checklist:

1. Confirm the version or commit being used.
2. Confirm the install path:
   - `go install`
   - release archive
   - local source build
   - first-party GitHub Action
3. Check the release notes and upgrade notes:
   - [docs/releasing.md](releasing.md)
   - [docs/upgrade-notes/README.md](upgrade-notes/README.md)
4. Verify whether the report, history, or CI workflow uses a documented
   compatibility surface.
5. Ask whether the repository is following the closest supported rollout
   playbook or a custom unsupported workflow.

## Known Limits To State Explicitly

Maintainers should say these limits plainly instead of letting adopters infer
stronger support than the repo actually claims:

- distributed execution is still experimental
- cross-branch history merge semantics are not built in
- current external validation is real but still bounded
- runtime variance across repositories is expected
- semantic triage reduces noise but does not remove the need for human review
- one-click zero-tuning adoption across every repository shape is not promised

Those limits should be stated early when they are relevant, not only after a
user is already frustrated.

## Evidence Bundle To Request

For a useful maintainer triage pass, ask adopters for a report bundle rather
than a prose-only description.

Minimum useful bundle:

- `cervomut.yaml`
- `mutation-report.json`
- `summary.txt`
- `survivors.txt` when survivor review is part of the question
- `history-dashboard.json` or `history-dashboard.html` when the issue is about
  trend movement, survivor age, or historical comparison
- CI job URL or terminal log excerpt
- `cervomut doctor` output
- OS, Go version, install path, and exact command used

When the issue is branch or release comparison related, also ask for:

- the older preserved report directory or artifact reference
- the newer candidate report directory or artifact reference
- whether baseline `diff`, `compare`, `accept`, or `promote` were used

## Which Issue Template To Use

Use the existing GitHub issue paths deliberately:

- [`Adoption feedback`](../.github/ISSUE_TEMPLATE/adoption-feedback.yml) when
  the problem is rollout evidence, noise, runtime behavior, or workflow friction
- normal issue filing when there is a concrete reproducible defect or a targeted
  feature request

If the same adoption finding repeats, maintainers should link it into
[docs/evaluations/follow-up-ledger.md](evaluations/follow-up-ledger.md) so it
turns into tracked product signal rather than scattered anecdotes.

## Triage Decision Flow

Use this flow for first-pass maintainer triage:

1. Confirm the adopter is on a supported or honestly-documented environment.
2. Confirm whether the issue is baseline-first adoption, PR lane review,
   nightly widening, campaign behavior, or experimental distributed execution.
3. Check whether the closest rollout playbook was followed.
4. Review the artifact bundle before classifying the issue as product, docs, or
   unsupported workflow.
5. Record the repository profile, adoption stage, primary blocker, and outcome
   type from the issue form so the feedback remains aggregatable.
6. If the same friction already exists in docs, point to the current guidance.
7. If the docs are insufficient, file or link a documentation follow-up.
8. If the behavior contradicts supported docs or compatibility claims, file or
   link a product or code follow-up.

## Report Artifact Cheat Sheet

Use these artifact surfaces consistently during triage:

- `mutation-report.json`: canonical structured result for one run
- `summary.txt`: compact human-readable outcome
- `survivors.txt`: targeted survivor review surface
- `history-dashboard.json`: structured trend export
- `history-dashboard.html`: human-readable historical dashboard
- SARIF or GitHub summary outputs: CI-native review context

If a question can be answered from one of those artifacts, prefer that evidence
over re-running the repo blind.

## Related Guides

- [docs/adoption-guide.md](adoption-guide.md)
- [docs/rollout-playbooks.md](rollout-playbooks.md)
- [docs/feedback-intake.md](feedback-intake.md)
- [docs/adoption-analytics.md](adoption-analytics.md)
- [docs/go-toolchain-compatibility.md](go-toolchain-compatibility.md)
- [docs/releasing.md](releasing.md)
- [docs/branch-release-comparisons.md](branch-release-comparisons.md)
