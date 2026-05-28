# Phase 0 + 1: CLI Refactoring + Config Tests Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor CLI business logic into internal packages, then raise internal/config coverage from 69.1% to 90%+ using table-driven tests.

**Architecture:** Phase 0 moves `urlToProjectName`/`projectName` → `internal/git`, `isNewWorktree` → `internal/engine`, config management functions → `internal/config`, and self-update logic → `internal/selfupdate`. Phase 1 adds ~16 table-driven test cases covering uncovered functions: `Parse`, `ParseFile` error paths, `PathStrategy.UnmarshalYAML`, `StepsOrLegacy`, `Merge` edges, `LoadHierarchy` error paths, and path helpers.

**Tech Stack:** Go 1.26.1, std `testing`, `github.com/stretchr/testify` (assert + require)

**Spec:** `docs/superpowers/specs/2026-05-28-testing-strategy-design.md`

---

### Task 0: Add testify dependency

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Add testify dependency**

```bash
go get github.com/stretchr/testify
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: builds without error.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add testify test dependency"
```

---

### Task 1: Move urlToProjectName and projectName to internal/git

**Files:**
- Create: `internal/git/project.go`
- Modify: `cmd/cli/root.go`
- Modify: `cmd/cli/init_cmd.go`
- Modify: `cmd/cli/add.go`
- Modify: `cmd/cli/switch.go`

- [ ] **Step 1: Write the new file internal/git/project.go**

```go
package git

import (
	"os/exec"
	"strings"
)

// URLToProjectName converts a git remote URL to a project name identifier.
func URLToProjectName(url string) string {
	url = strings.TrimSpace(url)
	url = strings.TrimPrefix(url, "git@")
	url = strings.TrimPrefix(url, "https://")
	url = strings.ReplaceAll(url, ":", "/")
	url = strings.TrimSuffix(url, ".git")
	parts := strings.Split(url, "/")
	if len(parts) >= 3 {
		last := parts[len(parts)-3:]
		return strings.Join(last, "-")
	}
	return strings.ReplaceAll(url, "/", "-")
}

// ProjectName detects the current git repo's project name from origin remote.
func ProjectName(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return URLToProjectName(strings.TrimSpace(string(out))), nil
}
```

- [ ] **Step 2: Build to verify compilation**

```bash
go build ./...
```

- [ ] **Step 3: Update cmd/cli/root.go — remove urlToProjectName/projectName, delegate to git package**

Remove functions `urlToProjectName` and `projectName`. Replace `projectName()` calls with `gitpkg.ProjectName(getRepoDir())`.

Replace the current `projectName` function body:

```go
func projectName() string {
	name, err := gitpkg.ProjectName(getRepoDir())
	if err != nil {
		return ""
	}
	return name
}
```

And remove `urlToProjectName` entirely.

Remove unused imports after the change.

- [ ] **Step 4: Build to verify compilation**

```bash
go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add internal/git/project.go cmd/cli/root.go cmd/cli/init_cmd.go cmd/cli/add.go cmd/cli/switch.go
git commit -m "refactor: move urlToProjectName and projectName to internal/git"
```

---

### Task 2: Move isNewWorktree to internal/engine

**Files:**
- Create: `internal/engine/newworktree.go`
- Modify: `cmd/cli/run.go`

- [ ] **Step 1: Write internal/engine/newworktree.go**

```go
package engine

import "strings"

// IsNewWorktree returns true if prevHead looks like a new worktree reference
// (all zeros, meaning no previous HEAD).
func IsNewWorktree(prevHead string) bool {
	if len(prevHead) >= 40 {
		return strings.Count(prevHead, "0") == 40
	}
	return prevHead == "0000000000000000000000000000000000000000"
}
```

- [ ] **Step 2: Build to verify**

```bash
go build ./...
```

- [ ] **Step 3: Update cmd/cli/run.go — remove isNewWorktree, delegate to engine**

Remove the `isNewWorktree` function from `run.go`. Replace the call site:

```go
if detectCreate {
    if len(os.Args) >= 3 && engine.IsNewWorktree(os.Args[len(os.Args)-3]) {
        currentDir, _ := os.Getwd()
        return eng.RunPostCreate(cfg, currentDir)
    }
    return eng.RunPostCheckout(cfg, ".")
}
```

Remove unused imports after the change.

- [ ] **Step 4: Build to verify**

```bash
go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add internal/engine/newworktree.go cmd/cli/run.go
git commit -m "refactor: move isNewWorktree to internal/engine"
```

---

### Task 3: Move config management functions to internal/config

**Files:**
- Create: `internal/config/manage.go`
- Modify: `cmd/cli/config_cmd.go`

- [ ] **Step 1: Write internal/config/manage.go**

```go
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
```

