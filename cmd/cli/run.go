package main

import (
	"fmt"
	"os"

	"github.com/relaxtortoise/worktree-setup/internal/engine"
	gitpkg "github.com/relaxtortoise/worktree-setup/internal/git"
	"github.com/spf13/cobra"
)

var detectCreate bool

var runCmd = &cobra.Command{
	Use:   "run <event>",
	Short: "Execute configured event (called by git hooks)",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		mainWT := cfg.MainWorktree
		if mainWT == "" {
			mainWT, _ = gitpkg.FindMainWorktree()
		}

		eng := engine.New(mainWT)
		event := args[0]

		switch event {
		case "post-checkout":
			if detectCreate {
				// args[1:] are the 3 hook args: <prev-head> <new-head> <is-branch-checkout>
				if len(args) >= 2 && engine.IsNewWorktree(args[1]) {
					currentDir, _ := os.Getwd()
					return eng.RunPostCreate(cfg, currentDir)
				}
				return eng.RunPostCheckout(cfg, ".")
			}
			return eng.RunPostCheckout(cfg, ".")
		default:
			return fmt.Errorf("unknown event: %s", event)
		}
	},
}

func init() {
	runCmd.Flags().BoolVar(&detectCreate, "detect-create", false, "Detect new worktree from post-checkout args")
	rootCmd.AddCommand(runCmd)
}
