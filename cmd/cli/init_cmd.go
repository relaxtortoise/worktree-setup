package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/relaxtortoise/worktree-setup/internal/config"
	gitpkg "github.com/relaxtortoise/worktree-setup/internal/git"
	"github.com/relaxtortoise/worktree-setup/internal/tui"
	"github.com/spf13/cobra"
)

var (
	initMainWorktree string
	initPathStrategy string
	initNoSaveVCS    bool
	initPostCreate   []string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize .worktree.yaml and project config",
	Long: `Initialize worktree-setup configuration for the current project.

By default, runs an interactive wizard that guides you through
main_worktree, path_strategy, post-create steps, and whether to
save event configuration to .worktree.yaml (VCS).

Pass CLI flags to skip the wizard and write directly.`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVar(&initMainWorktree, "main-worktree", "", "Main worktree path")
	initCmd.Flags().StringVar(&initPathStrategy, "path-strategy", "", "Path strategy: sibling, nested, or template")
	initCmd.Flags().BoolVar(&initNoSaveVCS, "no-save-vcs", false, "Save everything to user config (disable VCS)")
	initCmd.Flags().StringArrayVar(&initPostCreate, "post-create-run", nil, "Add a post-create run step (repeatable)")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	repoDir := getRepoDir()

	// Pre-check: must be in a git repo with remote origin
	projName := projectName()
	if projName == "" {
		return fmt.Errorf("not in a git repository with a remote origin")
	}

	// Detect main worktree for default
	detectedWT, err := gitpkg.FindMainWorktree()
	if err != nil {
		detectedWT = repoDir
	}

	// Pre-check: existing files (interactive only)
	hasFlags := initMainWorktree != "" || initPathStrategy != "" || initNoSaveVCS || len(initPostCreate) > 0
	if !hasFlags {
		wtPath := filepath.Join(repoDir, ".worktree.yaml")
		projCfgPath := config.ProjectConfigPath(projName)
		if err := checkOverwrite(wtPath, ".worktree.yaml"); err != nil {
			return err
		}
		if err := checkOverwrite(projCfgPath, "project config"); err != nil {
			return err
		}
	}

	var result tui.WizardResult

	if hasFlags {
		// Non-interactive mode: build result from flags
		result = tui.WizardResult{
			MainWorktree: initMainWorktree,
			PathStrategy: initPathStrategy,
			Events:       initPostCreate,
			SaveWithVCS:  !initNoSaveVCS,
		}
		// Apply defaults for unspecified values
		if result.MainWorktree == "" {
			result.MainWorktree = detectedWT
		}
		if result.PathStrategy == "" {
			result.PathStrategy = "sibling"
		}
	} else {
		// Interactive mode: launch TUI wizard
		result = tui.RunInitWizard(detectedWT)
		if result.Cancelled {
			fmt.Println("cancelled")
			return nil
		}
	}

	// Write config files
	return writeInitConfig(repoDir, projName, result)
}

func checkOverwrite(path, label string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	fmt.Printf("%s already exists. Overwrite? [y/N]: ", label)
	var answer string
	_, _ = fmt.Scanln(&answer)
	if strings.ToLower(strings.TrimSpace(answer)) != "y" {
		fmt.Printf("skipping %s\n", label)
		return nil
	}
	return nil
}

func writeInitConfig(repoDir, projName string, r tui.WizardResult) error {
	wtPath := filepath.Join(repoDir, ".worktree.yaml")
	projDir := config.ProjectConfigDir(projName)
	projCfgPath := filepath.Join(projDir, "config.yaml")

	if err := os.MkdirAll(projDir, 0755); err != nil {
		return err
	}

	if r.SaveWithVCS {
		// .worktree.yaml ← events (always write to repo to signal VCS config)
		wtCfg := &config.Config{}
		if len(r.Events) > 0 {
			wtCfg.On = &config.Events{
				PostCreate: &config.Event{},
			}
			for _, ev := range r.Events {
				wtCfg.On.PostCreate.Steps = append(wtCfg.On.PostCreate.Steps, config.Step{Run: ev})
			}
		}
		if err := config.WriteFile(wtPath, wtCfg); err != nil {
			return err
		}
		fmt.Printf("created %s\n", wtPath)
		slog.With("repo", repoDir).Info("config saved", "config_file", wtPath, "main_worktree", r.MainWorktree)

		// project config ← main_worktree + path_strategy
		var sb strings.Builder
		fmt.Fprintf(&sb, "main_worktree: %s\n", r.MainWorktree)
		sb.WriteString(formatPathStrategy(r.PathStrategy, r.CustomTemplate))
		if err := os.WriteFile(projCfgPath, []byte(sb.String()), 0644); err != nil {
			return err
		}
		fmt.Printf("created %s\n", projCfgPath)
		slog.With("repo", repoDir).Info("config saved", "config_file", projCfgPath, "main_worktree", r.MainWorktree)
	} else {
		// Everything → project config
		var sb strings.Builder
		fmt.Fprintf(&sb, "main_worktree: %s\n", r.MainWorktree)
		sb.WriteString(formatPathStrategy(r.PathStrategy, r.CustomTemplate))
		if len(r.Events) > 0 {
			sb.WriteString("on:\n  post-create:\n    steps:\n")
			for _, ev := range r.Events {
				fmt.Fprintf(&sb, "      - run: %s\n", ev)
			}
		}
		if err := os.WriteFile(projCfgPath, []byte(sb.String()), 0644); err != nil {
			return err
		}
		fmt.Printf("created %s\n", projCfgPath)
		slog.With("repo", repoDir).Info("config saved", "config_file", projCfgPath, "main_worktree", r.MainWorktree)
	}

	return nil
}

func formatPathStrategy(strategy, customTemplate string) string {
	if strategy == "custom" {
		return fmt.Sprintf("path_strategy:\n  template: %s\n", customTemplate)
	}
	if strategy == "" {
		strategy = "sibling"
	}
	return fmt.Sprintf("path_strategy: %s\n", strategy)
}
