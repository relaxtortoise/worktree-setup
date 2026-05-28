# worktree-setup 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建 `wt` CLI 工具，增强 git worktree 的自动化配置能力（创建时复制文件/软链接/执行命令）。

**Architecture:** 采用 cobra CLI + bubbletea TUI + shell-out git 操作。配置三层合并（全局 < 项目个人 < 仓库 `.worktree.yaml`）。事件引擎按 pre-create/post-create/post-checkout/pre-delete/post-delete 分发执行 action（run/copy/symlink）。

**Tech Stack:** Go 1.26, cobra, bubbletea + bubbles, gopkg.in/yaml.v3, shell-out git

---

### Task 1: 项目脚手架

**Files:**
- Modify: `go.mod`
- Create: `cmd/cli/main.go`

- [ ] **Step 1: 初始化 go.mod 依赖**

```bash
cd /home/lairui/mytools/worktree-setup
go get github.com/spf13/cobra@latest
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/bubbles@latest
go get gopkg.in/yaml.v3@latest
```

- [ ] **Step 2: 创建最小入口**

`cmd/cli/main.go`:
```go
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

`cmd/cli/root.go`:
```go
package main

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "wt",
	Short: "Enhanced git worktree management",
	Long:  "wt enhances git worktree with automated setup via .worktree.yaml config files.",
}
```

- [ ] **Step 3: 验证能编译并显示帮助**

```bash
go build ./cmd/cli/ && ./cli --help
```

期望输出：显示 `wt` 帮助信息。

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum cmd/cli/
git commit -m "feat: scaffold project with cobra root command"
```

---

### Task 2: 配置 Schema 定义

**Files:**
- Create: `internal/config/schema.go`

- [ ] **Step 1: 定义完整的配置类型**

`internal/config/schema.go`:
```go
package config

// Config 表示完整配置（任意一层）
type Config struct {
	MainWorktree string       `yaml:"main_worktree,omitempty"`
	PathStrategy *PathStrategy `yaml:"path_strategy,omitempty"`
	On           *Events       `yaml:"on,omitempty"`
}

// PathStrategy 可以是字符串或模板对象
type PathStrategy struct {
	Name     string `yaml:"-"`
	Template string `yaml:"-"`
}

func (p *PathStrategy) UnmarshalYAML(unmarshal func(any) error) error {
	var name string
	if err := unmarshal(&name); err == nil {
		p.Name = name
		return nil
	}
	var obj struct {
		Template string `yaml:"template"`
	}
	if err := unmarshal(&obj); err != nil {
		return err
	}
	p.Template = obj.Template
	return nil
}

type Events struct {
	PreCreate    *Event `yaml:"pre-create,omitempty"`
	PostCreate   *Event `yaml:"post-create,omitempty"`
	PostCheckout *Event `yaml:"post-checkout,omitempty"`
	PreDelete    *Event `yaml:"pre-delete,omitempty"`
	PostDelete   *Event `yaml:"post-delete,omitempty"`
}

type Event struct {
	Run    []string    `yaml:"run,omitempty"`
	Copy   *CopyItems  `yaml:"copy,omitempty"`
	Symlink *CopyItems `yaml:"symlink,omitempty"`
}

// CopyAction 单个复制/软链接条目
type CopyAction struct {
	From string
	To   string
}

// CopyItems 支持 map 或 list 两种 YAML 形式，解析时统一为 []CopyAction
type CopyItems struct {
	Items []CopyAction
}

func (c *CopyItems) UnmarshalYAML(unmarshal func(any) error) error {
	// 尝试 map 形式
	var m map[string]string
	if err := unmarshal(&m); err == nil {
		for from, to := range m {
			if to == "" {
				to = from
			}
			c.Items = append(c.Items, CopyAction{From: from, To: to})
		}
		return nil
	}

	// 尝试 list 形式
	var raw []any
	if err := unmarshal(&raw); err != nil {
		return err
	}
	for _, item := range raw {
		switch v := item.(type) {
		case string:
			from, to := parseColonShorthand(v)
			c.Items = append(c.Items, CopyAction{From: from, To: to})
		case map[string]any:
			from, _ := v["from"].(string)
			to, _ := v["to"].(string)
			c.Items = append(c.Items, CopyAction{From: from, To: to})
		}
	}
	return nil
}

func parseColonShorthand(s string) (string, string) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return s, s
}
```

需要添加 `import "strings"`。

- [ ] **Step 2: 验证编译**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/config/schema.go
git commit -m "feat: add config schema types"
```

---

### Task 3: Config 解析器

**Files:**
- Create: `internal/config/parser.go`
- Create: `internal/config/parser_test.go`

- [ ] **Step 1: 编写解析测试**

`internal/config/parser_test.go`:
```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseWorktreeYAML_MapForm(t *testing.T) {
	yaml := `
main_worktree: /home/me/projects/myapp
path_strategy: sibling
on:
  post-create:
    run:
      - "go mod download"
    copy:
      ".env.example": ".env"
    symlink:
      "../main/node_modules": "node_modules"
`
	dir := t.TempDir()
	path := filepath.Join(dir, ".worktree.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MainWorktree != "/home/me/projects/myapp" {
		t.Errorf("MainWorktree = %q", cfg.MainWorktree)
	}
	if cfg.On.PostCreate == nil {
		t.Fatal("post-create is nil")
	}
	if len(cfg.On.PostCreate.Run) != 1 {
		t.Errorf("run count = %d", len(cfg.On.PostCreate.Run))
	}
	if len(cfg.On.PostCreate.Copy.Items) != 1 {
		t.Errorf("copy items = %d", len(cfg.On.PostCreate.Copy.Items))
	}
	if len(cfg.On.PostCreate.Symlink.Items) != 1 {
		t.Errorf("symlink items = %d", len(cfg.On.PostCreate.Symlink.Items))
	}
}

