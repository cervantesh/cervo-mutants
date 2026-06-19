# Releasing

CervoMutants releases are tag-driven.

The release workflow publishes:

- cross-platform binary archives
- `SHA256SUMS`
- `release-manifest.json`
- GitHub release notes composed from `CHANGELOG.md`
- per-version upgrade notes from `docs/upgrade-notes/`

## Before Tagging

1. Update `CHANGELOG.md` with a new `## [vX.Y.Z] - YYYY-MM-DD` section.
2. Add `docs/upgrade-notes/vX.Y.Z.md`.
3. If the release changes a supported CLI or report surface, update
   [compatibility-policy.md](compatibility-policy.md) or confirm the change is
   already covered by that policy.
4. Ensure `main` is green, including the upgrade smoke that exercises the
   latest public release archive against the current source tree.
5. Ensure the matching upgrade note contains `## Summary`,
   `## Operator Action`, and `## Rollback`.

## Create The Release

```powershell
git tag vX.Y.Z
git push origin vX.Y.Z
```

The GitHub Actions release workflow then:

1. runs `go vet ./...`
2. runs `go test ./...`
3. verifies the documented `go install` path and report-reading flow from an installed binary
4. builds release archives for the supported target set
5. writes `SHA256SUMS`
6. composes release notes from the changelog section and upgrade note file
7. verifies archive contents, checksums, and generated release notes
8. verifies the archive-install path and report-reading flow from the built Linux release archive
9. writes `release-manifest.json` for auditability
10. publishes or updates the GitHub release for that tag

## Artifacts

The workflow currently publishes:

- `cervomut_<version>_linux_amd64.tar.gz`
- `cervomut_<version>_linux_arm64.tar.gz`
- `cervomut_<version>_darwin_amd64.tar.gz`
- `cervomut_<version>_darwin_arm64.tar.gz`
- `cervomut_<version>_windows_amd64.zip`
- `SHA256SUMS`
- `release-manifest.json`

Each archive includes:

- the `cervomut` binary
- `LICENSE`
- `NOTICE`
- `TRADEMARKS.md`
- `README.md`
- `CHANGELOG.md`

If the changelog section or upgrade note file is missing, the workflow fails
instead of publishing an incomplete release.

The workflow also fails before publication if the documented `go install` or
archive-install path cannot run a bounded mutation lane and read back the
generated report surfaces from the installed binary.

## Release Evidence Trail

After a public release is published, update
[release-evidence-trail.md](release-evidence-trail.md) with:

- the release URL
- the published `release-manifest.json`
- the published `SHA256SUMS`
- the matching upgrade note
- the release-specific hosted-wave or validation evidence

That keeps the release-confidence story comparable across tags instead of
forcing maintainers to reconstruct it from scattered dated notes later.

## Rollback Guidance

Public release tags should be treated as immutable once announced.

Use this rollback posture:

1. Do not move or overwrite a published release tag.
2. If a release is bad but already public, direct users back to the previous
   known-good tag and publish a follow-up patch release instead of rewriting the
   broken tag.
3. If the issue is install-path specific, roll back using the same install
   family:
   - `go install github.com/cervantesh/cervo-mutants/cmd/cervomut@<previous>`
   - reinstall the previous GitHub release archive
4. If the release changed a supported CLI or report surface, use the matching
   per-version `## Rollback` section in `docs/upgrade-notes/vX.Y.Z.md` as the
   operator-facing rollback instruction set.

The goal is to make rollback explicit and auditable, not a tribal-memory step.

## Release-State Drift Check

After a public release is published, do one short release-state drift pass so
the live guidance stays aligned with the latest tag.

### Live surfaces to check

These should reflect the latest public release when they describe the current
install, CI, or maturity state:

- [docs/install.md](install.md)
- [docs/github-action.md](github-action.md)
- [docs/project-maturity-assessment.md](project-maturity-assessment.md)
- [README.md](../README.md) when it contains current release guidance
- [Adoption feedback issue template](../.github/ISSUE_TEMPLATE/adoption-feedback.yml) when it shows example version placeholders

### Historical surfaces to leave dated

These are evidence artifacts or versioned notes. Do not rewrite them just to
mirror the newest tag unless a brief clarifying note is necessary:

- [docs/evaluations/](evaluations/)
- [docs/upgrade-notes/](upgrade-notes/)
- [docs/distribution-audit.md](distribution-audit.md)
- dated release or validation notes under `docs/`

### Minimum post-release checklist

1. Confirm the latest tag and assets with `gh release view <tag>`.
2. Check live install examples for the current tag or the intended `@latest`
   posture.
3. Check GitHub Action examples and versioning notes for the latest supported
   tag.
4. Check any current maturity or release-state guidance for stale references to
   older "latest release" claims or already-closed active follow-ups.
5. Leave dated evidence artifacts alone unless readers would otherwise confuse
   them for current release guidance.

This check is intentionally small. The goal is to catch obvious live-doc drift
early without rewriting historical evidence or turning release publication into
an unbounded documentation sweep.
