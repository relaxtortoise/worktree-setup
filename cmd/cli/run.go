package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/relaxtortoise/worktree-setup/internal/engine"
	gitpkg "github.com/relaxtortoise/worktree-setup/internal/git"
)

var detectCreate bool

var runCmd = &cobra.Command{
	Use:   "run <event>",
	Short: "Execute configured event (called by git hooks)",
	Args:  cobra.ExactArgs(1),
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
				// post-checkout hook args are after the event name in os.Args
				// Find the 3 hook args: <prev-head> <new-head> <is-branch-checkout>
				if len(os.Args) >= 3 && isNewWorktree(os.Args[len(os.Args)-3]) {
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

func isNewWorktree(prevHead string) bool {
	if len(prevHead) >= 40 {
		return strings.Count(prevHead, "0") == 40
	}
	return prevHead == "0000000000000000000000000000000000000000"
}

func init() {
	runCmd.Flags().BoolVar(&detectCreate, "detect-create", false, "Detect new worktree from post-checkout args")
	rootCmd.AddCommand(runCmd)
}
