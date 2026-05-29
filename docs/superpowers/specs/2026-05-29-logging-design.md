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

**Slog handler：** `slog.NewJSONHandler(lumberjackWriter, &slog.HandlerOptions{Level: level})`

## 范围

### 从 stderr 迁移到 slog

| 文件 | 当前 | 改为 |
|------|------|------|
| `internal/actions/symlink.go:25-26` | `fmt.Fprintf(os.Stderr, "Symlink failed...")` | `slog.Warn("symlink failed, offering copy fallback", "to", item.To, "err", err)` |
| `internal/actions/symlink.go:33` | `fmt.Fprintf(os.Stderr, "Copied instead...")` | `slog.Info("copy fallback used", "to", item.To)` |

`symlink.go:26` 的交互式提示（`"Downgrade to copy? [y/N]"`）保持 `fmt.Fprintf(os.Stderr, ...)` 不变——这是用户交互，不是日志。

### 新增日志点（关键操作事件）

```
worktree create      → slog.Info("worktree created", "path", wtPath, "repo", projectName)
worktree remove      → slog.Info("worktree removed", "path", wtPath)
worktree switch      → slog.Info("worktree switched", "path", newPath)
config save          → slog.Info("config saved", "path", cfgPath)
hook install         → slog.Info("hook installed", "hook", hookName)
selfupdate apply     → slog.Info("update applied", "from", oldVer, "to", newVer)
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
