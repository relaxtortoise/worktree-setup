package actions

import (
	"fmt"
	"os"
	"os/exec"
)

func ExecuteRun(commands []string, workDir string, dryRun bool) error {
	for _, cmd := range commands {
		if dryRun {
			continue
		}
		c := exec.Command("sh", "-c", cmd)
		c.Dir = workDir
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Env = os.Environ()
		if err := c.Run(); err != nil {
			return fmt.Errorf("command failed: %w", err)
		}
	}
	return nil
}
