# Example Workspaces

Tracking issue: #82

This document is the public entry point for the maintained example workspaces
that demonstrate how to adopt CervoMutants across repository sizes.

The first delivery is intentionally versioned inside this repository under
`examples/`. That keeps the examples reviewable, testable, and release-aligned
before splitting them into standalone public repos becomes worth the extra
maintenance overhead.

## Pick The Right Starting Point

| Example | Copy first when | Default policy posture | Primary CI idea |
| --- | --- | --- | --- |
| `examples/small-library` | You need the simplest baseline-first rollout. | `ci-fast` with compact outputs. | Single PR mutation lane. |
| `examples/medium-service` | You have multiple packages and need richer review output. | `ci-balanced` on PRs and `nightly` on schedule. | PR lane plus nightly lane. |
| `examples/large-repo-ci` | You need deterministic slicing before broad mutation coverage. | `ci-fast` for PR shards and `nightly` for wider shards. | Matrix sharding and bounded density. |

## What Each Example Includes

Each workspace includes:

- a standalone `go.mod`
- a runnable `cervomut.yaml`
- a sample GitHub Actions workflow under `.github/workflows/`
- repository-local README notes that explain the adoption tradeoffs

The repository test suite validates that:

- the example configs remain parseable through `pkg/config`
- each example Go module still passes `go test ./...`
- the README/workflow assets remain present so docs do not drift away from the
  actual example files

## Recommended Adoption Order

1. Copy the closest example workspace into a branch or sandbox repo.
2. Run the documented local dry-run command first.
3. Establish a baseline before introducing hard score expectations.
4. Keep the example workflow shape until your own repo proves it needs a
   different policy, target, or sharding strategy.

## Why These Are In-Repo First

Three separate external example repositories would create more operational
overhead than learning value at the current maturity stage. The in-repo model
still provides:

- versioned examples tied to released behavior
- reviewable diffs alongside core product changes
- test coverage that fails if the examples become stale

If public adoption later justifies it, these workspaces can still be promoted
into standalone repositories without losing the documented rollout patterns.
