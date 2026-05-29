package main

import (
	"os"
)

var (
	Version   = "dev" // exported for internal/selfupdate
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
