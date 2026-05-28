package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of wt",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("wt v%s (%s/%s) commit %s built at %s\n",
			version, runtime.GOOS, runtime.GOARCH, commit, buildTime)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
