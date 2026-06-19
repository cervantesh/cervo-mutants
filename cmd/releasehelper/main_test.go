package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cervantesh/cervo-mutants/internal/compatmatrix"
)

func TestExtractMarkdownSection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "CHANGELOG.md")
	body := `# Changelog

## [Unreleased]

## [v0.3.0] - 2026-06-17
### Added
- item one

## [v0.2.0] - 2026-06-17
- older
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	section, err := extractMarkdownSection(path, "v0.3.0")
	if err != nil {
		t.Fatalf("extractMarkdownSection returned error: %v", err)
	}
	for _, want := range []string{"### Added", "- item one"} {
		if !strings.Contains(section, want) {
			t.Fatalf("section missing %q: %s", want, section)
		}
	}
}

func TestBuildReleaseNotesStripsUpgradeHeading(t *testing.T) {
	notes := buildReleaseNotes("v0.3.0", "### Added\n- item one", "# Upgrade Notes for v0.3.0\n\n- migrate config\n")
	for _, want := range []string{"# CervoMutants v0.3.0", "## Changelog", "## Upgrade Notes", "- migrate config"} {
		if !strings.Contains(notes, want) {
			t.Fatalf("release notes missing %q:\n%s", want, notes)
		}
	}
	if strings.Contains(notes, "# Upgrade Notes for v0.3.0") {
		t.Fatalf("release notes kept top-level upgrade heading:\n%s", notes)
	}
}

func TestCmdNotesWritesOutput(t *testing.T) {
	dir := t.TempDir()
	changelogPath := filepath.Join(dir, "CHANGELOG.md")
	upgradeDir := filepath.Join(dir, "upgrade")
	if err := os.MkdirAll(upgradeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(changelogPath, []byte("# Changelog\n\n## [v0.3.0] - 2026-06-17\n- note\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(upgradeDir, "v0.3.0.md"), []byte("# Upgrade Notes\n\n- migrate\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(dir, "dist", "release-notes.md")
	if err := cmdNotes([]string{"--version", "v0.3.0", "--changelog", changelogPath, "--upgrade-dir", upgradeDir, "--out", outPath}); err != nil {
		t.Fatalf("cmdNotes returned error: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read release notes: %v", err)
	}
	if !strings.Contains(string(data), "- note") || !strings.Contains(string(data), "- migrate") {
		t.Fatalf("unexpected release notes body:\n%s", data)
	}
}

func TestCmdVerifyCompatAcceptsAlignedFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module fixture\n\ngo "+compatmatrix.SupportedGoVersion+"\n")
	writeFile(t, filepath.Join(dir, "docs", "go-toolchain-compatibility.md"), "# Go And OS Compatibility Matrix\n\n"+
		"| OS | Go version | Status | Automated validation | Notes |\n"+
		"| --- | --- | --- | --- | --- |\n"+
		"| Linux | `1.25.x` | Supported | GitHub Actions core lane | Primary validation lane. |\n"+
		"| Windows | `1.25.x` | Supported | GitHub Actions compatibility smoke | Windows lane. |\n"+
		"| macOS | `1.25.x` | Supported | GitHub Actions compatibility smoke | macOS lane. |\n"+
		"| Any OS | `< 1.25` | Unsupported | Not validated | Current `go.mod` baseline is `go 1.25.6`. |\n")
	writeFile(t, filepath.Join(dir, ".github", "workflows", "test.yml"), `jobs:
  core-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25.6"
  compatibility-smoke:
    strategy:
      matrix:
        include:
          - os: ubuntu-latest
            go-version: "1.25.6"
          - os: windows-latest
            go-version: "1.25.6"
          - os: macos-latest
            go-version: "1.25.6"
  github-action-smoke:
    runs-on: ubuntu-latest
    steps:
      - name: Run local GitHub Action source
        uses: ./
        with:
          go-version: "1.25.6"
`)
	writeFile(t, filepath.Join(dir, ".github", "workflows", "release.yml"), `jobs:
  compatibility-smoke:
    strategy:
      matrix:
        include:
          - os: ubuntu-latest
            go-version: "1.25.6"
          - os: windows-latest
            go-version: "1.25.6"
          - os: macos-latest
            go-version: "1.25.6"
  publish:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25.6"
      - run: go run ./cmd/releasehelper verify-compat
      - run: GOBIN="$PWD/.verify-install/gobin" GOPROXY=direct GOSUMDB=off go install github.com/cervantesh/cervo-mutants/cmd/cervomut@${GITHUB_REF_NAME}
      - run: go run ./cmd/releasehelper verify-install --binary .verify-install/gobin/cervomut
      - run: go run ./cmd/releasehelper verify-release
      - run: go run ./cmd/releasehelper verify-install --archive dist/cervomut_${GITHUB_REF_NAME}_linux_amd64.tar.gz
