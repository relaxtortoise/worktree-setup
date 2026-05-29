package logging

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/relaxtortoise/worktree-setup/internal/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Init configures the default slog logger to write JSON Lines to
// <UserConfigDir>/log.jsonl with size-based rotation.
// Log level is controlled by WT_LOG_LEVEL env var (default "info").
// Returns error only if the log directory cannot be created.
func Init() error {
	dir := config.UserConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	level := parseLevel(os.Getenv("WT_LOG_LEVEL"))

	writer := &lumberjack.Logger{
		Filename:   filepath.Join(dir, "log.jsonl"),
		MaxSize:    10,
		MaxBackups: 3,
		Compress:   true,
	}

	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	})

	slog.SetDefault(slog.New(handler))
	return nil
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
