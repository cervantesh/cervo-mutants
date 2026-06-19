package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/version"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cervantesh/cervo-mutants/pkg/engine"
)

const defaultModulePath = "github.com/cervantesh/cervo-mutants/cmd/cervomut"

type installPlan struct {
	ActionPath string `json:"action_path,omitempty"`
	Mode       string `json:"mode"`
	Target     string `json:"target,omitempty"`
	Version    string `json:"version,omitempty"`
}

type goVersionResolution struct {
	GoVersion          string `json:"go_version"`
	GoVersionRequested string `json:"go_version_requested,omitempty"`
	GoVersionTarget    string `json:"go_version_target,omitempty"`
	GoVersionActionMin string `json:"go_version_action_min"`
}

type failureDebugArtifact struct {
	SchemaVersion string                      `json:"schema_version"`
	Kind          string                      `json:"kind"`
	Message       string                      `json:"message"`
	CorrelationID string                      `json:"correlation_id"`
	Command       []string                    `json:"command,omitempty"`
	Targets       []string                    `json:"targets,omitempty"`
	StackTrace    string                      `json:"stack_trace,omitempty"`
	RunnerResult  *engine.FailureRunnerResult `json:"runner_result,omitempty"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: actionhelper <install-plan|report-dir|resolve-go-version|failure-from-debug|build-wave-result|render-wave-result-markdown|build-wave-summary|render-wave-summary-markdown|build-adoption-summary|render-adoption-summary-markdown>")
	}
	switch args[0] {
	case "install-plan":
		return cmdInstallPlan(args[1:], stdout)
	case "report-dir":
		return cmdReportDir(args[1:], stdout)
	case "resolve-go-version":
		return cmdResolveGoVersion(args[1:], stdout)
	case "failure-from-debug":
		return cmdFailureFromDebug(args[1:], stdout)
	case "build-wave-result":
		return cmdBuildWaveResult(args[1:], stdout)
	case "render-wave-result-markdown":
		return cmdRenderWaveResultMarkdown(args[1:], stdout)
	case "build-wave-summary":
		return cmdBuildWaveSummary(args[1:], stdout)
	case "render-wave-summary-markdown":
		return cmdRenderWaveSummaryMarkdown(args[1:], stdout)
	case "build-adoption-summary":
		return cmdBuildAdoptionSummary(args[1:], stdout)
	case "render-adoption-summary-markdown":
		return cmdRenderAdoptionSummaryMarkdown(args[1:], stdout)
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

func cmdResolveGoVersion(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("resolve-go-version", flag.ContinueOnError)
	requested := fs.String("requested", "", "optional requested Go version from the manifest")
	targetGoMod := fs.String("target-gomod", "", "path to the target repository go.mod")
	actionGoMod := fs.String("action-gomod", "", "path to the action source go.mod")
	defaultActionMin := fs.String("default-action-min", "1.25.6", "fallback action minimum when action go.mod is unavailable")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	resolution, err := resolveGoVersion(*requested, *targetGoMod, *actionGoMod, *defaultActionMin)
	if err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(resolution)
}

func cmdFailureFromDebug(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("failure-from-debug", flag.ContinueOnError)
	path := fs.String("path", "", "path to failure-debug.json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	failure, err := failureFromDebugFile(*path)
	if err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(failure)
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

func resolveGoVersion(requested, targetGoMod, actionGoMod, defaultActionMin string) (goVersionResolution, error) {
	requested = normalizeGoVersion(requested)
	targetVersion, err := extractGoVersionFromGoMod(targetGoMod)
	if err != nil {
		return goVersionResolution{}, err
	}
	actionMin := normalizeGoVersion(defaultActionMin)
	if actionGoMod != "" {
		parsedActionMin, err := extractGoVersionFromGoMod(actionGoMod)
		if err != nil {
			return goVersionResolution{}, err
		}
		if parsedActionMin != "" {
			actionMin = parsedActionMin
		}
	}
	if actionMin == "" {
		return goVersionResolution{}, fmt.Errorf("action minimum Go version could not be determined")
	}

	resolved := requested
	if resolved == "" {
		resolved = targetVersion
	}
	if resolved == "" {
		resolved = actionMin
	} else {
		resolved = maxGoVersion(actionMin, resolved)
	}
	return goVersionResolution{
		GoVersion:          resolved,
		GoVersionRequested: requested,
		GoVersionTarget:    targetVersion,
		GoVersionActionMin: actionMin,
	}, nil
}

func extractGoVersionFromGoMod(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	var fallback string
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "toolchain":
			return normalizeGoVersion(fields[1]), nil
		case "go":
			if fallback == "" {
				fallback = normalizeGoVersion(fields[1])
			}
		}
	}
	return fallback, nil
}

func normalizeGoVersion(raw string) string {
	return strings.TrimPrefix(strings.TrimSpace(raw), "go")
}

func maxGoVersion(left, right string) string {
	left = normalizeGoVersion(left)
	right = normalizeGoVersion(right)
	switch {
	case left == "":
		return right
	case right == "":
		return left
	case version.Compare("go"+left, "go"+right) >= 0:
		return left
	default:
		return right
	}
}

func failureFromDebugFile(path string) (*engine.Failure, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var artifact failureDebugArtifact
	if err := json.Unmarshal(data, &artifact); err != nil {
		return nil, err
	}
	if artifact.Kind == "" && artifact.Message == "" && artifact.CorrelationID == "" {
		return nil, nil
	}
	failure := &engine.Failure{
		Kind:          artifact.Kind,
		Message:       artifact.Message,
		CorrelationID: artifact.CorrelationID,
		Command:       append([]string{}, artifact.Command...),
		Targets:       append([]string{}, artifact.Targets...),
		DebugArtifact: filepath.Base(path),
	}
	if artifact.RunnerResult != nil {
		cloned := *artifact.RunnerResult
		cloned.Command = append([]string{}, artifact.RunnerResult.Command...)
		failure.RunnerResult = &cloned
	}
	return failure, nil
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
