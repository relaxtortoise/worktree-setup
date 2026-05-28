package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	_ = os.MkdirAll(filepath.Join(repoDir, ".git"), 0755)
	repoYAML := `
path_strategy: nested
on:
  post-create:
    run:
      - "echo repo"
`
	_ = os.WriteFile(filepath.Join(repoDir, ".worktree.yaml"), []byte(repoYAML), 0644)

	// 项目个人配置
	cfgDir := t.TempDir()
	projDir := filepath.Join(cfgDir, "projects", "github.com-owner-repo")
	_ = os.MkdirAll(projDir, 0755)
	projYAML := `main_worktree: /home/me/projects/myapp`
	_ = os.WriteFile(filepath.Join(projDir, "config.yaml"), []byte(projYAML), 0644)

	// 全局配置
	globalYAML := `path_strategy: sibling`
	_ = os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte(globalYAML), 0644)

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

func TestMerge_AllNil(t *testing.T) {
	result := Merge(nil, nil, nil)
	require.NotNil(t, result)
	assert.Equal(t, &Config{}, result)
}

func TestMerge_PartialEvents(t *testing.T) {
	first := &Config{On: &Events{
		PreCreate:  &Event{Run: []string{"first"}},
		PostCreate: &Event{Run: []string{"second"}},
	}}
	second := &Config{On: &Events{
		PostCheckout: &Event{Run: []string{"third"}},
		PreDelete:    &Event{Run: []string{"fourth"}},
	}}
	result := Merge(first, second)
	assert.NotNil(t, result.On.PreCreate)
	assert.NotNil(t, result.On.PostCreate)
	assert.NotNil(t, result.On.PostCheckout)
	assert.NotNil(t, result.On.PreDelete)
	assert.Nil(t, result.On.PostDelete)
}

func TestMerge_EventOverride(t *testing.T) {
	first := &Config{On: &Events{
		PostCreate: &Event{Run: []string{"first version"}},
	}}
	second := &Config{On: &Events{
		PostCreate: &Event{Run: []string{"second version"}},
	}}
	result := Merge(first, second)
	assert.Equal(t, []string{"second version"}, result.On.PostCreate.Run)
}

func TestLoadHierarchy_NoConfigFiles(t *testing.T) {
	repoDir := t.TempDir()
	os.MkdirAll(filepath.Join(repoDir, ".git"), 0755)
	cfgDir := t.TempDir()

	cfg, err := LoadHierarchy(repoDir, cfgDir, "no-such-project")
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Empty(t, cfg.MainWorktree)
}

func TestLoadHierarchy_BrokenYAML(t *testing.T) {
	repoDir := t.TempDir()
	os.MkdirAll(filepath.Join(repoDir, ".git"), 0755)
	os.WriteFile(filepath.Join(repoDir, ".worktree.yaml"), []byte("{{{broken"), 0644)

	cfgDir := t.TempDir()

	cfg, err := LoadHierarchy(repoDir, cfgDir, "no-such-project")
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Empty(t, cfg.MainWorktree)
}

func TestUserConfigPaths(t *testing.T) {
	home := os.Getenv("HOME")
	assert.NotEmpty(t, home)

	userDir := UserConfigDir()
	assert.Contains(t, userDir, ".config/worktree-setup")

	globalPath := GlobalConfigPath()
	assert.Contains(t, globalPath, "config.yaml")

	projDir := ProjectConfigDir("github.com-owner-repo")
	assert.Contains(t, projDir, "projects/github.com-owner-repo")

	projPath := ProjectConfigPath("github.com-owner-repo")
	assert.Contains(t, projPath, "projects/github.com-owner-repo/config.yaml")
}
