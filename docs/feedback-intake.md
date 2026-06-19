# Adoption Feedback Intake

Tracking issues: #136, #190

This document defines the supported path for collecting external adoption
feedback without relying on private chat notes or maintainer memory.

## Default Intake Path

Open a GitHub issue with the
[`Adoption feedback`](../.github/ISSUE_TEMPLATE/adoption-feedback.yml) template
whenever a real repository rollout surfaces useful feedback about:

- mutation signal quality
- review UX
- runtime or resource behavior
- rollout friction

The template intentionally asks for the same fields every time so feedback stays
comparable across repositories and releases.

It now also captures structured rollout dimensions such as repository profile,
adoption stage, install path, closest rollout playbook, primary blocker class,
and the suggested outcome. Those fields are the basis for release-level
adoption analytics rather than a maintainer trying to reconstruct patterns from
free-text issues later.

## What Good Feedback Looks Like

A useful adoption-feedback issue should include:

- the repository and mutation target
- environment details such as OS, Go version, and local versus CI execution
- the CervoMutants version or commit used
- the workflow shape: policy, budget, workers, reports, baseline posture
- concrete notes about survivor usefulness or noise
- runtime or resource observations
- rollout friction and artifact links

Short unsupported opinions like "too noisy" or "too slow" are not enough on
their own. The goal is a repeatable product signal, not just sentiment.

## Triage Expectations

Maintainers should classify each adoption-feedback issue into one of these
outcomes:

- already explained by current docs or known limits
- needs a documentation or workflow follow-up
- needs a product or code follow-up
- needs a narrower reproduction before acting

If the same finding repeats, link it from
[docs/evaluations/follow-up-ledger.md](evaluations/follow-up-ledger.md) so it
becomes part of the explicit backlog instead of remaining buried in individual
issue threads.

For the analytics model, promotion rules, and release-level review loop that
turn those issues into durable product signal, see
[docs/adoption-analytics.md](adoption-analytics.md).

For the maintainer-side support bundle, triage flow, and upgrade checklist used
when processing external adoption feedback, see
[docs/maintainer-operations-pack.md](maintainer-operations-pack.md).

## Scope Boundary

This intake path is for real adoption feedback, not generic feature requests.

Use a normal issue instead when the report is mainly:

- a concrete defect with a local reproduction
- a targeted feature proposal
- an internal implementation task with no external rollout evidence
