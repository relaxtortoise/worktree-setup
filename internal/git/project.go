package git

import (
	"os/exec"
	"strings"
)

// URLToProjectName converts a git remote URL to a project name identifier.
func URLToProjectName(url string) string {
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

// ProjectName detects the current git repo's project name from origin remote.
func ProjectName(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return URLToProjectName(strings.TrimSpace(string(out))), nil
}