`)

	err := cmdVerifyCompat([]string{
		"--go-mod", filepath.Join(dir, "go.mod"),
		"--doc", filepath.Join(dir, "docs", "go-toolchain-compatibility.md"),
		"--test-workflow", filepath.Join(dir, ".github", "workflows", "test.yml"),
		"--release-workflow", filepath.Join(dir, ".github", "workflows", "release.yml"),
	})
	if err != nil {
		t.Fatalf("cmdVerifyCompat returned error: %v", err)
	}
}

func TestCmdVerifyCompatRejectsMismatchedWorkflowVersion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module fixture\n\ngo "+compatmatrix.SupportedGoVersion+"\n")
	writeFile(t, filepath.Join(dir, "docs", "go-toolchain-compatibility.md"), "| Linux | `1.25.x` | Supported |\n"+
		"| Windows | `1.25.x` | Supported |\n"+
		"| macOS | `1.25.x` | Supported |\n"+
		"Current `go.mod` baseline is `go 1.25.6`.\n")
	writeFile(t, filepath.Join(dir, ".github", "workflows", "test.yml"), `jobs:
  core-tests:
    steps:
      - uses: actions/setup-go@v5
        with:
          go-version: "1.24.9"
  compatibility-smoke:
    strategy:
      matrix:
        include:
          - os: ubuntu-latest
            go-version: "1.25.6"
          - os: windows-latest
            go-version: "1.25.6"
          - os: macos-latest
            go-version: "1.25.6"
  github-action-smoke:
    steps:
      - name: Run local GitHub Action source
        uses: ./
        with:
          go-version: "1.25.6"
`)
	writeFile(t, filepath.Join(dir, ".github", "workflows", "release.yml"), `jobs:
  compatibility-smoke:
    strategy:
      matrix:
        include:
          - os: ubuntu-latest
            go-version: "1.25.6"
          - os: windows-latest
            go-version: "1.25.6"
          - os: macos-latest
            go-version: "1.25.6"
  publish:
    steps:
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25.6"
      - run: go run ./cmd/releasehelper verify-release
`)

	err := cmdVerifyCompat([]string{
		"--go-mod", filepath.Join(dir, "go.mod"),
		"--doc", filepath.Join(dir, "docs", "go-toolchain-compatibility.md"),
		"--test-workflow", filepath.Join(dir, ".github", "workflows", "test.yml"),
		"--release-workflow", filepath.Join(dir, ".github", "workflows", "release.yml"),
	})
	if err == nil || !strings.Contains(err.Error(), "core-tests") {
		t.Fatalf("expected core-tests mismatch error, got %v", err)
	}
}

func TestCmdVerifyCompatRejectsMissingInstallGate(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module fixture\n\ngo "+compatmatrix.SupportedGoVersion+"\n")
	writeFile(t, filepath.Join(dir, "docs", "go-toolchain-compatibility.md"), "# Go And OS Compatibility Matrix\n\n"+
		"| OS | Go version | Status | Automated validation | Notes |\n"+
		"| --- | --- | --- | --- | --- |\n"+
		"| Linux | `1.25.x` | Supported | GitHub Actions core lane | Primary validation lane. |\n"+
		"| Windows | `1.25.x` | Supported | GitHub Actions compatibility smoke | Windows lane. |\n"+
		"| macOS | `1.25.x` | Supported | GitHub Actions compatibility smoke | macOS lane. |\n"+
		"| Any OS | `< 1.25` | Unsupported | Not validated | Current `go.mod` baseline is `go 1.25.6`. |\n")
	writeFile(t, filepath.Join(dir, ".github", "workflows", "test.yml"), `jobs:
  core-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25.6"
  compatibility-smoke:
    strategy:
      matrix:
        include:
          - os: ubuntu-latest
            go-version: "1.25.6"
          - os: windows-latest
            go-version: "1.25.6"
          - os: macos-latest
            go-version: "1.25.6"
  github-action-smoke:
    runs-on: ubuntu-latest
    steps:
      - name: Run local GitHub Action source
        uses: ./
        with:
          go-version: "1.25.6"
`)
	writeFile(t, filepath.Join(dir, ".github", "workflows", "release.yml"), `jobs:
  compatibility-smoke:
    strategy:
      matrix:
        include:
          - os: ubuntu-latest
            go-version: "1.25.6"
          - os: windows-latest
            go-version: "1.25.6"
          - os: macos-latest
            go-version: "1.25.6"
  publish:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25.6"
      - run: go run ./cmd/releasehelper verify-compat
      - run: go run ./cmd/releasehelper verify-release
`)
	err := cmdVerifyCompat([]string{
		"--go-mod", filepath.Join(dir, "go.mod"),
		"--doc", filepath.Join(dir, "docs", "go-toolchain-compatibility.md"),
		"--test-workflow", filepath.Join(dir, ".github", "workflows", "test.yml"),
		"--release-workflow", filepath.Join(dir, ".github", "workflows", "release.yml"),
	})
	if err == nil || !strings.Contains(err.Error(), "verify-install") {
		t.Fatalf("expected verify-install gate error, got %v", err)
	}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
