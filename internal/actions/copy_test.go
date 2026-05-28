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
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "root.txt"), []byte("root"), 0644)
	os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte("nested"), 0644)

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

	os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(srcDir, "b.txt"), []byte("b"), 0644)

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
	os.MkdirAll(deepDir, 0755)
	os.WriteFile(filepath.Join(deepDir, "deep.txt"), []byte("deep"), 0644)

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

	os.WriteFile(filepath.Join(srcDir, "target.txt"), []byte("data"), 0644)

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

	os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("hello"), 0644)

	// Copy to a nested destination that doesn't exist yet
	_, err := ExecuteCopy(dstDir, srcDir, []config.CopyAction{
		{From: "file.txt", To: "sub/dir/file.txt"},
	})
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dstDir, "sub", "dir", "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))
}