- [ ] **Step 2: Build to verify**

```bash
go build ./...
```

- [ ] **Step 3: Update cmd/cli/config_cmd.go — remove functions, delegate to config package**

Remove `printConfigValue`, `setConfigValue`, `printConfigFile`, `writeConfigFile` from `config_cmd.go`.

Replace calls:
- `printConfigValue(&cfg, args[1])` → `config.PrintValue(&cfg, args[1])`
- `setConfigValue(&cfg, args[1], args[2])` → `config.SetValue(&cfg, args[1], args[2])`
- `printConfigFile(cfgPath)` → `config.PrintFile(cfgPath)`
- `writeConfigFile(cfgPath, &cfg)` → `config.WriteFile(cfgPath, &cfg)`

Remove unused imports after the change.

- [ ] **Step 4: Build to verify**

```bash
go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add internal/config/manage.go cmd/cli/config_cmd.go
git commit -m "refactor: move config management functions to internal/config"
```

---

### Task 4: Move self-update logic to internal/selfupdate

**Files:**
- Create: `internal/selfupdate/selfupdate.go`
- Modify: `cmd/cli/selfupdate.go`
- Modify: `cmd/cli/main.go` (export version variables)

- [ ] **Step 1: Write internal/selfupdate/selfupdate.go**

```go
package selfupdate

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
	Version    string
	Owner      string
	Repo       string
	yesFlag    bool
	checkFlag  bool
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download binary: HTTP %d", resp.StatusCode)
	}

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
	return nil
}

func (u *Updater) getLatestRelease() (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", u.Owner, u.Repo)
	resp, err := DoHTTPGet(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("release not found for tag %s", tag)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}
```

- [ ] **Step 2: Add version vars export in cmd/cli/main.go**

Add this line above `main()`:

```go
// Version is the application version, injected via ldflags.
var Version = "dev"
```

- [ ] **Step 3: Rewrite cmd/cli/selfupdate.go — thin cobra wrapper**

Replace the entire file content with:

```go
package main

import (
	"github.com/relaxtortoise/worktree-setup/internal/selfupdate"
	"github.com/spf13/cobra"
)

var (
	selfUpdateYes   bool
	selfUpdateCheck bool
)

var selfUpdateCmd = &cobra.Command{
	Use:   "self-update [version]",
	Short: "Update wt to the latest or specified version",
	Long: `Download and replace the current wt binary from GitHub Releases.

Without arguments, updates to the latest release.
With a version argument (e.g. v1.2.0), updates to that specific version.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		updater := selfupdate.New(Version, "relaxtortoise", "worktree-setup")
		return updater.Run(args, selfUpdateYes, selfUpdateCheck)
	},
}

func init() {
	selfUpdateCmd.Flags().BoolVarP(&selfUpdateYes, "yes", "y", false, "Skip confirmation prompt")
	selfUpdateCmd.Flags().BoolVar(&selfUpdateCheck, "check", false, "Only check for updates, do not download")
	rootCmd.AddCommand(selfUpdateCmd)
}
```

- [ ] **Step 4: Update cmd/cli/main.go to export version as Version**

The `version` var in `main.go` is referenced by `selfupdate` as `Version`. Make sure to keep `version` for the `versionCmd` and add `var Version = version` or directly replace `version` with `Version`.

Actually the simplest: rename `version` to `Version` in main.go (the ldflags already inject as `main.version`, keep that and add a `var Version = version` for internal use, or just rename the var).

Rename in `main.go`:
```go
var (
	Version   = "dev"  // exported for internal/selfupdate
	commit    = "unknown"
	buildTime = "unknown"
)
```

And update `version.go` to use `Version`:

```go
fmt.Printf("wt v%s (%s/%s) commit %s built at %s\n",
    Version, runtime.GOOS, runtime.GOARCH, commit, buildTime)
