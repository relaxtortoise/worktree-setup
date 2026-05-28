package main

import (
	"fmt"

	gitpkg "github.com/relaxtortoise/worktree-setup/internal/git"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all worktrees",
	RunE: func(cmd *cobra.Command, args []string) error {
		wts, err := gitpkg.ListWorktrees()
		if err != nil {
			return err
		}
		for _, wt := range wts {
			n := min(8, len(wt.Head))
			fmt.Printf("%s\t%s\n", wt.Head[:n], wt.Path)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
