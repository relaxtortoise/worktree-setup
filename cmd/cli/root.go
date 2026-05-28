package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/relaxtortoise/worktree-setup/internal/config"
	"github.com/spf13/cobra"
)

var (
	configDir    string
	noFetch      bool
	explicitPath string
)

func init() {
	configDir = config.UserConfigDir()
}

var rootCmd = &cobra.Command{
	Use:   "wt",
	Short: "Enhanced git worktree management",
	Long:  "wt enhances git worktree with automated setup via .worktree.yaml config files.",
}

func getRepoDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return dir
}

func projectName() string {
	dir := getRepoDir()
	out, err := runGitCmd(dir, "remote", "get-url", "origin")
	if err != nil {
		return ""
	}
	return urlToProjectName(strings.TrimSpace(out))
}

func urlToProjectName(url string) string {
	url = strings.TrimSpace(url)
	url = strings.TrimPrefix(url, "git@")
	url = strings.TrimPrefix(url, "https://")
	url = strings.ReplaceAll(url, ":", "/")
	url = strings.TrimSuffix(url, ".git")
	parts := strings.Split(url, "/")
	if len(parts) >= 3 {
		last := parts[len(parts)-3:]
		return strings.Join(last, "-")
	}
	return strings.ReplaceAll(url, "/", "-")
}

func loadConfig() (*config.Config, error) {
	repoDir := getRepoDir()
	projName := projectName()
	if projName == "" {
		return nil, fmt.Errorf("not in a git repository with a remote origin")
	}
	return config.LoadHierarchy(repoDir, configDir, projName)
}

func runGitCmd(dir string, args ...string) (string, error) {
	cmd := execCommand("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

var execCommand = func(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
