# Go And OS Compatibility Matrix

Tracking issue: https://github.com/cervantesh/cervo-mutants/issues/91

CervoMutants is sensitive to toolchain and filesystem behavior because mutation
runs execute many `go test` commands, may use Go overlay files, and generate
reports and coverage artifacts across OS-specific path semantics.

The official support claim is therefore tied to an explicit CI-validated
matrix, not only to local anecdotal success.

The repository also enforces this matrix directly in CI and release preparation
through:

```powershell
go run ./cmd/releasehelper verify-compat
```

That check fails if `go.mod`, this document, or the main test/release workflows
drift away from the supported Go/OS matrix.

## Official Matrix

| OS | Go version | Status | Automated validation | Notes |
| --- | --- | --- | --- | --- |
| Linux | `1.25.x` | Supported | GitHub Actions core lane: `go vet`, full `go test`, race tests, CLI smoke | Primary validation lane. |
| Windows | `1.25.x` | Supported | GitHub Actions compatibility smoke | Validates CLI and report generation on Windows-native paths. |
| macOS | `1.25.x` | Supported | GitHub Actions compatibility smoke | Validates CLI and report generation on macOS. |
| Any OS | `< 1.25` | Unsupported | Not validated | Current `go.mod` baseline is `go 1.25.6`. |
| Any OS | `> 1.25` | Untested future | Not validated yet | `cervomut doctor` warns until explicitly added to the matrix. |

## What The Workflows Validate

The primary Linux lane validates:

- `go vet ./...`
- `go test ./...`
- `go test -race ./pkg/engine ./pkg/mutator ./pkg/report`
- `go run ./cmd/cervomut list-mutators`
- bounded CLI smoke runs that write and re-read mutation reports

The Windows and macOS compatibility lanes validate:

- `go test ./...`
- `go run ./cmd/cervomut doctor`
- `go run ./cmd/cervomut list-mutators`
- a bounded `cervomut fast` smoke run
- `cervomut report summary` against the generated report

That split is intentional: all supported cells get real automated validation,
but only the primary Linux lane pays the cost of the deeper race and broader
mutation smoke checks.

## Doctor Checks

`cervomut doctor` reports:

- `go version`
- `go-version-compatibility`
- `go-overlay`
- `go-env`
- `go-toolchain-auto` warning when `GOTOOLCHAIN=auto`
- `go-flags-coverprofile` warning when global `GOFLAGS` can conflict with
  CervoMutants coverage output
- runtime OS/arch/temp metadata
- Windows path and resource-control warnings
- WSL/systemd-run guidance when applicable

Example:

```powershell
cervomut doctor
```

## Reproducibility Guidance

For CI and release automation, prefer pinned toolchains over implicit downloads:

```powershell
$env:GOTOOLCHAIN = "go1.25.6"
go test ./...
cervomut fast ./... --budget 10m
```

Avoid setting global `GOFLAGS=-coverprofile=...` for mutation jobs. CervoMutants
manages coverage profiles itself when `selection.mode: coverage` or
`selection.prefilter: true` is enabled.

## Overlay Support

Go overlay isolation requires Go 1.14 or newer. The current supported matrix is
already above that requirement, so an overlay failure usually points to command
flags, path handling, or filesystem behavior rather than the Go version itself.
