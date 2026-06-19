package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveInstallPlanPrefersLocalActionPath(t *testing.T) {
	plan, err := resolveInstallPlan(defaultModulePath, "", "/tmp/action", "refs/heads/feature/slash-name")
	if err != nil {
		t.Fatalf("resolveInstallPlan returned error: %v", err)
	}
	if plan.Mode != "local-source" {
		t.Fatalf("expected local-source mode, got %q", plan.Mode)
	}
	if plan.ActionPath != "/tmp/action" {
		t.Fatalf("expected local action path to be preserved, got %q", plan.ActionPath)
	}
	if plan.Target != "" {
		t.Fatalf("expected no go-install target for local source mode, got %q", plan.Target)
	}
}

func TestResolveInstallPlanNormalizesTagAndHeadRefs(t *testing.T) {
	tests := []struct {
		name      string
		actionRef string
		want      string
	}{
		{name: "tag ref", actionRef: "refs/tags/v0.3.0", want: defaultModulePath + "@v0.3.0"},
		{name: "main branch ref", actionRef: "refs/heads/main", want: defaultModulePath + "@main"},
		{name: "commit sha", actionRef: "0123456789abcdef0123456789abcdef01234567", want: defaultModulePath + "@0123456789abcdef0123456789abcdef01234567"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := resolveInstallPlan(defaultModulePath, "", "", tc.actionRef)
			if err != nil {
				t.Fatalf("resolveInstallPlan returned error: %v", err)
			}
			if plan.Mode != "go-install" {
				t.Fatalf("expected go-install mode, got %q", plan.Mode)
			}
			if plan.Target != tc.want {
				t.Fatalf("expected target %q, got %q", tc.want, plan.Target)
			}
		})
	}
}

