package main

import (
	"fmt"
	"os"

	"github.com/relaxtortoise/worktree-setup/internal/tui"
	"github.com/relaxtortoise/worktree-setup/internal/worktree"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add [branch]",
	Short: "Create a new worktree",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		branch := ""
		if len(args) > 0 {
			branch = args[0]
		}

		if branch == "" {
			autoFetch := !noFetch
			if os.Getenv("WT_NO_FETCH") == "1" {
				autoFetch = false
			}
			branch, err = tui.RunBranchSelector(autoFetch)
			if err != nil {
				return err
			}
		}

		projName := projectName()
		autoFetch := !noFetch
		if os.Getenv("WT_NO_FETCH") == "1" {
			autoFetch = false
		}

		path, err := worktree.Create(branch, explicitPath, projName, cfg, autoFetch)
		if err != nil {
			return err
		}
		fmt.Println(path)
		return nil
	},
}

func init() {
	addCmd.Flags().BoolVar(&noFetch, "no-fetch", false, "Skip git fetch")
	addCmd.Flags().StringVar(&explicitPath, "path", "", "Explicit worktree path")
	rootCmd.AddCommand(addCmd)
}
