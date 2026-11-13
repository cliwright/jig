package jig

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Bump copies the latest version of a template to v<N+1> and returns the new
// version directory — the starting point for the template's next iteration.
// name resolves exactly as in ResolveTemplate. The copy refuses symlinks,
// matching the default render policy.
func Bump(templatesDir, name string) (string, error) {
	latest, err := ResolveTemplate(templatesDir, name, 0)
	if err != nil {
		return "", err
	}
	n, err := strconv.Atoi(strings.TrimPrefix(filepath.Base(latest), "v"))
	if err != nil {
		return "", fmt.Errorf("template %q: latest version directory %s is not v<N>", name, filepath.Base(latest))
	}
	next := filepath.Join(filepath.Dir(latest), "v"+strconv.Itoa(n+1))
	if _, err := os.Stat(next); err == nil {
		return "", fmt.Errorf("%s already exists", next)
	}
	if err := copyTree(latest, next); err != nil {
		return "", err
	}
	return next, nil
}

// copyTree recursively copies src to dst, preserving file modes. Symlinks
// are rejected outright.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dst, 0o755)
		}
		out := filepath.Join(dst, rel)
		if d.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("template contains symlink %s: bump refuses to copy symlinks", path)
		}
		if d.IsDir() {
			return os.MkdirAll(out, 0o755)
		}
		return copyFile(path, out, d)
	})
}
