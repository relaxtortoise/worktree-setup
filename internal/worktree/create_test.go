package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relaxtortoise/worktree-setup/internal/config"
	gitpkg "github.com/relaxtortoise/worktree-setup/internal/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// setupWorktreeRepo creates a real git repo with a main branch.
func setupWorktreeRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test")
	runGit(t, dir, "config", "user.name", "test")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0644))
	runGit(t, dir, "add", "README.md")
	runGit(t, dir, "commit", "-m", "initial")
	return dir
}

// runGit runs a native git command in the given directory (bypasses git.CmdFn).
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed in %s: %s\n%s", args, dir, err, out)
	}
}

// overrideCmd replaces git.CmdFn so that every git command:
//   - runs inside mainDir (so we don't need os.Chdir)
//   - strips "origin/<branch>" arguments from "git worktree add" calls
//     because test repos don't have a remote.
//
// Usage: defer overrideCmd(t, mainDir)()
func overrideCmd(t *testing.T, mainDir string) func() {
	t.Helper()
	oldFn := gitpkg.CmdFn
	gitpkg.CmdFn = func(name string, args ...string) *exec.Cmd {
		clean := make([]string, 0, len(args))
		for _, a := range args {
			if !strings.HasPrefix(a, "origin/") {
				clean = append(clean, a)
			}
		}
		cmd := exec.Command(name, clean...)
		cmd.Dir = mainDir
		return cmd
	}
	return func() { gitpkg.CmdFn = oldFn }
}

// ---------------------------------------------------------------------------
// Create tests
// ---------------------------------------------------------------------------

func TestCreate_Success(t *testing.T) {
	mainDir := setupWorktreeRepo(t)
	cfg := &config.Config{MainWorktree: mainDir}
	defer overrideCmd(t, mainDir)()

	path, err := Create("feature-test", "", "github.com-owner-repo", cfg, false)
	require.NoError(t, err)
	assert.DirExists(t, path)
	assert.Contains(t, path, "feature-test")
}

func TestCreate_ExplicitPath(t *testing.T) {
	mainDir := setupWorktreeRepo(t)
	wtDir := filepath.Join(t.TempDir(), "my-worktree")
	cfg := &config.Config{MainWorktree: mainDir}
	defer overrideCmd(t, mainDir)()

	path, err := Create("feature-explicit", wtDir, "github.com-owner-repo", cfg, false)
	require.NoError(t, err)
	assert.Equal(t, wtDir, path)
	assert.DirExists(t, path)
}

func TestCreate_AutoFetch(t *testing.T) {
	mainDir := setupWorktreeRepo(t)
	remoteDir := t.TempDir()
	runGit(t, remoteDir, "init", "--bare", "-b", "main")
	runGit(t, mainDir, "remote", "add", "origin", remoteDir)
	cfg := &config.Config{MainWorktree: mainDir}
	defer overrideCmd(t, mainDir)()

	path, err := Create("feature-fetch", "", "github.com-owner-repo", cfg, true)
	require.NoError(t, err)
	assert.DirExists(t, path)
}

func TestCreate_NoMainWorktree_AutoDetect(t *testing.T) {
	mainDir := setupWorktreeRepo(t)
	cfg := &config.Config{MainWorktree: ""} // auto-detect
	defer overrideCmd(t, mainDir)()

	// Chdir so that FindMainWorktree can also rely on cwd as a fallback
	oldDir, _ := os.Getwd()
	_ = os.Chdir(mainDir)
	defer func() { _ = os.Chdir(oldDir) }()

	path, err := Create("feature-auto", "", "github.com-owner-repo", cfg, false)
	require.NoError(t, err)
	assert.DirExists(t, path)
}

