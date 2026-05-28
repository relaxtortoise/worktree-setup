package actions

import (
	"fmt"
	"strings"

	"github.com/relaxtortoise/worktree-setup/internal/config"
)

type Runner struct {
	MainWorktree string
}

func NewRunner(mainWorktree string) *Runner {
	return &Runner{MainWorktree: mainWorktree}
}

func (r *Runner) ExecuteEvent(event *config.Event, worktreeDir string) error {
	if event == nil {
		return nil
	}
	steps := event.StepsOrLegacy()
	for i, step := range steps {
		if err := r.executeStep(step, worktreeDir); err != nil {
			return fmt.Errorf("step %d: %w", i+1, err)
		}
	}
	return nil
}

func (r *Runner) executeStep(step config.Step, worktreeDir string) error {
	if step.Copy != nil {
		if _, err := ExecuteCopy(worktreeDir, r.MainWorktree, step.Copy.Items); err != nil {
			return fmt.Errorf("copy: %w", err)
		}
	}
	if step.Symlink != nil {
		if _, err := ExecuteSymlink(worktreeDir, r.MainWorktree, step.Symlink.Items); err != nil {
			return fmt.Errorf("symlink: %w", err)
		}
	}
	if step.Run != "" {
		for _, cmd := range strings.Split(step.Run, "\n") {
			cmd = strings.TrimSpace(cmd)
			if cmd == "" {
				continue
			}
			if err := ExecuteRun([]string{cmd}, worktreeDir, false); err != nil {
				return fmt.Errorf("run %q: %w", cmd, err)
			}
		}
	}
	return nil
}

func (r *Runner) ExecutePreCreate(event *config.Event) error {
	if event == nil {
		return nil
	}
	for _, cmd := range event.Run {
		if err := ExecuteRun([]string{cmd}, r.MainWorktree, false); err != nil {
			return fmt.Errorf("pre-create run %q: %w", cmd, err)
		}
	}
	return nil
}
