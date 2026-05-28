package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// initGitRepo creates a temp git repo and returns its path.
func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runRawGit(t, dir, "init", "-b", "main")
	runRawGit(t, dir, "config", "user.email", "test@test")
	runRawGit(t, dir, "config", "user.name", "test")
	// Create an initial commit so the repo has a HEAD
	writeFile(t, dir, "README.md", "# test")
	runRawGit(t, dir, "add", "README.md")
	runRawGit(t, dir, "commit", "-m", "initial")
	return dir
}

// initBareRepo creates a bare repo (simulates a remote)
func initBareRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runRawGit(t, dir, "init", "--bare", "-b", "main")
	return dir
}

// runRawGit runs a git command natively in the given directory.
func runRawGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed in %s: %s\n%s", args, dir, err, out)
	}
	return string(out)
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// parsePorcelain tests
// ---------------------------------------------------------------------------

func TestParsePorcelain(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int // number of worktrees parsed
	}{
		{
			name: "single worktree",
			input: `worktree /home/me/projects/app
HEAD abc123def456
branch refs/heads/main

`,
			want: 1,
		},
		{
			name: "bare and non-bare",
			input: `worktree /home/me/projects/app
HEAD abc123
branch refs/heads/main

worktree /home/me/projects/app2
HEAD def456
branch refs/heads/feature
bare

`,
			want: 2,
		},
		{
			name:  "empty input",
			input: "",
			want:  0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wts := parsePorcelain(tt.input)
			assert.Len(t, wts, tt.want)
		})
	}
}

func TestParsePorcelain_Fields(t *testing.T) {
	input := `worktree /path/to/repo
HEAD abc123def456789
branch refs/heads/feature-x

`
	wts := parsePorcelain(input)
	require.Len(t, wts, 1)
	assert.Equal(t, "/path/to/repo", wts[0].Path)
	assert.Equal(t, "abc123def456789", wts[0].Head)
	assert.Equal(t, "refs/heads/feature-x", wts[0].Branch)
	assert.False(t, wts[0].Bare)
}

func TestParsePorcelain_Bare(t *testing.T) {
	input := `worktree /bare/repo
HEAD abc123
bare

`
	wts := parsePorcelain(input)
	require.Len(t, wts, 1)
	assert.True(t, wts[0].Bare)
}

// ---------------------------------------------------------------------------
// URLToProjectName tests
// ---------------------------------------------------------------------------

func TestURLToProjectName(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "https github url",
			url:  "https://github.com/owner/repo.git",
			want: "github.com-owner-repo",
		},
		{
			name: "git ssh url",
			url:  "git@github.com:owner/repo.git",
			want: "github.com-owner-repo",
		},
		{
			name: "url with no .git suffix",
			url:  "https://github.com/owner/repo",
			want: "github.com-owner-repo",
		},
		{
			name: "two segment url",
			url:  "https://example.com/owner",
			want: "example.com-owner",
		},
		{
			name: "url with extra path segments",
			url:  "https://github.com/org/team/repo.git",
			want: "org-team-repo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := URLToProjectName(tt.url)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// ProjectName tests
// ---------------------------------------------------------------------------

func TestProjectName(t *testing.T) {
	dir := initGitRepo(t)
	runRawGit(t, dir, "remote", "add", "origin", "https://github.com/relaxtortoise/worktree-setup.git")

	name, err := ProjectName(dir)
	require.NoError(t, err)
	assert.Equal(t, "github.com-relaxtortoise-worktree-setup", name)
}

func TestProjectName_NoRemote(t *testing.T) {
	dir := initGitRepo(t)
	// No remote added — expect error
	_, err := ProjectName(dir)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Run and RunInternal tests
// ---------------------------------------------------------------------------

func TestRun_Success(t *testing.T) {
	dir := initGitRepo(t)

	// Use CmdFn override to run in our temp dir
	oldCmdFn := CmdFn
	defer func() { CmdFn = oldCmdFn }()

	CmdFn = func(name string, args ...string) *exec.Cmd {
		cmd := exec.Command(name, args...)
		// Set working dir for git commands
		if name == "git" {
			cmd.Dir = dir
		}
		return cmd
	}

	out, err := Run("rev-parse", "--abbrev-ref", "HEAD")
	require.NoError(t, err)
	assert.Equal(t, "main", out)
}

func TestRun_Failure(t *testing.T) {
	oldCmdFn := CmdFn
	defer func() { CmdFn = oldCmdFn }()

	CmdFn = func(name string, args ...string) *exec.Cmd {
		return exec.Command(name, args...)
	}

	_, err := Run("invalid-git-command-xyz")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git")
}

func TestRunInternal_WTInternal(t *testing.T) {
	dir := initGitRepo(t)

	oldCmdFn := CmdFn
	defer func() { CmdFn = oldCmdFn }()

	var capturedCmd *exec.Cmd
	CmdFn = func(name string, args ...string) *exec.Cmd {
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		capturedCmd = cmd // capture pointer so Env is visible after RunInternal sets it
		return cmd
	}

	_, err := RunInternal("rev-parse", "HEAD")
	require.NoError(t, err)
	assert.Contains(t, capturedCmd.Env, "WT_INTERNAL=1")
}

// ---------------------------------------------------------------------------
// Worktree operation tests
// ---------------------------------------------------------------------------

func TestAddWorktree_Success(t *testing.T) {
	dir := initGitRepo(t)
	wtDir := filepath.Join(t.TempDir(), "worktree")

	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)

	out, err := Run("worktree", "list")
	require.NoError(t, err, "listing before add: %s", out)

	err = AddWorktree(wtDir, "", "")
	require.NoError(t, err)
	assert.DirExists(t, wtDir)
}

func TestRemoveWorktree_Success(t *testing.T) {
	dir := initGitRepo(t)
	wtDir := filepath.Join(t.TempDir(), "worktree")

	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)

	err := AddWorktree(wtDir, "", "")
	require.NoError(t, err)

	err = RemoveWorktree(wtDir, false)
	require.NoError(t, err)
	assert.NoDirExists(t, wtDir)
}

