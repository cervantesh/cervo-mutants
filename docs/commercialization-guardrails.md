# Commercialization Guardrails

Tracking issue: #194

This document defines the product and architecture boundaries that should keep
any future commercial offering adjacent to the CervoMutants OSS core instead of
turning the mutation engine into a fragmented or partially closed product.

It is intentionally narrower than the license and trademark documents:

- [LICENSE](../LICENSE) governs code use rights under Apache License 2.0.
- [TRADEMARKS.md](../TRADEMARKS.md) governs branding and naming rights.
- this document governs product-shape and architecture decisions when support,
  hosted services, enterprise packs, or other commercial offers are explored.

## Core Position

CervoMutants remains an OSS-first, CLI-first mutation testing product.

The commercial-adjacent rule is simple:

- the mutation engine, report model, baseline workflow, compatibility policy,
  and first-party CI integrations remain public product surfaces;
- commercial work may package, host, support, govern, or extend those surfaces;
- commercial work must not quietly become a required dependency for the
  published OSS workflow.

## What Must Stay In The OSS Core

The following capabilities belong in the public core repository and should not
be turned into enterprise-only behavior:

- mutation generation and execution semantics
- public CLI workflows such as `run`, `baseline`, `report`, `compare`, and
  `doctor`
- supported report outputs and their documented compatibility contracts
- first-party GitHub Action behavior required for normal public CI adoption
- baseline, quarantine, and history data flows needed by documented OSS
  workflows
- compatibility policy, release notes, upgrade notes, and support matrix
- fixes required to keep documented public workflows working

If a change alters the meaning of the public product, it belongs in OSS first.

## Acceptable Adjacent Commercial Offers

The following are acceptable directions because they stay downstream of the OSS
core rather than replacing it:

- paid support, onboarding, tuning, and rollout help
- enterprise policy packs or reviewed configuration bundles
- managed campaign hosting built on documented OSS inputs and outputs
- dashboards or reporting layers that consume exported report or history data
- compliance, audit, or retention workflows layered around OSS artifacts
- training, certification, and operational playbooks
- priority response or hosted convenience around already-public capabilities

These offers may be commercial, but they must not redefine what the open
product is.

## Red Lines

The following moves are out of bounds unless the OSS core changes first:

- moving core mutation behavior behind a private service or private API
- making the documented CLI depend on a hosted account for normal supported use
- shipping enterprise-only report formats that become the de facto required
  review surface
- withholding compatibility-critical fixes from the OSS core while selling them
  only in a separate offering
- treating experimental surfaces as commercially supported before the OSS
  compatibility and support story is ready
- forking the public mutation engine into a separate closed implementation with
  incompatible semantics while still presenting it as the same product

## Architecture Guardrails

When evaluating a new hosted or commercial proposal, use these rules:

1. If it changes mutation semantics, report schema, baseline semantics, or
   supported CI behavior, land that change in OSS first.
2. If it depends on private state, private services, or proprietary policy
   logic, keep the dependency optional and outside the core CLI path.
3. If it needs extra data, prefer consuming documented exports instead of adding
   hidden coupling to internal repository state.
4. If it introduces a new integration contract, document whether that contract
   is public, experimental, or commercial-only.
5. If it needs branding distinct from upstream, follow
   [TRADEMARKS.md](../TRADEMARKS.md) and use separate product naming unless
   explicit permission says otherwise.

## Decision Rule For New Proposals

Use this test before approving a commercial-adjacent initiative:

- If the feature makes the OSS product itself better, safer, or more compatible
  for every user, it belongs in core.
- If the feature primarily adds convenience, governance, hosting, support, or
  operational leverage around documented OSS outputs, it may live adjacent to
  core.
- If the feature would make the OSS path second-class, blocked, or misleading,
  reject it or redesign it.

## Relationship To Hosted-Service Evaluation

This document does not say that CervoMutants should launch a hosted service. It
only defines the boundaries for evaluating one.

The question of whether a hosted or reporting layer is justified belongs to
issue #193. The current evaluation is documented in
[docs/hosted-reporting-evaluation.md](hosted-reporting-evaluation.md). If that
work proceeds, it should be judged against the guardrails above instead of
inventing a separate product boundary later.
