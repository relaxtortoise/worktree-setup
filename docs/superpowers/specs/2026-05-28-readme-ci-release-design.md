# worktree-setup README + CI/CD + Release 设计

> 2026-05-28 | approved

## 目标

为 `github.com/relaxtortoise/worktree-setup` 添加项目文档、CI/CD 流水线、以及基于 Git tag 的自动发版机制。

## 文件清单

### 新增文件

| 文件 | 用途 |
|------|------|
| `README.md` | 项目主页文档 |
| `docs/architecture.md` | 架构设计说明 |
| `docs/configuration.md` | 完整配置参考 |
| `.github/workflows/ci.yml` | CI 流水线（push/PR 触发） |
| `.github/workflows/release.yml` | 发版流水线（`v*` tag 触发） |
| `.golangci.yml` | golangci-lint 配置 |

### 修改文件

| 文件 | 变更 |
|------|------|
| `scripts/install.sh` | 无修改（已存在） |

---

## 1. README.md

### 结构

```
Badges (CI status, latest release)
简介 — wt 是什么，解决什么问题
安装
  - curl | sh 一键脚本
  - go install
快速开始
  - wt init
  - wt hooks
  - wt add feature-x
命令参考（表格）
  - add, remove, switch, list, init, hooks, run, config
配置概要 → 详见 docs/configuration.md
架构设计 → 详见 docs/architecture.md
```

### Badges

```markdown
[![CI](https://github.com/relaxtortoise/worktree-setup/actions/workflows/ci.yml/badge.svg)](https://github.com/relaxtortoise/worktree-setup/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/relaxtortoise/worktree-setup)](https://github.com/relaxtortoise/worktree-setup/releases/latest)
```

---

## 2. docs/architecture.md

### 内容大纲

- **分层架构**
  ```
  ┌─────────────────────────────┐
  │  CLI (cobra)                │
  │  add / remove / switch / …  │
  ├─────────────────────────────┤
  │  Worktree (create/remove)   │  ← 流程编排
  ├──────────────┬──────────────┤
  │  Engine      │  TUI         │  ← 事件引擎 + 交互
  ├──────────────┼──────────────┤
  │  Actions     │  Git         │  ← 执行器 + shell-out
  ├──────────────┴──────────────┤
  │  Config                     │  ← 配置解析/合并
  └─────────────────────────────┘
  ```

- **配置三层合并**：全局(~/.config/worktree-setup/config.yaml) < 项目个人(projects/<name>/config.yaml) < 仓库(.worktree.yaml)，后面覆盖前面

- **事件引擎流程**：
  ```
  pre-create → git worktree add → post-create
  post-checkout (via git hook)
  pre-delete → git worktree remove → post-delete
  ```

- **路径策略**：
  - sibling: `{main_parent}/{repo_name}@{branch}`
  - nested: `{main}/.worktrees/{branch}`
  - home: `~/worktrees/{project_name}/{branch}`
  - 自定义模板: 支持 `{main}`, `{branch}`, `{project_name}` 等变量

---

## 3. docs/configuration.md

### 内容大纲

- `.worktree.yaml` 完整参考
  - `main_worktree`
  - `path_strategy`（字符串或模板对象）
  - `on.<event>.steps[]`（推荐）vs 三段式（run/copy/symlink）
  - step 三种形式：裸字符串 run、copy 对象、symlink 对象
  - copy/symlink 的 map/list 两种写法

- `~/.config/worktree-setup/` 目录结构
  - `config.yaml`（全局）
  - `projects/<project-name>/config.yaml`（项目个人）

- `wt config` 命令用法
  - `wt config list`
  - `wt config get <key>`
  - `wt config set <key> <value>`
  - `--global` 标志

- 完整示例

---

## 4. CI Workflow (`.github/workflows/ci.yml`)

### 触发条件

```yaml
on:
  push:
    branches: [master, main]
  pull_request:
    branches: [master, main]
```

### Jobs

| Job | 内容 |
|-----|------|
| **test** | `go test ./... -race -v`，Go 1.26 |
| **lint** | `golangci-lint run`，v2 最新 |
| **build** | `go build ./cmd/cli/` |
| **cross-build** | GOOS/GOARCH 矩阵：linux/amd64, darwin/amd64, darwin/arm64, windows/amd64，仅编译不产出 artifact |

交叉编译使用 `go build` 的 GOOS/GOARCH 环境变量，无需 CGO。

---

## 5. Release Workflow (`.github/workflows/release.yml`)

### 触发条件

```yaml
on:
  push:
    tags:
      - 'v*'
```

### Jobs

单个 job，包含以下步骤：

1. **Checkout** + 设置 Go 1.26
2. **跨平台编译**（矩阵或循环）：
   - linux/amd64 → `wt-linux-amd64`
   - linux/arm64 → `wt-linux-arm64`
   - darwin/amd64 → `wt-darwin-amd64`
   - darwin/arm64 → `wt-darwin-arm64`
   - windows/amd64 → `wt-windows-amd64.exe`
3. **生成 Changelog**：提取当前 tag 与前一个 tag 之间的 commit 历史，生成 markdown 列表
4. **创建 GitHub Release**：使用 `softprops/action-gh-release`，上传所有二进制 + changelog 作为 body

### Changelog 生成方式

```bash
# 获取上一个 tag
PREV_TAG=$(git describe --tags --abbrev=0 HEAD^ 2>/dev/null || echo "")

if [ -n "$PREV_TAG" ]; then
  git log --oneline --no-merges $PREV_TAG..HEAD
else
  git log --oneline --no-merges
fi
```

---

## 6. golangci-lint 配置 (`.golangci.yml`)

启用 linters：
- `errcheck`, `gosimple`, `govet`, `ineffassign`, `staticcheck`, `unused`
- `goimports`, `misspell`, `gofmt`

排除规则：
- 测试文件放宽 `errcheck`
- 排除 `ST1000`（package comment）

---

## 实现顺序

1. `.golangci.yml`
2. `.github/workflows/ci.yml`
3. `.github/workflows/release.yml`
4. `docs/architecture.md`
5. `docs/configuration.md`
6. `README.md`
7. 提交验证