func TestCreate_PreCreateHook(t *testing.T) {
	mainDir := setupWorktreeRepo(t)
	outFile := filepath.Join(t.TempDir(), "precreate-out.txt")
	cfg := &config.Config{
		MainWorktree: mainDir,
		On: &config.Events{
			PreCreate: &config.Event{
				Run: []string{"echo precreate-done > " + outFile},
			},
		},
	}
	defer overrideCmd(t, mainDir)()

	path, err := Create("feature-precreate", "", "github.com-owner-repo", cfg, false)
	require.NoError(t, err)
	assert.DirExists(t, path)
	data, err := os.ReadFile(outFile)
	require.NoError(t, err)
	assert.Contains(t, string(data), "precreate-done")
}

func TestCreate_PostCreateHook(t *testing.T) {
	mainDir := setupWorktreeRepo(t)
	outFile := filepath.Join(t.TempDir(), "postcreate-out.txt")
	cfg := &config.Config{
		MainWorktree: mainDir,
		On: &config.Events{
			PostCreate: &config.Event{
				Run: []string{"echo postcreate-done > " + outFile},
			},
		},
	}
	defer overrideCmd(t, mainDir)()

	path, err := Create("feature-postcreate", "", "github.com-owner-repo", cfg, false)
	require.NoError(t, err)
	assert.DirExists(t, path)
	data, err := os.ReadFile(outFile)
	require.NoError(t, err)
	assert.Contains(t, string(data), "postcreate-done")
}

