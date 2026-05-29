package main

import (
	"log/slog"
	"os"

	"github.com/relaxtortoise/worktree-setup/internal/logging"
)

var (
	Version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	if err := logging.Init(); err != nil {
		// Silent degradation — slog stays at default text handler
	} else {
		// Set global attributes on the default logger
		slog.SetDefault(slog.Default().With("app", "wt", "version", Version))
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
