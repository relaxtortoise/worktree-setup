package worktree

import (
	"fmt"
	"log/slog"

	"github.com/relaxtortoise/worktree-setup/internal/config"
	"github.com/relaxtortoise/worktree-setup/internal/engine"
	gitpkg "github.com/relaxtortoise/worktree-setup/internal/git"
)

func Remove(path string, cfg *config.Config, force bool) error {
	mainWT := cfg.MainWorktree
	if mainWT == "" {
		var err error
		mainWT, err = gitpkg.FindMainWorktree()
		if err != nil {
			return fmt.Errorf("cannot determine main worktree: %w", err)
		}
	}

	eng := engine.New(mainWT)

	if err := eng.RunPreDelete(cfg, path); err != nil {
		return fmt.Errorf("pre-delete: %w", err)
	}

	if err := gitpkg.RemoveWorktree(path, force); err != nil {
		return fmt.Errorf("git worktree remove: %w", err)
	}

	if err := eng.RunPostDelete(cfg); err != nil {
		return fmt.Errorf("post-delete: %w", err)
	}

	slog.Info("worktree removed", "worktree", path)
	return nil
}
