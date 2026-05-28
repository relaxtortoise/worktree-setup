package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/relaxtortoise/worktree-setup/internal/config"
	"github.com/spf13/cobra"
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

		cfg, err := config.ParseFile(cfgPath)
		if err != nil {
			cfg = &config.Config{}
		}

		switch action {
		case "get":
			if len(args) < 2 {
				return fmt.Errorf("usage: wt config get <key>")
			}
			config.PrintValue(cfg, args[1])
		case "set":
			if len(args) < 3 {
				return fmt.Errorf("usage: wt config set <key> <value>")
			}
			config.SetValue(cfg, args[1], args[2])
			_ = os.MkdirAll(filepath.Dir(cfgPath), 0755)
			return config.WriteFile(cfgPath, cfg)
		case "list":
			config.PrintFile(cfgPath)
		}
		return nil
	},
}

func init() {
	configCmd.Flags().BoolVar(&globalConfig, "global", false, "Use global config instead of project config")
	rootCmd.AddCommand(configCmd)
}
