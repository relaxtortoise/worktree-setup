package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/relaxtortoise/worktree-setup/internal/selfupdate"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetFlags resets package-level flag variables and cobra help flags to their
// zero/default values. Cobra's flag parsing modifies package-level vars and
// the help bool on FlagSets during Execute(), and these persist across tests
// because the FlagSet pointer is shared.
func resetFlags() {
	globalConfig = false
	removeForce = false
	selfUpdateYes = false
	selfUpdateCheck = false
	noFetch = false
	explicitPath = ""
	detectCreate = false
	noGitignore = false

	// Reset cobra's help flag on every command to prevent --help from previous
	// tests leaking into subsequent tests via the shared FlagSet pointer.
	resetCommandHelp(rootCmd)
}

func resetCommandHelp(cmd *cobra.Command) {
	if cmd == nil {
		return
	}
	if f := cmd.Flags().Lookup("help"); f != nil {
		_ = f.Value.Set("false")
	}
	for _, child := range cmd.Commands() {
		resetCommandHelp(child)
	}
}

// executeCommand runs the given cobra command with args and returns stdout,
// stderr, and error. This uses rootCmd directly (not a copy) and relies on
// resetFlags() being called before each test to clear persisted state.
func executeCommand(args ...string) (string, string, error) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	rootCmd.SetArgs(args)
	err := rootCmd.Execute()

	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufOut, bufErr bytes.Buffer
	_, _ = io.Copy(&bufOut, rOut)
	_, _ = io.Copy(&bufErr, rErr)

	return bufOut.String(), bufErr.String(), err
}

// initGitRepo creates a git repo in dir with an initial commit.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	runGitCmd(t, dir, "init", "-b", "main")
	runGitCmd(t, dir, "config", "user.email", "test@test")
	runGitCmd(t, dir, "config", "user.name", "test")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0644))
	runGitCmd(t, dir, "add", "README.md")
	runGitCmd(t, dir, "commit", "-m", "initial")
}

// runGitCmd runs a git command in the given directory.
func runGitCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v failed: %s", args, out)
	return string(out)
}

// chdir changes to dir and returns a function to restore the original directory.
func chdir(t *testing.T, dir string) func() {
	t.Helper()
	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	return func() { os.Chdir(oldDir) }
}

// ---------------------------------------------------------------------------
// Table-driven help tests
// ---------------------------------------------------------------------------

func TestCommands_Help(t *testing.T) {
	tests := []struct {
		name string
		args []string
		sub  string
	}{
		{name: "root", args: []string{"--help"}, sub: "wt"},
		{name: "version", args: []string{"version", "--help"}, sub: "version"},
		{name: "list", args: []string{"list", "--help"}, sub: "worktrees"},
		{name: "add", args: []string{"add", "--help"}, sub: "worktree"},
		{name: "remove", args: []string{"remove", "--help"}, sub: "worktree"},
		{name: "switch", args: []string{"switch", "--help"}, sub: "worktree"},
		{name: "hooks", args: []string{"hooks", "--help"}, sub: "hooks"},
		{name: "init", args: []string{"init", "--help"}, sub: "worktree"},
		{name: "config", args: []string{"config", "--help"}, sub: "config"},
		{name: "run", args: []string{"run", "--help"}, sub: "event"},
		{name: "self-update", args: []string{"self-update", "--help"}, sub: "self-update"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetFlags()
			out, _, err := executeCommand(tt.args...)
			require.NoError(t, err)
			assert.Contains(t, out, tt.sub)
		})
	}
}

// ---------------------------------------------------------------------------
// Table-driven flag existence tests
// ---------------------------------------------------------------------------

