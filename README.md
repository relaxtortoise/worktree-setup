# wt — Enhanced Git Worktree Management

[![CI](https://github.com/relaxtortoise/worktree-setup/actions/workflows/ci.yml/badge.svg)](https://github.com/relaxtortoise/worktree-setup/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/relaxtortoise/worktree-setup)](https://github.com/relaxtortoise/worktree-setup/releases/latest)

`wt` enhances `git worktree` with automated setup via `.worktree.yaml` config files. Define post-create scripts, file copies, and symlinks — and every new worktree is ready to use immediately.

## Installation

### One-liner

```bash
curl -fsSL https://raw.githubusercontent.com/relaxtortoise/worktree-setup/master/scripts/install.sh | sh
```

### Go install

```bash
go install github.com/relaxtortoise/worktree-setup/cmd/cli@latest
```

### Download binary

Prebuilt binaries are available on the [releases page](https://github.com/relaxtortoise/worktree-setup/releases).

## Quick Start

```bash
# 1. Initialize config for your repo
cd your-project
wt init

# 2. Install git hooks (enables auto-detection of worktree creation)
wt hooks

# 3. Edit .worktree.yaml to add setup steps
#    (see configuration reference below)

# 4. Create a worktree
wt add feature-x
```

## Commands

| Command | Description |
|---------|-------------|
| `wt add [branch]` | Create a new worktree (interactive branch picker if omitted) |
| `wt remove <name\|path>` | Remove a worktree |
| `wt switch [path]` | Switch to a worktree (interactive picker across projects) |
| `wt list` | List all worktrees |
| `wt init` | Initialize `.worktree.yaml` and project config |
| `wt hooks` | Install git hooks for auto-detection |
| `wt run <event>` | Execute configured event steps |
| `wt config [get\|set\|list]` | Manage personal config |

## How It Works

When you run `wt add feature-x`, `wt`:

1. Loads and merges config from three layers: global → project → repo
2. Optionally launches a TUI branch picker (if no branch specified)
3. Fires the `pre-create` event
4. Runs `git worktree add` with the computed path
5. Fires `post-create` — running your configured steps (scripts, copies, symlinks)

Worktrees created outside of `wt` (e.g. `git worktree add` directly) are detected by the git hook installed via `wt hooks`, so your setup still runs automatically.

## Configuration

See [docs/configuration.md](docs/configuration.md) for the complete configuration reference.

Example `.worktree.yaml`:

```yaml
on:
  post-create:
    steps:
      - run: cp .env.example .env
      - run: make install
      - copy:
          - node_modules: node_modules
```

## Architecture

See [docs/architecture.md](docs/architecture.md) for the full architecture design.

## License

MIT — see [LICENSE](LICENSE).
