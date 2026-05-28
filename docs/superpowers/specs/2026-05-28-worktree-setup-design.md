# worktree-setup 设计文档

## 概述

`wt` 是一个增强 `git worktree` 的 Go CLI 工具。创建新 worktree 时可以自动从主 worktree 复制文件、创建软链接（如 `vendor/`、`node_modules/`）以及执行自定义命令 —— 全部由事件驱动的 YAML 配置文件控制。

## CLI 命令

| 命令 | 说明 |
|---|---|
| `wt add [branch]` | 创建 worktree。带分支参数直接创建（跳过 TUI），无参数进入 fuzzy 选择器 |
| `wt remove [name]` | 删除 worktree，触发 pre/post-delete 事件 |
| `wt switch [name]` | 切换到已有 worktree，输出路径供 shell 集成。无参数进入 TUI 选择器 |
| `wt list` | 列出所有 worktree |
| `wt init` | 生成 `.worktree.yaml` 模板及项目个人配置 |
| `wt install` | 往 `.git/hooks/` 安装 hook 脚本 |
| `wt run <event>` | 执行配置中对应事件，供 git hooks 内部调用 |
| `wt config [get/set/list]` | 管理项目个人配置 `~/.config/worktree-setup/projects/<name>/config.yaml` |
| `wt config -g \| --global [get/set/list]` | 管理全局个人配置 `~/.config/worktree-setup/config.yaml` |

### 参数

- `wt add --path /custom/path` — 显式指定 worktree 路径（覆盖 path_strategy）
- `wt add --no-fetch` — 跳过自动 `git fetch origin`，也可通过环境变量 `WT_NO_FETCH=1` 全局停用

### Shell 集成

`wt switch` 输出目标目录路径，由 shell 函数包装实现 `cd`：

```bash
wt() {
    if [[ "$1" == "switch" ]]; then
        local dir=$(command wt switch "${@:2}")
        [[ -n "$dir" ]] && cd "$dir"
    else
        command wt "$@"
    fi
}
```

## 架构

```
cmd/cli/
  └── main.go              # 入口，cobra 命令路由

internal/
  ├── config/               # 配置解析（三层合并）
  │   ├── parser.go         # .worktree.yaml 解析
  │   ├── hierarchy.go      # 优先级合并
  │   └── schema.go         # 配置结构定义
  ├── engine/               # 事件引擎
  │   └── engine.go         # 按事件名分发执行 actions
  ├── actions/              # 动作实现
  │   ├── run.go            # shell 命令执行
  │   ├── copy.go           # 文件复制
  │   └── symlink.go        # 跨平台软链接
  ├── git/                  # git 操作封装（shell out）
  │   ├── worktree.go       # add/remove/list worktree
  │   └── branch.go         # fetch、列出远程分支
  ├── tui/                  # 分支/worktree 选择器
  │   └── selector.go       # bubbletea + bubbles fuzzy 选择器
  ├── hooks/                # git hooks 管理
  │   └── installer.go      # wt install 实现
  └── worktree/             # worktree 生命周期管理
      ├── create.go         # wt add 完整流程
      └── remove.go         # wt remove 完整流程
```

### 技术选型

- **TUI**：`bubbletea` + `bubbles` 实现跨平台 fuzzy 选择器（无需系统安装 `fzf`）
- **Git 操作**：shell out 调用原生 `git`（始终可用，worktree 支持可靠）
- **CLI 框架**：`cobra`
- **配置解析**：`gopkg.in/yaml.v3`
- **分发方式**：GoReleaser 构建单文件静态二进制

## 配置层级

由低到高优先级：

1. `~/.config/worktree-setup/config.yaml` —— 全局默认配置（最低）
2. `~/.config/worktree-setup/projects/<name>/config.yaml` —— 项目个人配置
3. `<repo>/.worktree.yaml` —— 仓库配置，随代码提交（最高）

其中 `<name>` 由 `git remote get-url origin` 提取：`{host}/{owner}/{repo}`。

`main_worktree` 与本地环境强相关，`wt init` 默认将其写入项目个人配置而非 `.worktree.yaml`。若 `.worktree.yaml` 中显式声明了 `main_worktree`，其优先级高于个人配置。

## .worktree.yaml 格式

```yaml
# 可选：显式指定主 worktree 路径。
# 通常此配置保存在个人项目配置中（与本地环境绑定），不随仓库提交。
# 若在 .worktree.yaml 中显式声明也会被识别（优先级高于个人配置）。
# 省略则自动检测（通过 git worktree list 查找 main/master 分支）。
main_worktree: "/home/me/projects/myapp"

# 可选：worktree 目录放置策略
path_strategy: sibling

on:
  pre-create:
    run:
      - "git fetch origin --prune"

  post-create:
    copy:
      ".env.example": ".env"
      "config/dev.yaml": "config/dev.yaml"
    symlink:
      "../main/node_modules": "node_modules"
      "../main/vendor": "vendor"
    run:
      - "go mod download"

  post-checkout:
    run:
      - "git submodule update --init --recursive"

  pre-delete:
    run:
      - "docker compose -f docker-compose.dev.yml down"

  post-delete:
    run: []
```

### Action 语法

`copy` 和 `symlink` 在每个事件块中仅支持两种形式之一（map 或 list，不允许混用）。list 内支持混合 string 和 object 项。

**Map 形式：**

```yaml
copy:
  ".env.example": ".env"
symlink:
  "../main/node_modules": "node_modules"
```

**List 形式（string 与 object 可混合）：**

