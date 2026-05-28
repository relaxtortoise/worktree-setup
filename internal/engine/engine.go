package engine

import (
	"github.com/relaxtortoise/worktree-setup/internal/actions"
	"github.com/relaxtortoise/worktree-setup/internal/config"
)

type Engine struct {
	Runner *actions.Runner
}

func New(mainWorktree string) *Engine {
	return &Engine{Runner: actions.NewRunner(mainWorktree)}
}

func (e *Engine) RunPreCreate(cfg *config.Config) error {
	if cfg.On == nil {
		return nil
	}
	return e.Runner.ExecutePreCreate(cfg.On.PreCreate)
}

func (e *Engine) RunPostCreate(cfg *config.Config, worktreeDir string) error {
	if cfg.On == nil {
		return nil
	}
	return e.Runner.ExecuteEvent(cfg.On.PostCreate, worktreeDir)
}

func (e *Engine) RunPostCheckout(cfg *config.Config, worktreeDir string) error {
	if cfg.On == nil {
		return nil
	}
	return e.Runner.ExecuteEvent(cfg.On.PostCheckout, worktreeDir)
}

func (e *Engine) RunPreDelete(cfg *config.Config, worktreeDir string) error {
	if cfg.On == nil {
		return nil
	}
	return e.Runner.ExecuteEvent(cfg.On.PreDelete, worktreeDir)
}

func (e *Engine) RunPostDelete(cfg *config.Config) error {
	if cfg.On == nil {
		return nil
	}
	return e.Runner.ExecuteEvent(cfg.On.PostDelete, e.Runner.MainWorktree)
}
