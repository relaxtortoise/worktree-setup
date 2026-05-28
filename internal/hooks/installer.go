package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const hookScript = `#!/bin/sh
# Installed by wt hooks

if [ -n "$WT_INTERNAL" ]; then
    exit 0
fi
wt run post-checkout "$@" --detect-create
`

func Install(repoDir string) ([]string, error) {
	hooksDir := filepath.Join(repoDir, ".git", "hooks")

	var installed []string
	hookPath := filepath.Join(hooksDir, "post-checkout")
	if err := os.WriteFile(hookPath, []byte(hookScript), 0755); err != nil {
		return installed, fmt.Errorf("write %s: %w", hookPath, err)
	}
	installed = append(installed, "post-checkout")

	return installed, nil
}

func IsInstalled(repoDir string) bool {
	hookPath := filepath.Join(repoDir, ".git", "hooks", "post-checkout")
	data, err := os.ReadFile(hookPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "Installed by wt")
}