```yaml
copy:
  - "go.mod"                     # 相同路径直拷
  - ".env.example:.env"          # 冒号简洁语法（from:to）
  - from: "scripts/hooks.sh"     # 对象形式
    to: ".git/hooks/pre-commit"
```

冒号在所有主流 OS 中均为非法文件名字符，因此不会产生歧义。

### 事件

| 事件 | 触发时机 | 可用 action |
|---|---|---|
| `pre-create` | `git worktree add` 之前 | 仅 `run`（目标目录尚不存在） |
| `post-create` | `git worktree add` 之后 | `run`、`copy`、`symlink` |
| `post-checkout` | 已有 worktree 中 checkout 后 | `run`、`copy`、`symlink` |
| `pre-delete` | `git worktree remove` 之前 | 仅 `run` |
| `post-delete` | `git worktree remove` 之后 | 仅 `run` |

## 路径策略

控制新 worktree 目录的创建位置。在任意配置层级中均可设置。

| 策略 | 路径公式 | 示例 |
|---|---|---|
| `sibling`（默认） | `{main_parent}/{repo_name}@{branch}` | `/home/me/projects/myapp@feature-x` |
| `nested` | `{main}/.worktrees/{branch}` | `/home/me/projects/myapp/.worktrees/feature-x` |
| `home` | `~/worktrees/{project_name}/{branch}` | `~/worktrees/myapp/feature-x` |

**自定义模板：**

```yaml
path_strategy:
  template: "/data/worktrees/{project_name}/{branch}"
```

可用变量：`{main}`、`{main_parent}`、`{repo_name}`、`{project_name}`、`{branch}`、`{host}`、`{owner}`。

优先级：CLI `--path` 参数 > `.worktree.yaml` > 项目配置 > 全局配置 > 默认 `sibling`。

## TUI（Fuzzy 选择器）

```
┌──────────────────────────────────────────────────┐
│ Search: feat▌                                     │
│                                                    │
│  origin/feature/user-auth      (2 days ago)  alice │
│  origin/feature/api-cache      (5 days ago)  bob   │
│  origin/feature/login-v2       (1 week ago)  alice │
│                                                    │
│  4 matches │ ↑↓ navigate │ Enter select │ Esc quit │
└──────────────────────────────────────────────────┘
```

- **Fuzzy 匹配**：大小写不敏感，多字符模糊匹配分支名
- **自动 fetch**：打开时自动执行 `git fetch origin`（`--no-fetch` 或 `WT_NO_FETCH=1` 跳过）
- **分支列表**：仅显示 origin 远程分支，按最近提交时间降序，排除 HEAD 和已 checkout 的分支
- **每行信息**：分支名、最近提交相对时间、作者
- **快捷键**：`↑↓` 或 `Ctrl+j/k` 导航，`Enter` 确认，`Esc` 退出
- **快捷路径**：`wt add <branch>` 传入完整分支名时跳过 TUI 直接创建

## Git Hooks

`wt install` 写入轻量 shell 包装脚本到 `.git/hooks/`：

```sh
#!/bin/sh
# .git/hooks/post-checkout（由 wt install 安装）
wt run post-checkout "$@" --detect-create
```

安装的 hook：`post-checkout`。

### Hook 触发事件原理

原生 git 中仅 `post-checkout` 这一个 hook 与 worktree 操作相关。`pre-create`、`pre-delete`、`post-delete` 没有对应的原生 hook —— 它们仅在直接使用 `wt add` / `wt remove` 时触发。

当 `wt run post-checkout` 通过 hook 被调用时（带 `--detect-create` 标志），它检查 git 传入的 previous HEAD 值：

- **previous HEAD = `0000...`**（全零）→ 表示 worktree 刚被创建 → 执行 `post-create` 动作
- **previous HEAD ≠ `0000...`** → 普通分支切换 → 执行 `post-checkout` 动作

这确保了通过任意工具（IDE、手动 `git worktree add` 等）创建的 worktree 都能正确执行 `post-create` 配置中的自动化步骤。

## 跨平台软链接

- **Linux/macOS**：`os.Symlink(source, target)`
- **Windows**：尝试 `os.Symlink`，需要开发者模式或管理员权限。若无权限则警告并回退为复制目录。

## Worktree 删除

`wt remove` 完整流程：
1. 执行 `pre-delete` 动作
2. 运行 `git worktree remove <path>`
3. 执行 `post-delete` 动作

## 安装

### scripts/install.sh

一行命令从 GitHub Releases 下载最新二进制：

```sh
curl -fsSL https://github.com/relaxtortoise/worktree-setup/releases/latest/download/install.sh | sh
```

- `WT_INSTALL_DIR` 环境变量覆盖安装路径（默认 `/usr/local/bin`）
- `WT_VERSION` 环境变量指定版本（默认 `latest`）
- GoReleaser 构建多平台二进制，`install.sh` 自动选择 OS/arch

### 安装后步骤

1. `wt init` —— 生成 `.worktree.yaml`
2. `wt install` —— 安装 git hooks
3. 将 `wt` shell 函数添加到 `.zshrc`/`.bashrc`（用于 `wt switch` 的 cd 集成）

## 错误处理

- 配置解析错误：报告文件名、行号及具体问题；终止执行
- 非 git 仓库：尽早检测并给出明确错误提示
- Action 失败：报告哪个 action 失败、输出内容，以及是否继续或终止（可在 action 级别配置 `on_error: continue|abort`，默认 `abort`）
- 权限错误（Windows 上 symlink）：警告、建议解决方案、回退为复制
