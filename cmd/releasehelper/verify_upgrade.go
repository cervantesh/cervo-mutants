package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cervantesh/cervo-mutants/internal/compatmatrix"
	"github.com/cervantesh/cervo-mutants/pkg/baseline"
	"github.com/cervantesh/cervo-mutants/pkg/engine"
)

type upgradeVerificationOptions struct {
	PreviousVersion string
	PreviousArchive string
	CurrentBinary   string
	WorkDir         string
}

type upgradeCommandRunner interface {
	Run(dir, binary string, args []string) (string, error)
}

type execUpgradeRunner struct{}

const upgradeMutationReportFileName = "mutation-report.json"

func cmdVerifyUpgrade(args []string) error {
	fs := flag.NewFlagSet("verify-upgrade", flag.ContinueOnError)
	previousVersion := fs.String("previous-version", "", "latest supported public release version, for example v0.3.0")
	previousArchive := fs.String("previous-archive", "", "path to a previous release archive such as cervomut_v0.3.0_linux_amd64.tar.gz")
	currentBinary := fs.String("current-binary", "", "path to the current cervomut binary to verify")
	workDir := fs.String("work-dir", "", "scratch directory for the upgrade smoke workspace; defaults to a temp directory")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	return verifyUpgrade(upgradeVerificationOptions{
		PreviousVersion: *previousVersion,
		PreviousArchive: *previousArchive,
		CurrentBinary:   *currentBinary,
		WorkDir:         *workDir,
	}, execUpgradeRunner{})
}

func verifyUpgrade(opts upgradeVerificationOptions, runner upgradeCommandRunner) error {
	if strings.TrimSpace(opts.PreviousVersion) == "" {
		return fmt.Errorf("verify-upgrade requires --previous-version")
	}
	if strings.TrimSpace(opts.PreviousArchive) == "" {
		return fmt.Errorf("verify-upgrade requires --previous-archive")
	}
	if strings.TrimSpace(opts.CurrentBinary) == "" {
		return fmt.Errorf("verify-upgrade requires --current-binary")
	}
	currentBinary, err := filepath.Abs(opts.CurrentBinary)
	if err != nil {
		return err
	}
	if _, err := os.Stat(currentBinary); err != nil {
		return fmt.Errorf("current binary %s: %w", filepath.ToSlash(currentBinary), err)
	}

	workDir, err := prepareUpgradeWorkDir(opts.WorkDir)
	if err != nil {
		return err
	}
	previousRoot := filepath.Join(workDir, "previous")
	if err := os.MkdirAll(previousRoot, 0o755); err != nil {
		return err
	}
	if err := extractReleaseArchive(opts.PreviousArchive, previousRoot); err != nil {
		return err
	}
	previousBinary, err := findCervomutBinary(previousRoot)
	if err != nil {
		return err
	}
	if err := ensureExecutable(previousBinary); err != nil {
		return err
	}
	if err := ensureExecutable(currentBinary); err != nil {
		return err
	}

	fixtureDir := filepath.Join(workDir, "fixture")
	if err := writeUpgradeFixture(fixtureDir); err != nil {
		return err
	}
	reportDir := filepath.Join(fixtureDir, ".cervomut", "reports")
	baselinePath := filepath.Join(fixtureDir, ".cervomut", "baseline.json")
	if err := runUpgradeSmokeWorkflow(runner, previousBinary, currentBinary, fixtureDir, reportDir, baselinePath); err != nil {
		return err
	}

	fmt.Printf("verified upgrade compatibility from %s using %s and %s\n", opts.PreviousVersion, filepath.ToSlash(previousBinary), filepath.ToSlash(currentBinary))
	fmt.Printf("upgrade smoke workspace: %s\n", filepath.ToSlash(workDir))
	return nil
}

func (execUpgradeRunner) Run(dir, binary string, args []string) (string, error) {
	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%s %s failed: %w\n%s", filepath.Base(binary), strings.Join(args, " "), err, string(out))
	}
	return string(out), nil
}

