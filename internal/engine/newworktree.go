package engine

import "strings"

// IsNewWorktree returns true if prevHead looks like a new worktree reference
// (all zeros, meaning no previous HEAD).
func IsNewWorktree(prevHead string) bool {
	if len(prevHead) >= 40 {
		return strings.Count(prevHead, "0") == 40
	}
	return prevHead == "0000000000000000000000000000000000000000"
}