func TestParseWorktreeYAML_ListForm(t *testing.T) {
	yaml := `
on:
  post-create:
    copy:
      - "go.mod"
      - ".env.example:.env"
      - from: "scripts/hooks.sh"
        to: ".git/hooks/pre-commit"
`
	dir := t.TempDir()
	path := filepath.Join(dir, ".worktree.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	items := cfg.On.PostCreate.Copy.Items
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[0].From != "go.mod" || items[0].To != "go.mod" {
		t.Errorf("string item: %+v", items[0])
	}
	if items[1].From != ".env.example" || items[1].To != ".env" {
		t.Errorf("colon item: %+v", items[1])
	}
	if items[2].From != "scripts/hooks.sh" || items[2].To != ".git/hooks/pre-commit" {
		t.Errorf("object item: %+v", items[2])
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
go test ./internal/config/ -v
```
期望：`FAIL: undefined: ParseFile`

- [ ] **Step 3: 实现解析器**

`internal/config/parser.go`:
```go
package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

func ParseFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Parse(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test ./internal/config/ -v
```
期望：PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: add config YAML parser"
```

---

### Task 4: 配置层级合并

**Files:**
- Create: `internal/config/hierarchy.go`
- Create: `internal/config/hierarchy_test.go`

- [ ] **Step 1: 编写合并测试**

`internal/config/hierarchy_test.go`:
```go
package config

import (
	"os"
	"path/filepath"
	"testing"
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
	os.MkdirAll(filepath.Join(repoDir, ".git"), 0755)
	repoYAML := `
path_strategy: nested
on:
  post-create:
    run:
      - "echo repo"
`
	os.WriteFile(filepath.Join(repoDir, ".worktree.yaml"), []byte(repoYAML), 0644)

	// 项目个人配置
	cfgDir := t.TempDir()
	projDir := filepath.Join(cfgDir, "projects", "github.com-owner-repo")
	os.MkdirAll(projDir, 0755)
	projYAML := `main_worktree: /home/me/projects/myapp`
	os.WriteFile(filepath.Join(projDir, "config.yaml"), []byte(projYAML), 0644)

	// 全局配置
	globalYAML := `path_strategy: sibling`
	os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte(globalYAML), 0644)

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
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
go test ./internal/config/ -v -run TestMerge
```
期望：FAIL

- [ ] **Step 3: 实现合并逻辑**

`internal/config/hierarchy.go`:
```go
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
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test ./internal/config/ -v
```
期望：全部 PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/hierarchy.go internal/config/hierarchy_test.go
git commit -m "feat: add config hierarchy merging"
```

---

### Task 5: Git 操作封装

**Files:**
- Create: `internal/git/worktree.go`
- Create: `internal/git/branch.go`

- [ ] **Step 1: 实现 worktree 操作**

`internal/git/worktree.go`:
```go
package git

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Worktree struct {
	Path   string
	Head   string
	Branch string
	Bare   bool
}

func Run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), string(ee.Stderr))
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// RunInternal 执行 git 命令并设置 WT_INTERNAL 标记
func RunInternal(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Env = append(os.Environ(), "WT_INTERNAL=1")
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), string(ee.Stderr))
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func AddWorktree(path, branch, baseBranch string) error {
	args := []string{"worktree", "add", path}
	if branch != "" {
		args = append(args, branch)
	}
	if baseBranch != "" {
		args = append(args, baseBranch)
	}
	_, err := RunInternal(args...)
	return err
}

func RemoveWorktree(path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)
	_, err := RunInternal(args...)
	return err
}

func ListWorktrees() ([]Worktree, error) {
	out, err := Run("worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return parsePorcelain(out), nil
}

func parsePorcelain(out string) []Worktree {
	var wts []Worktree
	var cur *Worktree
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			if cur != nil {
				wts = append(wts, *cur)
				cur = nil
			}
			continue
		}
		if strings.HasPrefix(line, "worktree ") {
			cur = &Worktree{Path: strings.TrimPrefix(line, "worktree ")}
		} else if strings.HasPrefix(line, "HEAD ") {
			if cur != nil {
				cur.Head = strings.TrimPrefix(line, "HEAD ")
			}
		} else if strings.HasPrefix(line, "branch ") {
			if cur != nil {
				cur.Branch = strings.TrimPrefix(line, "branch ")
			}
		} else if strings.HasPrefix(line, "bare") {
			if cur != nil {
				cur.Bare = true
			}
		}
	}
	if cur != nil {
		wts = append(wts, *cur)
	}
	return wts
}

// FindMainWorktree 自动检测 main worktree
func FindMainWorktree() (string, error) {
	wts, err := ListWorktrees()
	if err != nil {
		return "", err
	}
	// 优先找 main/master 分支
	for _, wt := range wts {
		b := strings.TrimPrefix(wt.Branch, "refs/heads/")
		if b == "main" || b == "master" {
			return wt.Path, nil
		}
	}
	// 退而求其次找第一个非 bare
	for _, wt := range wts {
		if !wt.Bare {
			return wt.Path, nil
		}
	}
	return "", fmt.Errorf("no main worktree found")
}

func CurrentWorktreePath() (string, error) {
	out, err := Run("rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", err
	}
	// git-common-dir 指向 .git 或 main 的 .git
	// 需要去掉 /.git 后缀
	dir := strings.TrimSuffix(out, "/.git")
	dir = strings.TrimSuffix(dir, "\\.git")
	return dir, nil
}
```

`internal/git/branch.go`:
```go
package git

import (
	"sort"
	"strings"
	"time"
)

type Branch struct {
	Name      string
	LastCommit time.Time
	Author    string
	CheckedOut bool
}

func FetchOrigin() error {
	_, err := Run("fetch", "origin", "--prune")
	return err
}