func TestResolveInstallPlanRejectsSlashQualifiedBranchWithoutActionPath(t *testing.T) {
	_, err := resolveInstallPlan(defaultModulePath, "", "", "refs/heads/release/hotfix")
	if err == nil {
		t.Fatal("expected an error for slash-qualified branch ref without action path")
	}
	if !strings.Contains(err.Error(), "cannot be used as a go install version") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveInstallPlanUsesExplicitVersion(t *testing.T) {
	plan, err := resolveInstallPlan(defaultModulePath, "latest", "", "")
	if err != nil {
		t.Fatalf("resolveInstallPlan returned error: %v", err)
	}
	if plan.Target != defaultModulePath+"@latest" {
		t.Fatalf("unexpected explicit target: %q", plan.Target)
	}
}

func TestResolveReportDir(t *testing.T) {
	workspace := filepath.Join(string(filepath.Separator), "workspace")
	if runtime.GOOS == "windows" {
		workspace = `C:\workspace`
	}

	got, err := resolveReportDir(workspace, "repo/subdir", ".cervomut/pr")
	if err != nil {
		t.Fatalf("resolveReportDir returned error: %v", err)
	}
	want := filepath.Clean(filepath.Join(workspace, "repo/subdir", ".cervomut/pr"))
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResolveReportDirPreservesAbsoluteOutput(t *testing.T) {
	outDir := filepath.Join(string(filepath.Separator), "tmp", "reports")
	if runtime.GOOS == "windows" {
		outDir = `D:\tmp\reports`
	}

	got, err := resolveReportDir("", ".", outDir)
	if err != nil {
		t.Fatalf("resolveReportDir returned error: %v", err)
	}
	if got != filepath.Clean(outDir) {
		t.Fatalf("expected %q, got %q", filepath.Clean(outDir), got)
	}
}

func TestCmdInstallPlanWritesJSON(t *testing.T) {
	var out bytes.Buffer
	if err := cmdInstallPlan([]string{"--version", "v0.3.0"}, &out); err != nil {
		t.Fatalf("cmdInstallPlan returned error: %v", err)
	}

	var plan installPlan
	if err := json.Unmarshal(out.Bytes(), &plan); err != nil {
		t.Fatalf("cmdInstallPlan did not emit valid JSON: %v", err)
	}
	if plan.Target != defaultModulePath+"@v0.3.0" {
		t.Fatalf("unexpected target: %q", plan.Target)
	}
}

func TestCmdReportDirWritesPath(t *testing.T) {
	workspace := filepath.Join(string(filepath.Separator), "workspace")
	if runtime.GOOS == "windows" {
		workspace = `C:\workspace`
	}

	var out bytes.Buffer
	if err := cmdReportDir([]string{"--workspace", workspace, "--working-directory", "repo", "--out", ".cervomut/reports"}, &out); err != nil {
		t.Fatalf("cmdReportDir returned error: %v", err)
	}

	got := strings.TrimSpace(out.String())
	want := filepath.Clean(filepath.Join(workspace, "repo", ".cervomut/reports"))
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResolveGoVersionUsesTargetWhenTargetNeedsNewerToolchain(t *testing.T) {
	dir := t.TempDir()
	targetGoMod := filepath.Join(dir, "target.go.mod")
	actionGoMod := filepath.Join(dir, "action.go.mod")
	writeGoModForTest(t, targetGoMod, "module example.com/target\n\ngo 1.25.0\ntoolchain go1.26.0\n")
	writeGoModForTest(t, actionGoMod, "module github.com/cervantesh/cervo-mutants\n\ngo 1.25.6\n")

	resolution, err := resolveGoVersion("", targetGoMod, actionGoMod, "1.25.6")
	if err != nil {
		t.Fatalf("resolveGoVersion returned error: %v", err)
	}
	if resolution.GoVersion != "1.26.0" {
		t.Fatalf("resolved go version = %q, want 1.26.0", resolution.GoVersion)
	}
	if resolution.GoVersionTarget != "1.26.0" {
		t.Fatalf("target go version = %q, want 1.26.0", resolution.GoVersionTarget)
	}
	if resolution.GoVersionActionMin != "1.25.6" {
		t.Fatalf("action minimum = %q, want 1.25.6", resolution.GoVersionActionMin)
	}
}

func TestResolveGoVersionFloorsAtActionMinimum(t *testing.T) {
	dir := t.TempDir()
	targetGoMod := filepath.Join(dir, "target.go.mod")
	actionGoMod := filepath.Join(dir, "action.go.mod")
	writeGoModForTest(t, targetGoMod, "module example.com/target\n\ngo 1.12\n")
	writeGoModForTest(t, actionGoMod, "module github.com/cervantesh/cervo-mutants\n\ngo 1.25.6\n")

	resolution, err := resolveGoVersion("", targetGoMod, actionGoMod, "1.25.6")
	if err != nil {
		t.Fatalf("resolveGoVersion returned error: %v", err)
	}
	if resolution.GoVersion != "1.25.6" {
		t.Fatalf("resolved go version = %q, want 1.25.6", resolution.GoVersion)
	}
	if resolution.GoVersionTarget != "1.12" {
		t.Fatalf("target go version = %q, want 1.12", resolution.GoVersionTarget)
	}
}

func TestResolveGoVersionHonorsRequestedVersionWhenHigher(t *testing.T) {
	dir := t.TempDir()
	targetGoMod := filepath.Join(dir, "target.go.mod")
	actionGoMod := filepath.Join(dir, "action.go.mod")
	writeGoModForTest(t, targetGoMod, "module example.com/target\n\ngo 1.26.0\n")
	writeGoModForTest(t, actionGoMod, "module github.com/cervantesh/cervo-mutants\n\ngo 1.25.6\n")

	resolution, err := resolveGoVersion("1.27.1", targetGoMod, actionGoMod, "1.25.6")
	if err != nil {
		t.Fatalf("resolveGoVersion returned error: %v", err)
	}
	if resolution.GoVersion != "1.27.1" {
		t.Fatalf("resolved go version = %q, want 1.27.1", resolution.GoVersion)
	}
	if resolution.GoVersionRequested != "1.27.1" {
		t.Fatalf("requested go version = %q, want 1.27.1", resolution.GoVersionRequested)
	}
}

func TestCmdResolveGoVersionWritesJSON(t *testing.T) {
	dir := t.TempDir()
	targetGoMod := filepath.Join(dir, "target.go.mod")
	actionGoMod := filepath.Join(dir, "action.go.mod")
	writeGoModForTest(t, targetGoMod, "module example.com/target\n\ngo 1.12\n")
	writeGoModForTest(t, actionGoMod, "module github.com/cervantesh/cervo-mutants\n\ngo 1.25.6\n")

	var out bytes.Buffer
	if err := cmdResolveGoVersion([]string{
		"--requested", "1.24.0",
		"--target-gomod", targetGoMod,
		"--action-gomod", actionGoMod,
	}, &out); err != nil {
		t.Fatalf("cmdResolveGoVersion returned error: %v", err)
	}

	var resolution goVersionResolution
	if err := json.Unmarshal(out.Bytes(), &resolution); err != nil {
		t.Fatalf("cmdResolveGoVersion did not emit valid JSON: %v", err)
	}
	if resolution.GoVersion != "1.25.6" {
		t.Fatalf("resolved go version = %q, want 1.25.6", resolution.GoVersion)
	}
	if resolution.GoVersionRequested != "1.24.0" || resolution.GoVersionTarget != "1.12" || resolution.GoVersionActionMin != "1.25.6" {
		t.Fatalf("unexpected go version resolution payload: %+v", resolution)
	}
}

func writeGoModForTest(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write go.mod %s: %v", path, err)
	}
}
