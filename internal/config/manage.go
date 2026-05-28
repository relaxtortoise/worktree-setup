package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// PrintValue prints a single config key's value.
func PrintValue(cfg *Config, key string) {
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

// SetValue sets a config key to the given value.
func SetValue(cfg *Config, key, value string) {
	switch key {
	case "main_worktree":
		cfg.MainWorktree = value
	case "path_strategy":
		cfg.PathStrategy = &PathStrategy{Name: value}
	}
}

// PrintFile prints the contents of a config file.
func PrintFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("(no config)")
		return
	}
	fmt.Print(string(data))
}

// WriteFile writes a Config to a YAML file.
func WriteFile(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
