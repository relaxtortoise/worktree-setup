package worktree

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/relaxtortoise/worktree-setup/internal/config"
)

func ComputePath(mainWorktree, branch, projectName string, ps *config.PathStrategy) string {
	template := "{main_parent}/{repo_name}@{branch}"
	if ps != nil {
		if ps.Template != "" {
			template = ps.Template
		} else {
			switch ps.Name {
			case "nested":
				template = "{main}/.worktrees/{branch}"
			case "home":
				template = "~/worktrees/{project_name}/{branch}"
			}
		}
	}

	result := template
	result = strings.ReplaceAll(result, "{main}", mainWorktree)
	result = strings.ReplaceAll(result, "{main_parent}", filepath.Dir(mainWorktree))
	result = strings.ReplaceAll(result, "{repo_name}", filepath.Base(mainWorktree))
	result = strings.ReplaceAll(result, "{project_name}", projectName)
	result = strings.ReplaceAll(result, "{branch}", sanitizeBranch(branch))
	if strings.HasPrefix(result, "~/") {
		home, _ := os.UserHomeDir()
		result = filepath.Join(home, result[2:])
	}
	return result
}

func sanitizeBranch(branch string) string {
	return strings.ReplaceAll(branch, "/", "-")
}
