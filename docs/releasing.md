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
3. builds release archives for the supported target set
4. writes `SHA256SUMS`
5. composes release notes from the changelog section and upgrade note file
6. verifies archive contents, checksums, and generated release notes
7. writes `release-manifest.json` for auditability
8. publishes or updates the GitHub release for that tag

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
