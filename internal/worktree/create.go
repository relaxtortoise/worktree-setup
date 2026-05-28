package worktree

import (
	"fmt"
	"os"

	"github.com/relaxtortoise/worktree-setup/internal/config"
	"github.com/relaxtortoise/worktree-setup/internal/engine"
	gitpkg "github.com/relaxtortoise/worktree-setup/internal/git"
)

func Create(branch, explicitPath, projectName string, cfg *config.Config, autoFetch bool) (string, error) {
	mainWT := cfg.MainWorktree
	if mainWT == "" {
		var err error
		mainWT, err = gitpkg.FindMainWorktree()
		if err != nil {
			return "", fmt.Errorf("cannot determine main worktree: %w", err)
		}
	}

	if autoFetch {
		gitpkg.FetchOrigin()
	}

	targetPath := explicitPath
	if targetPath == "" {
		targetPath = ComputePath(mainWT, branch, projectName, cfg.PathStrategy)
	}

	eng := engine.New(mainWT)
	if err := eng.RunPreCreate(cfg); err != nil {
		return "", fmt.Errorf("pre-create: %w", err)
	}

	if err := gitpkg.AddWorktree(targetPath, branch, "origin/"+branch); err != nil {
		return "", fmt.Errorf("git worktree add: %w", err)
	}

	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return "", fmt.Errorf("worktree directory not found after creation: %s", targetPath)
	}

	if err := eng.RunPostCreate(cfg, targetPath); err != nil {
		return targetPath, fmt.Errorf("post-create: %w", err)
	}

	return targetPath, nil
}