func TestCommands_Flags(t *testing.T) {
	tests := []struct {
		name string
		cmd  *cobra.Command
		flag string
	}{
		{name: "add/no-fetch", cmd: addCmd, flag: "no-fetch"},
		{name: "add/path", cmd: addCmd, flag: "path"},
		{name: "remove/force", cmd: removeCmd, flag: "force"},
		{name: "run/detect-create", cmd: runCmd, flag: "detect-create"},
		{name: "config/global", cmd: configCmd, flag: "global"},
		{name: "self-update/yes", cmd: selfUpdateCmd, flag: "yes"},
		{name: "self-update/check", cmd: selfUpdateCmd, flag: "check"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.cmd.Flags().Lookup(tt.flag)
			require.NotNil(t, f, "flag --%s should exist on %s", tt.flag, tt.cmd.Name())
		})
	}
}

// ---------------------------------------------------------------------------
// Version
// ---------------------------------------------------------------------------

func TestVersionCmd(t *testing.T) {
	resetFlags()
	out, _, err := executeCommand("version")
	require.NoError(t, err)
	assert.Contains(t, out, "wt v")
}

// ---------------------------------------------------------------------------
// Switch
// ---------------------------------------------------------------------------

func TestSwitchCmd_WithArg(t *testing.T) {
	resetFlags()
	out, _, err := executeCommand("switch", "mybranch")
	require.NoError(t, err)
	assert.Equal(t, "mybranch\n", out)
}

// ---------------------------------------------------------------------------
// Config (global)
// ---------------------------------------------------------------------------

func TestConfigCmd_Global_NoConfig(t *testing.T) {
	resetFlags()
	t.Setenv("HOME", t.TempDir())
	out, _, err := executeCommand("config", "--global", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "(no config)")
}

func TestConfigCmd_Global_SetAndGet(t *testing.T) {
	resetFlags()
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Set main_worktree
	out, _, err := executeCommand("config", "--global", "set", "main_worktree", "/tmp/test")
	require.NoError(t, err)
	assert.Empty(t, out)

	// Get it back
	out, _, err = executeCommand("config", "--global", "get", "main_worktree")
	require.NoError(t, err)
	assert.Equal(t, "/tmp/test\n", out)

	// Set path_strategy (note: PathStrategy fields have yaml:"-" tags so they
	// don't round-trip through YAML serialization; this is a known limitation)
	_, _, err = executeCommand("config", "--global", "set", "path_strategy", "sibling")
	require.NoError(t, err)

	// List should contain main_worktree
	out, _, err = executeCommand("config", "--global", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "main_worktree")
	assert.Contains(t, out, "/tmp/test")
}

func TestConfigCmd_Global_Get_NoKey(t *testing.T) {
	resetFlags()
	t.Setenv("HOME", t.TempDir())
	_, _, err := executeCommand("config", "--global", "get")
	require.Error(t, err)
}

func TestConfigCmd_Global_Set_NoValue(t *testing.T) {
	resetFlags()
	t.Setenv("HOME", t.TempDir())
	_, _, err := executeCommand("config", "--global", "set", "main_worktree")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Config (project)
// ---------------------------------------------------------------------------

func TestConfigCmd_Project_NoGitRepo(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	defer chdir(t, dir)()

	_, _, err := executeCommand("config", "list")
	require.Error(t, err)
}

func TestConfigCmd_Project_List_NoConfig(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	initGitRepo(t, dir)
	runGitCmd(t, dir, "remote", "add", "origin", "https://github.com/owner/repo.git")
	t.Setenv("HOME", t.TempDir())
	defer chdir(t, dir)()

	out, _, err := executeCommand("config", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "(no config)")
}

func TestConfigCmd_Project_SetAndGet(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	initGitRepo(t, dir)
	runGitCmd(t, dir, "remote", "add", "origin", "https://github.com/owner/repo.git")
	t.Setenv("HOME", t.TempDir())
	defer chdir(t, dir)()

	// Set main_worktree
	out, _, err := executeCommand("config", "set", "main_worktree", "/tmp/main")
	require.NoError(t, err)
	assert.Empty(t, out)

	// Get it back
	out, _, err = executeCommand("config", "get", "main_worktree")
	require.NoError(t, err)
	assert.Equal(t, "/tmp/main\n", out)
}

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------

func TestHooksCmd(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	initGitRepo(t, dir)
	defer chdir(t, dir)()

	out, _, err := executeCommand("hooks")
	require.NoError(t, err)
	assert.Contains(t, out, "installed:")
	require.FileExists(t, filepath.Join(dir, ".git", "hooks", "post-checkout"))
}

func TestHooksCmd_NoGitRepo(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	defer chdir(t, dir)()

	_, _, err := executeCommand("hooks")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Init
// ---------------------------------------------------------------------------

func TestInitCmd(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	initGitRepo(t, dir)
	runGitCmd(t, dir, "remote", "add", "origin", "https://github.com/owner/repo.git")
	t.Setenv("HOME", t.TempDir())
	defer chdir(t, dir)()

	out, _, err := executeCommand("init")
	require.NoError(t, err)
	assert.Contains(t, out, "created")
	require.FileExists(t, filepath.Join(dir, ".worktree.yaml"))
}

func TestInitCmd_AlreadyExists(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	initGitRepo(t, dir)
	runGitCmd(t, dir, "remote", "add", "origin", "https://github.com/owner/repo.git")
	home := t.TempDir()
	t.Setenv("HOME", home)
	defer chdir(t, dir)()

	// Pre-create .worktree.yaml
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".worktree.yaml"), []byte("existing"), 0644))

	// Pre-create project config so the "already exists" path is also covered
	projDir := filepath.Join(home, ".config", "worktree-setup", "projects", "github.com-owner-repo")
	require.NoError(t, os.MkdirAll(projDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(projDir, "config.yaml"), []byte("main_worktree: /tmp/main\n"), 0644))

	out, _, err := executeCommand("init")
	require.NoError(t, err)
	assert.Contains(t, out, "skipping")
}

