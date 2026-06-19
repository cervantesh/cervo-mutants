package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cervantesh/cervo-mutants/pkg/baseline"
)

func TestVerifyUpgradeAcceptsPreviousReleaseArtifacts(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "dist", "cervomut_v0.3.0_linux_amd64.tar.gz")
	target := releaseTarget{GOOS: "linux", GOARCH: "amd64", Format: "tar.gz"}
	writeReleaseArchive(t, archivePath, target, "v0.3.0")

	currentBinary := filepath.Join(dir, "current", "cervomut")
	writeFile(t, currentBinary, "placeholder")

	runner := &fakeUpgradeRunner{currentBinary: currentBinary}
	err := verifyUpgrade(upgradeVerificationOptions{
		PreviousVersion: "v0.3.0",
		PreviousArchive: archivePath,
		CurrentBinary:   currentBinary,
		WorkDir:         filepath.Join(dir, "work"),
	}, runner)
	if err != nil {
		t.Fatalf("verifyUpgrade returned error: %v", err)
	}
	if len(runner.calls) != 9 {
		t.Fatalf("verifyUpgrade ran %d commands, want 9: %#v", len(runner.calls), runner.calls)
	}
	for _, want := range []string{
		"run ./... --policy ci-fast --max-mutants 1 --workers 1 --sample deterministic --out .cervomut/reports",
		"baseline update",
		"report summary --out .cervomut/reports",
		"report survivors --out .cervomut/reports --actionable-only",
		"baseline compare",
		"baseline diff --json",
		"baseline accept",
		"baseline promote",
	} {
		if !containsString(runner.calls, want) {
			t.Fatalf("verifyUpgrade missing command %q in %#v", want, runner.calls)
		}
	}
}

func TestVerifyUpgradeRejectsMissingPreviousReport(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "dist", "cervomut_v0.3.0_linux_amd64.tar.gz")
	target := releaseTarget{GOOS: "linux", GOARCH: "amd64", Format: "tar.gz"}
	writeReleaseArchive(t, archivePath, target, "v0.3.0")

	currentBinary := filepath.Join(dir, "current", "cervomut")
	writeFile(t, currentBinary, "placeholder")

	runner := &fakeUpgradeRunner{currentBinary: currentBinary, skipPreviousReport: true}
	err := verifyUpgrade(upgradeVerificationOptions{
		PreviousVersion: "v0.3.0",
		PreviousArchive: archivePath,
		CurrentBinary:   currentBinary,
		WorkDir:         filepath.Join(dir, "work"),
	}, runner)
	if err == nil || !strings.Contains(err.Error(), "did not produce") {
		t.Fatalf("expected missing previous report error, got %v", err)
	}
}

func TestEnsureExecutableAddsExecBits(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("Windows does not expose POSIX execute bits through os.Stat")
	}
	path := filepath.Join(t.TempDir(), "cervomut")
	if err := os.WriteFile(path, []byte("placeholder"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ensureExecutable(path); err != nil {
		t.Fatalf("ensureExecutable returned error: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("ensureExecutable did not set execute bits: mode=%#o", info.Mode().Perm())
	}
}

func TestExtractReleaseArchiveRejectsEscapingZipEntry(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "escape.zip")
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("../escape.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("bad")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	err = extractReleaseArchive(archivePath, filepath.Join(dir, "out"))
	if err == nil || !strings.Contains(err.Error(), "escapes extraction root") {
		t.Fatalf("expected archive escape error, got %v", err)
	}
}

func TestExtractReleaseArchiveRejectsEscapingTarEntry(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "escape.tar.gz")
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	gzw := gzip.NewWriter(f)
	tw := tar.NewWriter(gzw)
	body := []byte("bad")
	hdr := &tar.Header{Name: "../escape.txt", Mode: 0o644, Size: int64(len(body))}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	err = extractReleaseArchive(archivePath, filepath.Join(dir, "out"))
	if err == nil || !strings.Contains(err.Error(), "escapes extraction root") {
		t.Fatalf("expected archive escape error, got %v", err)
	}
}

type fakeUpgradeRunner struct {
	currentBinary      string
	skipPreviousReport bool
	calls              []string
}

func (f *fakeUpgradeRunner) Run(dir, binary string, args []string) (string, error) {
	cmd := strings.Join(args, " ")
	f.calls = append(f.calls, cmd)
	reportPath := filepath.Join(dir, ".cervomut", "reports", upgradeMutationReportFileName)
	baselinePath := filepath.Join(dir, ".cervomut", "baseline.json")
	candidatePath := baseline.CandidatePath(baselinePath)

	switch {
	case args[0] == "run" && binary != f.currentBinary:
		if !f.skipPreviousReport {
			if err := writeFixtureFile(reportPath, upgradeFixtureReportJSON); err != nil {
				return "", err
			}
		}
		return "", nil
	case len(args) == 2 && args[0] == "baseline" && args[1] == "update":
		data, err := os.ReadFile(reportPath)
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(filepath.Dir(baselinePath), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(baselinePath, data, 0o644); err != nil {
			return "", err
		}
		return "", nil
	case len(args) >= 2 && args[0] == "report" && args[1] == "summary":
		return "Raw score: 100.00%\n", nil
	case len(args) >= 2 && args[0] == "report" && args[1] == "survivors":
		return "No actionable survivors.\n", nil
	case args[0] == "run" && binary == f.currentBinary:
		if err := writeFixtureFile(reportPath, upgradeFixtureReportJSON); err != nil {
			return "", err
		}
		return "", nil
	case len(args) == 2 && args[0] == "baseline" && args[1] == "compare":
		return `{"enabled":true,"previous_score":100,"current_score":100,"regression":false}`, nil
	case len(args) == 3 && args[0] == "baseline" && args[1] == "diff" && args[2] == "--json":
		return `{"baseline_found":true,"previous_score":100,"current_score":100,"score_delta":0,"previous_actionable_score":100,"current_actionable_score":100,"actionable_score_delta":0}`, nil
	case len(args) == 2 && args[0] == "baseline" && args[1] == "accept":
		data, err := os.ReadFile(reportPath)
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(filepath.Dir(candidatePath), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(candidatePath, data, 0o644); err != nil {
			return "", err
		}
		return "", nil
	case len(args) == 2 && args[0] == "baseline" && args[1] == "promote":
		data, err := os.ReadFile(candidatePath)
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(baselinePath, data, 0o644); err != nil {
			return "", err
		}
		if err := os.Remove(candidatePath); err != nil {
			return "", err
		}
		return "", nil
	default:
		return "", nil
	}
}

const upgradeFixtureReportJSON = `{
  "schema_version": "1",
  "summary": {
    "score": 100,
    "actionable": {
      "actionable_score": 100
    }
  },
  "mutants": []
}`

func writeFixtureFile(path, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o644)
}
