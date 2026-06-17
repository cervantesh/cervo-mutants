# GitHub Review Workflow

Tracking issue: #142

This document explains the intended pull-request and CI review workflow for
CervoMutants using the repository's maintained GitHub Actions examples.

The goal is not to promote one universal pipeline. The goal is to give teams a
clear default operating model that is already represented by committed example
workflows.

For the first-party installation and run wrapper itself, see
[docs/github-action.md](github-action.md).

## Validated Workflow Shapes

The current public workflow guidance is grounded in three maintained examples:

- [`examples/small-library/.github/workflows/cervomut.yml`](../examples/small-library/.github/workflows/cervomut.yml):
  a single pull-request mutation lane with `ci-fast`
- [`examples/medium-service/.github/workflows/cervomut.yml`](../examples/medium-service/.github/workflows/cervomut.yml):
  a pull-request lane plus a scheduled nightly lane
- [`examples/large-repo-ci/.github/workflows/cervomut.yml`](../examples/large-repo-ci/.github/workflows/cervomut.yml):
  sharded pull-request and nightly lanes for larger repositories

Those examples are intentionally different. They show that the recommended
workflow depends on repository size and runtime budget, not on one fixed YAML
shape.

## Default Pull-Request Lane

The PR lane should answer one question fast:

> Did this change introduce new actionable survivors or regress the current
> baseline?

Recommended defaults:

- `ci-fast` or `ci-balanced`
- bounded runtime budget
- `github-summary` enabled
- baseline-first governance
- JSON and JUnit retained for automation

Example command shape:

```powershell
cervomut run ./... `
  --policy ci-fast `
  --budget 5m `
  --report summary,json,junit,github-summary `
  --out .cervomut/pr
```

On GitHub Actions, `github-summary` writes the same markdown to
`$GITHUB_STEP_SUMMARY`, so the PR lane can expose a compact review summary
without forcing humans into the raw JSON first.

## Nightly Or Scheduled Lane

The nightly lane is where broader depth belongs.

Recommended defaults:

- `nightly` policy
- HTML and SARIF enabled in addition to PR-lane artifacts
- longer budget
- optional slicing or sharding for larger repositories

Example command shape:

```powershell
cervomut run ./... `
  --policy nightly `
  --budget 20m `
  --report summary,json,junit,html,sarif,github-summary `
  --out .cervomut/nightly
```

This keeps the PR workflow focused while still allowing deeper survivor review,
history, and code-scanning style output outside the fast review lane.

## Large-Repo Pattern

For larger repositories, the intended GitHub workflow is to shard broad mutation
work instead of pretending one monolithic run will remain fast and stable.

The maintained `large-repo-ci` example demonstrates:

- PR shards with `--slice-by package`
- nightly shards with `--slice-by file`
- bounded density controls such as `--max-files-per-run` and
  `--max-mutants-per-package`
- deterministic sampling to keep shard behavior reviewable

That pattern is the default public answer for larger repositories until the
experimental daemon/worker path becomes a supported execution surface.

## What Reviewers Should Look At

For normal PR review, the intended order is:

1. GitHub step summary from `github-summary`
2. baseline or actionable survivor deltas from the stored report artifacts
3. HTML workbench or survivor reports when a change needs deeper inspection

For scheduled or campaign review, add:

1. SARIF/code-scanning style output when enabled
2. HTML workbench for grouped survivor review
3. history or governance artifacts when the repo is already using those layers

## Codex Review Gate On Main

Pull requests into `main` are expected to wait for Codex review before merge.

The repository now enforces this in two layers:

- a required `codex-review-gate` check waits for a Codex response on the
  current PR head
- unresolved Codex review threads must be resolved before merge

That means a pull request should not merge while:

- Codex has not yet responded to the current head commit, or
- Codex has an unresolved review thread on the current head commit

The intended workflow is:

1. wait for Codex to react or comment on the current head
2. address the feedback or explicitly document why it is being declined
3. resolve the Codex review conversation in GitHub
4. merge only after `codex-review-gate` is green

If Codex leaves a no-finding thumbs-up reaction, the gate accepts that as the
review response for the current head commit.

If Codex leaves a review comment, resolving the conversation is required after
the feedback is handled. Do not merge while the thread is still open.

## Baseline Rule

The GitHub workflow should stay baseline-first by default.

That means:

- do not fail an existing repository on a raw score threshold on day one
- fail on baseline regressions or policy-driven new survivors instead
- introduce nightly depth only after the PR lane is understandable

This rule matches the product defaults, the example repos, and the broader
signal-first design of CervoMutants.

## Optional GitHub Add-Ons

If a repository wants GitHub code-scanning ingestion for SARIF, add a follow-up
step after the mutation run in the nightly lane:

```yaml
- name: Upload SARIF
  if: always()
  uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: .cervomut/nightly/mutation-report.sarif
```

That is optional on purpose. The core workflow remains useful even when a team
only wants step summaries, artifacts, and local HTML review.

## Decision Guide

Use the workflow shape that matches your repository:

- copy `small-library` first when you want the lowest-friction PR lane
- copy `medium-service` when you need separate PR and nightly behavior
- copy `large-repo-ci` when sharding is required before mutation breadth grows

If your repository cannot keep a bounded PR lane understandable, narrow the
target or shard the run before increasing policy depth.
