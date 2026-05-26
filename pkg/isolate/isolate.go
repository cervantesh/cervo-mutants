package isolate

import (
	"io"
	"os"
	"path/filepath"
)

func CopyModule(moduleDir string) (string, error) {
	tmp, err := os.MkdirTemp("", "cervomut-*")
	if err != nil {
		return "", err
	}
	if err := copyTree(moduleDir, tmp); err != nil {
		_ = os.RemoveAll(tmp)
		return "", err
	}
	return tmp, nil
}

func Cleanup(path string) error {
	if path == "" {
		return nil
	}
	return os.RemoveAll(path)
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if d.IsDir() && excludedDir(d.Name()) {
			return filepath.SkipDir
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func excludedDir(name string) bool {
	switch name {
	case ".git", ".cervomut", "vendor", "node_modules", "dist", "build", "coverage":
		return true
	default:
		return false
	}
}
