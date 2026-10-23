# Contributing

Thanks for contributing to CervoMutants.

This repository expects contributions to be auditable, issue-linked, and safe
for report consumers and CI users.

The project is in a maintainer pause. Contributions are welcome, but review
times and releases are not guaranteed. Start with the dated
[maintainer handoff](docs/maintainer-handoff.md) to distinguish shipped work
from candidate follow-ups; do not assume a historical roadmap item is open.

## Before You Change Code

1. Use an existing GitHub issue or open one before implementation starts.
2. Read [AGENTS.md](AGENTS.md) for repository-specific workflow rules.
3. Read [docs/compatibility-policy.md](docs/compatibility-policy.md) if your
   change touches CLI behavior, report outputs, or protocol surfaces.
4. Read [docs/contributing-technical.md](docs/contributing-technical.md) for
   package boundaries and validation expectations.

## Pull Request Expectations

- Reference the issue in the branch, commit, or PR when practical.
- Keep the issue updated with scope, decisions, tests, and deviations.
- Update public docs when behavior, contracts, or workflows change.
- Update `CHANGELOG.md` and `docs/upgrade-notes/` when a release-relevant
  supported surface changes.

## Validation

At minimum, contributors should run:

```powershell
go test ./...
go vet ./...
```

When changing compatibility surfaces, also update or verify the relevant
documentation and fixtures:

- CLI or flags: `README.md`, compatibility policy, and user-facing docs
- report schema or rendered outputs: schema docs, report tests, golden fixtures
- release process: `docs/releasing.md`, changelog, and upgrade notes
- daemon/worker: `docs/daemon-worker.md` and compatibility policy

## Special Cases

### Mutation tool comparisons

If your change affects comparisons with Gremlins, gomu, go-mutesting, or other
external mutation tools, follow:

- [docs/evaluations/tool-comparison-protocol.md](docs/evaluations/tool-comparison-protocol.md)
- [docs/evaluations/comparison-harness.md](docs/evaluations/comparison-harness.md)

Do not summarize those comparisons without recording target semantics and
denominator health.

### Experimental surfaces

Experimental surfaces are opt-in and do not yet carry backward-compatibility
guarantees. Do not promote them to supported behavior casually; align any
change with [docs/compatibility-policy.md](docs/compatibility-policy.md) first.