```

- [ ] **Step 5: Build to verify**

```bash
go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add internal/selfupdate/selfupdate.go cmd/cli/selfupdate.go cmd/cli/main.go cmd/cli/version.go
git commit -m "refactor: move self-update logic to internal/selfupdate"
```

---

### Task 5: Test go.mod integration — verify Phase 0 didn't break anything

- [ ] **Step 1: Run all tests**

```bash
go test ./... -count=1
```

Expected: All existing tests pass, no new failures.

- [ ] **Step 2: Build the binary**

```bash
go build -o /tmp/wt ./cmd/cli/
```

Expected: Binary builds successfully.

- [ ] **Step 3: Commit (if any cleanup was needed)**

Only if tests revealed issues.

---

### Task 6: Add Parse and ParseFile error path tests to parser_test.go

**Files:**
- Create: `internal/config/config_test.go`
- Modify: `internal/config/parser_test.go`

- [ ] **Step 1: Add Parse tests to parser_test.go**

Append to `internal/config/parser_test.go`:

```go
func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    *Config
		wantErr bool
	}{
		{
			name:  "valid yaml",
			input: []byte("main_worktree: /home/me/projects/app\npath_strategy: sibling\n"),
			want: &Config{
				MainWorktree: "/home/me/projects/app",
				PathStrategy: &PathStrategy{Name: "sibling"},
			},
		},
		{
			name:    "invalid yaml",
			input:   []byte("{{{invalid"),
			wantErr: true,
		},
		{
			name:  "empty input",
			input: []byte{},
			want:  &Config{},
		},
		{
			name: "only on section",
			input: []byte(`on:
  post-create:
    run:
      - "echo hello"
`),
			want: &Config{
				On: &Events{
					PostCreate: &Event{
						Run: []string{"echo hello"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseFile_ErrorPaths(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr bool
	}{
		{
			name: "file not found",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent.yaml")
			},
			wantErr: true,
		},
		{
			name: "invalid yaml content",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, ".worktree.yaml")
				os.WriteFile(path, []byte("{{{bad"), 0644)
				return path
			},
			wantErr: true,
		},
		{
			name: "empty file is valid",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, ".worktree.yaml")
				os.WriteFile(path, []byte{}, 0644)
				return path
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			_, err := ParseFile(path)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
```

Add these imports to `parser_test.go`:
```go
import (
	"path/filepath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)
```

- [ ] **Step 2: Run tests to verify they fail (some may pass)**

```bash
go test ./internal/config/ -v -run "TestParse|TestParseFile_ErrorPaths"
```

- [ ] **Step 3: Commit**

```bash
git add internal/config/parser_test.go
git commit -m "test(config): add Parse and ParseFile error path tests"
```

---

### Task 7: Write schema_test.go — UnmarshalYAML and StepsOrLegacy tests

**Files:**
- Create: `internal/config/schema_test.go`

- [ ] **Step 1: Write schema_test.go**

```go
package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPathStrategy_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want PathStrategy
	}{
		{
			name: "sibling string",
			yaml: "sibling",
			want: PathStrategy{Name: "sibling"},
		},
		{
			name: "nested string",
			yaml: "nested",
			want: PathStrategy{Name: "nested"},
		},
		{
			name: "home string",
			yaml: "home",
			want: PathStrategy{Name: "home"},
		},
		{
			name: "custom template",
			yaml: "template: /data/worktrees/{project_name}/{branch}",
			want: PathStrategy{Template: "/data/worktrees/{project_name}/{branch}"},
		},
		{
			name: "empty template",
			yaml: "template: ''",
			want: PathStrategy{Template: ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ps PathStrategy
			err := yaml.Unmarshal([]byte(tt.yaml), &ps)
			require.NoError(t, err)
			assert.Equal(t, tt.want, ps)
		})
	}
}

func TestStep_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want Step
	}{
		{
			name: "string becomes implicit run",
			yaml: `"echo hello"`,
			want: Step{Run: "echo hello"},
		},
		{
			name: "object with run",
			yaml: `run: "echo hello"`,
			want: Step{Run: "echo hello"},
		},
		{
			name: "object with copy",
			yaml: `copy:
  "a.txt": "b.txt"`,
			want: Step{Copy: &CopyItems{Items: []CopyAction{{From: "a.txt", To: "b.txt"}}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s Step
			err := yaml.Unmarshal([]byte(tt.yaml), &s)
			require.NoError(t, err)
			assert.Equal(t, tt.want, s)
		})
	}
}

func TestCopyItems_UnmarshalYAML_EmptyList(t *testing.T) {
	var ci CopyItems
	err := yaml.Unmarshal([]byte("[]"), &ci)
	require.NoError(t, err)
	assert.Nil(t, ci.Items)
}

func TestCopyItems_UnmarshalYAML_InvalidElement(t *testing.T) {
	var ci CopyItems
	err := yaml.Unmarshal([]byte("[123]"), &ci)
	require.Error(t, err)
}

func TestCopyItems_UnmarshalYAML_MapForm(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want []CopyAction
	}{
		{
			name: "simple map",
			yaml: `".env.example": ".env"`,
			want: []CopyAction{{From: ".env.example", To: ".env"}},
		},
		{
			name: "same from and to",
			yaml: `"go.mod": ""`,
			want: []CopyAction{{From: "go.mod", To: "go.mod"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ci CopyItems
			err := yaml.Unmarshal([]byte(tt.yaml), &ci)
			require.NoError(t, err)
			assert.Equal(t, tt.want, ci.Items)
		})
	}
}

func TestEvent_StepsOrLegacy(t *testing.T) {
	tests := []struct {
		name  string
		event *Event
		want  int // number of steps expected
	}{
		{
			name:  "nil event",
			event: nil,
			want:  0,
		},
		{
			name:  "empty event",
			event: &Event{},
			want:  0,
		},
		{
			name: "steps take priority",
			event: &Event{
				Steps: []Step{{Run: "step1"}},
				Run:   []string{"legacy"},
			},
			want: 1,
		},
		{
			name: "legacy copy only",
			event: &Event{
				Copy: &CopyItems{Items: []CopyAction{{From: "a", To: "b"}}},
			},
			want: 1,
		},
		{
			name: "legacy symlink only",
			event: &Event{
				Symlink: &CopyItems{Items: []CopyAction{{From: "a", To: "b"}}},
			},
			want: 1,
		},
		{
			name: "legacy run only",
			event: &Event{
				Run: []string{"cmd1", "cmd2"},
			},
			want: 1,
		},
		{
			name: "legacy all three",
			event: &Event{
				Copy:    &CopyItems{Items: []CopyAction{{From: "a", To: "b"}}},
				Symlink: &CopyItems{Items: []CopyAction{{From: "c", To: "d"}}},
				Run:     []string{"cmd1"},
			},
			want: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.event.StepsOrLegacy()
			assert.Len(t, got, tt.want)
		})
	}
}

func TestParseColonShorthand(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantFrom  string
		wantTo    string
	}{
		{
			name:     "with colon",
			input:    ".env.example:.env",
			wantFrom: ".env.example",
			wantTo:   ".env",
		},
		{
			name:     "without colon",
			input:    "go.mod",
			wantFrom: "go.mod",
			wantTo:   "go.mod",
		},
		{
			name:     "path with multiple colons",
			input:    "a:b:c",
			wantFrom: "a",
			wantTo:   "b:c",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, to := parseColonShorthand(tt.input)
			assert.Equal(t, tt.wantFrom, from)
			assert.Equal(t, tt.wantTo, to)
		})
	}
}
```

- [ ] **Step 2: Run new tests**

```bash
go test ./internal/config/ -v -run "TestPathStrategy|TestStep_|TestCopyItems|TestEvent|TestParseColon"
```

Expected: All pass.

- [ ] **Step 3: Commit**

```bash
git add internal/config/schema_test.go
git commit -m "test(config): add UnmarshalYAML, StepsOrLegacy, and parseColonShorthand tests"
```

---

### Task 8: Write hierarchy tests — Merge edges, LoadHierarchy error paths, path helpers

**Files:**
- Modify: `internal/config/hierarchy_test.go`

- [ ] **Step 1: Append tests to hierarchy_test.go**

```go
func TestMerge_AllNil(t *testing.T) {
	result := Merge(nil, nil, nil)
	require.NotNil(t, result)
	assert.Equal(t, &Config{}, result)
}

