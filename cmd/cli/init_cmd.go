package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/relaxtortoise/worktree-setup/internal/config"
	gitpkg "github.com/relaxtortoise/worktree-setup/internal/git"
)

var noGitignore bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize .worktree.yaml and project config",
	RunE: func(cmd *cobra.Command, args []string) error {
		repoDir := getRepoDir()
		projName := projectName()
		if projName == "" {
			return fmt.Errorf("not in a git repository with a remote origin")
		}

		template := `# wt worktree configuration
# See: https://github.com/relaxtortoise/worktree-setup

on:
  post-create:
    run: []
  post-checkout:
    run: []
`
		wtPath := filepath.Join(repoDir, ".worktree.yaml")
		if _, err := os.Stat(wtPath); os.IsNotExist(err) {
			os.WriteFile(wtPath, []byte(template), 0644)
			fmt.Println("created .worktree.yaml")
		} else {
			fmt.Println(".worktree.yaml already exists, skipping")
		}

		projDir := config.ProjectConfigDir(projName)
		os.MkdirAll(projDir, 0755)
		cfgPath := filepath.Join(projDir, "config.yaml")

		if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
			mainWT, err := gitpkg.FindMainWorktree()
			if err != nil {
				mainWT = repoDir
			}
			content := fmt.Sprintf("main_worktree: %s\npath_strategy: sibling\n", mainWT)
			os.WriteFile(cfgPath, []byte(content), 0644)
			fmt.Printf("created %s\n", cfgPath)
		} else {
			fmt.Printf("%s already exists, skipping\n", cfgPath)
		}

		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&noGitignore, "no-gitignore", false, "Do not add .worktree.yaml to .gitignore")
	rootCmd.AddCommand(initCmd)
}