func ListRemoteBranches() ([]Branch, error) {
	out, err := Run("for-each-ref", "--format=%(refname:short)%00%(committerdate:iso8601)%00%(authorname)",
		"refs/remotes/origin/", "--sort=-committerdate")
	if err != nil {
		return nil, err
	}

	// 获取已 checkout 的分支
	checkedOut := checkedOutBranches()

	var branches []Branch
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\x00", 3)
		if len(parts) < 3 {
			continue
		}
		name := strings.TrimPrefix(parts[0], "origin/")
		if name == "HEAD" {
			continue
		}
		t, _ := time.Parse("2006-01-02 15:04:05 -0700", parts[1])
		branches = append(branches, Branch{
			Name:       name,
			LastCommit: t,
			Author:     parts[2],
			CheckedOut: checkedOut[name],
		})
	}
	sort.Slice(branches, func(i, j int) bool {
		return branches[i].LastCommit.After(branches[j].LastCommit)
	})
	return branches, nil
}

func checkedOutBranches() map[string]bool {
	out, err := Run("worktree", "list")
	if err != nil {
		return nil
	}
	result := make(map[string]bool)
	for _, line := range strings.Split(out, "\n") {
		// 每行格式：/path   <hash> [branch_name]
		start := strings.LastIndex(line, "[")
		end := strings.LastIndex(line, "]")
		if start >= 0 && end > start {
			result[line[start+1:end]] = true
		}
	}
	return result
}
```

- [ ] **Step 2: 验证编译**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/git/
git commit -m "feat: add git operation wrappers"
```

---

### Task 6: Action 执行器

**Files:**
- Create: `internal/actions/run.go`
- Create: `internal/actions/copy.go`
- Create: `internal/actions/symlink.go`
- Create: `internal/actions/runner.go`
- Create: `internal/actions/runner_test.go`

- [ ] **Step 1: 编写测试**

`internal/actions/runner_test.go`:
```go
package actions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/relaxtortoise/worktree-setup/internal/config"
)

func TestRunCommand(t *testing.T) {
	dir := t.TempDir()
	err := ExecuteRun([]string{"echo hello > test.txt"}, dir, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCopyFiles(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("hello"), 0644)

	_, err := ExecuteCopy(dstDir, srcDir, []config.CopyAction{
		{From: "a.txt", To: "a.txt"},
	})
	if err != nil {
		t.Fatalf("copy error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dstDir, "a.txt"))
	if err != nil {
		t.Fatal("file not created")
	}
	if string(data) != "hello" {
		t.Errorf("content = %q", string(data))
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
go test ./internal/actions/ -v
```

- [ ] **Step 3: 实现 action 执行器**

`internal/actions/runner.go`:
```go
package actions

import (
	"fmt"
	"os"

	"github.com/relaxtortoise/worktree-setup/internal/config"
)

type Runner struct {
	MainWorktree string
}

func NewRunner(mainWorktree string) *Runner {
	return &Runner{MainWorktree: mainWorktree}
}

func (r *Runner) ExecuteEvent(event *config.Event, worktreeDir string) error {
	if event == nil {
		return nil
	}

	for _, cmd := range event.Run {
		if err := ExecuteRun([]string{cmd}, worktreeDir, false); err != nil {
			return fmt.Errorf("run %q: %w", cmd, err)
		}
	}

	if event.Copy != nil {
		if _, err := ExecuteCopy(worktreeDir, r.MainWorktree, event.Copy.Items); err != nil {
			return fmt.Errorf("copy: %w", err)
		}
	}

	if event.Symlink != nil {
		if _, err := ExecuteSymlink(worktreeDir, r.MainWorktree, event.Symlink.Items); err != nil {
			return fmt.Errorf("symlink: %w", err)
		}
	}

	return nil
}

func (r *Runner) ExecutePreCreate(event *config.Event) error {
	if event == nil {
		return nil
	}
	for _, cmd := range event.Run {
		if err := ExecuteRun([]string{cmd}, r.MainWorktree, false); err != nil {
			return fmt.Errorf("pre-create run %q: %w", cmd, err)
		}
	}
	return nil
}
```

`internal/actions/run.go`:
```go
package actions

import (
	"os"
	"os/exec"
	"strings"
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
```

Need to add `"fmt"` import.

`internal/actions/copy.go`:
```go
package actions

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/relaxtortoise/worktree-setup/internal/config"
)

func ExecuteCopy(dstDir, srcDir string, items []config.CopyAction) ([]string, error) {
	var copied []string
	for _, item := range items {
		src := filepath.Join(srcDir, item.From)
		dst := filepath.Join(dstDir, item.To)

		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return copied, err
		}

		srcInfo, err := os.Stat(src)
		if err != nil {
			return copied, fmt.Errorf("stat %s: %w", src, err)
		}

		if srcInfo.IsDir() {
			if err := copyDir(src, dst); err != nil {
				return copied, err
			}
		} else {
			if err := copyFile(src, dst); err != nil {
				return copied, err
			}
		}
		copied = append(copied, item.To)
	}
	return copied, nil
}

func copyFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer d.Close()

	_, err = io.Copy(d, s)
	return err
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		dest := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(dest, info.Mode())
		}
		return copyFile(path, dest)
	})
}
```

`internal/actions/symlink.go`:
```go
package actions

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/relaxtortoise/worktree-setup/internal/config"
)

func ExecuteSymlink(dstDir, srcDir string, items []config.CopyAction) ([]string, error) {
	var linked []string
	for _, item := range items {
		src := filepath.Join(srcDir, item.From)
		dst := filepath.Join(dstDir, item.To)

		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return linked, err
		}

		err := os.Symlink(src, dst)
		if err != nil && runtime.GOOS == "windows" {
			fmt.Fprintf(os.Stderr, "Symlink failed for %s: %v\n", item.To, err)
			fmt.Fprintf(os.Stderr, "Downgrade to copy? This requires developer mode or admin privileges for symlink. [y/N]: ")
			var answer string
			fmt.Scanln(&answer)
			if strings.ToLower(strings.TrimSpace(answer)) == "y" {
				if err := copyDir(src, dst); err != nil {
					return linked, fmt.Errorf("fallback copy failed: %w", err)
				}
				fmt.Fprintf(os.Stderr, "Copied instead: %s\n", item.To)
			}
		} else if err != nil {
			return linked, fmt.Errorf("symlink %s: %w", item.To, err)
		}
		linked = append(linked, item.To)
	}
	return linked, nil
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test ./internal/actions/ -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/actions/
git commit -m "feat: add action executors (run, copy, symlink)"
```

