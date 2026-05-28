# worktree-setup Design Spec

## Overview

`wt` is a Go CLI tool that enhances `git worktree` with automated setup workflows. When creating a new worktree, it can copy files from the main worktree, create symlinks (e.g., `vendor/`, `node_modules/`), and execute custom commands — all driven by an event-based YAML config.

## CLI Commands

| Command | Description |
|---|---|
| `wt add [branch]` | Create a worktree. With branch arg: direct create (skips TUI). Without: fuzzy TUI selector. |
| `wt remove [name]` | Remove a worktree, triggering pre/post-delete events. |
| `wt switch [name]` | Switch to an existing worktree. Outputs path for shell integration. Without arg: TUI selector. |
| `wt list` | List all worktrees. |
| `wt init` | Generate `.worktree.yaml` template and project personal config. |
| `wt install` | Install git hook scripts into `.git/hooks/`. |
| `wt run <event>` | Execute a configured event. Called internally by git hooks. |
| `wt config [get/set/list]` | Manage personal config under `~/.config/worktree-setup/`. |

### Flags

- `wt add --path /custom/path` — explicit worktree path (overrides path_strategy)
- `wt add --no-fetch` — skip automatic `git fetch origin`

### Shell Integration

`wt switch` outputs the target directory path. A shell function wraps it to actually `cd`:

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

## Architecture

```
cmd/wt/
  └── main.go              # entry point, cobra command routing

internal/
  ├── config/               # config parsing (3-tier merge)
  │   ├── parser.go         # .worktree.yaml parsing
  │   ├── hierarchy.go      # priority-based config merge
  │   └── schema.go         # config struct definitions
  ├── engine/               # event engine
  │   └── engine.go         # dispatch actions by event name
  ├── actions/              # action implementations
  │   ├── run.go            # shell command execution
  │   ├── copy.go           # file copy
  │   └── symlink.go        # cross-platform symlink
  ├── git/                  # git operation wrappers (shell out)
  │   ├── worktree.go       # add/remove/list worktree
  │   └── branch.go         # fetch, list remote branches
  ├── tui/                  # branch/worktree selector
  │   └── selector.go       # bubbletea + bubbles fuzzy selector
  ├── hooks/                # git hooks management
  │   └── installer.go      # wt install implementation
  └── worktree/             # worktree lifecycle management
      ├── create.go         # wt add full flow
      └── remove.go         # wt remove full flow
```

### Tech Choices

- **TUI**: `bubbletea` + `bubbles` for cross-platform fuzzy selector (no system `fzf` dependency)
- **Git operations**: shell out to native `git` (always available, reliable worktree support)
- **CLI**: `cobra` for command routing
- **Config**: `gopkg.in/yaml.v3` for YAML parsing
- **Distribution**: single static binary via GoReleaser

## Config Hierarchy

Lowest to highest priority:

1. `~/.config/worktree-setup/config.yaml` — global defaults
2. `~/.config/worktree-setup/projects/<name>/config.yaml` — project personal config
3. `<repo>/.worktree.yaml` — repo config (committed)

Where `<name>` is derived from `git remote get-url origin`: `{host}/{owner}/{repo}`.

## .worktree.yaml Format

```yaml
# Optional: explicit main worktree path. Auto-detected if omitted.
main_worktree: "/home/me/projects/myapp"

# Optional: worktree directory placement strategy
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

### Action Syntax

`copy` and `symlink` each support ONE of two forms per event block (map or list, not mixed). Lists support mixing string and object items.

**Map form:**

```yaml
copy:
  ".env.example": ".env"
symlink:
  "../main/node_modules": "node_modules"
```

**List form (string + object mix allowed):**

```yaml
copy:
  - "go.mod"                     # same path
  - ".env.example:.env"          # colon shorthand (from:to)
  - from: "scripts/hooks.sh"     # object form
    to: ".git/hooks/pre-commit"
