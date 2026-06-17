# Branch And Release Comparisons

Tracking issue: #152

This guide explains the current supported workflow for comparing mutation health
across branches, releases, and time windows.

The current model is artifact-oriented. It is intentionally narrower than a
shared multi-branch history database.

## What Is Supported Today

The current supported pieces are:

- `cervomut baseline compare`
- `cervomut baseline diff`
- `cervomut baseline accept`
- `cervomut baseline promote`
- `cervomut report history --out DIR`
- persisted `history-dashboard.json` and `history-dashboard.html`
- preserved output directories per branch, lane, or release

Together they give a repeatable comparison workflow without pretending the tool
already merges cross-branch history automatically.

## Branch Comparison Workflow

Use separate worktrees or checkouts so each branch keeps its own normal
`.cervomut/` state.

Example:

```powershell
# In a main worktree or checkout
cervomut run ./... --policy ci-balanced

# In a feature-branch worktree or checkout
cervomut run ./... --policy ci-balanced
```

Then compare the feature worktree run against the accepted branch baseline:

```powershell
cervomut baseline diff
cervomut baseline compare
```

For review flows where a new branch result should be inspected before replacing
the baseline:

```powershell
cervomut baseline accept
cervomut baseline diff --candidate
cervomut baseline promote
```

Preserve older `.cervomut/` directories as artifacts if you also want a frozen
copy of the exact report outputs for later inspection.

That makes branch review explicit instead of silently redefining the baseline.

## Release Comparison Workflow

Treat each release as a preserved report directory plus its historical exports.

Recommended release artifacts:

- `mutation-report.json`
- `summary.txt`
- `survivors.txt`
- `history-dashboard.json`
- `history-dashboard.html`

To compare releases:

1. Preserve each release report directory as a CI artifact or release asset.
2. Keep the accepted baseline that matched the older release.
3. Run the newer release candidate against the same target and policy.
4. Use `baseline diff` and `baseline compare` to inspect score movement,
   survivor deltas, and actionable-score changes.

## Time-Window Trend Workflow

For repeated runs in one lane, use the persisted history store and report view:

```powershell
cervomut report history --out .cervomut/nightly
```

The matching `history-dashboard.json` and `history-dashboard.html` expose:

- raw score trend
- actionable score trend
- survivor aging
- timeout movement
- operator useful-survivor yield

This is the supported way to inspect longitudinal movement inside one lineage.

## Current Non-Goals

The current implementation does not claim:

- one shared history store with automatic branch merge semantics
- built-in release labels inside every history entry
- automatic cross-branch conflict resolution
- one-command arbitrary diffing between any two unrelated report directories

Those limits are intentional. They keep the workflow honest while still making
branch, release, and time-window comparisons repeatable with preserved
artifacts and baseline governance.
