package actions

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/relaxtortoise/worktree-setup/internal/config"
)

func ExecuteSymlink(dstDir, srcDir string, items []config.CopyAction) ([]string, error) {
	var linked []string
	for _, item := range items {
		src := filepath.Join(srcDir, item.From)
		dst := filepath.Join(dstDir, item.To)

		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return linked, err
		}

		err := os.Symlink(src, dst)
		if err != nil && runtime.GOOS == "windows" {
			fmt.Fprintf(os.Stderr, "Symlink failed for %s: %v\n", item.To, err)
			fmt.Fprintf(os.Stderr, "Downgrade to copy? This requires developer mode or admin privileges for symlink. [y/N]: ")
			var answer string
			_, _ = fmt.Scanln(&answer)
			if strings.ToLower(strings.TrimSpace(answer)) == "y" {
				if err := copyDir(src, dst); err != nil {
					return linked, fmt.Errorf("fallback copy failed: %w", err)
				}
				fmt.Fprintf(os.Stderr, "Copied instead: %s\n", item.To)
			}
		} else if err != nil {
			return linked, fmt.Errorf("symlink %s: %w", item.To, err)
		}
		linked = append(linked, item.To)
	}
	return linked, nil
}
