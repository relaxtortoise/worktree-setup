package selfupdate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var DoHTTPGet = func(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "wt-self-update/1.0")
	return http.DefaultClient.Do(req)
}

type Release struct {
	TagName string `json:"tag_name"`
}

type Updater struct {
	Version   string
	Owner     string
	Repo      string
	yesFlag   bool
	checkFlag bool
}

func New(version, owner, repo string) *Updater {
	return &Updater{Version: version, Owner: owner, Repo: repo}
}

func (u *Updater) Run(args []string, yes, check bool) error {
	u.yesFlag = yes
	u.checkFlag = check

	var targetTag string
	if len(args) > 0 {
		targetTag = args[0]
		if !strings.HasPrefix(targetTag, "v") {
			targetTag = "v" + targetTag
		}
		_, err := u.getReleaseByTag(targetTag)
		if err != nil {
			return fmt.Errorf("version %s not found", targetTag)
		}
	} else {
		release, err := u.getLatestRelease()
		if err != nil {
			return fmt.Errorf("failed to get latest release: %w", err)
		}
		targetTag = release.TagName
	}

	current := "v" + u.Version
	if current == targetTag {
		fmt.Println("Already up to date.")
		return nil
	}

	if u.checkFlag {
		fmt.Printf("Current version: %s (%s/%s)\n", current, runtime.GOOS, runtime.GOARCH)
		fmt.Printf("Latest version:  %s (%s/%s)\n", targetTag, runtime.GOOS, runtime.GOARCH)
		fmt.Println("An update is available.")
		os.Exit(1)
	}

	if !u.yesFlag {
		fmt.Printf("Current version: %s (%s/%s)\n", current, runtime.GOOS, runtime.GOARCH)
		fmt.Printf("New version:     %s (%s/%s)\n", targetTag, runtime.GOOS, runtime.GOARCH)
		fmt.Print("Update? [y/N]: ")
		var answer string
		_, _ = fmt.Scanln(&answer)
		answer = strings.ToLower(strings.TrimSpace(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Update cancelled.")
			return nil
		}
	}

	osName := runtime.GOOS
	archName := runtime.GOARCH
	ext := ""
	if osName == "windows" {
		ext = ".exe"
	}
	downloadURL := fmt.Sprintf(
		"https://github.com/%s/%s/releases/download/%s/wt-%s-%s%s",
		u.Owner, u.Repo, targetTag, osName, archName, ext,
	)

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	fmt.Printf("Downloading %s...\n", downloadURL)
	resp, err := DoHTTPGet(downloadURL)
	if err != nil {
		return errors.New("failed to connect to GitHub, check your network")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download binary: HTTP %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "wt-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to download binary: %w", err)
	}
	_ = tmpFile.Close()

	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	if err := os.Rename(tmpPath, exePath); err != nil {
		data, copyErr := os.ReadFile(tmpPath)
		if copyErr != nil {
			return fmt.Errorf("failed to read downloaded binary: %w", copyErr)
		}
		if copyErr := os.WriteFile(exePath, data, 0755); copyErr != nil {
			return fmt.Errorf("failed to replace binary: %w (try running with sudo)", copyErr)
		}
	}

	fmt.Printf("Updated to %s\n", targetTag)
	slog.Info("update applied", "from_version", current, "to_version", targetTag)
	return nil
}

func (u *Updater) getLatestRelease() (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", u.Owner, u.Repo)
	resp, err := DoHTTPGet(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("no releases found")
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

func (u *Updater) getReleaseByTag(tag string) (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", u.Owner, u.Repo, tag)
	resp, err := DoHTTPGet(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("release not found for tag %s", tag)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}
