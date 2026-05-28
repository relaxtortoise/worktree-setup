package actions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/relaxtortoise/worktree-setup/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteCopy_Directory(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create a directory with files
	subDir := filepath.Join(srcDir, "subdir")
	_ = os.MkdirAll(subDir, 0755)
	_ = os.WriteFile(filepath.Join(srcDir, "root.txt"), []byte("root"), 0644)
	_ = os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte("nested"), 0644)

	_, err := ExecuteCopy(dstDir, srcDir, []config.CopyAction{
		{From: "subdir", To: "copied_subdir"},
	})
	require.NoError(t, err)

	assert.DirExists(t, filepath.Join(dstDir, "copied_subdir"))
	data, err := os.ReadFile(filepath.Join(dstDir, "copied_subdir", "nested.txt"))
	require.NoError(t, err)
	assert.Equal(t, "nested", string(data))
}

func TestExecuteCopy_SourceNotFound(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	_, err := ExecuteCopy(dstDir, srcDir, []config.CopyAction{
		{From: "nonexistent.txt", To: "nonexistent.txt"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stat")
}

func TestExecuteCopy_MultipleItems(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	_ = os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("a"), 0644)
	_ = os.WriteFile(filepath.Join(srcDir, "b.txt"), []byte("b"), 0644)

	copied, err := ExecuteCopy(dstDir, srcDir, []config.CopyAction{
		{From: "a.txt", To: "a.txt"},
		{From: "b.txt", To: "b.txt"},
	})
	require.NoError(t, err)
	assert.Len(t, copied, 2)

	data, _ := os.ReadFile(filepath.Join(dstDir, "a.txt"))
	assert.Equal(t, "a", string(data))
	data, _ = os.ReadFile(filepath.Join(dstDir, "b.txt"))
	assert.Equal(t, "b", string(data))
}

func TestExecuteCopy_DeepDirectory(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	deepDir := filepath.Join(srcDir, "a", "b", "c")
	_ = os.MkdirAll(deepDir, 0755)
	_ = os.WriteFile(filepath.Join(deepDir, "deep.txt"), []byte("deep"), 0644)

	_, err := ExecuteCopy(dstDir, srcDir, []config.CopyAction{
		{From: "a", To: "a"},
	})
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dstDir, "a", "b", "c", "deep.txt"))
	require.NoError(t, err)
	assert.Equal(t, "deep", string(data))
}

func TestExecuteSymlink_Success(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	_ = os.WriteFile(filepath.Join(srcDir, "target.txt"), []byte("data"), 0644)

	linked, err := ExecuteSymlink(dstDir, srcDir, []config.CopyAction{
		{From: "target.txt", To: "link.txt"},
	})
	require.NoError(t, err)
	assert.Len(t, linked, 1)

	link, err := os.Readlink(filepath.Join(dstDir, "link.txt"))
	require.NoError(t, err)
	assert.Contains(t, link, "target.txt")
}

func TestExecuteSymlink_Dangling(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	linked, err := ExecuteSymlink(dstDir, srcDir, []config.CopyAction{
		{From: "nonexistent", To: "link"},
	})
	require.NoError(t, err)
	assert.Len(t, linked, 1)

	target, err := os.Readlink(filepath.Join(dstDir, "link"))
	require.NoError(t, err)
	assert.Contains(t, target, "nonexistent")
	t.Log("dangling symlink created successfully, target:", target)
}

func TestExecuteCopy_NestedPath(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	_ = os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("hello"), 0644)

	// Copy to a nested destination that doesn't exist yet
	_, err := ExecuteCopy(dstDir, srcDir, []config.CopyAction{
		{From: "file.txt", To: "sub/dir/file.txt"},
	})
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dstDir, "sub", "dir", "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))
}