func TestRemoveWorktree_Force(t *testing.T) {
	dir := initGitRepo(t)
	wtDir := filepath.Join(t.TempDir(), "worktree")

	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)

	err := AddWorktree(wtDir, "", "")
	require.NoError(t, err)

	// Dirty the worktree
	err = os.WriteFile(filepath.Join(wtDir, "dirty.txt"), []byte("hello"), 0644)
	require.NoError(t, err)

	err = RemoveWorktree(wtDir, true)
	require.NoError(t, err)
}

func TestListWorktrees(t *testing.T) {
	dir := initGitRepo(t)
	wtDir := filepath.Join(t.TempDir(), "worktree")

	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)

	err := AddWorktree(wtDir, "feature-x", "main")
	require.NoError(t, err)

	wts, err := ListWorktrees()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(wts), 2) // main + feature-x

	found := false
	for _, wt := range wts {
		if wt.Path == wtDir {
			found = true
			assert.Equal(t, "refs/heads/feature-x", wt.Branch)
		}
	}
	assert.True(t, found, "worktree not found in list")
}

func TestFindMainWorktree_Main(t *testing.T) {
	dir := initGitRepo(t)

	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)

	path, err := FindMainWorktree()
	require.NoError(t, err)
	assert.Equal(t, dir, path)
}

func TestFindMainWorktree_Master(t *testing.T) {
	// Create a repo with master instead of main
	dir := t.TempDir()
	runRawGit(t, dir, "init", "-b", "master")
	runRawGit(t, dir, "config", "user.email", "test@test")
	runRawGit(t, dir, "config", "user.name", "test")
	writeFile(t, dir, "README.md", "# test")
	runRawGit(t, dir, "add", "README.md")
	runRawGit(t, dir, "commit", "-m", "initial")

	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)

	path, err := FindMainWorktree()
	require.NoError(t, err)
	assert.Equal(t, dir, path)
}

func TestCheckedOutBranches(t *testing.T) {
	dir := initGitRepo(t)

	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)

	branches := checkedOutBranches()
	assert.NotNil(t, branches)
	// The main branch should be checked out (we just created it)
	assert.True(t, branches["main"], "main branch should be checked out")
}

// ---------------------------------------------------------------------------
// Remote operation tests
// ---------------------------------------------------------------------------

// setupRemote creates a local repo with a bare remote, adds a commit to remote, returns (localDir, remoteDir)
func setupRemote(t *testing.T) (string, string) {
	t.Helper()
	remoteDir := initBareRepo(t)
	localDir := initGitRepo(t)
	runRawGit(t, localDir, "remote", "add", "origin", remoteDir)
	return localDir, remoteDir
}

// pushToRemote pushes the current branch to origin
func pushToRemote(t *testing.T, dir string) {
	t.Helper()
	runRawGit(t, dir, "push", "-u", "origin", "main")
}

func TestFetchOrigin(t *testing.T) {
	localDir, remoteDir := setupRemote(t)
	pushToRemote(t, localDir)

	// Make a new commit in another clone and push to remote
	otherDir := t.TempDir()
	runRawGit(t, otherDir, "clone", remoteDir, otherDir)
	runRawGit(t, otherDir, "config", "user.email", "other@test")
	runRawGit(t, otherDir, "config", "user.name", "other")
	os.WriteFile(filepath.Join(otherDir, "other.txt"), []byte("other"), 0644)
	runRawGit(t, otherDir, "add", "other.txt")
	runRawGit(t, otherDir, "commit", "-m", "other commit")
	runRawGit(t, otherDir, "push", "origin", "main")

	// Now fetch from local
	oldDir, _ := os.Getwd()
	os.Chdir(localDir)
	defer os.Chdir(oldDir)

	err := FetchOrigin()
	require.NoError(t, err)
}

