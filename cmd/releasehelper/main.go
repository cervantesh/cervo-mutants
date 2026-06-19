package main

import (
	"flag"
	"fmt"
	"github.com/cervantesh/cervo-mutants/internal/compatmatrix"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: releasehelper <notes|verify-compat|verify-release|verify-upgrade>")
	}
	switch args[0] {
	case "notes":
		return cmdNotes(args[1:])
	case "verify-compat":
		return cmdVerifyCompat(args[1:])
	case "verify-release":
		return cmdVerifyRelease(args[1:])
	case "verify-upgrade":
		return cmdVerifyUpgrade(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func cmdNotes(args []string) error {
	fs := flag.NewFlagSet("notes", flag.ContinueOnError)
	version := fs.String("version", "", "release version such as v0.3.0")
	changelogPath := fs.String("changelog", "CHANGELOG.md", "path to changelog")
	upgradeDir := fs.String("upgrade-dir", filepath.Join("docs", "upgrade-notes"), "directory containing per-version upgrade notes")
	out := fs.String("out", "", "optional output path")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if strings.TrimSpace(*version) == "" {
		return fmt.Errorf("notes requires --version")
	}
	changelogSection, err := extractMarkdownSection(*changelogPath, *version)
	if err != nil {
		return err
	}
	upgradePath := filepath.Join(*upgradeDir, *version+".md")
	upgradeBody, err := os.ReadFile(upgradePath)
	if err != nil {
		return fmt.Errorf("read upgrade notes %s: %w", filepath.ToSlash(upgradePath), err)
	}
	notes := buildReleaseNotes(*version, changelogSection, string(upgradeBody))
	if strings.TrimSpace(*out) == "" {
		fmt.Print(notes)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		return err
	}
	return os.WriteFile(*out, []byte(notes), 0o644)
}

func extractMarkdownSection(path, version string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	var start int = -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "## ["+version+"]" || strings.HasPrefix(trimmed, "## ["+version+"] ") || trimmed == "## "+version || strings.HasPrefix(trimmed, "## "+version+" ") {
			start = i + 1
			break
		}
	}
	if start == -1 {
		return "", fmt.Errorf("version %s not found in %s", version, filepath.ToSlash(path))
	}
	end := len(lines)
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "## ") {
			end = i
			break
		}
	}
	section := strings.TrimSpace(strings.Join(lines[start:end], "\n"))
	if section == "" {
		return "", fmt.Errorf("version %s section in %s is empty", version, filepath.ToSlash(path))
	}
	return section, nil
}

func buildReleaseNotes(version, changelogSection, upgradeNotes string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# CervoMutants %s\n\n", version)
	b.WriteString("## Changelog\n\n")
	b.WriteString(strings.TrimSpace(changelogSection))
	b.WriteString("\n\n## Upgrade Notes\n\n")
	b.WriteString(stripTopHeading(strings.TrimSpace(upgradeNotes)))
	b.WriteString("\n")
	return b.String()
}

func stripTopHeading(body string) string {
	if body == "" {
		return body
	}
	lines := strings.Split(body, "\n")
	if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[0]), "#") {
		return strings.TrimSpace(strings.Join(lines[1:], "\n"))
	}
	return body
}

type workflowDoc struct {
	Jobs map[string]workflowJob `yaml:"jobs"`
}

type workflowJob struct {
	RunsOn   string         `yaml:"runs-on"`
	Strategy workflowConfig `yaml:"strategy"`
	Steps    []workflowStep `yaml:"steps"`
}

type workflowConfig struct {
	Matrix workflowMatrix `yaml:"matrix"`
}

type workflowMatrix struct {
	Include []map[string]string `yaml:"include"`
}

type workflowStep struct {
	Name string            `yaml:"name"`
	Uses string            `yaml:"uses"`
	With map[string]string `yaml:"with"`
}

func cmdVerifyCompat(args []string) error {
	fs := flag.NewFlagSet("verify-compat", flag.ContinueOnError)
	goModPath := fs.String("go-mod", compatmatrix.GoModPath, "path to go.mod")
	docPath := fs.String("doc", compatmatrix.CompatibilityDocPath, "path to the compatibility matrix document")
	testWorkflowPath := fs.String("test-workflow", compatmatrix.TestWorkflowPath, "path to the main test workflow")
	releaseWorkflowPath := fs.String("release-workflow", compatmatrix.ReleaseWorkflowPath, "path to the release workflow")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	if err := verifyGoModBaseline(*goModPath); err != nil {
		return err
	}
	if err := verifyCompatibilityDoc(*docPath); err != nil {
		return err
	}
	if err := verifyTestWorkflow(*testWorkflowPath); err != nil {
		return err
	}
	return verifyReleaseWorkflow(*releaseWorkflowPath)
}

