# Hosted And Advanced Reporting Layer Evaluation

Tracking issue: #193

Date: 2026-06-19

This document evaluates whether the current public evidence justifies building a
hosted service or advanced reporting layer for CervoMutants.

## Short Decision

Not yet.

The current OSS product already provides enough report and history surfaces to
support local, CI, and artifact-oriented review. The evidence does not yet
justify turning a hosted or advanced reporting layer into an active product
commitment.

The correct posture today is:

- keep the OSS core self-sufficient;
- allow future hosted exploration only as an optional downstream layer;
- revisit the decision after stronger adoption and feedback evidence exists.

## What Already Exists In The OSS Core

The current public product already exposes several capabilities that reduce the
need for a hosted layer as an immediate priority:

- structured mutation reports in JSON, JUnit, SARIF, summary, and HTML
- persisted local history in `.cervomut/history.json`
- exported history dashboards in `history-dashboard.json` and
  `history-dashboard.html`
- branch, release, and time-window comparison workflows based on preserved
  artifacts and baseline governance
- a first-party GitHub Action and documented rollout patterns for small,
  medium, and large repositories

That means the product already has machine-consumable exports and human review
surfaces without requiring a service backend.

## What Public Evidence Proves Today

The current evidence is strong enough to justify continued OSS investment, but
not a hosted product commitment.

What is proven:

- the tool works in local and CI mutation workflows
- bounded external validation succeeded across five public Go repositories
- historical dashboards and comparison artifacts already exist in the OSS core
- baseline-first governance, compatibility policy, and adoption guidance are
  publicly documented

What is not yet proven:

- direct maintainer demand for centralized hosted review
- repeated multi-team demand for shared dashboards or campaign hosting
- a stable operational need that cannot be met with preserved artifacts and
  current exports
- support expectations for a hosted control plane, identity model, tenancy, or
  retention policy

## Why A Hosted Layer Is Not Yet Justified

The missing proof is not technical possibility. It is product necessity.

Three constraints dominate today:

1. Adoption evidence is still early. The repo has a real external validation
   wave, but not broad recurring maintainer feedback that proves centralized
   workflow pain.
2. The current OSS exports already cover many of the obvious reporting use
   cases. A hosted layer would need to solve something materially beyond
   exporting JSON, HTML, history dashboards, and preserved artifacts.
3. Operational scope would grow much faster than current product evidence.
   Hosted reporting implies decisions about auth, tenancy, privacy, retention,
   import/export contracts, and support obligations that the current evidence
   does not yet justify.

## What Could Justify Reopening The Decision

Revisit hosted or advanced reporting work when at least some of these signals
are true:

- repeated adoption feedback shows teams want shared cross-run dashboards rather
  than artifact-only review
- multiple real adopters request managed history retention, campaign hosting, or
  multi-user governance
- operational friction around preserved artifacts shows up across more than one
  independent team
- the OSS export surfaces stabilize enough that a downstream service can depend
  on them without hidden coupling
- maintainers can name a narrow first hosted use case with clear value and low
  support ambiguity

## If Revisited Later, Start Narrow

If this issue is reopened in the future, the first hosted layer should stay
small and downstream:

- read-only ingestion of published report or history artifacts
- dashboards built from documented JSON exports
- optional campaign orchestration around documented CLI and report contracts
- no hidden dependency in the OSS CLI path
- no new private mutation semantics

That path keeps the service additive rather than redefining the product.

## Relationship To Other Documents

- [docs/commercialization-guardrails.md](commercialization-guardrails.md):
  defines the product and architecture boundaries for any commercial-adjacent
  offering
- [docs/history-data.md](history-data.md): defines the current history and
  dashboard contract already available in OSS
- [docs/branch-release-comparisons.md](branch-release-comparisons.md): defines
  the currently supported artifact-oriented comparison workflow
- [docs/adoption-guide.md](adoption-guide.md): defines what current public
  evidence does and does not prove

## Current Conclusion

As of 2026-06-19, CervoMutants should not commit to building a hosted or
advanced reporting layer as a near-term product promise.

The better next move is to keep strengthening:

- OSS report quality
- adoption evidence
- feedback intake
- comparison workflows
- governance and history usability

If later evidence proves that preserved artifacts and current exports are not
enough, a hosted layer can be reconsidered under the guardrails above.
