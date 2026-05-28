package git

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CmdFn is the function used to create exec.Cmd instances.
// It can be overridden in tests for dependency injection.
var CmdFn = exec.Command

type Worktree struct {
	Path   string
	Head   string
	Branch string
	Bare   bool
}

func Run(args ...string) (string, error) {
	cmd := CmdFn("git", args...)
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), string(ee.Stderr))
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// RunInternal 执行 git 命令并设置 WT_INTERNAL 标记
func RunInternal(args ...string) (string, error) {
	cmd := CmdFn("git", args...)
	cmd.Env = append(os.Environ(), "WT_INTERNAL=1")
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), string(ee.Stderr))
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func AddWorktree(path, branch, baseBranch string) error {
	args := []string{"worktree", "add", path}
	if branch != "" {
		args = append(args, branch)
	}
	if baseBranch != "" {
		args = append(args, baseBranch)
	}
	_, err := RunInternal(args...)
	return err
}

func RemoveWorktree(path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)
	_, err := RunInternal(args...)
	return err
}

func ListWorktrees() ([]Worktree, error) {
	out, err := Run("worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return parsePorcelain(out), nil
}

func parsePorcelain(out string) []Worktree {
	var wts []Worktree
	var cur *Worktree
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			if cur != nil {
				wts = append(wts, *cur)
				cur = nil
			}
			continue
		}
		if strings.HasPrefix(line, "worktree ") {
			cur = &Worktree{Path: strings.TrimPrefix(line, "worktree ")}
		} else if strings.HasPrefix(line, "HEAD ") {
			if cur != nil {
				cur.Head = strings.TrimPrefix(line, "HEAD ")
			}
		} else if strings.HasPrefix(line, "branch ") {
			if cur != nil {
				cur.Branch = strings.TrimPrefix(line, "branch ")
			}
		} else if strings.HasPrefix(line, "bare") {
			if cur != nil {
				cur.Bare = true
			}
		}
	}
	if cur != nil {
		wts = append(wts, *cur)
	}
	return wts
}

// FindMainWorktree 自动检测 main worktree
func FindMainWorktree() (string, error) {
	wts, err := ListWorktrees()
	if err != nil {
		return "", err
	}
	// 优先找 main/master 分支
	for _, wt := range wts {
		b := strings.TrimPrefix(wt.Branch, "refs/heads/")
		if b == "main" || b == "master" {
			return wt.Path, nil
		}
	}
	// 退而求其次找第一个非 bare
	for _, wt := range wts {
		if !wt.Bare {
			return wt.Path, nil
		}
	}
	return "", fmt.Errorf("no main worktree found")
}

func CurrentWorktreePath() (string, error) {
	out, err := Run("rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", err
	}
	dir := strings.TrimSuffix(out, "/.git")
	dir = strings.TrimSuffix(dir, "\\.git")
	return dir, nil
}
