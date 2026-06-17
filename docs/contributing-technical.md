# Technical Contribution Guide

This guide exists so outside contributors can make safe changes without relying
on thread history or unwritten maintainer context.

## Package Boundaries

Use these boundaries as the default architecture:

- `cmd/cervomut`: CLI entrypoints, flag parsing, and command dispatch. Keep
  product heuristics out of command parsing when possible.
- `cmd/releasehelper`: release-note assembly helper used by the release
  workflow.
- `pkg/config`: config model, defaults, validation, and policy application.
- `pkg/engine`: run orchestration, schema types, checkpointing, history,
  summary, ranking hooks, and execution flow.
- `pkg/mutator`: mutation generation and operator definitions.
- `pkg/triage`: semantic actionability heuristics and survivor-priority signal.
- `pkg/report`: human and machine report projections such as summary, JSON,
  HTML, JUnit, SARIF, governance, and history dashboard outputs.
- `pkg/pool`: multi-repo smoke, compare, benchmark, and campaign orchestration.
- `pkg/extcompare`: normalization/parsing for external mutation tools.
- `pkg/daemon`: experimental JSON-lines protocol runner.
- `pkg/runner`, `pkg/isolate`, `pkg/selecttest`, `pkg/cache`,
  `pkg/quarantine`, `pkg/baseline`, `pkg/doctor`, `pkg/discover`, and
  `pkg/eval`: focused support packages that should stay narrower than
  `pkg/engine`.

## Change Placement Rules

- New semantic review heuristics belong in `pkg/triage`, not directly in
  `pkg/mutator`.
- New public report fields should be introduced through `pkg/engine` types and
  rendered in `pkg/report`, not bolted onto one output only.
- External tool comparison logic belongs in `pkg/extcompare` or `pkg/pool`,
  not inside generic mutation execution paths.
- Experimental daemon/worker changes must stay aligned with
  [daemon-worker.md](daemon-worker.md) and must not imply supported protocol
  guarantees.

## When To Update Docs

Update docs in the same change when you touch:

- CLI behavior or flags:
  `README.md`, [compatibility-policy.md](compatibility-policy.md)
- pull request review or merge policy:
  [github-review-workflow.md](github-review-workflow.md)
- report schema or output contracts:
  `README.md`, [compatibility-policy.md](compatibility-policy.md), relevant
  report docs or golden fixtures
- release process:
  [releasing.md](releasing.md), `CHANGELOG.md`, `docs/upgrade-notes/`
- architecture or contributor expectations:
  this guide and `docs/adr/`

## ADR Expectations

Add or update an ADR when a change:

- changes a long-lived architectural boundary
- changes the default product doctrine
- introduces a new compatibility promise
- deliberately keeps a surface experimental

Start from [docs/adr/README.md](adr/README.md).

## Validation Expectations

Default validation:

```powershell
go test ./...
go vet ./...
```

Then add targeted checks when relevant:

- report changes: report tests and golden fixtures
- workflow or release changes: workflow YAML parse and release-helper tests
- PR review gate changes: validate the GitHub workflow plus the live branch
  protection or ruleset state
- comparison harness changes: protocol docs plus pool/extcompare tests
- Windows-sensitive execution changes: `doctor`, runner, and path-sensitive
  tests

## Public Contract Changes

Before changing a public surface, check:

- [compatibility-policy.md](compatibility-policy.md)
- [go-toolchain-compatibility.md](go-toolchain-compatibility.md)
- [releasing.md](releasing.md)

If the change is breaking, do not hide it inside an implementation PR. The
change must be explicit in docs, changelog, and upgrade notes.

## Comparison Work

For external mutation-tool studies, follow:

- [evaluations/tool-comparison-protocol.md](evaluations/tool-comparison-protocol.md)
- [evaluations/comparison-harness.md](evaluations/comparison-harness.md)

Every fairness claim must record target semantics and denominator health.