---

### Task 7: 事件引擎

**Files:**
- Create: `internal/engine/engine.go`

- [ ] **Step 1: 实现事件引擎**

`internal/engine/engine.go`:
```go
package engine

import (
	"github.com/relaxtortoise/worktree-setup/internal/actions"
	"github.com/relaxtortoise/worktree-setup/internal/config"
)

type Engine struct {
	Runner *actions.Runner
}

func New(mainWorktree string) *Engine {
	return &Engine{Runner: actions.NewRunner(mainWorktree)}
}

func (e *Engine) RunPreCreate(cfg *config.Config) error {
	if cfg.On == nil {
		return nil
	}
	return e.Runner.ExecutePreCreate(cfg.On.PreCreate)
}

func (e *Engine) RunPostCreate(cfg *config.Config, worktreeDir string) error {
	if cfg.On == nil {
		return nil
	}
	return e.Runner.ExecuteEvent(cfg.On.PostCreate, worktreeDir)
}

func (e *Engine) RunPostCheckout(cfg *config.Config, worktreeDir string) error {
	if cfg.On == nil {
		return nil
	}
	return e.Runner.ExecuteEvent(cfg.On.PostCheckout, worktreeDir)
}

func (e *Engine) RunPreDelete(cfg *config.Config, worktreeDir string) error {
	if cfg.On == nil {
		return nil
	}
	return e.Runner.ExecuteEvent(cfg.On.PreDelete, worktreeDir)
}

func (e *Engine) RunPostDelete(cfg *config.Config) error {
	if cfg.On == nil {
		return nil
	}
	return e.Runner.ExecuteEvent(cfg.On.PostDelete, e.Runner.MainWorktree)
}
```

- [ ] **Step 2: 验证编译**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/engine/
git commit -m "feat: add event engine"
```

---

### Task 8: 路径策略计算

**Files:**
- Create: `internal/worktree/path.go`
- Create: `internal/worktree/path_test.go`

- [ ] **Step 1: 编写测试**

`internal/worktree/path_test.go`:
```go
package worktree

import (
	"testing"

	"github.com/relaxtortoise/worktree-setup/internal/config"
)