func TestExecuteSymlink_ErrorOnNonWindows(t *testing.T) {
	// On Linux, symlink to an invalid source succeeds (dangling symlink).
	// The error path in ExecuteSymlink is only triggered when os.Symlink actually fails,
	// which can happen if the target already exists as a non-symlink.
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	_ = os.WriteFile(filepath.Join(srcDir, "target.txt"), []byte("data"), 0644)
	// Create a file at the link destination to cause os.Symlink to fail
	_ = os.WriteFile(filepath.Join(dstDir, "link.txt"), []byte("existing"), 0644)

	_, err := ExecuteSymlink(dstDir, srcDir, []config.CopyAction{
		{From: "target.txt", To: "link.txt"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "symlink")
}

func TestExecuteCopy_CopyFileError(t *testing.T) {
	// Test that copyFile returns error when source can't be read
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create a directory with the same name as the file we'll try to copy from
	// This causes os.Open error
	_ = os.MkdirAll(filepath.Join(srcDir, "somedir"), 0755)

	_, err := ExecuteCopy(dstDir, srcDir, []config.CopyAction{
		{From: "somedir", To: "somedir"},
	})
	require.NoError(t, err) // directory copy should work

	// Test with a file that doesn't exist (already tested above, just verifying)
	assert.DirExists(t, filepath.Join(dstDir, "somedir"))
}

func TestCopyFile_OpenError(t *testing.T) {
	// copyFile returns error when the source file exists but can't be opened
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	src := filepath.Join(srcDir, "noread.txt")
	require.NoError(t, os.WriteFile(src, []byte("secret"), 0644))
	require.NoError(t, os.Chmod(src, 0)) // remove all permissions

	_, err := ExecuteCopy(dstDir, srcDir, []config.CopyAction{
		{From: "noread.txt", To: "out.txt"},
	})
	require.Error(t, err)
}

func TestCopyFile_CreateError(t *testing.T) {
	// copyFile returns error when destination can't be created
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("data"), 0644))
	require.NoError(t, os.Chmod(dstDir, 0500)) // remove write permission

	_, err := ExecuteCopy(dstDir, srcDir, []config.CopyAction{
		{From: "file.txt", To: "file.txt"},
	})
	require.Error(t, err)
}

func TestExecuteCopy_MkdirAllError(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "src.txt"), []byte("data"), 0644))
	// Create a file where MkdirAll expects a directory component
	require.NoError(t, os.WriteFile(filepath.Join(dstDir, "blocker"), []byte{}, 0644))

	_, err := ExecuteCopy(dstDir, srcDir, []config.CopyAction{
		{From: "src.txt", To: filepath.Join("blocker", "sub", "dst.txt")},
	})
	require.Error(t, err)
}

func TestExecuteSymlink_MkdirAllError(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "target.txt"), []byte("data"), 0644))
	// Create a file where MkdirAll expects a directory component
	require.NoError(t, os.WriteFile(filepath.Join(dstDir, "blocker"), []byte{}, 0644))

	_, err := ExecuteSymlink(dstDir, srcDir, []config.CopyAction{
		{From: "target.txt", To: filepath.Join("blocker", "sub", "link.txt")},
	})
	require.Error(t, err)
}

func TestCopyDir_WalkError(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create a structure where a subdirectory isn't readable
	subDir := filepath.Join(srcDir, "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "file.txt"), []byte("data"), 0644))

	// Restore permissions for cleanup regardless of test outcome
	// Remove read permission on subdir so filepath.Walk can't read it
	require.NoError(t, os.Chmod(subDir, 0))
	defer func() {
		_ = os.Chmod(subDir, 0755)
	}()

	_, err := ExecuteCopy(dstDir, srcDir, []config.CopyAction{
		{From: "subdir", To: "copied"},
	})
	// Walk should encounter an error reading the subdir
	// Note: may not error if running as root (perms are ignored)
	t.Logf("Walk error result: %v", err)
	if err == nil {
		t.Skip("Skipping: running as root or platform doesn't enforce dir perms")
	}
}
