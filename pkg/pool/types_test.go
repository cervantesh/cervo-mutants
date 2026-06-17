package pool

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cervantesh/cervo-mutants/internal/testharness"
)

func TestLoadManifestAndFilterRepos(t *testing.T) {
	fixture := testharness.NewDir(t)
	manifestPath := fixture.WriteJSON(t, "manifest.json", Manifest{
		SchemaVersion: "1",
		Repos: []Repo{
			{Name: "alpha", URL: "https://example.com/alpha.git", Target: "./..."},
			{Name: "beta", URL: "https://example.com/beta.git", Target: "./pkg"},
		},
	})

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	if manifest.SchemaVersion != "1" || len(manifest.Repos) != 2 {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}

	filtered := FilterRepos(manifest, []string{" beta ", "", "missing"}, 0)
	if len(filtered) != 1 || filtered[0].Name != "beta" {
		t.Fatalf("FilterRepos names mismatch: %+v", filtered)
	}

	all := FilterRepos(manifest, nil, 1)
	if len(all) != 1 || all[0].Name != "alpha" {
		t.Fatalf("FilterRepos limit mismatch: %+v", all)
	}
	all[0].Name = "changed"
	if manifest.Repos[0].Name != "alpha" {
		t.Fatalf("FilterRepos should copy source repos: %+v", manifest.Repos)
	}
}

func TestPoolFileAndPathHelpers(t *testing.T) {
	fixture := testharness.NewDir(t)

	jsonPath := filepath.Join(fixture.Root, "nested", "summary.json")
	if err := writeJSON(jsonPath, []string{"one", "two"}); err != nil {
		t.Fatalf("writeJSON returned error: %v", err)
	}
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("summary missing: %v", err)
	}
	if string(data[len(data)-1]) != "\n" {
		t.Fatalf("writeJSON should add trailing newline: %q", string(data))
	}

	src := fixture.WriteFile(t, "src/report.txt", "copied")
	dst := filepath.Join(fixture.Root, "out", "copied.txt")
	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile returned error: %v", err)
	}
	if got := readFile(dst); got != "copied" {
		t.Fatalf("readFile copied text mismatch: %q", got)
	}
	if got := readFile(filepath.Join(fixture.Root, "missing.txt")); got != "" {
		t.Fatalf("readFile missing file = %q, want empty", got)
	}

	if got := defaultPath(" custom ", "fallback"); got != " custom " {
		t.Fatalf("defaultPath preserved explicit value = %q", got)
	}
	if got := defaultPath("   ", "fallback"); got != "fallback" {
		t.Fatalf("defaultPath fallback mismatch: %q", got)
	}
	if got := summaryPath(fixture.Root); got != filepath.Join(fixture.Root, "summary.json") {
		t.Fatalf("summaryPath = %q", got)
	}
	if got, err := requiredBinary("git", "git"); err != nil || got != "git" {
		t.Fatalf("requiredBinary explicit value = %q, %v", got, err)
	}
	if _, err := requiredBinary("git", "   "); err == nil {
		t.Fatal("requiredBinary accepted blank path")
	}

	started := time.Now().Add(-1500 * time.Millisecond)
	if got := seconds(started); got < 1.0 {
		t.Fatalf("seconds = %.3f, want elapsed value", got)
	}
}

func TestRunSimpleCommandReturnsExitAndError(t *testing.T) {
	runner := &fakeRunner{run: func(spec CommandSpec) (CommandResult, error) {
		if spec.Path == "fail" {
			return CommandResult{}, errors.New("boom")
		}
		return CommandResult{ExitCode: 7}, nil
	}}
	exit, err := runSimpleCommand(context.Background(), runner, CommandSpec{Path: "ok"})
	if err != nil || exit != 7 {
		t.Fatalf("runSimpleCommand success = (%d, %v), want (7, nil)", exit, err)
	}
	if _, err := runSimpleCommand(context.Background(), runner, CommandSpec{Path: "fail"}); err == nil {
		t.Fatal("runSimpleCommand accepted runner error")
	}
}
