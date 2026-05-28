package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var (
	selfUpdateYes   bool
	selfUpdateCheck bool
)

type githubRelease struct {
	TagName string `json:"tag_name"`
}

const (
	githubOwner = "relaxtortoise"
	githubRepo  = "worktree-setup"
)

var selfUpdateCmd = &cobra.Command{
	Use:   "self-update [version]",
	Short: "Update wt to the latest or specified version",
	Long: `Download and replace the current wt binary from GitHub Releases.

Without arguments, updates to the latest release.
With a version argument (e.g. v1.2.0), updates to that specific version.`,
	RunE: runSelfUpdate,
}

func init() {
	selfUpdateCmd.Flags().BoolVarP(&selfUpdateYes, "yes", "y", false, "Skip confirmation prompt")
	selfUpdateCmd.Flags().BoolVar(&selfUpdateCheck, "check", false, "Only check for updates, do not download")
	rootCmd.AddCommand(selfUpdateCmd)
}

func runSelfUpdate(cmd *cobra.Command, args []string) error {
	// 1. Determine target version
	var targetTag string
	if len(args) > 0 {
		targetTag = args[0]
		if !strings.HasPrefix(targetTag, "v") {
			targetTag = "v" + targetTag
		}
		_, err := getReleaseByTag(targetTag)
		if err != nil {
			return fmt.Errorf("version %s not found", targetTag)
		}
	} else {
		release, err := getLatestRelease()
		if err != nil {
			return fmt.Errorf("failed to get latest release: %w", err)
		}
		targetTag = release.TagName
	}

	// 2. Compare with current version
	current := "v" + version
	if current == targetTag {
		fmt.Println("Already up to date.")
		return nil
	}

	// 3. Check mode
	if selfUpdateCheck {
		fmt.Printf("Current version: %s (%s/%s)\n", current, runtime.GOOS, runtime.GOARCH)
		fmt.Printf("Latest version:  %s (%s/%s)\n", targetTag, runtime.GOOS, runtime.GOARCH)
		os.Exit(1)
	}

	// 4. Confirmation
	if !selfUpdateYes {
		fmt.Printf("Current version: %s (%s/%s)\n", current, runtime.GOOS, runtime.GOARCH)
		fmt.Printf("New version:     %s (%s/%s)\n", targetTag, runtime.GOOS, runtime.GOARCH)
		fmt.Print("Update? [y/N]: ")
		var answer string
		fmt.Scanln(&answer)
		answer = strings.ToLower(strings.TrimSpace(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Update cancelled.")
			return nil
		}
	}

	// 5. Build download URL
	osName := runtime.GOOS
	archName := runtime.GOARCH
	ext := ""
	if osName == "windows" {
		ext = ".exe"
	}
	downloadURL := fmt.Sprintf(
		"https://github.com/%s/%s/releases/download/%s/wt-%s-%s%s",
		githubOwner, githubRepo, targetTag, osName, archName, ext,
	)

	// 6. Get current executable path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	// 7. Download binary
	fmt.Printf("Downloading %s...\n", downloadURL)
	resp, err := http.Get(downloadURL)
	if err != nil {
		return errors.New("failed to connect to GitHub, check your network")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download binary: HTTP %d", resp.StatusCode)
	}

	// 8. Write to temp file
	tmpFile, err := os.CreateTemp("", "wt-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to download binary: %w", err)
	}
	tmpFile.Close()

	// 9. Make executable and replace
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	if err := os.Rename(tmpPath, exePath); err != nil {
		return errors.New("permission denied, try running with sudo")
	}

	fmt.Printf("Updated to %s\n", targetTag)
	return nil
}

func getLatestRelease() (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", githubOwner, githubRepo)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("no releases found")
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

func getReleaseByTag(tag string) (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", githubOwner, githubRepo, tag)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("release not found for tag %s", tag)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}
