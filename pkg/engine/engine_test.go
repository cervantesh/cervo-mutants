package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/config"
)

func writeFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module fixture\n\ngo 1.25.6\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "calc.go"), []byte(`package fixture

func IsPositiveOrZero(n int) bool {
	return n >= 0
}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "calc_test.go"), []byte(`package fixture

import "testing"

func TestIsPositiveOrZero(t *testing.T) {
	if !IsPositiveOrZero(1) {
		t.Fatal("want positive")
	}
}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestRunDryRunDiscoversMutantsWithoutChangingWorkspace(t *testing.T) {
	dir := writeFixture(t)
	before, err := os.ReadFile(filepath.Join(dir, "calc.go"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Defaults()
	cfg.Tests.Command = []string{"go", "test", "./..."}
	cfg.Reports.Output = filepath.Join(dir, ".cervomut", "reports")
	cfg.Cache.Path = filepath.Join(dir, ".cervomut", "cache")

	result, err := New(cfg).Run(context.Background(), RunRequest{Targets: []string{dir}, DryRun: true})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Summary.Total == 0 {
		t.Fatal("dry-run discovered no mutants")
	}
	after, err := os.ReadFile(filepath.Join(dir, "calc.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("dry-run changed source workspace")
	}
}

func TestRunClassifiesSurvivorAndWritesReports(t *testing.T) {
	dir := writeFixture(t)
	cfg := config.Defaults()
	cfg.Tests.Command = []string{"go", "test", "./..."}
	cfg.Tests.Timeout = 10_000_000_000
	cfg.Reports.Output = filepath.Join(dir, ".cervomut", "reports")
	cfg.Cache.Path = filepath.Join(dir, ".cervomut", "cache")
	cfg.Execution.Workers = 1

	result, err := New(cfg).Run(context.Background(), RunRequest{Targets: []string{dir}})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Summary.Total == 0 {
		t.Fatal("run discovered no mutants")
	}
	if result.Summary.Survived == 0 {
		t.Fatalf("expected weak fixture test to leave a survivor: %+v", result.Summary)
	}
	reportPath := filepath.Join(cfg.Reports.Output, "mutation-report.json")
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("report was not written: %v", err)
	}
	if !strings.Contains(string(data), `"schema_version": "1"`) {
		t.Fatalf("report missing schema version: %s", data)
	}
}
