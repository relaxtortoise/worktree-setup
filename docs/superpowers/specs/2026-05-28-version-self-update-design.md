# wt version & self-update Design Spec

## Overview

Add `wt version` and `wt self-update` commands to the CLI tool. Version is injected via ldflags at build time. Self-update downloads and replaces the binary from GitHub Releases.

## File Changes

```
cmd/cli/
  main.go          # Add var declarations for ldflags injection (version, commit, buildTime)
  version.go       # New: wt version command
  selfupdate.go    # New: wt self-update command

.github/workflows/release.yml  # Inject ldflags with version/commit/buildTime
```

## `wt version`

### Output Format

```
wt v1.2.3 (linux/amd64) commit abc1234 built at 2026-01-15T12:00:00Z
```

Local dev builds (no ldflags) output:

```
wt dev (linux/amd64) commit unknown built at unknown
```

### Implementation

Three package-level `var` strings in `main.go`:

```go
var (
    version   = "dev"
    commit    = "unknown"
    buildTime = "unknown"
)
```

`version.go` creates a cobra command that prints the formatted string. Uses `runtime.GOOS`/`runtime.GOARCH` for platform info — not injected.

## `wt self-update`

### Usage

```
wt self-update [target-version]   # Default: latest release
wt self-update v1.2.0            # Specific version
```

### Flags

| Flag | Description |
|------|-------------|
| `--yes` / `-y` | Skip confirmation prompt |
| `--check` | Check only — prints versions and exits. Exit code 1 if update available. |

### Flow

1. **Determine target version**
   - No arg: `GET /repos/relaxtortoise/worktree-setup/releases/latest` → extract `tag_name`
   - With arg: `GET /repos/relaxtortoise/worktree-setup/releases/tags/{tag}` → verify exists
   - If current version == target version: print "Already up to date" and exit 0

2. **Check mode** (`--check`)
   - Print current and target version, exit 0 if same, exit 1 if different
   - No download performed

3. **Confirmation** (skipped with `--yes`)
   ```
   Current version: v1.2.3 (linux/amd64)
   New version:     v1.3.0 (linux/amd64)
   Update? [y/N]:
   ```

4. **Download & replace**
   - Download URL: `https://github.com/relaxtortoise/worktree-setup/releases/download/{tag}/wt-{os}-{arch}`
   - Download to temp file
   - `os.Chmod` to make executable
   - `os.Rename` to replace current binary
   - Print success message

### Edge Cases

- **Windows**: `os.Rename` works even on running binaries; no special handling needed
- **Network error**: Print "Failed to connect to GitHub, check your network" and exit 1
- **Permission denied**: Print "Permission denied, try running with sudo" and exit 1
- **No GitHub releases found**: Print "No releases found" and exit 1
- **HTTP 404 on version**: Print "Version {tag} not found" and exit 1
- **Current version is `dev`**: Still allow update; prompt shows `dev → {target}`

### Dependencies

No new dependencies. Uses only `net/http`, `os`, `fmt`, `runtime` from stdlib.

## Build & CI

### ldflags

```
go build -ldflags "\
  -X main.version=${VERSION} \
  -X main.commit=${COMMIT} \
  -X main.buildTime=${BUILD_TIME}" \
  ./cmd/cli/
```

### Release Workflow Changes

In `.github/workflows/release.yml`, before the build step:

```yaml
- name: Extract version
  run: |
    echo "VERSION=${GITHUB_REF_NAME#v}" >> $GITHUB_ENV
    echo "COMMIT=$(git rev-parse --short HEAD)" >> $GITHUB_ENV
```

Build step adds `-ldflags` with the three variables. `buildTime` uses `date -u +%Y-%m-%dT%H:%M:%SZ`.

### Local Development

No changes needed. Devs run `go build ./cmd/cli/` as before, version defaults to `dev`.

## Testing

- `wt version` prints expected format with `dev` when built without ldflags
- `wt self-update --check` with `dev` version reports update available
- `wt self-update --check` with current latest version reports up to date
- Self-update dry-run (no actual binary replacement test in CI, verify manually)
