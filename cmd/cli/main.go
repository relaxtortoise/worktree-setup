package main

import (
	"fmt"
	"os"
)

var (
	Version   = "dev" // exported for internal/selfupdate
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