func TestInitCmd_FindMainWorktreeFallback(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	// Bare repos have no worktree branches, so FindMainWorktree fails
	// and the fallback mainWT = repoDir is used.
	runGitCmd(t, dir, "init", "--bare")
	runGitCmd(t, dir, "remote", "add", "origin", "https://github.com/owner/repo.git")
	t.Setenv("HOME", t.TempDir())
	defer chdir(t, dir)()

	out, _, err := executeCommand("init")
	require.NoError(t, err)
	assert.Contains(t, out, "created")
}

func TestInitCmd_NoGitRepo(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	defer chdir(t, dir)()

	_, _, err := executeCommand("init")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestListCmd_InGitRepo(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	initGitRepo(t, dir)
	defer chdir(t, dir)()

	out, _, err := executeCommand("list")
	require.NoError(t, err)
	assert.Contains(t, out, dir)
}

func TestListCmd_NoGitRepo(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	defer chdir(t, dir)()

	_, _, err := executeCommand("list")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Run
// ---------------------------------------------------------------------------

func TestRunCmd_UnknownEvent(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	initGitRepo(t, dir)
	runGitCmd(t, dir, "remote", "add", "origin", "https://github.com/owner/repo.git")
	defer chdir(t, dir)()

	_, _, err := executeCommand("run", "unknown-event")
	require.Error(t, err)
}

func TestRunCmd_PostCheckout(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	initGitRepo(t, dir)
	runGitCmd(t, dir, "remote", "add", "origin", "https://github.com/owner/repo.git")
	defer chdir(t, dir)()

	out, _, err := executeCommand("run", "post-checkout")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestRunCmd_PostCheckout_DetectCreate(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	initGitRepo(t, dir)
	runGitCmd(t, dir, "remote", "add", "origin", "https://github.com/owner/repo.git")
	defer chdir(t, dir)()

	// With --detect-create but not in a real git hook context, it falls
	// through to RunPostCheckout because IsNewWorktree returns false.
	out, _, err := executeCommand("run", "post-checkout", "--detect-create")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestRunCmd_NoGitRepo(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	defer chdir(t, dir)()

	// loadConfig fails when not in a git repo with remote origin.
	_, _, err := executeCommand("run", "post-checkout")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Remove
// ---------------------------------------------------------------------------

func TestRemoveCmd_MissingArg(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	defer chdir(t, dir)()

	_, _, err := executeCommand("remove")
	require.Error(t, err)
}

func TestRemoveCmd_MissingArg_InGitRepo(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	initGitRepo(t, dir)
	runGitCmd(t, dir, "remote", "add", "origin", "https://github.com/owner/repo.git")
	defer chdir(t, dir)()

	// Override configDir to avoid reading user's real config
	oldConfigDir := configDir
	configDir = t.TempDir()
	defer func() { configDir = oldConfigDir }()

	_, _, err := executeCommand("remove")
	require.Error(t, err)
}

func TestRemoveCmd_NonExistentPath(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	initGitRepo(t, dir)
	runGitCmd(t, dir, "remote", "add", "origin", "https://github.com/owner/repo.git")
	defer chdir(t, dir)()

	oldConfigDir := configDir
	configDir = t.TempDir()
	defer func() { configDir = oldConfigDir }()

	// loadConfig succeeds, worktree.Remove is called and fails because the
	// path does not correspond to any registered worktree.
	_, _, err := executeCommand("remove", "/nonexistent-worktree-path")
	require.Error(t, err)
}

func TestRemoveCmd_NonExistentPath_Force(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	initGitRepo(t, dir)
	runGitCmd(t, dir, "remote", "add", "origin", "https://github.com/owner/repo.git")
	defer chdir(t, dir)()

	oldConfigDir := configDir
	configDir = t.TempDir()
	defer func() { configDir = oldConfigDir }()

	_, _, err := executeCommand("remove", "/nonexistent-worktree-path", "--force")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Self-update
// ---------------------------------------------------------------------------

func TestSelfUpdateCmd_VersionNotFound(t *testing.T) {
	resetFlags()

	// Mock HTTP to prevent network access and return a known error
	oldDoHTTPGet := selfupdate.DoHTTPGet
	selfupdate.DoHTTPGet = func(url string) (*http.Response, error) {
		return nil, errors.New("mock error")
	}
	defer func() { selfupdate.DoHTTPGet = oldDoHTTPGet }()

	_, _, err := executeCommand("self-update", "v99.99.99", "--yes")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Add
// ---------------------------------------------------------------------------

func TestAddCmd_NoGitRepo(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	defer chdir(t, dir)()

	_, _, err := executeCommand("add", "mybranch")
	require.Error(t, err)
}

func TestAddCmd_WithBranch(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	initGitRepo(t, dir)
	runGitCmd(t, dir, "remote", "add", "origin", "https://github.com/owner/repo.git")
	defer chdir(t, dir)()

	oldConfigDir := configDir
	configDir = t.TempDir()
	defer func() { configDir = oldConfigDir }()

	// Use --no-fetch to prevent FetchOrigin() from hanging on a fake remote.
	_, _, err := executeCommand("add", "my-test-branch", "--no-fetch")
	require.Error(t, err)
}

func TestAddCmd_WT_NO_FETCH(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	initGitRepo(t, dir)
	runGitCmd(t, dir, "remote", "add", "origin", "https://github.com/owner/repo.git")
	defer chdir(t, dir)()

	oldConfigDir := configDir
	configDir = t.TempDir()
	defer func() { configDir = oldConfigDir }()

	// WT_NO_FETCH env var disables fetch without needing --no-fetch
	t.Setenv("WT_NO_FETCH", "1")
	_, _, err := executeCommand("add", "my-test-branch")
	require.Error(t, err)
}

func TestAddCmd_WithPath(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	initGitRepo(t, dir)
	runGitCmd(t, dir, "remote", "add", "origin", "https://github.com/owner/repo.git")
	defer chdir(t, dir)()

	oldConfigDir := configDir
	configDir = t.TempDir()
	defer func() { configDir = oldConfigDir }()

	_, _, err := executeCommand("add", "my-test-branch", "--path", "/tmp/nonexistent-wt", "--no-fetch")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Root execution
// ---------------------------------------------------------------------------

func TestRootCmd_NoArgs(t *testing.T) {
	resetFlags()
	out, _, err := executeCommand()
	require.NoError(t, err)
	assert.Contains(t, out, "wt")
	assert.Contains(t, out, "worktree")
}
