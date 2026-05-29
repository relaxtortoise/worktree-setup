# Interactive `wt init` Design

## Overview

Replace the current hardcoded `wt init` with an interactive Bubble Tea TUI wizard that
guides users through configuring their project. The wizard asks about `main_worktree`,
`path_strategy`, post-create events, and whether to save events to VCS (`.worktree.yaml`)
or keep everything in user config.

## Current Behavior

`wt init` creates two files with hardcoded defaults:

- `.worktree.yaml` — only `on.post-create.run: []` and `on.post-checkout.run: []`
- `~/.config/worktree-setup/projects/<project>/config.yaml` — `main_worktree` and `path_strategy: sibling`

No prompts, no customization.

## Target Flow

```
wt init
  │
  ├─ Pre-checks (before TUI)
  │   ├─ Not in a git repo? → error, exit
  │   ├─ No remote origin? → error, exit
  │   ├─ .worktree.yaml exists? → prompt overwrite [y/N]
  │   └─ project config exists? → prompt overwrite [y/N]
  │
  ├─ CLI flags provided? → skip TUI, write directly
  │
  └─ Run InitWizard TUI
      ├─ Step 1: main_worktree (text input, pre-filled)
      ├─ Step 2: path_strategy (single select)
      ├─ Step 3: post-create events (multi-select presets + custom)
      ├─ Step 4: Save With VCS? (default Yes)
      └─ Step 5: Review & confirm
          │
          └─ Write files, print summary
```

## CLI Flags (non-interactive mode)

Providing any of these flags skips the TUI:

| Flag | Description |
|------|-------------|
| `--main-worktree <path>` | Main worktree path |
| `--path-strategy <name>` | Path strategy: `sibling`, `nested`, or a template |
| `--no-save-vcs` | Save everything to user config (default: save-vcs) |
| `--post-create-run <cmd>` | Add a post-create run step (repeatable) |

## Config Save Rules

**Save With VCS (default):**
- `.worktree.yaml` ← only `on:` (event configuration)
- `~/.config/worktree-setup/projects/<project>/config.yaml` ← `main_worktree`, `path_strategy`

**Save Without VCS (`--no-save-vcs`):**
- `~/.config/worktree-setup/projects/<project>/config.yaml` ← everything
- `.worktree.yaml` is NOT created

## Overwrite Behavior

- **Interactive mode**: Before launching TUI, prompt `[y/N]` for each existing file. N skips that file.
- **Non-interactive mode** (CLI flags): Overwrite without prompting.

## TUI Pages

### Page 1 — main_worktree

Text input pre-filled with auto-detected main worktree path. Press enter to confirm or edit first.

### Page 2 — path_strategy

Single-select: `sibling` (default), `nested`, `custom`. Choosing `custom` reveals a Go template input field.

### Page 3 — post-create events

Multi-select with presets:

```
[x] cp .env.example .env
[ ] make install
[ ] npm install
[ ] yarn install
[ ] pnpm install
[ ] pip install -r requirements.txt
[ ] go mod download
[ ] bundle install
[+] Add custom command...
```

Space toggles selection, enter confirms. Selecting "+" opens a text input for a custom command.

### Page 4 — Save With VCS?

```
> Yes — events → .worktree.yaml, personal settings → user config
  No  — everything saved to user config only
```

Default: Yes.

### Page 5 — Review

Shows what will be written to each file. Enter to confirm, esc to go back.

## Code Changes

### Modified: `cmd/cli/init_cmd.go`

Rewrite. New responsibilities:
- Pre-check helpers (git repo, remote origin, existing files)
- CLI flag parsing
- Branch: CLI flags → direct write, else → launch `tui.RunInitWizard()`
- Write config to appropriate files based on VCS decision

### New: `internal/tui/init_wizard.go`

Bubble Tea multi-step model:

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
    // ... sub-models for select/multiselect
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

Reuses existing `textinput` and selector patterns from `internal/tui/selector.go`.

### New: `internal/tui/init_wizard_test.go`

- Step progression (each step → next, ESC → cancelled)
- Default values (main_worktree pre-filled, path_strategy=sibling, Save VCS=true)
- Multi-select toggle and custom command entry
- Custom path_strategy reveals template input
- Review page reflects prior selections

### CLI test additions (`cmd/cli/cli_test.go`)

- `wt init` without args
- `wt init --main-worktree /x --path-strategy nested`
- `wt init --no-save-vcs`
- `wt init --post-create-run "make install"`
- Error: not in git repo
- Error: no remote origin

## Error Handling

| Scenario | Behavior |
|----------|----------|
| Not in git repo | Error before TUI: `"not in a git repository"` |
| No remote origin | Error before TUI: `"no remote origin configured"` |
| Existing files | Prompt `[y/N]` per file (interactive); overwrite (non-interactive) |
| User presses ESC | `Cancelled=true`, no files written |
| File write failure | Return error, cobra prints it |
| Empty events list | Allowed — writes `on.post-create.run: []` |
