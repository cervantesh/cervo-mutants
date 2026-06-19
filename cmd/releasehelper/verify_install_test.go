package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyInstallAcceptsBinarySmoke(t *testing.T) {
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "bin", "cervomut")
	writeFile(t, binaryPath, "placeholder")

	runner := &fakeInstallRunner{}
	err := verifyInstall(installVerificationOptions{
		Binary:     binaryPath,
		SourceRoot: dir,
		Target:     "./pkg/config",
		WorkDir:    filepath.Join(dir, "work"),
		Label:      "go-install",
	}, runner)
	if err != nil {
		t.Fatalf("verifyInstall returned error: %v", err)
	}
	for _, want := range []string{
		"doctor",
		"list-mutators",
		"fast ./pkg/config --policy ci-fast --report summary,json,junit,html,sarif,github-summary --max-mutants 1 --workers 1",
		"report summary",
		"report survivors",
		"report recommendations",
		"report governance",
		"report history",
		"report sarif",
		"report github-summary",
	} {
		if !containsPrefix(runner.calls, want) {
			t.Fatalf("verifyInstall missing command prefix %q in %#v", want, runner.calls)
		}
	}
}

func TestVerifyInstallAcceptsArchiveSmoke(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "dist", "cervomut_v9.9.9_linux_amd64.tar.gz")
	target := releaseTarget{GOOS: "linux", GOARCH: "amd64", Format: "tar.gz"}
	writeReleaseArchive(t, archivePath, target, "v9.9.9")

	runner := &fakeInstallRunner{}
	err := verifyInstall(installVerificationOptions{
		Archive:    archivePath,
		SourceRoot: dir,
		Target:     "./pkg/config",
		WorkDir:    filepath.Join(dir, "work"),
		Label:      "archive-install",
	}, runner)
	if err != nil {
		t.Fatalf("verifyInstall returned error: %v", err)
	}
	if len(runner.binaries) == 0 || !strings.Contains(filepath.ToSlash(runner.binaries[0]), "/work/archive/") {
		t.Fatalf("verifyInstall did not run the extracted archive binary: %#v", runner.binaries)
	}
}

func TestVerifyInstallRejectsMissingReportArtifact(t *testing.T) {
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "bin", "cervomut")
	writeFile(t, binaryPath, "placeholder")

	runner := &fakeInstallRunner{omitFile: "mutation-report.sarif"}
	err := verifyInstall(installVerificationOptions{
		Binary:     binaryPath,
		SourceRoot: dir,
		Target:     "./pkg/config",
		WorkDir:    filepath.Join(dir, "work"),
	}, runner)
	if err == nil || !strings.Contains(err.Error(), "mutation-report.sarif") {
		t.Fatalf("expected missing artifact error, got %v", err)
	}
}

type fakeInstallRunner struct {
	calls    []string
	binaries []string
	omitFile string
}

func (f *fakeInstallRunner) Run(dir, binary string, args []string) (string, error) {
	f.calls = append(f.calls, strings.Join(args, " "))
	f.binaries = append(f.binaries, filepath.ToSlash(binary))
	if len(args) == 0 {
		return "", nil
	}
	switch args[0] {
	case "doctor":
		return "ok\n", nil
	case "list-mutators":
		return "[]\n", nil
	case "fast":
		outDir := findArgValue(args, "--out")
		if outDir == "" {
			return "", nil
		}
		required := map[string]string{
			"mutation-report.json":  `{"schema_version":"1"}`,
			"summary.txt":           "summary\n",
			"junit.xml":             "<testsuite></testsuite>\n",
			"index.html":            "<html></html>\n",
			"mutation-report.sarif": `{"version":"2.1.0"}`,
			"github-summary.md":     "summary\n",
		}
		for name, body := range required {
			if name == f.omitFile {
				continue
			}
			if err := writeInstallFixtureFile(filepath.Join(outDir, name), body); err != nil {
				return "", err
			}
		}
		return "", nil
	case "report":
		return "ok\n", nil
	default:
		return "", nil
	}
}

func containsPrefix(items []string, want string) bool {
	for _, item := range items {
		if strings.HasPrefix(item, want) {
			return true
		}
	}
	return false
}

func findArgValue(args []string, flag string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag {
			return args[i+1]
		}
	}
	return ""
}

func writeInstallFixtureFile(path, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o644)
}