func TestComputePath_Sibling(t *testing.T) {
	ps := &config.PathStrategy{Name: "sibling"}
	path := ComputePath("/home/me/projects/myapp", "feature-x", "github.com-owner-myapp", ps)
	expected := "/home/me/projects/myapp@feature-x"
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestComputePath_CustomTemplate(t *testing.T) {
	ps := &config.PathStrategy{Template: "/data/worktrees/{project_name}/{branch}"}
	path := ComputePath("/home/me/projects/myapp", "feature-x", "myapp", ps)
	expected := "/data/worktrees/myapp/feature-x"
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestComputePath_DefaultSibling(t *testing.T) {
	path := ComputePath("/home/me/projects/myapp", "feature-x", "proj", nil)
	expected := "/home/me/projects/myapp@feature-x"
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
go test ./internal/worktree/ -v
```

- [ ] **Step 3: 实现路径策略**

`internal/worktree/path.go`:
```go
package worktree

import (
	"path/filepath"
	"strings"

	"github.com/relaxtortoise/worktree-setup/internal/config"
)

func ComputePath(mainWorktree, branch, projectName string, ps *config.PathStrategy) string {
	template := "{main_parent}/{repo_name}@{branch}" // 默认 sibling
	if ps != nil {
		if ps.Template != "" {
			template = ps.Template
		} else {
			switch ps.Name {
			case "nested":
				template = "{main}/.worktrees/{branch}"
			case "home":
				template = "~/worktrees/{project_name}/{branch}"
			case "sibling", "":
				template = "{main_parent}/{repo_name}@{branch}"
			}
		}
	}

	result := template
	result = strings.ReplaceAll(result, "{main}", mainWorktree)
	result = strings.ReplaceAll(result, "{main_parent}", filepath.Dir(mainWorktree))
	result = strings.ReplaceAll(result, "{repo_name}", filepath.Base(mainWorktree))
	result = strings.ReplaceAll(result, "{project_name}", projectName)
	result = strings.ReplaceAll(result, "{branch}", sanitizeBranch(branch))
	if strings.HasPrefix(result, "~/") {
		home, _ := getHomeDir()
		result = filepath.Join(home, result[2:])
	}
	return result
}

func sanitizeBranch(branch string) string {
	return strings.ReplaceAll(branch, "/", "-")
}

var getHomeDir = func() (string, error) {
	return filepath.Join("/home", "placeholder"), nil // 测试中覆盖
}
```

需要在文件顶部添加 `os/user` 导入改用 `os.UserHomeDir`：

实际上，直接用 `os.UserHomeDir()` 即可。修正：

```go
package worktree

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/relaxtortoise/worktree-setup/internal/config"
)

func ComputePath(mainWorktree, branch, projectName string, ps *config.PathStrategy) string {
	template := "{main_parent}/{repo_name}@{branch}"
	if ps != nil {
		if ps.Template != "" {
			template = ps.Template
		} else {
			switch ps.Name {
			case "nested":
				template = "{main}/.worktrees/{branch}"
			case "home":
				template = "~/worktrees/{project_name}/{branch}"
			}
		}
	}

	result := template
	result = strings.ReplaceAll(result, "{main}", mainWorktree)
	result = strings.ReplaceAll(result, "{main_parent}", filepath.Dir(mainWorktree))
	result = strings.ReplaceAll(result, "{repo_name}", filepath.Base(mainWorktree))
	result = strings.ReplaceAll(result, "{project_name}", projectName)
	result = strings.ReplaceAll(result, "{branch}", sanitizeBranch(branch))
	if strings.HasPrefix(result, "~/") {
		home, _ := os.UserHomeDir()
		result = filepath.Join(home, result[2:])
	}
	return result
}

func sanitizeBranch(branch string) string {
	return strings.ReplaceAll(branch, "/", "-")
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test ./internal/worktree/ -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/worktree/
git commit -m "feat: add path strategy computation"
```

---

### Task 9: TUI Fuzzy 选择器

**Files:**
- Create: `internal/tui/selector.go`

- [ ] **Step 1: 实现 TUI 选择器**

`internal/tui/selector.go`:
```go
package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/relaxtortoise/worktree-setup/internal/git"
)

type SelectItem struct {
	Name    string
	Detail  string
	Payload any // Branch 或 Worktree
}

type SelectType int

const (
	SelectBranch   SelectType = iota
	SelectWorktree
)

type model struct {
	textInput textinput.Model
	items     []SelectItem
	filtered  []int
	cursor    int
	selType   SelectType
	quitting  bool
}

type SelectedMsg struct {
	Item SelectItem
}

func RunBranchSelector(autoFetch bool) (string, error) {
	if autoFetch {
		git.FetchOrigin()
	}
	branches, err := git.ListRemoteBranches()
	if err != nil {
		return "", err
	}
	var items []SelectItem
	for _, b := range branches {
		detail := fmt.Sprintf("%s  %s", formatTime(b.LastCommit), b.Author)
		items = append(items, SelectItem{
			Name:    "origin/" + b.Name,
			Detail:  detail,
			Payload: b,
		})
	}
	p := tea.NewProgram(initialModel(items, SelectBranch))
	m, err := p.Run()
	if err != nil {
		return "", err
	}
	if m, ok := m.(model); ok && !m.quitting {
		if len(m.filtered) > 0 {
			return m.items[m.filtered[m.cursor]].Payload.(git.Branch).Name, nil
		}
	}
	return "", fmt.Errorf("cancelled")
}

func RunWorktreeSelector(knownProjects []struct{ Name, Path string }) (string, error) {
	var items []SelectItem
	for _, proj := range knownProjects {
		wts, err := git.ListWorktrees()
		if err != nil {
			continue
		}
		for _, wt := range wts {
			if wt.Bare {
				continue
			}
			name := proj.Name + "/" + strings.TrimPrefix(wt.Branch, "refs/heads/")
			items = append(items, SelectItem{
				Name:    name,
				Detail:  wt.Path,
				Payload: wt,
			})
		}
	}
	p := tea.NewProgram(initialModel(items, SelectWorktree))
	m, err := p.Run()
	if err != nil {
		return "", err
	}
	if m, ok := m.(model); ok && !m.quitting {
		if len(m.filtered) > 0 {
			return m.items[m.filtered[m.cursor]].Payload.(git.Worktree).Path, nil
		}
	}
	return "", fmt.Errorf("cancelled")
}

func initialModel(items []SelectItem, selType SelectType) model {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.Focus()
	ti.CharLimit = 80

	filtered := make([]int, len(items))
	for i := range items {
		filtered[i] = i
	}

	return model{
		textInput: ti,
		items:     items,
		filtered:  filtered,
		selType:   selType,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			return m, tea.Quit
		case "up", "ctrl+k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "ctrl+j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)

	// 过滤
	query := strings.ToLower(m.textInput.Value())
	m.filtered = nil
	for i, item := range m.items {
		if query == "" || strings.Contains(strings.ToLower(item.Name), query) {
			m.filtered = append(m.filtered, i)
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}

	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	s := m.textInput.View() + "\n\n"
	for i, idx := range m.filtered {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}
		item := m.items[idx]
		s += fmt.Sprintf("%s %s  %s\n", cursor, item.Name, item.Detail)
	}

	s += fmt.Sprintf("\n%d matches | enter select | esc quit", len(m.filtered))
	return s
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%dw ago", int(d.Hours()/24/7))
	}
}
```

- [ ] **Step 2: 验证编译**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/tui/
git commit -m "feat: add TUI fuzzy selector (bubbletea)"
```

---

### Task 10: Git Hooks 安装器（`wt hooks`）

**Files:**
- Create: `internal/hooks/installer.go`

- [ ] **Step 1: 实现 hooks 安装**

`internal/hooks/installer.go`:
```go
package hooks

import (
	"fmt"
	"os"
	"path/filepath"
)

const hookScript = `#!/bin/sh
# Installed by wt hooks

if [ -n "$WT_INTERNAL" ]; then
    exit 0
fi
wt run post-checkout "$@" --detect-create
`

func Install(repoDir string) ([]string, error) {
	hooksDir := filepath.Join(repoDir, ".git", "hooks")

	var installed []string
	hookPath := filepath.Join(hooksDir, "post-checkout")
	if err := os.WriteFile(hookPath, []byte(hookScript), 0755); err != nil {
		return installed, fmt.Errorf("write %s: %w", hookPath, err)
	}
	installed = append(installed, "post-checkout")

	return installed, nil
}

func IsInstalled(repoDir string) bool {
	hookPath := filepath.Join(repoDir, ".git", "hooks", "post-checkout")
	data, err := os.ReadFile(hookPath)
	if err != nil {
		return false
	}
	// 检查是否是 wt 安装的
	return contains(string(data), "Installed by wt")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: 验证编译**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/hooks/
git commit -m "feat: add git hooks installer"
```

---

### Task 11: Worktree 创建流程

**Files:**
- Create: `internal/worktree/create.go`

- [ ] **Step 1: 实现创建流程**

`internal/worktree/create.go`:
```go
package worktree

import (
	"fmt"
	"os"

	"github.com/relaxtortoise/worktree-setup/internal/config"
	"github.com/relaxtortoise/worktree-setup/internal/engine"
	gitpkg "github.com/relaxtortoise/worktree-setup/internal/git"
)

func Create(branch, explicitPath, projectName string, cfg *config.Config, autoFetch bool) (string, error) {
	// 确定 main worktree
	mainWT := cfg.MainWorktree
	if mainWT == "" {
		var err error
		mainWT, err = gitpkg.FindMainWorktree()
		if err != nil {
			return "", fmt.Errorf("cannot determine main worktree: %w", err)
		}
	}

	// fetch
	if autoFetch {
		gitpkg.FetchOrigin()
	}

	// 计算路径
	targetPath := explicitPath
	if targetPath == "" {
		targetPath = ComputePath(mainWT, branch, projectName, cfg.PathStrategy)
	}

	// pre-create
	eng := engine.New(mainWT)
	if err := eng.RunPreCreate(cfg); err != nil {
		return "", fmt.Errorf("pre-create: %w", err)
	}

	// git worktree add
	if err := gitpkg.AddWorktree(targetPath, branch, "origin/"+branch); err != nil {
		return "", fmt.Errorf("git worktree add: %w", err)
	}

	// 确认目录存在
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return "", fmt.Errorf("worktree directory not found after creation: %s", targetPath)
	}

	// post-create
	if err := eng.RunPostCreate(cfg, targetPath); err != nil {
		return targetPath, fmt.Errorf("post-create: %w", err)
	}

	return targetPath, nil
}
```

- [ ] **Step 2: 验证编译**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/worktree/create.go
git commit -m "feat: add worktree creation flow"
```

---

### Task 12: Worktree 删除流程

**Files:**
- Create: `internal/worktree/remove.go`

- [ ] **Step 1: 实现删除流程**

`internal/worktree/remove.go`:
```go
package worktree

import (
	"fmt"

	"github.com/relaxtortoise/worktree-setup/internal/config"
	"github.com/relaxtortoise/worktree-setup/internal/engine"
	gitpkg "github.com/relaxtortoise/worktree-setup/internal/git"
)

func Remove(path string, cfg *config.Config, force bool) error {
	mainWT := cfg.MainWorktree
	if mainWT == "" {
		var err error
		mainWT, err = gitpkg.FindMainWorktree()
		if err != nil {
			return fmt.Errorf("cannot determine main worktree: %w", err)
		}
	}

	eng := engine.New(mainWT)

	// pre-delete
	if err := eng.RunPreDelete(cfg, path); err != nil {
		return fmt.Errorf("pre-delete: %w", err)
	}

	// git worktree remove
	if err := gitpkg.RemoveWorktree(path, force); err != nil {
		return fmt.Errorf("git worktree remove: %w", err)
	}

	// post-delete
	if err := eng.RunPostDelete(cfg); err != nil {
		return fmt.Errorf("post-delete: %w", err)
	}

	return nil
}
```

- [ ] **Step 2: 验证编译**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/worktree/remove.go
git commit -m "feat: add worktree removal flow"
```

---

### Task 13: CLI 命令（cobra 子命令）

**Files:**
- Create: `cmd/cli/add.go`
- Create: `cmd/cli/remove.go`
- Create: `cmd/cli/switch.go`
- Create: `cmd/cli/list.go`
- Create: `cmd/cli/init_cmd.go`
- Create: `cmd/cli/hooks.go`
- Create: `cmd/cli/run.go`
- Create: `cmd/cli/config_cmd.go`
- Modify: `cmd/cli/root.go`

- [ ] **Step 1: 修改 root.go 添加全局 flag 和共享逻辑**

`cmd/cli/root.go`:
```go
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/relaxtortoise/worktree-setup/internal/config"
)

var (
	configDir  string
	noFetch    bool
	explicitPath string
)

func init() {
	configDir = config.UserConfigDir()
}

func getRepoDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return dir
}

func projectName() string {
	dir := getRepoDir()
	out, err := runGitCmd(dir, "remote", "get-url", "origin")
	if err != nil {
		return ""
	}
	return urlToProjectName(strings.TrimSpace(out))
}

func urlToProjectName(url string) string {
	// git@github.com:owner/repo.git -> github.com-owner-repo
	// https://github.com/owner/repo.git -> github.com-owner-repo
	url = strings.TrimSpace(url)
	url = strings.TrimPrefix(url, "git@")
	url = strings.TrimPrefix(url, "https://")
	url = strings.ReplaceAll(url, ":", "/")
	url = strings.TrimSuffix(url, ".git")
	parts := strings.Split(url, "/")
	if len(parts) >= 3 {
		// parts[len(parts)-3:] = [host, owner, repo]
		last := parts[len(parts)-3:]
		return strings.Join(last, "-")
	}
	return strings.ReplaceAll(url, "/", "-")
}

func loadConfig() (*config.Config, error) {
	repoDir := getRepoDir()
	projName := projectName()
	if projName == "" {
		return nil, fmt.Errorf("not in a git repository with a remote origin")
	}
	return config.LoadHierarchy(repoDir, configDir, projName)
}

func runGitCmd(dir string, args ...string) (string, error) {
	cmd := execCommand("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

var execCommand = func(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
```

需要 `import "os/exec"`。

- [ ] **Step 2: 实现所有子命令**

`cmd/cli/add.go`:
```go
package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/relaxtortoise/worktree-setup/internal/tui"
	"github.com/relaxtortoise/worktree-setup/internal/worktree"
)

var addCmd = &cobra.Command{
	Use:   "add [branch]",
	Short: "Create a new worktree",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		branch := ""
		if len(args) > 0 {
			branch = args[0]
		}

		if branch == "" {
			autoFetch := !noFetch
			if os.Getenv("WT_NO_FETCH") == "1" {
				autoFetch = false
			}
			branch, err = tui.RunBranchSelector(autoFetch)
			if err != nil {
				return err
			}
		}

		projName := projectName()
		autoFetch := !noFetch
		if os.Getenv("WT_NO_FETCH") == "1" {
			autoFetch = false
		}

		path, err := worktree.Create(branch, explicitPath, projName, cfg, autoFetch)
		if err != nil {
			return err
		}
		fmt.Println(path)
		return nil
	},
}

func init() {
	addCmd.Flags().BoolVar(&noFetch, "no-fetch", false, "Skip git fetch")
	addCmd.Flags().StringVar(&explicitPath, "path", "", "Explicit worktree path")
	rootCmd.AddCommand(addCmd)
}
```

需要 `import "os"`。

`cmd/cli/remove.go`:
```go
package main

import (
	"github.com/spf13/cobra"
	"github.com/relaxtortoise/worktree-setup/internal/worktree"
)

var removeForce bool

var removeCmd = &cobra.Command{
	Use:   "remove [name|path]",
	Short: "Remove a worktree",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		if len(args) == 0 {
			return fmt.Errorf("worktree name or path required")
		}
		return worktree.Remove(args[0], cfg, removeForce)
	},
}

func init() {
	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "Force removal")
	rootCmd.AddCommand(removeCmd)
}
```

需要 `import "fmt"`。

`cmd/cli/switch.go`:
```go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/relaxtortoise/worktree-setup/internal/config"
	gitpkg "github.com/relaxtortoise/worktree-setup/internal/git"
	"github.com/relaxtortoise/worktree-setup/internal/tui"
)

var switchCmd = &cobra.Command{
	Use:   "switch [name]",
	Short: "Switch to a worktree (cross-project)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			// 直接匹配
			fmt.Println(args[0])
			return nil
		}

		var projects []struct{ Name, Path string }
		// 收集已知项目
		entries, err := os.ReadDir(filepath.Join(configDir, "projects"))
		if err == nil {
			for _, e := range entries {
				if e.IsDir() {
					projects = append(projects, struct{ Name, Path string }{
						Name: e.Name(), Path: filepath.Join(configDir, "projects", e.Name()),
					})
				}
			}
		}
		// 确保当前项目在列表中
		projName := projectName()
		hasCurrent := false
		for _, p := range projects {
			if p.Name == projName {
				hasCurrent = true
				break
			}
		}
		if !hasCurrent {
			projects = append(projects, struct{ Name, Path string }{Name: projName, Path: ""})
		}

		path, err := tui.RunWorktreeSelector(projects)
		if err != nil {
			return err
		}
		fmt.Println(path)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(switchCmd)
}
```

需要 `import "path/filepath"`.

`cmd/cli/list.go`:
```go
package main

import (
	"fmt"

	"github.com/spf13/cobra"
	gitpkg "github.com/relaxtortoise/worktree-setup/internal/git"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all worktrees",
	RunE: func(cmd *cobra.Command, args []string) error {
		wts, err := gitpkg.ListWorktrees()
		if err != nil {
			return err
		}
		for _, wt := range wts {
			fmt.Printf("%s\t%s\n", wt.Head[:min(8, len(wt.Head))], wt.Path)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
```

`cmd/cli/init_cmd.go`:
```go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/relaxtortoise/worktree-setup/internal/config"
	gitpkg "github.com/relaxtortoise/worktree-setup/internal/git"
)

var noGitignore bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize .worktree.yaml and project config",
	RunE: func(cmd *cobra.Command, args []string) error {
		repoDir := getRepoDir()
		projName := projectName()
		if projName == "" {
			return fmt.Errorf("not in a git repository with a remote origin")
		}

		// 创建 .worktree.yaml 模板
		template := `# wt worktree configuration
# See: https://github.com/relaxtortoise/worktree-setup

on:
  post-create:
    run: []
  post-checkout:
    run: []
`
		wtPath := filepath.Join(repoDir, ".worktree.yaml")
		if _, err := os.Stat(wtPath); os.IsNotExist(err) {
			os.WriteFile(wtPath, []byte(template), 0644)
			fmt.Println("created .worktree.yaml")
		} else {
			fmt.Println(".worktree.yaml already exists, skipping")
		}

		// 创建项目个人配置
		projDir := config.ProjectConfigDir(projName)
		os.MkdirAll(projDir, 0755)
		cfgPath := filepath.Join(projDir, "config.yaml")

		if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
			mainWT, err := gitpkg.FindMainWorktree()
			if err != nil {
				mainWT = repoDir
			}
			content := fmt.Sprintf("main_worktree: %s\npath_strategy: sibling\n", mainWT)
			os.WriteFile(cfgPath, []byte(content), 0644)
			fmt.Printf("created %s\n", cfgPath)
		} else {
			fmt.Printf("%s already exists, skipping\n", cfgPath)
		}

		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&noGitignore, "no-gitignore", false, "Do not add .worktree.yaml to .gitignore")
	rootCmd.AddCommand(initCmd)
}
```

`cmd/cli/hooks.go`:
```go
package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/relaxtortoise/worktree-setup/internal/hooks"
)

var hooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Install git hooks for auto-detecting worktree creation",
	RunE: func(cmd *cobra.Command, args []string) error {
		repoDir := getRepoDir()
		installed, err := hooks.Install(repoDir)
		if err != nil {
			return err
		}
		for _, h := range installed {
			fmt.Printf("installed: .git/hooks/%s\n", h)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(hooksCmd)
}
```

`cmd/cli/run.go`:
```go
package main

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/relaxtortoise/worktree-setup/internal/config"
	"github.com/relaxtortoise/worktree-setup/internal/engine"
	gitpkg "github.com/relaxtortoise/worktree-setup/internal/git"
)

var detectCreate bool

var runCmd = &cobra.Command{
	Use:   "run <event>",
	Short: "Execute configured event (called by git hooks)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		mainWT := cfg.MainWorktree
		if mainWT == "" {
			mainWT, _ = gitpkg.FindMainWorktree()
		}

		eng := engine.New(mainWT)
		event := args[0]

		switch event {
		case "post-checkout":
			if detectCreate {
				// 从 git hook 参数判断是否是新 worktree
				// post-checkout 参数: <prev-head> <new-head> <is-branch-checkout>
				hookArgs := cmd.Flags().Args()
				if len(hookArgs) >= 1 && isNewWorktree(hookArgs[0]) {
					// 新 worktree → 执行 post-create
					currentDir, _ := os.Getwd()
					return eng.RunPostCreate(cfg, currentDir)
				}
				return eng.RunPostCheckout(cfg, ".")
			}
			return eng.RunPostCheckout(cfg, ".")
		default:
			return fmt.Errorf("unknown event: %s", event)
		}
	},
}

func isNewWorktree(prevHead string) bool {
	// previous HEAD 全零表示新 worktree
	if len(prevHead) >= 40 {
		return strings.Count(prevHead, "0") == 40
	}
	return prevHead == "0000000000000000000000000000000000000000"
}

func init() {
	runCmd.Flags().BoolVar(&detectCreate, "detect-create", false, "Detect new worktree from post-checkout args")
	// cobra 不支持位置参数后的额外参数，post-checkout hook 会传 3 个额外的位置参数
	// 这些参数会出现在 os.Args 中，我们在 RunE 中通过 os.Args 访问
	rootCmd.AddCommand(runCmd)
}
```

需要 `import "fmt"`。

`cmd/cli/config_cmd.go`:
```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/relaxtortoise/worktree-setup/internal/config"
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
			os.MkdirAll(config.ProjectConfigDir(projName), 0755)
			cfgPath = config.ProjectConfigPath(projName)
		}

		action := "list"
		if len(args) > 0 {
			action = args[0]
		}

		var cfg config.Config
		if data, err := os.ReadFile(cfgPath); err == nil {
			yaml.Unmarshal(data, &cfg)
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
	os.MkdirAll(filepath.Dir(path), 0755)
	return os.WriteFile(path, data, 0644)
}
```

需要 `import "path/filepath"`。

`cmd/cli/config_cmd.go` — 修正 writeConfigFile 签名缺失的 `filepath`：

```go
func writeConfigFile(path string, cfg *config.Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
```

不需要 `filepath`，在 `set` case 中调用：

```go
		case "set":
			if len(args) < 3 {
				return fmt.Errorf("usage: wt config set <key> <value>")
			}
			setConfigValue(&cfg, args[1], args[2])
			os.MkdirAll(path.Dir(cfgPath), 0755)
			return writeConfigFile(cfgPath, &cfg)
```

- [ ] **Step 3: 验证编译**

```bash
go build ./cmd/cli/
```

- [ ] **Step 4: Commit**

```bash
git add cmd/cli/
git commit -m "feat: add all CLI subcommands"
```

---

### Task 14: install.sh 安装脚本

**Files:**
- Create: `scripts/install.sh`

- [ ] **Step 1: 编写安装脚本**

`scripts/install.sh`:
```sh
#!/bin/sh
set -e

REPO="relaxtortoise/worktree-setup"
BIN_NAME="wt"
INSTALL_DIR="${WT_INSTALL_DIR:-/usr/local/bin}"
VERSION="${WT_VERSION:-latest}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    arm64)   ARCH="arm64" ;;
esac

if [ "$VERSION" = "latest" ]; then
    URL="https://github.com/$REPO/releases/latest/download/${BIN_NAME}-${OS}-${ARCH}"
else
    URL="https://github.com/$REPO/releases/download/${VERSION}/${BIN_NAME}-${OS}-${ARCH}"
fi

echo "Downloading wt $VERSION for $OS/$ARCH..."
curl -fsSL "$URL" -o "/tmp/$BIN_NAME"
chmod +x "/tmp/$BIN_NAME"

if [ ! -d "$INSTALL_DIR" ]; then
    echo "Creating $INSTALL_DIR..."
    mkdir -p "$INSTALL_DIR"
fi

mv "/tmp/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
echo "wt installed to $INSTALL_DIR/$BIN_NAME"
echo ""
echo "Next steps:"
echo "  1. cd to your project and run: wt init"
echo "  2. Run: wt hooks"
echo "  3. Add shell integration to your .bashrc/.zshrc for 'wt switch'"
```

- [ ] **Step 2: 设置可执行权限**

```bash
chmod +x scripts/install.sh
```

- [ ] **Step 3: Commit**

```bash
git add scripts/install.sh
git commit -m "feat: add install.sh script"
```

---

### Task 15: 集成验证与收尾

- [ ] **Step 1: 最终编译验证**

```bash
go build ./cmd/cli/
```
期望：编译成功，无错误。

- [ ] **Step 2: 运行所有单元测试**

```bash
go test ./... -v
```
期望：全部 PASS。

- [ ] **Step 3: 在真实仓库中测试**

```bash
# 使用 ./cli init 初始化当前项目
./cli init
# 验证 .worktree.yaml 生成
cat .worktree.yaml
# 验证项目个人配置生成
cat ~/.config/worktree-setup/projects/github.com-relaxtortoise-worktree-setup/config.yaml
# 验证 hooks
./cli hooks
cat .git/hooks/post-checkout
# 验证 list
./cli list
```

- [ ] **Step 4: 清理测试生成的文件**

```bash
rm -f .worktree.yaml
git checkout .git/hooks/post-checkout 2>/dev/null || rm -f .git/hooks/post-checkout
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "chore: final integration verification"
```
