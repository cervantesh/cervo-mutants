package testharness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type Dir struct {
	Root string
}

func NewDir(t *testing.T) Dir {
	t.Helper()
	return Dir{Root: t.TempDir()}
}

func (d Dir) Path(parts ...string) string {
	all := append([]string{d.Root}, parts...)
	return filepath.Join(all...)
}

func (d Dir) WriteFile(t *testing.T, relativePath, contents string) string {
	t.Helper()
	path := d.Path(relativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func (d Dir) WriteJSON(t *testing.T, relativePath string, value any) string {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return d.WriteFile(t, relativePath, string(data)+"\n")
}

func (d Dir) WriteFiles(t *testing.T, files map[string]string) {
	t.Helper()
	for relativePath, contents := range files {
		d.WriteFile(t, relativePath, contents)
	}
}

func WriteGoModuleFixture(t *testing.T, root, module string, files map[string]string) {
	t.Helper()
	if module == "" {
		module = "fixture"
	}
	dir := Dir{Root: root}
	dir.WriteFile(t, "go.mod", "module "+module+"\n\ngo 1.25.6\n")
	dir.WriteFiles(t, files)
}

func WriteGoModuleTempDir(t *testing.T, module string, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	WriteGoModuleFixture(t, root, module, files)
	return root
}