func verifyGoModBaseline(path string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", filepath.ToSlash(path), err)
	}
	want := "go " + compatmatrix.SupportedGoVersion
	if !strings.Contains(string(body), want) {
		return fmt.Errorf("%s must declare %q to match the supported matrix", filepath.ToSlash(path), want)
	}
	return nil
}

func verifyCompatibilityDoc(path string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", filepath.ToSlash(path), err)
	}
	text := string(body)
	for _, target := range compatmatrix.Targets() {
		row := fmt.Sprintf("| %s | `%s` | Supported |", target.DocLabel, compatmatrix.SupportedGoSeries)
		if !strings.Contains(text, row) {
			return fmt.Errorf("%s is missing compatibility row %q", filepath.ToSlash(path), row)
		}
	}
	if !strings.Contains(text, "Current `go.mod` baseline is `go "+compatmatrix.SupportedGoVersion+"`.") {
		return fmt.Errorf("%s must mention go.mod baseline go %s", filepath.ToSlash(path), compatmatrix.SupportedGoVersion)
	}
	return nil
}

func verifyTestWorkflow(path string) error {
	workflow, err := loadWorkflow(path)
	if err != nil {
		return err
	}
	if err := verifySetupGoVersion(workflow, "core-tests", compatmatrix.SupportedGoVersion); err != nil {
		return err
	}
	if err := verifyActionInputVersion(workflow, "github-action-smoke", "Run local GitHub Action source", compatmatrix.SupportedGoVersion); err != nil {
		return err
	}
	return verifyCompatibilityMatrixJob(workflow, path, "compatibility-smoke")
}

func verifyReleaseWorkflow(path string) error {
	workflow, err := loadWorkflow(path)
	if err != nil {
		return err
	}
	if err := verifySetupGoVersion(workflow, "publish", compatmatrix.SupportedGoVersion); err != nil {
		return err
	}
	return verifyCompatibilityMatrixJob(workflow, path, "compatibility-smoke")
}

func loadWorkflow(path string) (workflowDoc, error) {
	var doc workflowDoc
	body, err := os.ReadFile(path)
	if err != nil {
		return doc, fmt.Errorf("read %s: %w", filepath.ToSlash(path), err)
	}
	if err := yaml.Unmarshal(body, &doc); err != nil {
		return doc, fmt.Errorf("parse %s: %w", filepath.ToSlash(path), err)
	}
	return doc, nil
}

func verifyCompatibilityMatrixJob(workflow workflowDoc, path, jobName string) error {
	job, ok := workflow.Jobs[jobName]
	if !ok {
		return fmt.Errorf("%s is missing job %q", filepath.ToSlash(path), jobName)
	}
	expected := map[string]string{}
	for _, target := range compatmatrix.Targets() {
		expected[target.Runner] = compatmatrix.SupportedGoVersion
	}
	if len(job.Strategy.Matrix.Include) != len(expected) {
		return fmt.Errorf("%s job %q must define %d compatibility targets, got %d", filepath.ToSlash(path), jobName, len(expected), len(job.Strategy.Matrix.Include))
	}
	for _, include := range job.Strategy.Matrix.Include {
		runner := include["os"]
		goVersion := include["go-version"]
		wantVersion, ok := expected[runner]
		if !ok {
			return fmt.Errorf("%s job %q contains unexpected runner %q", filepath.ToSlash(path), jobName, runner)
		}
		if goVersion != wantVersion {
			return fmt.Errorf("%s job %q runner %q uses Go %q, want %q", filepath.ToSlash(path), jobName, runner, goVersion, wantVersion)
		}
		delete(expected, runner)
	}
	if len(expected) > 0 {
		return fmt.Errorf("%s job %q is missing compatibility runners: %v", filepath.ToSlash(path), jobName, expected)
	}
	return nil
}

func verifySetupGoVersion(workflow workflowDoc, jobName, want string) error {
	job, ok := workflow.Jobs[jobName]
	if !ok {
		return fmt.Errorf("workflow is missing job %q", jobName)
	}
	for _, step := range job.Steps {
		if step.Uses == "actions/setup-go@v5" {
			if want == "" {
				return nil
			}
			if step.With["go-version"] != want {
				return fmt.Errorf("job %q uses actions/setup-go@v5 with go-version %q, want %q", jobName, step.With["go-version"], want)
			}
			return nil
		}
	}
	return fmt.Errorf("job %q is missing actions/setup-go@v5", jobName)
}

func verifyActionInputVersion(workflow workflowDoc, jobName, stepName, want string) error {
	job, ok := workflow.Jobs[jobName]
	if !ok {
		return fmt.Errorf("workflow is missing job %q", jobName)
	}
	for _, step := range job.Steps {
		if step.Name == stepName {
			if step.With["go-version"] != want {
				return fmt.Errorf("job %q step %q uses go-version %q, want %q", jobName, stepName, step.With["go-version"], want)
			}
			return nil
		}
	}
	return fmt.Errorf("job %q is missing step %q", jobName, stepName)
}