```

Colon shorthand is valid because `:` is illegal in filenames across all major OSes.

### Events

| Event | When | Valid actions |
|---|---|---|
| `pre-create` | Before `git worktree add` | `run` only (no target dir exists) |
| `post-create` | After `git worktree add` | `run`, `copy`, `symlink` |
| `post-checkout` | After checkout in existing worktree | `run`, `copy`, `symlink` |
| `pre-delete` | Before `git worktree remove` | `run` only |
| `post-delete` | After `git worktree remove` | `run` only |

## Path Strategy

Controls where new worktree directories are created. Configurable at any config tier.

| Strategy | Path formula | Example |
|---|---|---|
| `sibling` (default) | `{main_parent}/{repo_name}@{branch}` | `/home/me/projects/myapp@feature-x` |
| `nested` | `{main}/.worktrees/{branch}` | `/home/me/projects/myapp/.worktrees/feature-x` |
| `home` | `~/worktrees/{project_name}/{branch}` | `~/worktrees/myapp/feature-x` |

**Custom template:**

```yaml
path_strategy:
  template: "/data/worktrees/{project_name}/{branch}"
```

Available variables: `{main}`, `{main_parent}`, `{repo_name}`, `{project_name}`, `{branch}`, `{host}`, `{owner}`.

Priority: CLI `--path` flag > `.worktree.yaml` > project config > global config > `sibling` default.

## TUI (Fuzzy Selector)

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

- **Fuzzy match**: case-insensitive, multi-character fuzzy on branch name
- **Auto fetch**: runs `git fetch origin` on open (skip with `--no-fetch`)
- **Branch list**: origin remote branches only, sorted by last commit date desc, excludes HEAD and already-checked-out branches
- **Per row**: branch name, relative commit time, author
- **Keyboard**: `↑↓` or `Ctrl+j/k` navigate, `Enter` select, `Esc` quit
- **Quick path**: `wt add <branch>` with a full branch name skips TUI entirely

## Git Hooks

`wt install` writes thin shell wrappers into `.git/hooks/`:

```sh
#!/bin/sh
# .git/hooks/post-checkout (installed by wt)
wt run post-checkout "$@" --detect-create
```

Installed hooks: `post-checkout`.

### How hook-triggered events work

Native git only has `post-checkout` as a hook point relevant to worktree operations. `pre-create`, `pre-delete`, and `post-delete` have no corresponding native hooks — they only fire when using `wt add` / `wt remove` directly.

When `wt run post-checkout` is called from the hook (with `--detect-create`), it inspects the previous HEAD ref passed by git:

- **Previous HEAD = `0000...`** (all zeros) → worktree was just created → execute `post-create` actions
- **Previous HEAD ≠ `0000...`** → normal branch switch → execute `post-checkout` actions

This ensures that worktrees created by any tool (IDE, manual `git worktree add`, etc.) still get their `post-create` setup executed.

## Cross-platform Symlink

- **Linux/macOS**: `os.Symlink(source, target)`
- **Windows**: attempt `os.Symlink`, requiring developer mode or admin privileges. If permission denied, warn and fall back to copying the directory instead.

## Worktree Removal

`wt remove` triggers the full lifecycle:
1. Execute `pre-delete` actions
2. Run `git worktree remove <path>`
3. Execute `post-delete` actions

## Installation

### install.sh

One-liner for fetching the latest binary from GitHub Releases:

```sh
curl -fsSL https://github.com/relaxtortoise/worktree-setup/releases/latest/download/install.sh | sh
```

- `WT_INSTALL_DIR` env var overrides install path (default `/usr/local/bin`)
- `WT_VERSION` env var pins a specific version (default `latest`)
- GoReleaser builds multi-platform binaries, `install.sh` auto-selects OS/arch

### Post-install

After install, users run:
1. `wt init` — scaffold `.worktree.yaml`
2. `wt install` — install git hooks
3. Add `wt` shell function to `.zshrc`/`.bashrc` (for `wt switch` cd integration)

## Error Handling

- Config parse errors: report file, line, and specific issue; abort
- Missing git repo: detect early with clear error message
- Action failures: report which action failed, its output, and whether to continue or abort (configurable per-action `on_error: continue|abort`, default `abort`)
- Permission errors (symlink on Windows): warn, suggest fix, fall back to copy
