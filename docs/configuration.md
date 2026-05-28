# Configuration Reference

## Config Files

| File | Scope | Purpose |
|------|-------|---------|
| `~/.config/worktree-setup/config.yaml` | Global | Defaults for all projects |
| `~/.config/worktree-setup/projects/<name>/config.yaml` | Project | Per-project personal settings |
| `.worktree.yaml` (repo root) | Repository | Shared team config, version-controlled |

Layers merge with increasing priority: global < project < repo.

## Configuration Schema

### Top-Level Keys

```yaml
main_worktree: /path/to/main/worktree   # Absolute path
path_strategy: sibling                  # Strategy name or template
on:                                     # Event hooks
  pre-create: ...
  post-create: ...
  post-checkout: ...
  pre-delete: ...
  post-delete: ...
```

### `main_worktree`

Absolute path to the main (bare) worktree. If omitted, `wt` auto-detects it via `git worktree list`.

### `path_strategy`

Can be a string (strategy name) or an object with a `template` field:

```yaml
# Built-in strategies
path_strategy: sibling
path_strategy: nested
path_strategy: home

# Custom template
path_strategy:
  template: "{main_parent}/wt/{branch}"
```

#### Template Variables

| Variable | Example |
|----------|---------|
| `{main}` | `/home/user/project` |
| `{main_parent}` | `/home/user` |
| `{repo_name}` | `project` |
| `{project_name}` | `github.com-owner-repo` |
| `{branch}` | `feature-x` (slashes replaced with `-`) |

### Event Hooks (`on`)

Five lifecycle events are available:

| Event | When |
|-------|------|
| `pre-create` | Before `git worktree add` |
| `post-create` | After worktree is created |
| `post-checkout` | On checkout in a worktree (via git hook) |
| `pre-delete` | Before worktree removal |
| `post-delete` | After worktree removal (runs in main worktree) |

#### Steps Format (Recommended)

Each event has a `steps` list executed in order:

```yaml
on:
  post-create:
    steps:
      - run: cp .env.example .env
      - copy:
          - from: node_modules
            to: node_modules
      - symlink:
          - from: ../shared/logs
            to: logs
```

#### Legacy Format (also supported)

```yaml
on:
  post-create:
    run:
      - echo "hello"
    copy:
      .env.example: .env
    symlink:
      logs: ../shared/logs
```

Execution order: copy → symlink → run (in legacy mode).

### Step Types

#### `run`

Execute shell commands:

```yaml
# Implicit (bare string)
steps:
  - echo "hello"

# Explicit
steps:
  - run: echo "hello"
```

#### `copy`

Copy files or directories from the main worktree (or template path) to the new worktree:

```yaml
steps:
  - copy:
      # Map form (from: to)
      .env.example: .env
      config/dev.yaml: config/dev.yaml

      # List form
      - from: .env.example
        to: .env
      - .env.example:.env
```

#### `symlink`

Create symbolic links:

```yaml
steps:
  - symlink:
      # Map form
      ../shared/logs: logs
      /var/cache/project: cache

      # List form
      - from: ../shared/logs
        to: logs
```

## CLI: `wt config`

Manage personal config files:

```bash
wt config list              # Show project config
wt config list --global     # Show global config
wt config get <key>         # Get a specific value
wt config set <key> <val>   # Set a value
wt config set <key> <val> --global  # Set globally
```

Supported keys: `main_worktree`, `path_strategy`.

## Complete Example

Global config (`~/.config/worktree-setup/config.yaml`):

```yaml
path_strategy: sibling
```

Project config (`~/.config/worktree-setup/projects/github.com-myorg-myproject/config.yaml`):

```yaml
main_worktree: /home/user/work/myproject
```

Repo config (`.worktree.yaml`):

```yaml
on:
  pre-create:
    steps:
      - run: echo "Creating worktree..."
  post-create:
    steps:
      - run: cp .env.example .env
      - run: make install
      - copy:
          - node_modules: node_modules
          - .cache: .cache
  post-checkout:
    steps:
      - run: make deps
  pre-delete:
    steps:
      - run: echo "Cleaning up..."
```