func TestListRemoteBranches(t *testing.T) {
	localDir, remoteDir := setupRemote(t)

	// Create another branch remotely
	otherDir := t.TempDir()
	runRawGit(t, otherDir, "clone", remoteDir, otherDir)
	runRawGit(t, otherDir, "config", "user.email", "other@test")
	runRawGit(t, otherDir, "config", "user.name", "other")
	os.WriteFile(filepath.Join(otherDir, "feature.txt"), []byte("feature"), 0644)
	runRawGit(t, otherDir, "add", "feature.txt")
	runRawGit(t, otherDir, "commit", "-m", "feature commit")

	// Save commit time before push
	out := runRawGit(t, otherDir, "log", "-1", "--format=%aI")
	_ = out

	runRawGit(t, otherDir, "checkout", "-b", "feature-branch")
	runRawGit(t, otherDir, "push", "origin", "feature-branch")
	runRawGit(t, otherDir, "push", "origin", "main")

	// Fetch into local
	oldDir, _ := os.Getwd()
	os.Chdir(localDir)
	defer os.Chdir(oldDir)

	runRawGit(t, localDir, "fetch", "origin")

	branches, err := ListRemoteBranches()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(branches), 2) // main + feature-branch

	names := make(map[string]bool)
	for _, b := range branches {
		names[b.Name] = true
	}
	assert.True(t, names["main"])
	assert.True(t, names["feature-branch"])
}

func TestListRemoteBranches_NoRemote(t *testing.T) {
	dir := initGitRepo(t)
	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)

	branches, err := ListRemoteBranches()
	if err != nil {
		return
	}
	// No remote refs exist, so branches should be empty
	assert.Empty(t, branches)
}

func TestCurrentWorktreePath(t *testing.T) {
	dir := initGitRepo(t)

	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)

	path, err := CurrentWorktreePath()
	require.NoError(t, err)
	assert.Contains(t, path, filepath.Base(dir))
}

func TestListWorktrees_Error(t *testing.T) {
	// Test error path: Run fails
	oldCmdFn := CmdFn
	defer func() { CmdFn = oldCmdFn }()

	CmdFn = func(name string, args ...string) *exec.Cmd {
		return exec.Command("false") // always fails
	}

	_, err := ListWorktrees()
	require.Error(t, err)
}

func TestRun_ErrorPath_ExitError(t *testing.T) {
	oldCmdFn := CmdFn
	defer func() { CmdFn = oldCmdFn }()

	CmdFn = func(name string, args ...string) *exec.Cmd {
		// Run a git command that will fail with exit error
		return exec.Command("git", "merge", "--invalid-flag-that-doesnt-exist-xyz")
	}

	_, err := Run("merge", "--invalid-flag-that-doesnt-exist-xyz")
	require.Error(t, err)
	// Should contain "git" and the failed command
	assert.Contains(t, err.Error(), "git")
}

func TestRunInternal_ErrorPath(t *testing.T) {
	oldCmdFn := CmdFn
	defer func() { CmdFn = oldCmdFn }()

	CmdFn = func(name string, args ...string) *exec.Cmd {
		return exec.Command("nonexistent-command-that-will-fail")
	}

	_, err := RunInternal("nonexistent")
	require.Error(t, err)
}

func TestFindMainWorktree_NoWorktree(t *testing.T) {
	// Test the fallback when neither main nor master exists
	// Override CmdFn to simulate a worktree list with neither main nor master
	oldCmdFn := CmdFn
	defer func() { CmdFn = oldCmdFn }()

	CmdFn = func(name string, args ...string) *exec.Cmd {
		if len(args) >= 2 && args[0] == "worktree" && args[1] == "list" {
			// Return porcelain output with a bare repo and a repo with no main/master
			return exec.Command("echo", `worktree /some/path
HEAD abc
branch refs/heads/some-other-branch

worktree /bare/path
HEAD def
bare

`)
		}
		return exec.Command(name, args...)
	}

	path, err := FindMainWorktree()
	require.NoError(t, err)
	assert.Equal(t, "/some/path", path)
}

func TestFindMainWorktree_NoWorktreeAtAll(t *testing.T) {
	oldCmdFn := CmdFn
	defer func() { CmdFn = oldCmdFn }()

	CmdFn = func(name string, args ...string) *exec.Cmd {
		if len(args) >= 2 && args[0] == "worktree" && args[1] == "list" {
			return exec.Command("echo", "only bare repos\n")
		}
		return exec.Command(name, args...)
	}

	_, err := FindMainWorktree()
	require.Error(t, err)
}
