package git

import (
	"sort"
	"strings"
	"time"
)

type Branch struct {
	Name       string
	LastCommit time.Time
	Author     string
	CheckedOut bool
}

func FetchOrigin() error {
	_, err := Run("fetch", "origin", "--prune")
	return err
}

func ListRemoteBranches() ([]Branch, error) {
	out, err := Run("for-each-ref", "--format=%(refname:short)%00%(committerdate:iso8601)%00%(authorname)",
		"refs/remotes/origin/", "--sort=-committerdate")
	if err != nil {
		return nil, err
	}

	checkedOut := checkedOutBranches()

	var branches []Branch
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\x00", 3)
		if len(parts) < 3 {
			continue
		}
		name := strings.TrimPrefix(parts[0], "origin/")
		if name == "HEAD" {
			continue
		}
		t, _ := time.Parse("2006-01-02 15:04:05 -0700", parts[1])
		branches = append(branches, Branch{
			Name:       name,
			LastCommit: t,
			Author:     parts[2],
			CheckedOut: checkedOut[name],
		})
	}
	sort.Slice(branches, func(i, j int) bool {
		return branches[i].LastCommit.After(branches[j].LastCommit)
	})
	return branches, nil
}

func checkedOutBranches() map[string]bool {
	out, err := Run("worktree", "list")
	if err != nil {
		return nil
	}
	result := make(map[string]bool)
	for _, line := range strings.Split(out, "\n") {
		start := strings.LastIndex(line, "[")
		end := strings.LastIndex(line, "]")
		if start >= 0 && end > start {
			result[line[start+1:end]] = true
		}
	}
	return result
}
