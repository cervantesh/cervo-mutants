# Install And Upgrade

Tracking issue: #160

CervoMutants supports three practical installation paths today:

1. `go install` for the fastest cross-platform setup
2. GitHub release archives for pinned binary installs
3. local source builds when working inside the repository

## Option 1: `go install`

This is the simplest path on Windows, Linux, and macOS when Go is already
available:

```powershell
go install github.com/cervantesh/cervo-mutants/cmd/cervomut@latest
```

Pin a specific version when you need reproducible installs:

```powershell
go install github.com/cervantesh/cervo-mutants/cmd/cervomut@v0.4.1
```

Upgrade later with the same command, replacing the version suffix as needed.

## Option 2: GitHub Release Archives

The release workflow publishes archive names in this shape:

- `cervomut_<version>_linux_amd64.tar.gz`
- `cervomut_<version>_linux_arm64.tar.gz`
- `cervomut_<version>_darwin_amd64.tar.gz`
- `cervomut_<version>_darwin_arm64.tar.gz`
- `cervomut_<version>_windows_amd64.zip`
- `SHA256SUMS`

Replace `<version>` below with the tag you want, for example `v0.4.1`.

### Windows

```powershell
$version = "v0.4.1"
$name = "cervomut_${version}_windows_amd64.zip"
Invoke-WebRequest "https://github.com/cervantesh/cervo-mutants/releases/download/$version/$name" -OutFile $name
Expand-Archive $name -DestinationPath cervomut-install -Force
```

The extracted archive contains `cervomut.exe` plus the shipped license and
release metadata files.

### Linux

```bash
version=v0.4.1
name="cervomut_${version}_linux_amd64.tar.gz"
curl -LO "https://github.com/cervantesh/cervo-mutants/releases/download/${version}/${name}"
tar -xzf "${name}"
```

### macOS

```bash
version=v0.4.1
name="cervomut_${version}_darwin_arm64.tar.gz"
curl -LO "https://github.com/cervantesh/cervo-mutants/releases/download/${version}/${name}"
tar -xzf "${name}"
```

Pick the archive that matches your CPU architecture.

## Verify Release Downloads

Release archives are accompanied by `SHA256SUMS`.

After downloading the archive and checksum file, verify the checksum before
placing the binary on your normal PATH.

## Option 3: Build From Source

From this repository:

```powershell
go build ./cmd/cervomut
```

This is useful for local development, branch testing, or validating a change
before a public release is tagged.

Before a public release is published, the release workflow now validates both
the documented `go install` path and the Linux archive-install path with a real
installed binary, including report generation plus `cervomut report ...`
commands against the generated artifacts.

## Upgrade Guidance

Use one installation family consistently:

- if you installed with `go install`, upgrade with `go install`
- if you installed from a release archive, replace the binary with a newer
  archive from the next release tag
- if you build from source, rebuild from the target branch or tag you want

For CLI or report-surface changes, also read:

- [docs/releasing.md](releasing.md)
- [docs/compatibility-policy.md](compatibility-policy.md)
- [docs/upgrade-notes/README.md](upgrade-notes/README.md)
