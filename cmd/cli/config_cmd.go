package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/relaxtortoise/worktree-setup/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var globalConfig bool

var configCmd = &cobra.Command{
	Use:   "config [get|set|list]",
	Short: "Manage personal config",
	RunE: func(cmd *cobra.Command, args []string) error {
		var cfgPath string
		if globalConfig {
			cfgPath = config.GlobalConfigPath()
		} else {
			projName := projectName()
			if projName == "" {
				return fmt.Errorf("not in a git repository with a remote origin")
			}
			_ = os.MkdirAll(config.ProjectConfigDir(projName), 0755)
			cfgPath = config.ProjectConfigPath(projName)
		}

		action := "list"
		if len(args) > 0 {
			action = args[0]
		}

		var cfg config.Config
		if data, err := os.ReadFile(cfgPath); err == nil {
			_ = yaml.Unmarshal(data, &cfg)
		}

		switch action {
		case "get":
			if len(args) < 2 {
				return fmt.Errorf("usage: wt config get <key>")
			}
			printConfigValue(&cfg, args[1])
		case "set":
			if len(args) < 3 {
				return fmt.Errorf("usage: wt config set <key> <value>")
			}
			setConfigValue(&cfg, args[1], args[2])
			_ = os.MkdirAll(filepath.Dir(cfgPath), 0755)
			return writeConfigFile(cfgPath, &cfg)
		case "list":
			printConfigFile(cfgPath)
		}
		return nil
	},
}

func printConfigValue(cfg *config.Config, key string) {
	switch key {
	case "main_worktree":
		fmt.Println(cfg.MainWorktree)
	case "path_strategy":
		if cfg.PathStrategy != nil {
			if cfg.PathStrategy.Template != "" {
				fmt.Printf("template: %s\n", cfg.PathStrategy.Template)
			} else {
				fmt.Println(cfg.PathStrategy.Name)
			}
		}
	}
}

func setConfigValue(cfg *config.Config, key, value string) {
	switch key {
	case "main_worktree":
		cfg.MainWorktree = value
	case "path_strategy":
		cfg.PathStrategy = &config.PathStrategy{Name: value}
	}
}

func printConfigFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("(no config)")
		return
	}
	fmt.Print(string(data))
}

func writeConfigFile(path string, cfg *config.Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func init() {
	configCmd.Flags().BoolVar(&globalConfig, "global", false, "Use global config instead of project config")
	rootCmd.AddCommand(configCmd)
}
