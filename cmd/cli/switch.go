package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/relaxtortoise/worktree-setup/internal/tui"
	"github.com/spf13/cobra"
)

var switchCmd = &cobra.Command{
	Use:   "switch [name]",
	Short: "Switch to a worktree (cross-project)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			fmt.Println(args[0])
			return nil
		}

		var projects []struct{ Name, Path string }
		entries, err := os.ReadDir(filepath.Join(configDir, "projects"))
		if err == nil {
			for _, e := range entries {
				if e.IsDir() {
					projects = append(projects, struct{ Name, Path string }{
						Name: e.Name(), Path: filepath.Join(configDir, "projects", e.Name()),
					})
				}
			}
		}
		projName := projectName()
		hasCurrent := false
		for _, p := range projects {
			if p.Name == projName {
				hasCurrent = true
				break
			}
		}
		if !hasCurrent {
			projects = append(projects, struct{ Name, Path string }{Name: projName, Path: ""})
		}

		path, err := tui.RunWorktreeSelector(projects)
		if err != nil {
			return err
		}
		fmt.Println(path)
		repoDir := getRepoDir()
		slog.With("repo", repoDir).Info("worktree switched", "to", path, "from", repoDir)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(switchCmd)
}
