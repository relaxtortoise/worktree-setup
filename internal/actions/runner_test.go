package actions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/relaxtortoise/worktree-setup/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCommand(t *testing.T) {
	dir := t.TempDir()
	err := ExecuteRun([]string{"echo hello > test.txt"}, dir, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCopyFiles(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	_ = os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("hello"), 0644)

	_, err := ExecuteCopy(dstDir, srcDir, []config.CopyAction{
		{From: "a.txt", To: "a.txt"},
	})
	if err != nil {
		t.Fatalf("copy error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dstDir, "a.txt"))
	if err != nil {
		t.Fatal("file not created")
	}
	if string(data) != "hello" {
		t.Errorf("content = %q", string(data))
	}
}

func TestNewRunner(t *testing.T) {
	r := NewRunner("/home/me/projects/app")
	assert.NotNil(t, r)
	assert.Equal(t, "/home/me/projects/app", r.MainWorktree)
}

func TestExecuteEvent_NilEvent(t *testing.T) {
	r := NewRunner("/tmp")
	err := r.ExecuteEvent(nil, "/tmp/worktree")
	require.NoError(t, err)
}

func TestExecuteEvent_EmptySteps(t *testing.T) {
	r := NewRunner("/tmp")
	err := r.ExecuteEvent(&config.Event{}, "/tmp/worktree")
	require.NoError(t, err)
}

func TestExecuteEvent_RunStep(t *testing.T) {
	dir := t.TempDir()
	r := NewRunner(dir)
	event := &config.Event{
		Steps: []config.Step{
			{Run: "echo hello > " + filepath.Join(dir, "output.txt")},
		},
	}
	err := r.ExecuteEvent(event, dir)
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(dir, "output.txt"))
	require.NoError(t, err)
}

func TestExecuteEvent_CopyStep(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("hello"), 0644) //nolint:errcheck //nolint:errcheck

	r := NewRunner(srcDir)
	event := &config.Event{
		Steps: []config.Step{
			{Copy: &config.CopyItems{Items: []config.CopyAction{{From: "a.txt", To: "a.txt"}}}},
		},
	}
	err := r.ExecuteEvent(event, dstDir)
	require.NoError(t, err)
	data, _ := os.ReadFile(filepath.Join(dstDir, "a.txt"))
	assert.Equal(t, "hello", string(data))
}

func TestExecuteEvent_SymlinkStep(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "target.txt"), []byte("target"), 0644) //nolint:errcheck //nolint:errcheck

	r := NewRunner(srcDir)
	event := &config.Event{
		Steps: []config.Step{
			{Symlink: &config.CopyItems{Items: []config.CopyAction{{From: "target.txt", To: "link.txt"}}}},
		},
	}
	err := r.ExecuteEvent(event, dstDir)
	require.NoError(t, err)
	link, err := os.Readlink(filepath.Join(dstDir, "link.txt"))
	require.NoError(t, err)
	assert.Contains(t, link, "target.txt")
}

func TestExecuteEvent_StepFailure(t *testing.T) {
	r := NewRunner("/tmp")
	event := &config.Event{
		Steps: []config.Step{
			{Run: "exit 1"},
		},
	}
	err := r.ExecuteEvent(event, "/tmp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "step 1")
}

func TestExecuteEvent_MultiLineRun(t *testing.T) {
	dir := t.TempDir()
	r := NewRunner(dir)
	event := &config.Event{
		Steps: []config.Step{
			{Run: "echo first > " + filepath.Join(dir, "f.txt") + "\necho second >> " + filepath.Join(dir, "f.txt")},
		},
	}
	err := r.ExecuteEvent(event, dir)
	require.NoError(t, err)
	data, _ := os.ReadFile(filepath.Join(dir, "f.txt"))
	assert.Contains(t, string(data), "first")
	assert.Contains(t, string(data), "second")
}

func TestExecutePreCreate_NilEvent(t *testing.T) {
	r := NewRunner("/tmp")
	err := r.ExecutePreCreate(nil)
	require.NoError(t, err)
}

func TestExecutePreCreate_Run(t *testing.T) {
	dir := t.TempDir()
	r := NewRunner(dir)
	event := &config.Event{
		Run: []string{"echo precreate > " + filepath.Join(dir, "out.txt")},
	}
	err := r.ExecutePreCreate(event)
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(dir, "out.txt"))
	require.NoError(t, err)
}

func TestExecutePreCreate_Failure(t *testing.T) {
	r := NewRunner("/tmp")
	event := &config.Event{
		Run: []string{"exit 1"},
	}
	err := r.ExecutePreCreate(event)
	require.Error(t, err)
}

func TestExecuteRun_DryRun(t *testing.T) {
	dir := t.TempDir()
	err := ExecuteRun([]string{"touch " + filepath.Join(dir, "should-not-exist.txt")}, dir, true)
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(dir, "should-not-exist.txt"))
	assert.True(t, os.IsNotExist(err))
}

func TestExecuteRun_MultipleCommands(t *testing.T) {
	dir := t.TempDir()
	err := ExecuteRun([]string{
		"echo a > " + filepath.Join(dir, "a.txt"),
		"echo b > " + filepath.Join(dir, "b.txt"),
	}, dir, false)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, "a.txt"))
	assert.FileExists(t, filepath.Join(dir, "b.txt"))
}

func TestExecuteRun_EmptyCommands(t *testing.T) {
	err := ExecuteRun([]string{}, "/tmp", false)
	require.NoError(t, err)
}

func TestExecuteEvent_StepWithEmptyLines(t *testing.T) {
	dir := t.TempDir()
	r := NewRunner(dir)
	event := &config.Event{
		Steps: []config.Step{
			{Run: "\n  \necho hello > " + filepath.Join(dir, "out.txt") + "\n\n"},
		},
	}
	err := r.ExecuteEvent(event, dir)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, "out.txt"))
}

func TestExecuteEvent_StepWithCopyAndRun(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "src.txt"), []byte("data"), 0644) //nolint:errcheck //nolint:errcheck

	r := NewRunner(srcDir)
	event := &config.Event{
		Steps: []config.Step{
			{
				Copy: &config.CopyItems{Items: []config.CopyAction{{From: "src.txt", To: "dst.txt"}}},
				Run:  "echo done > " + filepath.Join(dstDir, "done.txt"),
			},
		},
	}
	err := r.ExecuteEvent(event, dstDir)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dstDir, "dst.txt"))
	assert.FileExists(t, filepath.Join(dstDir, "done.txt"))
}

func TestExecuteEvent_CopyStepFailure(t *testing.T) {
	dstDir := t.TempDir()
	r := NewRunner("/nonexistent-main")
	event := &config.Event{
		Steps: []config.Step{
			{Copy: &config.CopyItems{Items: []config.CopyAction{{From: "nope.txt", To: "nope.txt"}}}},
		},
	}
	err := r.ExecuteEvent(event, dstDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "copy")
}

func TestExecuteEvent_SymlinkStepFailure(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	// Create a file where the symlink would be placed to force os.Symlink to fail
	os.WriteFile(filepath.Join(dstDir, "link.txt"), []byte("existing"), 0644) //nolint:errcheck //nolint:errcheck

	r := NewRunner(srcDir)
	event := &config.Event{
		Steps: []config.Step{
			{Symlink: &config.CopyItems{Items: []config.CopyAction{{From: "target.txt", To: "link.txt"}}}},
		},
	}
	err := r.ExecuteEvent(event, dstDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "symlink")
}
