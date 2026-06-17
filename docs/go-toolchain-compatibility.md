# Go Toolchain Compatibility

Tracking issue: https://github.com/cervantesh/CervoMutants/issues/21

CervoMutants is sensitive to Go toolchain behavior because mutation runs execute
many `go test` commands, may use Go overlay files, and may generate coverage
profiles for test selection.

## Compatibility Matrix

| Go version | Status | Notes |
| --- | --- | --- |
| `1.25.x` | Tested | Matches the current supported target toolchain. |
| `1.24.x` | Compatible | Expected to work for normal runs; keep CI coverage if used by a project. |
| `< 1.24` | Unsupported | `cervomut doctor` fails the compatibility check. |
| `> 1.25` | Untested future | `cervomut doctor` warns until validated. |

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

For CI, prefer pinned toolchains over implicit downloads:

```powershell
$env:GOTOOLCHAIN = "go1.25.6"
go test ./...
cervomut fast ./... --budget 10m
```

Avoid setting global `GOFLAGS=-coverprofile=...` for mutation jobs. CervoMutants
manages coverage profiles itself when `selection.mode: coverage` or
`selection.prefilter: true` is enabled.

## Overlay Support

Go overlay isolation requires Go 1.14 or newer. The supported matrix is already
above that requirement, so an overlay failure usually points to command flags,
path handling, or filesystem behavior rather than the Go version itself.

