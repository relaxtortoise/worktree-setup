# wt — 增强版 Git Worktree 管理工具

[![CI](https://github.com/relaxtortoise/worktree-setup/actions/workflows/ci.yml/badge.svg)](https://github.com/relaxtortoise/worktree-setup/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/relaxtortoise/worktree-setup)](https://github.com/relaxtortoise/worktree-setup/releases/latest)

> [English](README_en.md)

`wt` 通过 `.worktree.yaml` 配置文件增强了 `git worktree`，支持自动化初始化。定义创建后脚本、文件复制和软链接 —— 每个新 worktree 开箱即用。

## 安装

### 一行命令

```bash
curl -fsSL https://raw.githubusercontent.com/relaxtortoise/worktree-setup/master/scripts/install.sh | sh
```

### Go 安装

```bash
go install github.com/relaxtortoise/worktree-setup/cmd/cli@latest
```

### 下载二进制

预编译二进制可从 [Releases 页面](https://github.com/relaxtortoise/worktree-setup/releases) 下载。

## 快速开始

```bash
# 1. 为你的仓库初始化配置
cd your-project
wt init

# 2. 安装 git hooks（启用自动检测 worktree 创建）
wt hooks

# 3. 编辑 .worktree.yaml 添加初始化步骤
#    （参见下方配置参考）

# 4. 创建一个 worktree
wt add feature-x
```

## 命令

| 命令 | 说明 |
|------|------|
| `wt add [branch]` | 创建新 worktree（省略分支名则启动交互式选择器） |
| `wt remove <name\|path>` | 删除 worktree |
| `wt switch [path]` | 切换到 worktree（跨项目交互式选择器） |
| `wt list` | 列出所有 worktree |
| `wt init` | 初始化 `.worktree.yaml` 和项目配置 |
| `wt hooks` | 安装 git hooks 用于自动检测 |
| `wt run <event>` | 执行已配置的事件步骤 |
| `wt config [get\|set\|list]` | 管理个人配置 |

## 工作原理

执行 `wt add feature-x` 时，`wt` 会：

1. 加载并合并三层配置：全局 → 项目 → 仓库
2. 可选启动 TUI 分支选择器（未指定分支时）
3. 触发 `pre-create` 事件
4. 执行 `git worktree add`（使用计算得到的路径）
5. 触发 `post-create` —— 运行你配置的步骤（脚本、复制、软链接）

通过 `git worktree add` 直接创建的 worktree 也会被 `wt hooks` 安装的 git hook 检测到，因此你的初始化步骤仍会自动执行。

## 配置

完整配置参考见 [docs/configuration.md](docs/configuration.md)。

`.worktree.yaml` 示例：

```yaml
on:
  post-create:
    steps:
      - run: cp .env.example .env
      - run: make install
      - copy:
          - node_modules: node_modules
```

## 架构

```
┌─────────────────────────────┐
│  CLI (cobra)                │
├─────────────────────────────┤
│  Worktree (create/remove)   │
├──────────────┬──────────────┤
│  Engine      │  TUI         │
├──────────────┼──────────────┤
│  Actions     │  Git         │
├──────────────┴──────────────┤
│  Config                     │
└─────────────────────────────┘
```

| 包 | 用途 |
|---|------|
| `cmd/cli/` | CLI 入口与命令定义 |
| `internal/config/` | 三层配置合并 |
| `internal/worktree/` | Worktree 创建/删除编排 |
| `internal/engine/` | 事件生命周期引擎 |
| `internal/actions/` | 步骤执行器（脚本、复制、软链接） |
| `internal/git/` | Git 命令封装 |
| `internal/hooks/` | Git hook 安装 |
| `internal/tui/` | 交互式选择器 |

完整架构设计见 [docs/architecture.md](docs/architecture.md)。

## License

MIT — 详见 [LICENSE](LICENSE)。
