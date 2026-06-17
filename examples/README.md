# Example Workspaces

These versioned example workspaces are the v1 delivery for roadmap issue `#82`.

They are intentionally kept inside the main repository so the project can ship
real, reviewable adoption assets now without waiting on the overhead of three
separate external repositories.

Use them as copy-first references:

| Example | Use when | Focus |
| --- | --- | --- |
| `small-library` | You want the lowest-friction first adoption path. | `ci-fast`, baseline-first rollout, compact PR workflow. |
| `medium-service` | You have a service with multiple packages and need richer review outputs. | `ci-balanced`, coverage-aware selection, nightly visibility. |
| `large-repo-ci` | You need shardable CI patterns before turning on broad mutation runs. | slicing, matrix workflows, bounded large-repo execution. |

Each example includes:

- a standalone `go.mod`
- a runnable `cervomut.yaml`
- a sample GitHub Actions workflow template
- adoption notes that explain why the commands are shaped that way

The repository test suite validates that these workspaces stay parseable and
that each nested Go module still passes `go test ./...`.
