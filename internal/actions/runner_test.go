package actions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/relaxtortoise/worktree-setup/internal/config"
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

	os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("hello"), 0644)

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
