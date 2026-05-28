# version & self-update Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `wt version` and `wt self-update` commands with ldflags-based version injection.

**Architecture:** Two new cobra command files (`version.go`, `selfupdate.go`) following the existing `init()` pattern. Three `var` strings in `main.go` injected via ldflags at build time. Self-update downloads from GitHub Releases API and replaces the running binary.

**Tech Stack:** Go 1.26, cobra, stdlib only (net/http, encoding/json, os, runtime)

---

### Task 1: Add ldflags variables to main.go

**Files:**
- Modify: `cmd/cli/main.go`

- [ ] **Step 1: Add var block for build-time variables**

Add after the `import` block in `cmd/cli/main.go`:

```go
var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)
```

The full file becomes:

```go
package main

import (
	"fmt"
	"os"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Verify build still works**

Run: `go build ./cmd/cli/`
Expected: No errors, binary produced at `cli` (or `cli.exe` on Windows)

- [ ] **Step 3: Commit**

```bash
git add cmd/cli/main.go
git commit -m "feat: add ldflags variable declarations for version injection"
```

---

### Task 2: Create version command

**Files:**
- Create: `cmd/cli/version.go`

- [ ] **Step 1: Create cmd/cli/version.go**

```go
package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of wt",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("wt v%s (%s/%s) commit %s built at %s\n",
			version, runtime.GOOS, runtime.GOARCH, commit, buildTime)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
```

- [ ] **Step 2: Build and test**

Run: `go build ./cmd/cli/ && ./cli version`
Expected output: `wt vdev (linux/amd64) commit unknown built at unknown`
(OS/arch will match your machine)

- [ ] **Step 3: Test with ldflags**

Run: `go build -ldflags "-X main.version=1.2.3 -X main.commit=abc1234 -X main.buildTime=2026-01-15T12:00:00Z" -o cli ./cmd/cli/ && ./cli version`
Expected output: `wt v1.2.3 (linux/amd64) commit abc1234 built at 2026-01-15T12:00:00Z`

- [ ] **Step 4: Commit**

```bash
git add cmd/cli/version.go
git commit -m "feat: add wt version command"
```

---

### Task 3: Create self-update command

**Files:**
- Create: `cmd/cli/selfupdate.go`

- [ ] **Step 1: Create cmd/cli/selfupdate.go**

```go
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
```

- [ ] **Step 2: Build**

Run: `go build ./cmd/cli/`
Expected: No errors

- [ ] **Step 3: Test --check mode (should detect update)**

Run: `go build -ldflags "-X main.version=0.0.1" -o cli ./cmd/cli/ && ./cli self-update --check; echo "Exit: $?"`
Expected: Prints current vs latest version, exits with code 1 (update available)

- [ ] **Step 4: Test --check mode with current version as "latest"**

Run: `./cli self-update --yes 2>&1 || true`
Expected: Either "Already up to date." or downloads (will likely fail on permission if not installed system-wide, or succeed if running local binary)

- [ ] **Step 5: Test invalid version**

Run: `./cli self-update v99999.0.0`
Expected: `version v99999.0.0 not found`

- [ ] **Step 6: Commit**

```bash
git add cmd/cli/selfupdate.go
git commit -m "feat: add wt self-update command"
```

---

### Task 4: Update release workflow

**Files:**
- Modify: `.github/workflows/release.yml`

- [ ] **Step 1: Add version extraction step before "Build binaries"**

Add this new step between `actions/setup-go` and `Build binaries`:

```yaml
      - name: Extract version
        run: |
          echo "VERSION=${GITHUB_REF_NAME#v}" >> $GITHUB_ENV
          echo "COMMIT=$(git rev-parse --short HEAD)" >> $GITHUB_ENV
```

- [ ] **Step 2: Update "Build binaries" step with ldflags**

Replace the `go build` line inside the loop. The updated step:

```yaml
      - name: Build binaries
        run: |
          mkdir -p dist
          BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
          targets="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64"
          for t in $targets; do
            GOOS=${t%/*}
            GOARCH=${t#*/}
            ext=""
            [ "$GOOS" = "windows" ] && ext=".exe"
            echo "Building wt-${GOOS}-${GOARCH}${ext}"
            GOOS=$GOOS GOARCH=$GOARCH go build \
              -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME}" \
              -o "dist/wt-${GOOS}-${GOARCH}${ext}" ./cmd/cli/
          done
```

- [ ] **Step 3: Verify YAML syntax**

Run: `yamllint .github/workflows/release.yml` (if yamllint available) or visually confirm indentation matches existing structure.

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci: inject version info via ldflags in release workflow"
```

---

### Task 5: Final build and verification

- [ ] **Step 1: Full build with all ldflags**

```bash
go build -ldflags "-X main.version=1.0.0 -X main.commit=test123 -X main.buildTime=2026-05-28T12:00:00Z" -o cli ./cmd/cli/
```

- [ ] **Step 2: Run version**

Run: `./cli version`
Expected output: `wt v1.0.0 (linux/amd64) commit test123 built at 2026-05-28T12:00:00Z`

- [ ] **Step 3: Verify self-update subcommand is registered**

Run: `./cli --help`
Expected: `self-update` and `version` appear in the commands list

- [ ] **Step 4: Run self-update --help**

Run: `./cli self-update --help`
Expected: Shows usage, `--yes`/`-y`, and `--check` flags

- [ ] **Step 5: Commit (if any leftover changes)**

```bash
git status
```
