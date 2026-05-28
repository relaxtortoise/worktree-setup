package main

import (
	"github.com/relaxtortoise/worktree-setup/internal/selfupdate"
	"github.com/spf13/cobra"
)

var (
	selfUpdateYes   bool
	selfUpdateCheck bool
)

var selfUpdateCmd = &cobra.Command{
	Use:   "self-update [version]",
	Short: "Update wt to the latest or specified version",
	Long: `Download and replace the current wt binary from GitHub Releases.

Without arguments, updates to the latest release.
With a version argument (e.g. v1.2.0), updates to that specific version.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		updater := selfupdate.New(Version, "relaxtortoise", "worktree-setup")
		return updater.Run(args, selfUpdateYes, selfUpdateCheck)
	},
}

func init() {
	selfUpdateCmd.Flags().BoolVarP(&selfUpdateYes, "yes", "y", false, "Skip confirmation prompt")
	selfUpdateCmd.Flags().BoolVar(&selfUpdateCheck, "check", false, "Only check for updates, do not download")
	rootCmd.AddCommand(selfUpdateCmd)
}