func prepareUpgradeWorkDir(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return os.MkdirTemp("", "cervomut-upgrade-*")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if abs == filepath.Dir(abs) {
		return "", fmt.Errorf("refusing to clear root directory %s", filepath.ToSlash(abs))
	}
	if err := os.RemoveAll(abs); err != nil {
		return "", err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return "", err
	}
	return abs, nil
}

func extractReleaseArchive(path, dest string) error {
	switch {
	case strings.HasSuffix(path, ".zip"):
		return extractZipArchive(path, dest)
	case strings.HasSuffix(path, ".tar.gz"):
		return extractTarGzArchive(path, dest)
	default:
		return fmt.Errorf("unsupported release archive %s", filepath.ToSlash(path))
	}
}

func extractZipArchive(path, dest string) error {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("open zip %s: %w", filepath.ToSlash(path), err)
	}
	defer reader.Close()
	for _, file := range reader.File {
		target, err := archiveExtractionTarget(dest, file.Name)
		if err != nil {
			return err
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := file.Open()
		if err != nil {
			return err
		}
		if err := writeExtractedFile(target, rc, file.Mode()); err != nil {
			rc.Close()
			return err
		}
		if err := rc.Close(); err != nil {
			return err
		}
	}
	return nil
}

func extractTarGzArchive(path, dest string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open tarball %s: %w", filepath.ToSlash(path), err)
	}
	defer f.Close()
	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("open gzip %s: %w", filepath.ToSlash(path), err)
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read tarball %s: %w", filepath.ToSlash(path), err)
		}
		target, err := archiveExtractionTarget(dest, hdr.Name)
		if err != nil {
			return err
		}
		if hdr.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := writeExtractedFile(target, tr, hdr.FileInfo().Mode()); err != nil {
			return err
		}
	}
}

func writeExtractedFile(path string, r io.Reader, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode|0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

func findCervomutBinary(root string) (string, error) {
	var found string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if name == "cervomut" || name == "cervomut.exe" {
			found = path
			return io.EOF
		}
		return nil
	})
	if err != nil && err != io.EOF {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("could not locate cervomut binary under %s", filepath.ToSlash(root))
	}
	return found, nil
}

func ensureExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	mode := info.Mode()
	if mode&0o111 != 0 {
		return nil
	}
	return os.Chmod(path, mode|0o755)
}

func archiveExtractionTarget(dest, name string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(name))
	if clean == "." {
		return "", fmt.Errorf("archive entry %q has no file path", name)
	}
	target := filepath.Join(dest, clean)
	rel, err := filepath.Rel(dest, target)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("archive entry %q escapes extraction root %s", name, filepath.ToSlash(dest))
	}
	return target, nil
}

