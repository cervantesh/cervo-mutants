package discover

import (
	"os"
	"path/filepath"
	"strings"
)

type File struct {
	ModuleDir string `json:"module_dir"`
	Package   string `json:"package"`
	Path      string `json:"path"`
	IsTest    bool   `json:"is_test"`
}

type Result struct {
	Modules []string `json:"modules"`
	Files   []File   `json:"files"`
}

func Discover(targets []string) (Result, error) {
	if len(targets) == 0 {
		targets = []string{"."}
	}
	var result Result
	seenModules := map[string]bool{}
	for _, target := range targets {
		root := strings.TrimSuffix(target, "/...")
		if root == "." || root == "./..." {
			root = "."
		}
		abs, err := filepath.Abs(root)
		if err != nil {
			return Result{}, err
		}
		moduleDir := findModule(abs)
		if moduleDir == "" {
			moduleDir = abs
		}
		if !seenModules[moduleDir] {
			seenModules[moduleDir] = true
			result.Modules = append(result.Modules, moduleDir)
		}
		err = filepath.WalkDir(abs, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() && excludedDir(d.Name()) {
				return filepath.SkipDir
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			name := filepath.Base(path)
			if strings.HasSuffix(name, "_generated.go") || strings.HasSuffix(name, ".pb.go") {
				return nil
			}
			rel, _ := filepath.Rel(moduleDir, filepath.Dir(path))
			pkg := "./" + filepath.ToSlash(rel)
			if rel == "." {
				pkg = "."
			}
			result.Files = append(result.Files, File{
				ModuleDir: moduleDir,
				Package:   pkg,
				Path:      path,
				IsTest:    strings.HasSuffix(name, "_test.go"),
			})
			return nil
		})
		if err != nil {
			return Result{}, err
		}
	}
	return result, nil
}

func findModule(start string) string {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func excludedDir(name string) bool {
	switch name {
	case ".git", ".cervomut", "vendor", "node_modules", "dist", "build", "coverage":
		return true
	default:
		return false
	}
}
