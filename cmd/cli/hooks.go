package main

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/relaxtortoise/worktree-setup/internal/hooks"
	"github.com/spf13/cobra"
)

var hooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Install git hooks for auto-detecting worktree creation",
	RunE: func(cmd *cobra.Command, args []string) error {
		repoDir := getRepoDir()
		installed, err := hooks.Install(repoDir)
		if err != nil {
			return err
		}
		for _, h := range installed {
			fmt.Printf("installed: .git/hooks/%s\n", h)
			slog.With("repo", repoDir).Info("hook installed",
				"hook", h, "git_dir", filepath.Join(repoDir, ".git"))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(hooksCmd)
}
