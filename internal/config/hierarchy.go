package config

import (
	"os"
	"path/filepath"
)

const (
	DefaultConfigDir  = ".config/worktree-setup"
	DefaultConfigFile = "config.yaml"
)

func UserConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, DefaultConfigDir)
}

func GlobalConfigPath() string {
	return filepath.Join(UserConfigDir(), DefaultConfigFile)
}

func ProjectConfigDir(projectName string) string {
	return filepath.Join(UserConfigDir(), "projects", projectName)
}

func ProjectConfigPath(projectName string) string {
	return filepath.Join(ProjectConfigDir(projectName), DefaultConfigFile)
}

// Merge 按优先级合并配置（后面参数优先）
func Merge(configs ...*Config) *Config {
	result := &Config{}
	for _, c := range configs {
		if c == nil {
			continue
		}
		if c.MainWorktree != "" {
			result.MainWorktree = c.MainWorktree
		}
		if c.PathStrategy != nil {
			result.PathStrategy = c.PathStrategy
		}
		if c.On != nil {
			if result.On == nil {
				result.On = &Events{}
			}
			if c.On.PreCreate != nil {
				result.On.PreCreate = c.On.PreCreate
			}
			if c.On.PostCreate != nil {
				result.On.PostCreate = c.On.PostCreate
			}
			if c.On.PostCheckout != nil {
				result.On.PostCheckout = c.On.PostCheckout
			}
			if c.On.PreDelete != nil {
				result.On.PreDelete = c.On.PreDelete
			}
			if c.On.PostDelete != nil {
				result.On.PostDelete = c.On.PostDelete
			}
		}
	}
	return result
}

// LoadHierarchy 加载并合并三层配置
func LoadHierarchy(repoDir, userConfigDir, projectName string) (*Config, error) {
	var globalCfg, projectCfg, repoCfg *Config

	// 全局配置
	if data, err := os.ReadFile(filepath.Join(userConfigDir, DefaultConfigFile)); err == nil {
		globalCfg, _ = Parse(data)
	}

	// 项目个人配置
	if data, err := os.ReadFile(filepath.Join(userConfigDir, "projects", projectName, DefaultConfigFile)); err == nil {
		projectCfg, _ = Parse(data)
	}

	// 仓库配置
	if data, err := os.ReadFile(filepath.Join(repoDir, ".worktree.yaml")); err == nil {
		repoCfg, _ = Parse(data)
	}

	return Merge(globalCfg, projectCfg, repoCfg), nil
}
