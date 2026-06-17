# Historical Data Contract

Tracking issue: #154

This document defines how CervoMutants stores, exports, retains, and audits
historical mutation data today.

It is intentionally narrower than a full branch-aware analytics platform. The
current goal is to make persisted history trustworthy and operable without
pretending that every future comparison workflow is already supported.

## Storage Model

History is enabled by default and stored at:

- `.cervomut/history.json`

The engine maintains that file as the local chronological history store for one
workspace and output lineage.

The stored data currently includes:

- stable mutant IDs and their latest known status
- first seen and last seen timestamps
- run counts such as survived, killed, not covered, compile error, and timed
  out
- aggregated run snapshots under `runs`

The public report JSON also mirrors the current historical summary under the
top-level `history` block so downstream consumers can inspect the active run
without reading the raw store directly.

Relevant public structures today include:

- `history.path`
- `history.updated_at`
- `history.loaded_mutants`
- `history.updated_mutants`
- `history.new_survivors`
- `history.long_standing_survivors`
- `history.operator_useful_survivor_yield`
- `history.runs[]`

## Export Surfaces

Historical data is exported through these reader-facing artifacts:

- `.cervomut/reports/history-dashboard.json`
- `.cervomut/reports/history-dashboard.html`
- `cervomut report history --out DIR`
- the current run's `mutation-report.json` under `history`

Those exports are derived from persisted history and are intended for CI,
humans, and automation.

`history-dashboard.json` is the structured export for dashboards or scripts.
`history-dashboard.html` is the readable review surface. `report history`
produces a compact text summary for terminals and CI logs.

## Retention Rules

CervoMutants does not currently enforce automatic pruning or retention windows
for history data.

Today the retention model is intentionally simple:

- the local history store is append-oriented
- users own deletion, reset, archiving, or rotation of `.cervomut/history.json`
- separate output directories or preserved artifacts are the correct way to
  keep long-lived history per lane, branch, or release snapshot

This means the current product supports historical persistence, but it does not
yet claim a built-in cross-branch retention or merge policy.

## Audit Expectations

Historical outputs should remain auditable rather than becoming opaque trend
numbers.

The current audit model depends on:

- stable mutant IDs
- explicit UTC timestamps such as `run_at`, `first_seen`, and `last_seen`
- persisted counts for survivor age and timeout classes
- raw score and actionable score staying visible together
- operator useful-survivor yield remaining attached to the recorded run

Historical summaries are not a replacement for the raw report. They are a
derived view that should stay explainable by the corresponding report artifacts
and stable mutant history.

## Current Guarantees

The current implementation guarantees:

- persisted local mutation history across runs when history is enabled
- exported dashboard JSON and HTML derived from that persisted history
- summary reporting for survivor aging, timeout movement, and operator yield
- additive exposure of historical fields in the public report schema

## Current Non-Guarantees

The current implementation does not yet guarantee:

- branch-aware history separation inside one shared store
- release labels embedded automatically into historical runs
- automatic retention pruning or expiration windows
- cross-repo or cross-branch merge semantics for history stores

Those gaps are intentional and keep the contract honest. They are part of the
remaining scope for branch/release comparison work rather than hidden behavior.
Use [docs/branch-release-comparisons.md](branch-release-comparisons.md) for the
current supported workflow.