func writeUpgradeFixture(dir string) error {
	if err := os.MkdirAll(filepath.Join(dir, "sample"), 0o755); err != nil {
		return err
	}
	files := map[string]string{
		"go.mod": "module example.com/cervomut-upgrade-fixture\n\ngo " + compatmatrix.SupportedGoVersion + "\n",
		"cervomut.yaml": `version: 1
tests:
  command: ["go", "test", "./..."]
  timeout: 10s
execution:
  workers: 1
  isolation: overlay
baseline:
  path: .cervomut/baseline.json
reports:
  output: .cervomut/reports
`,
		filepath.Join("sample", "sample.go"): `package sample

func Classify(v int) string {
	if v > 0 {
		return "positive"
	}
	return "zero"
}
`,
		filepath.Join("sample", "sample_test.go"): `package sample

import "testing"

func TestClassify(t *testing.T) {
	cases := []struct {
		name string
		in   int
		want string
	}{
		{name: "positive", in: 1, want: "positive"},
		{name: "zero", in: 0, want: "zero"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Classify(tc.in); got != tc.want {
				t.Fatalf("Classify(%d) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
`,
	}
	for rel, body := range files {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func runUpgradeSmokeWorkflow(runner upgradeCommandRunner, previousBinary, currentBinary, fixtureDir, reportDir, baselinePath string) error {
	reportArg := filepath.ToSlash(filepath.Join(".cervomut", "reports"))
	baseRunArgs := []string{"run", "./...", "--policy", "ci-fast", "--max-mutants", "1", "--workers", "1", "--sample", "deterministic", "--out", reportArg}

	if _, err := runner.Run(fixtureDir, previousBinary, baseRunArgs); err != nil {
		return fmt.Errorf("previous release run failed: %w", err)
	}
	if _, err := os.Stat(filepath.Join(reportDir, upgradeMutationReportFileName)); err != nil {
		return fmt.Errorf("previous release did not produce %s: %w", filepath.ToSlash(filepath.Join(reportDir, upgradeMutationReportFileName)), err)
	}
	if _, err := runner.Run(fixtureDir, previousBinary, []string{"baseline", "update"}); err != nil {
		return fmt.Errorf("previous release baseline update failed: %w", err)
	}
	if _, err := os.Stat(baselinePath); err != nil {
		return fmt.Errorf("previous release did not produce %s: %w", filepath.ToSlash(baselinePath), err)
	}

	if _, err := runner.Run(fixtureDir, currentBinary, []string{"report", "summary", "--out", reportArg}); err != nil {
		return fmt.Errorf("current report summary failed against previous release artifacts: %w", err)
	}
	if _, err := runner.Run(fixtureDir, currentBinary, []string{"report", "survivors", "--out", reportArg, "--actionable-only"}); err != nil {
		return fmt.Errorf("current actionable survivor view failed against previous release artifacts: %w", err)
	}
	if _, err := runner.Run(fixtureDir, currentBinary, baseRunArgs); err != nil {
		return fmt.Errorf("current run failed after upgrade: %w", err)
	}

	compareOut, err := runner.Run(fixtureDir, currentBinary, []string{"baseline", "compare"})
	if err != nil {
		return fmt.Errorf("current baseline compare failed against previous baseline: %w", err)
	}
	var comparison engine.BaselineComparison
	if err := json.Unmarshal([]byte(compareOut), &comparison); err != nil {
		return fmt.Errorf("parse baseline compare output: %w", err)
	}
	if !comparison.Enabled {
		return fmt.Errorf("baseline compare output did not enable comparison")
	}

	diffOut, err := runner.Run(fixtureDir, currentBinary, []string{"baseline", "diff", "--json"})
	if err != nil {
		return fmt.Errorf("current baseline diff failed against previous baseline: %w", err)
	}
	var diff baseline.Diff
	if err := json.Unmarshal([]byte(diffOut), &diff); err != nil {
		return fmt.Errorf("parse baseline diff output: %w", err)
	}
	if !diff.BaselineFound {
		return fmt.Errorf("baseline diff output did not confirm baseline availability")
	}

	if _, err := runner.Run(fixtureDir, currentBinary, []string{"baseline", "accept"}); err != nil {
		return fmt.Errorf("current baseline accept failed: %w", err)
	}
	candidatePath := baseline.CandidatePath(baselinePath)
	if _, err := os.Stat(candidatePath); err != nil {
		return fmt.Errorf("current baseline accept did not produce %s: %w", filepath.ToSlash(candidatePath), err)
	}
	if _, err := runner.Run(fixtureDir, currentBinary, []string{"baseline", "promote"}); err != nil {
		return fmt.Errorf("current baseline promote failed: %w", err)
	}
	if _, err := os.Stat(candidatePath); !os.IsNotExist(err) {
		if err == nil {
			return fmt.Errorf("current baseline promote left candidate file behind: %s", filepath.ToSlash(candidatePath))
		}
		return err
	}
	if _, err := os.Stat(baselinePath); err != nil {
		return fmt.Errorf("current baseline promote left no baseline file: %w", err)
	}
	return nil
}
