package main

import (
	"fmt"
	"os"

	"github.com/relaxtortoise/worktree-setup/internal/config"
	gitpkg "github.com/relaxtortoise/worktree-setup/internal/git"
	"github.com/spf13/cobra"
)

var (
	configDir    string
	noFetch      bool
	explicitPath string
)

func init() {
	configDir = config.UserConfigDir()
}

var rootCmd = &cobra.Command{
	Use:   "wt",
	Short: "Enhanced git worktree management",
	Long:  "wt enhances git worktree with automated setup via .worktree.yaml config files.",
}

func getRepoDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return dir
}

func projectName() string {
	name, err := gitpkg.ProjectName(getRepoDir())
	if err != nil {
		return ""
	}
	return name
}

func loadConfig() (*config.Config, error) {
	repoDir := getRepoDir()
	projName := projectName()
	if projName == "" {
		return nil, fmt.Errorf("not in a git repository with a remote origin")
	}
	return config.LoadHierarchy(repoDir, configDir, projName)
}

