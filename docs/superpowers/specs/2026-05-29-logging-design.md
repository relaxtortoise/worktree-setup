# 结构化日志设计

## 概述

将项目中零散的 `fmt.Fprintf(os.Stderr, ...)` 错误输出替换为统一的 JSON Lines 结构化日志，写入 `~/.config/worktree-setup/log.jsonl`。面向用户的 `fmt.Print*` 输出保持不变。

## 决策记录

| 决策 | 选择 | 原因 |
|------|------|------|
| 日志范围 | 错误 + 关键操作事件 | 见下方范围说明 |
| 日志库 | `log/slog` | 标准库，零依赖，内置 JSON handler |
| 用户输出 | 保持 `fmt.Print*` 不变 | 功能需求，非日志内容 |
| 日志轮转 | `lumberjack`（10MB / 3备份 / 压缩） | 事实标准，API 极简 |
| 日志级别控制 | `WT_LOG_LEVEL` 环境变量（debug/info/warn/error，默认 info） | 简单，无需 CLI flag |
| 错误处理 | 优雅降级——日志初始化失败不阻塞工具运行 | 日志是辅助功能 |

## 架构

新增包：`internal/logging`

```
internal/logging/
  logging.go      // Init、全局 logger 设置
  logging_test.go
```

**日志目录：** `config.UserConfigDir()`（即 `~/.config/worktree-setup`）。`Init()` 会自动创建目录。

**API：**

```go
// Init 配置 slog 将 JSON Lines 写入 <UserConfigDir>/log.jsonl
// 日志轮转：最大 10MB，保留 3 个备份，gzip 压缩
// 日志级别通过 WT_LOG_LEVEL 环境变量控制（默认 info）
// 仅在日志目录创建失败时返回 error
func Init() error
```

**启动流程（main.go）：**

```go
func main() {
    if err := logging.Init(); err != nil {
        // 静默降级 —— slog 保持默认 stderr text handler
    }
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

**轮转配置：** `lumberjack.Logger{MaxSize: 10, MaxBackups: 3, Compress: true}`

**Slog handler：** `slog.NewJSONHandler(lumberjackWriter, &slog.HandlerOptions{Level: level, AddSource: true})`

## 日志 Schema

所有日志统一 JSON 结构，每行一条记录。

### 通用字段（slog 自动生成）

| 字段 | 类型 | 说明 | 示例 |
|------|------|------|------|
| `time` | string | ISO 8601 时间戳 | `"2026-05-29T10:30:00+08:00"` |
| `level` | string | 日志级别 | `"INFO"` / `"WARN"` / `"ERROR"` / `"DEBUG"` |
| `msg` | string | 事件描述 | `"worktree created"` |
| `source` | string | 调用位置（文件:行号） | `"internal/worktree/create.go:42"` |

### 全局属性（Init 时设置，通过 `slog.With` 注入）

| 字段 | 来源 | 说明 |
|------|------|------|
| `app` | 常量 `"wt"` | 应用标识 |
| `version` | `Version` 变量 | 版本号 |

### 命令级属性（每条命令执行时通过 `slog.With` 注入）

| 字段 | 来源 | 说明 |
|------|------|------|
| `repo` | `os.Getwd()` | 当前 git 仓库路径 |

### 按事件的附加字段

| 事件 | msg | 额外字段 |
|------|-----|----------|
| worktree created | `"worktree created"` | `worktree`, `base_branch` |
| worktree removed | `"worktree removed"` | `worktree` |
| worktree switched | `"worktree switched"` | `from`, `to` |
| config saved | `"config saved"` | `config_file`, `main_worktree` |
| hook installed | `"hook installed"` | `hook`, `git_dir` |
| update applied | `"update applied"` | `from_version`, `to_version` |
| symlink fallback | `"symlink fallback"` | `item`, `error` |
| 通用错误 | per context | `error` |

### 完整示例

```json
{"time":"2026-05-29T10:30:00+08:00","level":"INFO","msg":"worktree created","source":"internal/worktree/create.go:42","app":"wt","version":"1.0.0","repo":"/home/lairui/projects/mytools","worktree":"/home/lairui/projects/mytools/.worktrees/feature-logging","base_branch":"master"}
```

### 实现方式

```go
// main.go: Init 返回 slog 会自动生成的属性，调用方用 With 补充全局属性
logger := slog.Default().With("app", "wt", "version", Version)

// 每条命令开始时补充 repo
cmdLogger := logger.With("repo", repoPath)

// 使用
cmdLogger.Info("worktree created", "worktree", wtPath, "base_branch", baseBranch)
```

## 范围

### 从 stderr 迁移到 slog

| 文件 | 当前 | 改为 |
|------|------|------|
| `internal/actions/symlink.go:25-26` | `fmt.Fprintf(os.Stderr, "Symlink failed...")` | `slog.Warn("symlink fallback", "item", item.To, "error", err)` |
| `internal/actions/symlink.go:33` | `fmt.Fprintf(os.Stderr, "Copied instead...")` | `slog.Info("symlink fallback used", "item", item.To)` |

`symlink.go:26` 的交互式提示（`"Downgrade to copy? [y/N]"`）保持 `fmt.Fprintf(os.Stderr, ...)` 不变——这是用户交互，不是日志。

### 新增日志点（关键操作事件）

```
worktree create      → slog.Info("worktree created", "worktree", wtPath, "base_branch", baseBranch)
worktree remove      → slog.Info("worktree removed", "worktree", wtPath)
worktree switch      → slog.Info("worktree switched", "from", oldPath, "to", newPath)
config save          → slog.Info("config saved", "config_file", cfgPath, "main_worktree", mw)
hook install         → slog.Info("hook installed", "hook", hookName, "git_dir", gitDir)
selfupdate apply     → slog.Info("update applied", "from_version", oldVer, "to_version", newVer)
```

### 不纳入日志

- TUI 交互（bubbletea 内部处理）
- `wt list` / `wt config` 命令输出（功能性，输出到 stdout）
- 交互式提示（"Overwrite? [y/N]"）
- `wt version` 输出

## 错误处理

- `Init()` 仅在目录创建失败时返回 error——`main.go` 不打印任何信息，slog 保持默认 handler
- 运行时写入失败（如磁盘满）：lumberjack 从 `Write()` 返回 error，slog 静默丢弃该条日志
- 无重试、无备选路径——日志是 best-effort，不阻塞工具运行

## 依赖

- `gopkg.in/natefinch/lumberjack.v2` —— 唯一新增依赖，稳定且广泛使用

## 测试策略

- `logging_test.go`：验证 `Init()` 创建日志文件并写入合法 JSON Lines
- 使用 `t.TempDir()` + 重写日志路径进行测试
- 测试日志目录不可写时的优雅降级
- 已有测试不变——测试不调用 `Init()`，因此不受全局 slog 配置影响
