package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMergeConfigs_HigherPriorityWins(t *testing.T) {
	global := &Config{PathStrategy: &PathStrategy{Name: "sibling"}}
	project := &Config{MainWorktree: "/home/me/projects/myapp"}
	repo := &Config{PathStrategy: &PathStrategy{Name: "nested"}}

	result := Merge(global, project, repo)
	if result.MainWorktree != "/home/me/projects/myapp" {
		t.Errorf("MainWorktree = %q", result.MainWorktree)
	}
	if result.PathStrategy.Name != "nested" {
		t.Errorf("PathStrategy = %q", result.PathStrategy.Name)
	}
}

func TestMergeConfigs_NilSkipped(t *testing.T) {
	result := Merge(nil, nil, &Config{MainWorktree: "/x"})
	if result.MainWorktree != "/x" {
		t.Errorf("MainWorktree = %q", result.MainWorktree)
	}
}

func TestLoadHierarchy(t *testing.T) {
	// 创建临时仓库
	repoDir := t.TempDir()
	os.MkdirAll(filepath.Join(repoDir, ".git"), 0755)
	repoYAML := `
path_strategy: nested
on:
  post-create:
    run:
      - "echo repo"
`
	os.WriteFile(filepath.Join(repoDir, ".worktree.yaml"), []byte(repoYAML), 0644)

	// 项目个人配置
	cfgDir := t.TempDir()
	projDir := filepath.Join(cfgDir, "projects", "github.com-owner-repo")
	os.MkdirAll(projDir, 0755)
	projYAML := `main_worktree: /home/me/projects/myapp`
	os.WriteFile(filepath.Join(projDir, "config.yaml"), []byte(projYAML), 0644)

	// 全局配置
	globalYAML := `path_strategy: sibling`
	os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte(globalYAML), 0644)

	hierarchy, err := LoadHierarchy(repoDir, cfgDir, "github.com-owner-repo")
	if err != nil {
		t.Fatalf("LoadHierarchy: %v", err)
	}
	if hierarchy.MainWorktree != "/home/me/projects/myapp" {
		t.Errorf("MainWorktree = %q", hierarchy.MainWorktree)
	}
	if hierarchy.PathStrategy.Name != "nested" {
		t.Errorf("PathStrategy = %q", hierarchy.PathStrategy.Name)
	}
}
