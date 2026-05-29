# 交互式 `wt init` 设计

## 概述

将当前硬编码的 `wt init` 替换为基于 Bubble Tea 的交互式 TUI 向导，引导用户完成项目配置。向导收集 `main_worktree`、`path_strategy`、post-create 事件，以及是否将事件保存到 VCS（`.worktree.yaml`）或全部保留在用户配置中。

## 当前行为

`wt init` 用硬编码默认值创建两个文件：

- `.worktree.yaml` — 仅 `on.post-create.run: []` 和 `on.post-checkout.run: []`
- `~/.config/worktree-setup/projects/<project>/config.yaml` — `main_worktree` 和 `path_strategy: sibling`

无提示，无自定义。

## 目标流程

```
wt init
  │
  ├─ 预检查（TUI 之前）
  │   ├─ 不在 git 仓库中？→ 报错退出
  │   ├─ 无 remote origin？→ 报错退出
  │   ├─ .worktree.yaml 已存在？→ 提示覆盖 [y/N]
  │   └─ project config 已存在？→ 提示覆盖 [y/N]
  │
  ├─ 提供了 CLI 参数？→ 跳过 TUI，直接写入
  │
  └─ 启动 InitWizard TUI
      ├─ 第 1 步：main_worktree（文本输入，预填检测值）
      ├─ 第 2 步：path_strategy（单选）
      ├─ 第 3 步：post-create 事件（多选预设 + 自定义）
      ├─ 第 4 步：Save With VCS?（默认 Yes）
      └─ 第 5 步：审核确认
          │
          └─ 写入文件，打印摘要
```

## CLI 参数（非交互模式）

提供以下任意参数即跳过 TUI：

| 参数 | 说明 |
|------|------|
| `--main-worktree <path>` | 主工作树路径 |
| `--path-strategy <name>` | 路径策略：`sibling`、`nested` 或模板 |
| `--no-save-vcs` | 所有配置保存到用户配置（默认 save-vcs） |
| `--post-create-run <cmd>` | 添加 post-create 运行步骤（可重复） |

## 配置保存规则

**Save With VCS（默认）：**
- `.worktree.yaml` ← 仅 `on:`（事件配置）
- `~/.config/worktree-setup/projects/<project>/config.yaml` ← `main_worktree`、`path_strategy`

**Save Without VCS（`--no-save-vcs`）：**
- `~/.config/worktree-setup/projects/<project>/config.yaml` ← 全部配置
- 不创建 `.worktree.yaml`

## 覆盖行为

- **交互模式**：TUI 启动前，对每个已存在文件提示 `[y/N]`，N 跳过该文件。
- **非交互模式**（有 CLI 参数）：直接覆盖，不提示。

## TUI 页面

### 第 1 页 — main_worktree

文本输入框，预填自动检测到的主工作树路径。回车确认，或编辑后确认。

### 第 2 页 — path_strategy

单选：`sibling`（默认）、`nested`、`custom`。选择 `custom` 后显示 Go 模板输入框。

### 第 3 页 — post-create 事件

多选预设列表：

```
[x] cp .env.example .env
[ ] make install
[ ] npm install
[ ] yarn install
[ ] pnpm install
[ ] pip install -r requirements.txt
[ ] go mod download
[ ] bundle install
[+] 添加自定义命令...
```

空格切换选中，回车确认。选择 "+" 弹出文本输入框输入自定义命令。

### 第 4 页 — Save With VCS?

```
> Yes — 事件 → .worktree.yaml，个人设置 → 用户配置
  No  — 全部保存到用户配置
```

默认 Yes。

### 第 5 页 — 审核确认

展示即将写入每个文件的内容。回车确认写入，esc 返回修改。

## 代码变更

### 修改：`cmd/cli/init_cmd.go`

重写。新职责：
- 预检查辅助逻辑（git 仓库、remote origin、已存在文件）
- CLI 参数解析
- 分支：有 CLI 参数 → 直接写入，否则 → 启动 `tui.RunInitWizard()`
- 根据 VCS 决定写入对应配置文件

### 新增：`internal/tui/init_wizard.go`

Bubble Tea 多步骤向导模型：

```go
type WizardModel struct {
    step           WizardStep
    mainWorktree   string
    pathStrategy   string
    customTemplate string
    selectedEvents []string
    saveWithVCS    bool
    cancelled      bool

    textInput textinput.Model
    // ... 单选/多选子模型
}

type WizardStep int
const (
    StepMainWorktree WizardStep = iota
    StepPathStrategy
    StepEvents
    StepSaveVCS
    StepReview
)

type WizardResult struct {
    MainWorktree   string
    PathStrategy   string
    CustomTemplate string
    Events         []string
    SaveWithVCS    bool
    Cancelled      bool
}

func RunInitWizard(detectedMainWT string) WizardResult
```

复用 `internal/tui/selector.go` 中已有的 `textinput` 和选择器模式。

### 新增：`internal/tui/init_wizard_test.go`

- 步骤流转（每步 → 下一步，ESC → 取消）
- 默认值（main_worktree 预填、path_strategy=sibling、Save VCS=true）
- 多选切换和自定义命令输入
- 选择 custom path_strategy 后显示模板输入框
- 审核页反映之前的选择

### CLI 测试补充（`cmd/cli/cli_test.go`）

- `wt init` 无参数
- `wt init --main-worktree /x --path-strategy nested`
- `wt init --no-save-vcs`
- `wt init --post-create-run "make install"`
- 错误：不在 git 仓库中
- 错误：无 remote origin

## 错误处理

| 场景 | 行为 |
|------|------|
| 不在 git 仓库 | TUI 前报错：`"不在 git 仓库中"` |
| 无 remote origin | TUI 前报错：`"未配置 remote origin"` |
| 已存在文件 | 交互模式逐文件提示 `[y/N]`；非交互模式直接覆盖 |
| 用户按 ESC | `Cancelled=true`，不写入任何文件 |
| 文件写入失败 | 返回 error，cobra 打印错误信息 |
| 事件列表为空 | 允许 — 写入 `on.post-create.run: []` |
