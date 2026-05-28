# Architecture

## Layered Design

```
┌─────────────────────────────────────┐
│  CLI (cobra)                        │
│  add / remove / switch / list / …   │
├─────────────────────────────────────┤
│  Worktree (create/remove)           │  ← workflow orchestration
├──────────────┬──────────────────────┤
│  Engine      │  TUI                 │  ← event engine + interaction
├──────────────┼──────────────────────┤
│  Actions     │  Git                 │  ← executors + shell-out
├──────────────┴──────────────────────┤
│  Config                             │  ← config parsing/merging
└─────────────────────────────────────┘
```

## Package Map

| Package | Purpose |
|---------|---------|
| `cmd/cli/` | CLI entry point and command definitions (cobra) |
| `internal/config/` | YAML config parsing, schema types, three-layer merge |
| `internal/worktree/` | Worktree create/remove orchestration, path computation |
| `internal/engine/` | Event lifecycle engine (pre-create, post-create, etc.) |
| `internal/actions/` | Step executors: run (shell), copy, symlink |
| `internal/git/` | Git command wrappers (worktree list, branch, main worktree detection) |
| `internal/hooks/` | Git hook installer (post-checkout) |
| `internal/tui/` | Interactive selectors (branch picker, worktree picker) |

## Configuration Hierarchy

Config is loaded from three layers and merged with increasing priority:

```
Global (~/.config/worktree-setup/config.yaml)
  ↓ overridden by
Project (~/.config/worktree-setup/projects/<name>/config.yaml)
  ↓ overridden by
Repo (.worktree.yaml)
```

Later layers override earlier ones per-key. See [configuration.md](configuration.md) for the full reference.

## Event Engine

The engine fires events at lifecycle points:

```
pre-create  →  git worktree add  →  post-create
                                      post-checkout (via git hook)
pre-delete  →  git worktree remove  →  post-delete
```

Each event can run a sequence of steps in order: `run` (shell commands), `copy` (file/directory copies), and `symlink`.

## Path Strategy

Worktree placement is controlled by the `path_strategy` config:

| Strategy | Template |
|----------|----------|
| `sibling` (default) | `{main_parent}/{repo_name}@{branch}` |
| `nested` | `{main}/.worktrees/{branch}` |
| `home` | `~/worktrees/{project_name}/{branch}` |
| custom | User-defined template string |

Template variables: `{main}`, `{main_parent}`, `{repo_name}`, `{project_name}`, `{branch}`.

## Git Hook Integration

`wt hooks` installs a `post-checkout` hook that invokes `wt run post-checkout --detect-create`. The hook auto-detects new worktree creation (prev-head is all zeros) and triggers the `post-create` event, enabling automated setup even when worktrees are created outside of `wt add`.
