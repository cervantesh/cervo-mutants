package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type installVerificationOptions struct {
	Binary     string
	Archive    string
	SourceRoot string
	Target     string
	WorkDir    string
	Label      string
}

func cmdVerifyInstall(args []string) error {
	fs := flag.NewFlagSet("verify-install", flag.ContinueOnError)
	binary := fs.String("binary", "", "path to an installed cervomut binary to verify")
	archive := fs.String("archive", "", "path to a release archive to extract and verify")
	sourceRoot := fs.String("source-root", ".", "source tree used as the smoke target")
	target := fs.String("target", "./pkg/config", "target passed to cervomut fast")
	workDir := fs.String("work-dir", "", "scratch directory for install and report smoke verification")
	label := fs.String("label", "", "label used in output messages")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	return verifyInstall(installVerificationOptions{
		Binary:     *binary,
		Archive:    *archive,
		SourceRoot: *sourceRoot,
		Target:     *target,
		WorkDir:    *workDir,
		Label:      *label,
	}, execUpgradeRunner{})
}

func verifyInstall(opts installVerificationOptions, runner upgradeCommandRunner) error {
	if strings.TrimSpace(opts.Binary) == "" && strings.TrimSpace(opts.Archive) == "" {
		return fmt.Errorf("verify-install requires --binary or --archive")
	}
	if strings.TrimSpace(opts.Binary) != "" && strings.TrimSpace(opts.Archive) != "" {
		return fmt.Errorf("verify-install accepts either --binary or --archive, not both")
	}

	sourceRoot, err := filepath.Abs(opts.SourceRoot)
	if err != nil {
		return err
	}
	if _, err := os.Stat(sourceRoot); err != nil {
		return fmt.Errorf("source root %s: %w", filepath.ToSlash(sourceRoot), err)
	}

	workDir, err := prepareUpgradeWorkDir(opts.WorkDir)
	if err != nil {
		return err
	}
	label := strings.TrimSpace(opts.Label)
	if label == "" {
		label = "install"
	}
	target := strings.TrimSpace(opts.Target)
	if target == "" {
		target = "./pkg/config"
	}

	binaryPath, err := installVerificationBinary(opts, workDir)
	if err != nil {
		return err
	}
	if err := ensureExecutable(binaryPath); err != nil {
		return err
	}

	if err := runInstalledBinarySmoke(runner, binaryPath, sourceRoot, target, filepath.Join(workDir, "reports")); err != nil {
		return fmt.Errorf("%s compatibility smoke failed: %w", label, err)
	}

	fmt.Printf("verified %s compatibility using %s\n", label, filepath.ToSlash(binaryPath))
	fmt.Printf("install and report smoke workspace: %s\n", filepath.ToSlash(workDir))
	return nil
}

func installVerificationBinary(opts installVerificationOptions, workDir string) (string, error) {
	if strings.TrimSpace(opts.Binary) != "" {
		binaryPath, err := filepath.Abs(opts.Binary)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(binaryPath); err != nil {
			return "", fmt.Errorf("binary %s: %w", filepath.ToSlash(binaryPath), err)
		}
		return binaryPath, nil
	}

	archivePath, err := filepath.Abs(opts.Archive)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(archivePath); err != nil {
		return "", fmt.Errorf("archive %s: %w", filepath.ToSlash(archivePath), err)
	}
	extractRoot := filepath.Join(workDir, "archive")
	if err := os.MkdirAll(extractRoot, 0o755); err != nil {
		return "", err
	}
	if err := extractReleaseArchive(archivePath, extractRoot); err != nil {
		return "", err
	}
	return findCervomutBinary(extractRoot)
}

func runInstalledBinarySmoke(runner upgradeCommandRunner, binaryPath, sourceRoot, target, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	outArg := filepath.ToSlash(outDir)
	if _, err := runner.Run(sourceRoot, binaryPath, []string{"doctor"}); err != nil {
		return fmt.Errorf("doctor failed: %w", err)
	}
	if _, err := runner.Run(sourceRoot, binaryPath, []string{"list-mutators"}); err != nil {
		return fmt.Errorf("list-mutators failed: %w", err)
	}
	if _, err := runner.Run(sourceRoot, binaryPath, []string{
		"fast", target,
		"--policy", "ci-fast",
		"--report", "summary,json,junit,html,sarif,github-summary",
		"--max-mutants", "1",
		"--workers", "1",
		"--out", outArg,
	}); err != nil {
		return fmt.Errorf("fast run failed: %w", err)
	}
	if err := verifyInstalledReportFiles(outDir); err != nil {
		return err
	}

	reportCommands := [][]string{
		{"report", "summary", "--out", outArg},
		{"report", "survivors", "--out", outArg, "--actionable-only"},
		{"report", "recommendations", "--out", outArg},
		{"report", "governance", "--out", outArg},
		{"report", "history", "--out", outArg},
		{"report", "sarif", "--out", outArg},
		{"report", "github-summary", "--out", outArg},
	}
	for _, args := range reportCommands {
		if _, err := runner.Run(sourceRoot, binaryPath, args); err != nil {
			return fmt.Errorf("%s failed: %w", strings.Join(args[:2], " "), err)
		}
	}
	return nil
}

func verifyInstalledReportFiles(outDir string) error {
	required := []string{
		"mutation-report.json",
		"summary.txt",
		"junit.xml",
		"index.html",
		"mutation-report.sarif",
		"github-summary.md",
	}
	for _, name := range required {
		path := filepath.Join(outDir, name)
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("expected report artifact %s: %w", filepath.ToSlash(path), err)
		}
	}

	reportPath := filepath.Join(outDir, "mutation-report.json")
	body, err := os.ReadFile(reportPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", filepath.ToSlash(reportPath), err)
	}
	var report struct {
		SchemaVersion string `json:"schema_version"`
	}
	if err := json.Unmarshal(body, &report); err != nil {
		return fmt.Errorf("parse %s: %w", filepath.ToSlash(reportPath), err)
	}
	if report.SchemaVersion != "1" {
		return fmt.Errorf("%s schema_version = %q, want %q", filepath.ToSlash(reportPath), report.SchemaVersion, "1")
	}
	return nil
}