func TestCreate_InvalidBranch(t *testing.T) {
	mainDir := setupWorktreeRepo(t)
	cfg := &config.Config{MainWorktree: mainDir}
	defer overrideCmd(t, mainDir)()

	// ".." is not allowed in git branch names.
	_, err := Create("feature..test", "", "github.com-owner-repo", cfg, false)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Create error-path tests
// ---------------------------------------------------------------------------

func TestCreate_MainWorktreeDetectionError(t *testing.T) {
	oldFn := gitpkg.CmdFn
	defer func() { gitpkg.CmdFn = oldFn }()
	gitpkg.CmdFn = func(name string, args ...string) *exec.Cmd {
		return exec.Command("git", "invalid-command-that-does-not-exist-xyz")
	}

	_, err := Create("feature-test", "", "github.com-owner-repo", &config.Config{MainWorktree: ""}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot determine main worktree")
}

func TestCreate_PreCreateHookError(t *testing.T) {
	mainDir := setupWorktreeRepo(t)
	cfg := &config.Config{
		MainWorktree: mainDir,
		On: &config.Events{
			PreCreate: &config.Event{
				Run: []string{"exit 1"},
			},
		},
	}
	defer overrideCmd(t, mainDir)()

	_, err := Create("feature-x", "", "github.com-owner-repo", cfg, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pre-create")
}

func TestCreate_PostCreateHookError(t *testing.T) {
	mainDir := setupWorktreeRepo(t)
	cfg := &config.Config{
		MainWorktree: mainDir,
		On: &config.Events{
			PostCreate: &config.Event{
				Run: []string{"exit 1"},
			},
		},
	}
	defer overrideCmd(t, mainDir)()

	path, err := Create("feature-y", "", "github.com-owner-repo", cfg, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "post-create")
	// Worktree directory should still exist even when the hook fails.
	assert.NotEmpty(t, path)
	assert.DirExists(t, path)
}

// ---------------------------------------------------------------------------
// Remove tests
// ---------------------------------------------------------------------------

func TestRemove_Success(t *testing.T) {
	mainDir := setupWorktreeRepo(t)
	cfg := &config.Config{MainWorktree: mainDir}
	defer overrideCmd(t, mainDir)()

	path, err := Create("to-remove", "", "github.com-owner-repo", cfg, false)
	require.NoError(t, err)
	assert.DirExists(t, path)

	err = Remove(path, cfg, false)
	require.NoError(t, err)
	assert.NoDirExists(t, path)
}

func TestRemove_Force(t *testing.T) {
	mainDir := setupWorktreeRepo(t)
	cfg := &config.Config{MainWorktree: mainDir}
	defer overrideCmd(t, mainDir)()

	path, err := Create("to-force-remove", "", "github.com-owner-repo", cfg, false)
	require.NoError(t, err)

	// Dirty the worktree so a normal remove would fail.
	require.NoError(t, os.WriteFile(filepath.Join(path, "dirty.txt"), []byte("dirty"), 0644))

	err = Remove(path, cfg, true)
	require.NoError(t, err)
	assert.NoDirExists(t, path)
}

func TestRemove_NoMainWorktree(t *testing.T) {
	mainDir := setupWorktreeRepo(t)
	cfg := &config.Config{MainWorktree: mainDir}
	defer overrideCmd(t, mainDir)()

	path, err := Create("auto-remove", "", "github.com-owner-repo", cfg, false)
	require.NoError(t, err)

	cfg2 := &config.Config{MainWorktree: ""} // auto-detect

	err = Remove(path, cfg2, false)
	require.NoError(t, err)
}

func TestRemove_WithHooks(t *testing.T) {
	mainDir := setupWorktreeRepo(t)
	preOut := filepath.Join(t.TempDir(), "predelete-out.txt")
	postOut := filepath.Join(t.TempDir(), "postdelete-out.txt")
	cfg := &config.Config{
		MainWorktree: mainDir,
		On: &config.Events{
			PreDelete:  &config.Event{Run: []string{"echo predelete > " + preOut}},
			PostDelete: &config.Event{Run: []string{"echo postdelete > " + postOut}},
		},
	}
	defer overrideCmd(t, mainDir)()

	path, err := Create("hook-remove", "", "github.com-owner-repo", cfg, false)
	require.NoError(t, err)

	err = Remove(path, cfg, false)
	require.NoError(t, err)

	data, err := os.ReadFile(preOut)
	require.NoError(t, err)
	assert.Contains(t, string(data), "predelete")

	data, err = os.ReadFile(postOut)
	require.NoError(t, err)
	assert.Contains(t, string(data), "postdelete")
}

// ---------------------------------------------------------------------------
// Remove error-path tests
// ---------------------------------------------------------------------------

func TestRemove_NonExistent(t *testing.T) {
	mainDir := setupWorktreeRepo(t)
	cfg := &config.Config{MainWorktree: mainDir}
	defer overrideCmd(t, mainDir)()

	err := Remove(filepath.Join(t.TempDir(), "nonexistent"), cfg, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git worktree remove")
}

func TestRemove_MainWorktreeDetectionError(t *testing.T) {
	oldFn := gitpkg.CmdFn
	defer func() { gitpkg.CmdFn = oldFn }()
	gitpkg.CmdFn = func(name string, args ...string) *exec.Cmd {
		return exec.Command("git", "invalid-command-that-does-not-exist-xyz")
	}

	err := Remove("/tmp/nonexistent", &config.Config{MainWorktree: ""}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot determine main worktree")
}

func TestRemove_PreDeleteHookError(t *testing.T) {
	mainDir := setupWorktreeRepo(t)
	cfg := &config.Config{
		MainWorktree: mainDir,
		On: &config.Events{
			PreDelete: &config.Event{
				Run: []string{"exit 1"},
			},
		},
	}
	defer overrideCmd(t, mainDir)()

	path, err := Create("predelete-fail", "", "github.com-owner-repo", cfg, false)
	require.NoError(t, err)

	err = Remove(path, cfg, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pre-delete")
}

func TestRemove_PostDeleteHookError(t *testing.T) {
	mainDir := setupWorktreeRepo(t)
	cfg := &config.Config{
		MainWorktree: mainDir,
		On: &config.Events{
			PostDelete: &config.Event{
				Run: []string{"exit 1"},
			},
		},
	}
	defer overrideCmd(t, mainDir)()

	path, err := Create("postdelete-fail", "", "github.com-owner-repo", cfg, false)
	require.NoError(t, err)

	err = Remove(path, cfg, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "post-delete")
}
