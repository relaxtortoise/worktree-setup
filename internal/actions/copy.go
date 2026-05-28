package actions

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/relaxtortoise/worktree-setup/internal/config"
)

func ExecuteCopy(dstDir, srcDir string, items []config.CopyAction) ([]string, error) {
	var copied []string
	for _, item := range items {
		src := filepath.Join(srcDir, item.From)
		dst := filepath.Join(dstDir, item.To)

		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return copied, err
		}

		srcInfo, err := os.Stat(src)
		if err != nil {
			return copied, fmt.Errorf("stat %s: %w", src, err)
		}

		if srcInfo.IsDir() {
			if err := copyDir(src, dst); err != nil {
				return copied, err
			}
		} else {
			if err := copyFile(src, dst); err != nil {
				return copied, err
			}
		}
		copied = append(copied, item.To)
	}
	return copied, nil
}

func copyFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()

	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = d.Close() }()

	_, err = io.Copy(d, s)
	return err
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		dest := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(dest, info.Mode())
		}
		return copyFile(path, dest)
	})
}
