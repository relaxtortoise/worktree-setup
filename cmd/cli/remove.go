package main

import (
	"fmt"

	"github.com/relaxtortoise/worktree-setup/internal/worktree"
	"github.com/spf13/cobra"
)

var removeForce bool

var removeCmd = &cobra.Command{
	Use:   "remove [name|path]",
	Short: "Remove a worktree",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		if len(args) == 0 {
			return fmt.Errorf("worktree name or path required")
		}
		return worktree.Remove(args[0], cfg, removeForce)
	},
}

func init() {
	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "Force removal")
	rootCmd.AddCommand(removeCmd)
}