func TestMerge_PartialEvents(t *testing.T) {
	// First config has PreCreate and PostCreate, second has PostCheckout and PreDelete.
	// Result should merge non-nil event fields.
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
```

Add imports to `hierarchy_test.go`:
```go
import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)
```

- [ ] **Step 2: Run tests**

```bash
go test ./internal/config/ -v -run "TestMerge_AllNil|TestMerge_Partial|TestMerge_EventOverride|TestLoadHierarchy_No|TestLoadHierarchy_Broken|TestUserConfigPaths"
```

Expected: All pass.

- [ ] **Step 3: Commit**

```bash
git add internal/config/hierarchy_test.go
git commit -m "test(config): add Merge edge cases, LoadHierarchy error paths, and path helper tests"
```

---

### Task 9: Verify config coverage reaches 90%+

- [ ] **Step 1: Run coverage check**

```bash
go test ./internal/config/ -coverprofile=/tmp/configcov.out
go tool cover -func=/tmp/configcov.out | tail -20
```

Expected: total coverage >= 90%.

- [ ] **Step 2: If not at 90%, identify gaps**

```bash
go tool cover -func=/tmp/configcov.out | grep -E "0\.0%|[0-7][0-9]\.[0-9]%"
```

Add any missing test cases and re-run.

- [ ] **Step 3: Run all tests to verify no regressions**

```bash
go test ./... -count=1
```

Expected: All tests pass.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "test(config): verify 90%+ coverage for internal/config"
```

---

### Task 10: Final integration verification

- [ ] **Step 1: Full test suite**

```bash
go test ./... -count=1 -cover
```

Expected: All tests pass, config coverage >= 90%.

- [ ] **Step 2: Build binary**

```bash
go build -o /tmp/wt ./cmd/cli/
/tmp/wt version
```

Expected: Version output works correctly.

- [ ] **Step 3: Commit any remaining changes**

```bash
git status
```
