package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const defaultModulePath = "github.com/cervantesh/cervo-mutants/cmd/cervomut"

type installPlan struct {
	ActionPath string `json:"action_path,omitempty"`
	Mode       string `json:"mode"`
	Target     string `json:"target,omitempty"`
	Version    string `json:"version,omitempty"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: actionhelper <install-plan|report-dir>")
	}
	switch args[0] {
	case "install-plan":
		return cmdInstallPlan(args[1:], stdout)
	case "report-dir":
		return cmdReportDir(args[1:], stdout)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func cmdInstallPlan(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("install-plan", flag.ContinueOnError)
	version := fs.String("version", "", "explicit cervomut version such as v0.3.0 or latest")
	actionPath := fs.String("action-path", os.Getenv("GITHUB_ACTION_PATH"), "composite action source path")
	actionRef := fs.String("action-ref", os.Getenv("GITHUB_ACTION_REF"), "ref pinned by the GitHub Action use site")
	modulePath := fs.String("module-path", defaultModulePath, "go install module path")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	plan, err := resolveInstallPlan(*modulePath, *version, *actionPath, *actionRef)
	if err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(plan)
}

func cmdReportDir(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("report-dir", flag.ContinueOnError)
	workspace := fs.String("workspace", os.Getenv("GITHUB_WORKSPACE"), "GitHub workspace root")
	workingDirectory := fs.String("working-directory", ".", "action working directory")
	outDir := fs.String("out", ".cervomut/reports", "report output directory")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	reportDir, err := resolveReportDir(*workspace, *workingDirectory, *outDir)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, reportDir)
	return err
}

func resolveInstallPlan(modulePath, explicitVersion, actionPath, actionRef string) (installPlan, error) {
	version := strings.TrimSpace(explicitVersion)
	if version != "" {
		return installPlan{
			Mode:    "go-install",
			Target:  modulePath + "@" + version,
			Version: version,
		}, nil
	}

	actionPath = strings.TrimSpace(actionPath)
	if actionPath != "" {
		return installPlan{
			Mode:       "local-source",
			ActionPath: actionPath,
		}, nil
	}

	version = normalizeActionRef(actionRef)
	if version == "" {
		return installPlan{}, fmt.Errorf("GITHUB_ACTION_PATH is not available; set cervomut-version explicitly")
	}
	if strings.Contains(version, "/") {
		return installPlan{}, fmt.Errorf("GITHUB_ACTION_PATH is not available and GITHUB_ACTION_REF %q cannot be used as a go install version. Set cervomut-version explicitly to a tag, commit SHA, or latest.", version)
	}

	return installPlan{
		Mode:    "go-install",
		Target:  modulePath + "@" + version,
		Version: version,
	}, nil
}

func normalizeActionRef(ref string) string {
	ref = strings.TrimSpace(ref)
	switch {
	case strings.HasPrefix(ref, "refs/tags/"):
		return strings.TrimPrefix(ref, "refs/tags/")
	case strings.HasPrefix(ref, "refs/heads/"):
		return strings.TrimPrefix(ref, "refs/heads/")
	default:
		return ref
	}
}

func resolveReportDir(workspace, workingDirectory, outDir string) (string, error) {
	outDir = strings.TrimSpace(outDir)
	if outDir == "" {
		return "", fmt.Errorf("out directory must not be empty")
	}
	if filepath.IsAbs(outDir) {
		return filepath.Clean(outDir), nil
	}

	workingDirectory = strings.TrimSpace(workingDirectory)
	if workingDirectory == "" {
		workingDirectory = "."
	}
	if !filepath.IsAbs(workingDirectory) {
		workspace = strings.TrimSpace(workspace)
		if workspace == "" {
			return "", fmt.Errorf("GITHUB_WORKSPACE is required when working-directory is relative")
		}
		workingDirectory = filepath.Join(workspace, workingDirectory)
	}

	return filepath.Clean(filepath.Join(workingDirectory, outDir)), nil
}
