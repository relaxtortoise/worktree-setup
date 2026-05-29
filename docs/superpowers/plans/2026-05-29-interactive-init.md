# Interactive `wt init` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace hardcoded `wt init` with an interactive Bubble Tea TUI wizard that collects main_worktree, path_strategy, post-create events, and VCS save decision.

**Architecture:** A new `InitWizard` Bubble Tea model with 7 steps in `internal/tui/init_wizard.go`. The `init_cmd.go` cobra command is rewritten to handle pre-checks, CLI flag parsing, and branching between direct-write (non-interactive) and the TUI wizard. Config files are written using a mix of `config.WriteFile` (for events `.worktree.yaml`) and raw string formatting (for project config, which includes `main_worktree`/`path_strategy`).

**Tech Stack:** Go, Cobra, Bubble Tea + Bubbles (textinput)

---

### Task 1: Add MarshalYAML to PathStrategy

**Files:**
- Modify: `internal/config/schema.go`

`PathStrategy` has `yaml:"-"` on both fields, so `yaml.Marshal` produces empty output. Add `MarshalYAML` to fix serialization — needed so `config.WriteFile` can write configs containing `path_strategy`.

- [ ] **Step 1: Write failing test**

In `internal/config/schema_test.go`, add:

```go
func TestPathStrategy_MarshalYAML(t *testing.T) {
	tests := []struct {
		name     string
		ps       *PathStrategy
		expected string
	}{
		{
			name:     "name form",
			ps:       &PathStrategy{Name: "sibling"},
			expected: "sibling\n",
		},
		{
			name:     "template form",
			ps:       &PathStrategy{Template: "../{{.Branch}}"},
			expected: "template: ../{{.Branch}}\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := yaml.Marshal(tt.ps)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(data))
		})
	}
}

func TestConfig_MarshalRoundTrip(t *testing.T) {
	cfg := &Config{
		MainWorktree: "/home/user/proj",
		PathStrategy: &PathStrategy{Name: "nested"},
	}
	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	cfg2, err := Parse(data)
	require.NoError(t, err)
	assert.Equal(t, "/home/user/proj", cfg2.MainWorktree)
	require.NotNil(t, cfg2.PathStrategy)
	assert.Equal(t, "nested", cfg2.PathStrategy.Name)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run "TestPathStrategy_MarshalYAML|TestConfig_MarshalRoundTrip" -v`
Expected: FAIL — `yaml:"-"` fields produce empty serialization

- [ ] **Step 3: Implement MarshalYAML**

In `internal/config/schema.go`, add right after `UnmarshalYAML`:

```go
func (p PathStrategy) MarshalYAML() (any, error) {
	if p.Template != "" {
		return map[string]string{"template": p.Template}, nil
	}
	return p.Name, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/ -run "TestPathStrategy_MarshalYAML|TestConfig_MarshalRoundTrip" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/schema.go internal/config/schema_test.go
git commit -m "fix: add MarshalYAML to PathStrategy for config round-trip"
```

---

### Task 2: Build the InitWizard Bubble Tea model

**Files:**
- Create: `internal/tui/init_wizard.go`

Build a multi-step Bubble Tea wizard. The model holds all field state and switches `Update`/`View` behavior based on the current step. Reuses `textinput.Model` from Bubbles (already a dependency).

- [ ] **Step 1: Create the file with types, constants, and presets**

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// WizardStep enumerates the pages in the init wizard.
type WizardStep int

const (
	StepMainWorktree   WizardStep = iota
	StepPathStrategy
	StepCustomTemplate
	StepEvents
	StepCustomCommand
	StepSaveVCS
	StepReview
)

// WizardResult holds the values collected by the init wizard.
type WizardResult struct {
	MainWorktree   string
	PathStrategy   string
	CustomTemplate string
	Events         []string
	SaveWithVCS    bool
	Cancelled      bool
}

// eventPresets defines the pre-configured post-create steps.
var eventPresets = []struct{ Label, Command string }{
	{"cp .env.example .env", "cp .env.example .env"},
	{"make install", "make install"},
	{"npm install", "npm install"},
	{"yarn install", "yarn install"},
	{"pnpm install", "pnpm install"},
	{"pip install -r requirements.txt", "pip install -r requirements.txt"},
	{"go mod download", "go mod download"},
	{"bundle install", "bundle install"},
}

type wizardModel struct {
	step      WizardStep
	width     int
	height    int

	mainWorktree   string
	pathStrategy   string
	customTemplate string
	customCommands []string
	saveWithVCS    bool
	cancelled      bool

	textInput textinput.Model
	cursor    int
	toggled   map[int]bool

	pathOptions []string
	vcsOptions  []string
}
```

- [ ] **Step 2: Add the RunInitWizard entry point**

```go
func RunInitWizard(detectedMainWT string) WizardResult {
	ti := textinput.New()
	ti.Placeholder = "Main worktree path"
	ti.CharLimit = 256
	ti.SetValue(detectedMainWT)
	ti.Focus()

	m := &wizardModel{
		step:         StepMainWorktree,
		mainWorktree: detectedMainWT,
		pathStrategy: "sibling",
		saveWithVCS:  true,
		textInput:    ti,
		toggled:      make(map[int]bool),
		pathOptions:  []string{"sibling", "nested", "custom"},
		vcsOptions:   []string{
			"Yes — events → .worktree.yaml, personal settings → user config",
			"No  — everything saved to user config only",
		},
	}

	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return WizardResult{Cancelled: true}
	}
	fm := final.(*wizardModel)
	if fm.cancelled {
		return WizardResult{Cancelled: true}
	}

	events := make([]string, 0)
	for i := range eventPresets {
		if fm.toggled[i] {
			events = append(events, eventPresets[i].Command)
		}
	}
	events = append(events, fm.customCommands...)

	return WizardResult{
		MainWorktree:   fm.mainWorktree,
		PathStrategy:   fm.pathStrategy,
		CustomTemplate: fm.customTemplate,
		Events:         events,
		SaveWithVCS:    fm.saveWithVCS,
	}
}
```

- [ ] **Step 3: Add Init, Update, and step-transition logic**

```go
func (m *wizardModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *wizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Always handle window size
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsm.Width
		m.height = wsm.Height
	}

	// Delegate to textInput for input-heavy steps
	if m.step == StepMainWorktree || m.step == StepCustomTemplate || m.step == StepCustomCommand {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		if _, ok := msg.(tea.KeyMsg); ok {
			return m.handleKey(msg.(tea.KeyMsg))
		}
		return m, cmd
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return m.handleKey(keyMsg)
	}
	return m, nil
}

func (m *wizardModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.cancelled = true
		return m, tea.Quit
	case "esc":
		return m.handleEscape()
	case "enter":
		return m.handleEnter()
	case "up", "ctrl+k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "ctrl+j":
		maxCursor := 0
		switch m.step {
		case StepPathStrategy:
			maxCursor = len(m.pathOptions) - 1
		case StepEvents:
			maxCursor = len(eventPresets) // presets + "+" item
		case StepSaveVCS:
			maxCursor = len(m.vcsOptions) - 1
		}
		if m.cursor < maxCursor {
			m.cursor++
		}
	case " ":
		if m.step == StepEvents && m.cursor < len(eventPresets) {
			m.toggled[m.cursor] = !m.toggled[m.cursor]
		}
	}
	return m, nil
}

func (m *wizardModel) handleEscape() (tea.Model, tea.Cmd) {
	switch m.step {
	case StepMainWorktree:
		m.cancelled = true
		return m, tea.Quit
	case StepPathStrategy, StepCustomTemplate:
		m.step = StepMainWorktree
		m.cursor = 0
	case StepEvents:
		m.step = StepPathStrategy
		m.cursor = 0
	case StepCustomCommand:
		m.step = StepEvents
		m.cursor = 0
	case StepSaveVCS:
		m.step = StepEvents
		m.cursor = 0
	case StepReview:
		m.step = StepSaveVCS
		m.cursor = 0
	}
	return m, nil
}

func (m *wizardModel) handleEnter() (tea.Model, tea.Cmd) {
	switch m.step {
	case StepMainWorktree:
		m.mainWorktree = m.textInput.Value()
		m.step = StepPathStrategy
		m.cursor = 0
	case StepPathStrategy:
		m.pathStrategy = m.pathOptions[m.cursor]
		if m.pathStrategy == "custom" {
			m.step = StepCustomTemplate
			m.textInput.SetValue("")
			m.textInput.Placeholder = "Go template (e.g. ../{{.Branch}})"
			m.textInput.Focus()
		} else {
			m.step = StepEvents
			m.cursor = 0
		}
	case StepCustomTemplate:
		m.customTemplate = m.textInput.Value()
		m.step = StepEvents
		m.cursor = 0
	case StepEvents:
		if m.cursor == len(eventPresets) {
			// "+ Add custom command..."
			m.step = StepCustomCommand
			m.textInput.SetValue("")
			m.textInput.Placeholder = "Enter custom command..."
			m.textInput.Focus()
		} else {
			m.step = StepSaveVCS
			m.cursor = 0
		}
	case StepCustomCommand:
		cmd := strings.TrimSpace(m.textInput.Value())
		if cmd != "" {
			m.customCommands = append(m.customCommands, cmd)
		}
		m.textInput.SetValue("")
		m.step = StepEvents
		m.cursor = 0
	case StepSaveVCS:
		m.saveWithVCS = (m.cursor == 0)
		m.step = StepReview
	case StepReview:
		return m, tea.Quit
	}
	return m, nil
}
```

- [ ] **Step 4: Add View methods for each step**

```go
func (m *wizardModel) View() string {
	switch m.step {
	case StepMainWorktree:
		return m.viewMainWorktree()
	case StepPathStrategy:
		return m.viewPathStrategy()
	case StepCustomTemplate:
		return m.viewCustomTemplate()
	case StepEvents:
		return m.viewEvents()
	case StepCustomCommand:
		return m.viewCustomCommand()
	case StepSaveVCS:
		return m.viewSaveVCS()
	case StepReview:
		return m.viewReview()
	}
	return ""
}

func (m *wizardModel) viewMainWorktree() string {
	s := "Main Worktree Path\n\n"
	s += "The primary working directory for this project.\n\n"
	s += m.textInput.View() + "\n\n"
	s += dim("[enter confirm | esc quit]")
	return s
}

func (m *wizardModel) viewPathStrategy() string {
	s := "Path Strategy\n\n"
	s += "How should new worktree directories be placed?\n\n"
	for i, opt := range m.pathOptions {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}
		s += fmt.Sprintf("%s %s\n", cursor, opt)
	}
	s += "\n" + dim("[enter select | esc back]")
	return s
}

func (m *wizardModel) viewCustomTemplate() string {
	s := "Path Strategy — Custom Template\n\n"
	s += "Enter a Go template for worktree paths.\n"
	s += "Available variables: {{.Branch}}, {{.ProjectName}}\n\n"
	s += m.textInput.View() + "\n\n"
	s += dim("[enter confirm | esc back]")
	return s
}

func (m *wizardModel) viewEvents() string {
	s := "Post-Create Steps\n\n"
	s += "Select steps to run after each worktree is created.\n"
	s += dim("[space] toggle  [enter] confirm") + "\n\n"
	for i, preset := range eventPresets {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}
		checked := "[ ]"
		if m.toggled[i] {
			checked = "[x]"
		}
		s += fmt.Sprintf("%s %s %s\n", cursor, checked, preset.Label)
	}
	cursor := " "
	if m.cursor == len(eventPresets) {
		cursor = ">"
	}
	s += fmt.Sprintf("%s [+] Add custom command...\n", cursor)

	selected := 0
	for _, v := range m.toggled {
		if v {
			selected++
		}
	}
	selected += len(m.customCommands)
	s += fmt.Sprintf("\n[%d selected | enter confirm | esc back]", selected)
	return s
}

func (m *wizardModel) viewCustomCommand() string {
	s := "Custom Command\n\n"
	s += "Enter a shell command to run after worktree creation.\n\n"
	s += m.textInput.View() + "\n\n"
	s += dim("[enter add | esc cancel]")
	return s
}

func (m *wizardModel) viewSaveVCS() string {
	s := "Save With VCS?\n\n"
	s += "Share event steps with your team via .worktree.yaml?\n\n"
	for i, opt := range m.vcsOptions {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}
		s += fmt.Sprintf("%s %s\n", cursor, opt)
	}
	s += "\n" + dim("[enter confirm | esc back]")
	return s
}

func (m *wizardModel) viewReview() string {
	s := "Review & Confirm\n\n"

	// Build events from toggled + custom
	events := make([]string, 0)
	for i := range eventPresets {
		if m.toggled[i] {
			events = append(events, eventPresets[i].Command)
		}
	}
	events = append(events, m.customCommands...)

	if m.saveWithVCS {
		s += "Will write to .worktree.yaml:\n"
		s += formatEventYAML(events)
		s += "\nWill write to user config:\n"
		s += fmt.Sprintf("  main_worktree: %s\n", m.mainWorktree)
		s += fmt.Sprintf("  path_strategy: %s\n", displayPathStrategy(m.pathStrategy, m.customTemplate))
	} else {
		s += "Will write to user config:\n"
		s += fmt.Sprintf("  main_worktree: %s\n", m.mainWorktree)
		s += fmt.Sprintf("  path_strategy: %s\n", displayPathStrategy(m.pathStrategy, m.customTemplate))
		if len(events) > 0 {
			s += formatEventYAML(events)
		}
	}

	s += "\n" + dim("[enter to write | esc back]")
	return s
}

func formatEventYAML(events []string) string {
	if len(events) == 0 {
		return "  on:\n    post-create:\n      run: []\n"
	}
	s := "  on:\n    post-create:\n      steps:\n"
	for _, ev := range events {
		s += fmt.Sprintf("        - run: %s\n", ev)
	}
	return s
}

func displayPathStrategy(strategy, customTemplate string) string {
	if strategy == "custom" {
		return fmt.Sprintf("template: %s", customTemplate)
	}
	return strategy
}

func dim(s string) string {
	return fmt.Sprintf("\033[2m%s\033[0m", s)
}
```

- [ ] **Step 5: Verify it compiles**

Run: `go build ./internal/tui/`
Expected: no errors

- [ ] **Step 6: Commit**

```bash
git add internal/tui/init_wizard.go
git commit -m "feat: add InitWizard Bubble Tea model for interactive wt init"
```

---

### Task 3: Add InitWizard unit tests

**Files:**
- Create: `internal/tui/init_wizard_test.go`

Test step progression, default values, ESC cancellation, event toggling, and result construction. These test the Bubble Tea model directly.

- [ ] **Step 1: Write step progression tests**

```go
package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWizard_Defaults(t *testing.T) {
	m := newWizardModel("/home/user/proj")
	assert.Equal(t, StepMainWorktree, m.step)
	assert.Equal(t, "/home/user/proj", m.mainWorktree)
	assert.Equal(t, "sibling", m.pathStrategy)
	assert.Equal(t, true, m.saveWithVCS)
}

func TestWizard_StepProgression_MainWorktreeToPathStrategy(t *testing.T) {
	m := newWizardModel("/home/user/proj")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, StepPathStrategy, m.step)
	assert.Equal(t, 0, m.cursor)
}

func TestWizard_StepProgression_PathStrategyToEvents(t *testing.T) {
	m := newWizardModel("/home/user/proj")
	m.step = StepPathStrategy
	// cursor=0 → "sibling"
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, StepEvents, m.step)
	assert.Equal(t, "sibling", m.pathStrategy)
}

func TestWizard_StepProgression_CustomPathStrategy(t *testing.T) {
	m := newWizardModel("/home/user/proj")
	m.step = StepPathStrategy
	m.cursor = 2 // "custom"
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, StepCustomTemplate, m.step)
	assert.Equal(t, "custom", m.pathStrategy)
}

func TestWizard_StepProgression_CustomTemplateToEvents(t *testing.T) {
	m := newWizardModel("/home/user/proj")
	m.step = StepCustomTemplate
	m.textInput.SetValue("../{{.Branch}}")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, StepEvents, m.step)
	assert.Equal(t, "../{{.Branch}}", m.customTemplate)
}

func TestWizard_StepProgression_EventsToSaveVCS(t *testing.T) {
	m := newWizardModel("/home/user/proj")
	m.step = StepEvents
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, StepSaveVCS, m.step)
	assert.Equal(t, 0, m.cursor)
}

func TestWizard_StepProgression_SaveVCSToReview(t *testing.T) {
	m := newWizardModel("/home/user/proj")
	m.step = StepSaveVCS
	m.cursor = 1 // "No"
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, StepReview, m.step)
	assert.Equal(t, false, m.saveWithVCS) // cursor 1 = No
}

func TestWizard_StepProgression_ReviewQuits(t *testing.T) {
	m := newWizardModel("/home/user/proj")
	m.step = StepReview
	final, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, tea.Quit(), cmd())
	fm := final.(*wizardModel)
	assert.False(t, fm.cancelled)
}
```

- [ ] **Step 2: Write ESC cancellation tests**

```go
func TestWizard_EscAtMainWorktree_Cancels(t *testing.T) {
	m := newWizardModel("/home/user/proj")
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.True(t, m.cancelled)
}

func TestWizard_EscGoesBack(t *testing.T) {
	m := newWizardModel("/home/user/proj")
	m.step = StepPathStrategy
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, StepMainWorktree, m.step)

	m.step = StepEvents
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, StepPathStrategy, m.step)

	m.step = StepSaveVCS
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, StepEvents, m.step)

	m.step = StepReview
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, StepSaveVCS, m.step)
}

func TestWizard_CtrlC_Cancels(t *testing.T) {
	m := newWizardModel("/home/user/proj")
	m.step = StepEvents
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	assert.True(t, m.cancelled)
}
```

- [ ] **Step 3: Write event toggling tests**

```go
func TestWizard_EventToggle(t *testing.T) {
	m := newWizardModel("/home/user/proj")
	m.step = StepEvents
	assert.False(t, m.toggled[0])

	// Space toggles item at cursor
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	assert.True(t, m.toggled[0])

	// Space again toggles off
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	assert.False(t, m.toggled[0])
}

func TestWizard_EventCursorNavigation(t *testing.T) {
	m := newWizardModel("/home/user/proj")
	m.step = StepEvents
	assert.Equal(t, 0, m.cursor)

	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m.cursor)

	m.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 0, m.cursor)

	// Cursor can't go below 0
	m.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 0, m.cursor)
}

func TestWizard_AddCustomCommand(t *testing.T) {
	m := newWizardModel("/home/user/proj")
	m.step = StepEvents
	m.cursor = len(eventPresets) // "+ Add custom command..."

	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, StepCustomCommand, m.step)

	m.textInput.SetValue("echo hello")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, StepEvents, m.step)
	assert.Contains(t, m.customCommands, "echo hello")
}

func TestWizard_AddCustomCommand_EmptyIsSkipped(t *testing.T) {
	m := newWizardModel("/home/user/proj")
	m.step = StepCustomCommand
	m.textInput.SetValue("  ")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, StepEvents, m.step)
	assert.Empty(t, m.customCommands)
}
```

- [ ] **Step 4: Write RunInitWizard result test**

```go
func newWizardModel(detectedMainWT string) *wizardModel {
	ti := textinput.New()
	ti.SetValue(detectedMainWT)
	return &wizardModel{
		step:         StepMainWorktree,
		mainWorktree: detectedMainWT,
		pathStrategy: "sibling",
		saveWithVCS:  true,
		textInput:    ti,
		toggled:      make(map[int]bool),
		pathOptions:  []string{"sibling", "nested", "custom"},
		vcsOptions: []string{
			"Yes — events → .worktree.yaml, personal settings → user config",
			"No  — everything saved to user config only",
		},
	}
}
```

Add this helper to `init_wizard.go` as an unexported function alongside `RunInitWizard` so tests in the same package can use it.

- [ ] **Step 5: Run tests**

Run: `go test ./internal/tui/ -run TestWizard -v`
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add internal/tui/init_wizard_test.go internal/tui/init_wizard.go
git commit -m "test: add InitWizard step progression and toggle tests"
```

---

### Task 4: Rewrite init_cmd.go

**Files:**
- Modify: `cmd/cli/init_cmd.go`

Rewrite the init command with pre-checks, CLI flags, branching to TUI or direct-write, and config file writing logic.

- [ ] **Step 1: Define new CLI flags and rewrite the command**

Replace the entire content of `cmd/cli/init_cmd.go`:

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/relaxtortoise/worktree-setup/internal/config"
	gitpkg "github.com/relaxtortoise/worktree-setup/internal/git"
	"github.com/relaxtortoise/worktree-setup/internal/tui"
	"github.com/spf13/cobra"
)

var (
	initMainWorktree string
	initPathStrategy string
	initNoSaveVCS    bool
	initPostCreate   []string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize .worktree.yaml and project config",
	Long: `Initialize worktree-setup configuration for the current project.

By default, runs an interactive wizard that guides you through
main_worktree, path_strategy, post-create steps, and whether to
save event configuration to .worktree.yaml (VCS).

Pass CLI flags to skip the wizard and write directly.`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVar(&initMainWorktree, "main-worktree", "", "Main worktree path")
	initCmd.Flags().StringVar(&initPathStrategy, "path-strategy", "", "Path strategy: sibling, nested, or template")
	initCmd.Flags().BoolVar(&initNoSaveVCS, "no-save-vcs", false, "Save everything to user config (disable VCS)")
	initCmd.Flags().StringArrayVar(&initPostCreate, "post-create-run", nil, "Add a post-create run step (repeatable)")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	repoDir := getRepoDir()

	// Pre-check: must be in a git repo with remote origin
	projName := projectName()
	if projName == "" {
		return fmt.Errorf("not in a git repository with a remote origin")
	}

	// Detect main worktree for default
	detectedWT, err := gitpkg.FindMainWorktree()
	if err != nil {
		detectedWT = repoDir
	}

	// Pre-check: existing files (interactive only)
	hasFlags := initMainWorktree != "" || initPathStrategy != "" || len(initPostCreate) > 0
	if !hasFlags {
		wtPath := filepath.Join(repoDir, ".worktree.yaml")
		projCfgPath := config.ProjectConfigPath(projName)
		if err := checkOverwrite(wtPath, ".worktree.yaml"); err != nil {
			return err
		}
		if err := checkOverwrite(projCfgPath, "project config"); err != nil {
			return err
		}
	}

	var result tui.WizardResult

	if hasFlags {
		// Non-interactive mode: build result from flags
		result = tui.WizardResult{
			MainWorktree: initMainWorktree,
			PathStrategy: initPathStrategy,
			Events:       initPostCreate,
			SaveWithVCS:  !initNoSaveVCS,
		}
		// Apply defaults for unspecified values
		if result.MainWorktree == "" {
			result.MainWorktree = detectedWT
		}
		if result.PathStrategy == "" {
			result.PathStrategy = "sibling"
		}
	} else {
		// Interactive mode: launch TUI wizard
		result = tui.RunInitWizard(detectedWT)
		if result.Cancelled {
			fmt.Println("cancelled")
			return nil
		}
	}

	// Write config files
	return writeInitConfig(repoDir, projName, result)
}

func checkOverwrite(path, label string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	fmt.Printf("%s already exists. Overwrite? [y/N]: ", label)
	var answer string
	fmt.Scanln(&answer)
	if strings.ToLower(strings.TrimSpace(answer)) != "y" {
		fmt.Printf("skipping %s\n", label)
		return nil
	}
	return nil
}

func writeInitConfig(repoDir, projName string, r tui.WizardResult) error {
	wtPath := filepath.Join(repoDir, ".worktree.yaml")
	projDir := config.ProjectConfigDir(projName)
	projCfgPath := filepath.Join(projDir, "config.yaml")

	if err := os.MkdirAll(projDir, 0755); err != nil {
		return err
	}

	if r.SaveWithVCS {
		// .worktree.yaml ← only events
		if len(r.Events) > 0 {
			wtCfg := &config.Config{
				On: &config.Events{
					PostCreate: &config.Event{},
				},
			}
			for _, ev := range r.Events {
				wtCfg.On.PostCreate.Steps = append(wtCfg.On.PostCreate.Steps, config.Step{Run: ev})
			}
			if err := config.WriteFile(wtPath, wtCfg); err != nil {
				return err
			}
			fmt.Printf("created %s\n", wtPath)
		}

		// project config ← main_worktree + path_strategy
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("main_worktree: %s\n", r.MainWorktree))
		sb.WriteString(formatPathStrategy(r.PathStrategy, r.CustomTemplate))
		if err := os.WriteFile(projCfgPath, []byte(sb.String()), 0644); err != nil {
			return err
		}
		fmt.Printf("created %s\n", projCfgPath)
	} else {
		// Everything → project config
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("main_worktree: %s\n", r.MainWorktree))
		sb.WriteString(formatPathStrategy(r.PathStrategy, r.CustomTemplate))
		if len(r.Events) > 0 {
			sb.WriteString("on:\n  post-create:\n    steps:\n")
			for _, ev := range r.Events {
				sb.WriteString(fmt.Sprintf("      - run: %s\n", ev))
			}
		}
		if err := os.WriteFile(projCfgPath, []byte(sb.String()), 0644); err != nil {
			return err
		}
		fmt.Printf("created %s\n", projCfgPath)
	}

	return nil
}

func formatPathStrategy(strategy, customTemplate string) string {
	if strategy == "custom" {
		return fmt.Sprintf("path_strategy:\n  template: %s\n", customTemplate)
	}
	if strategy == "" {
		strategy = "sibling"
	}
	return fmt.Sprintf("path_strategy: %s\n", strategy)
}
```

- [ ] **Step 2: Remove the package-level `noGitignore` variable**

The old `noGitignore` flag is no longer used. Remove its declaration and the `init()` function that registered it.

- [ ] **Step 3: Verify it compiles**

Run: `go build ./cmd/cli/`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add cmd/cli/init_cmd.go
git commit -m "feat: rewrite wt init with interactive wizard and CLI flags"
```

---

### Task 5: Update CLI tests for new init behavior

**Files:**
- Modify: `cmd/cli/cli_test.go`

Update existing init tests and add new ones for the interactive/non-interactive modes.

- [ ] **Step 1: Add `initNoSaveVCS` and `initPostCreate` to `resetFlags`**

In `cmd/cli/cli_test.go`, in `resetFlags()`, add:

```go
initNoSaveVCS = false
initPostCreate = nil
initMainWorktree = ""
initPathStrategy = ""
```

- [ ] **Step 2: Rewrite existing TestInitCmd to test non-interactive mode**

Replace `TestInitCmd`:

```go
func TestInitCmd(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	initGitRepo(t, dir)
	runGitCmd(t, dir, "remote", "add", "origin", "https://github.com/owner/repo.git")
	t.Setenv("HOME", t.TempDir())
	defer chdir(t, dir)()

	out, _, err := executeCommand("init",
		"--main-worktree", "/tmp/main",
		"--path-strategy", "nested",
		"--post-create-run", "make install",
	)
	require.NoError(t, err)
	assert.Contains(t, out, "created")
	require.FileExists(t, filepath.Join(dir, ".worktree.yaml"))

	// Verify .worktree.yaml content
	data, err := os.ReadFile(filepath.Join(dir, ".worktree.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "make install")

	// Verify project config content
	home := os.Getenv("HOME")
	projCfg := filepath.Join(home, ".config", "worktree-setup", "projects", "github.com-owner-repo", "config.yaml")
	require.FileExists(t, projCfg)
	data, err = os.ReadFile(projCfg)
	require.NoError(t, err)
	assert.Contains(t, string(data), "/tmp/main")
	assert.Contains(t, string(data), "nested")
}
```

- [ ] **Step 3: Replace TestInitCmd_AlreadyExists**

```go
func TestInitCmd_AlreadyExists(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	initGitRepo(t, dir)
	runGitCmd(t, dir, "remote", "add", "origin", "https://github.com/owner/repo.git")
	home := t.TempDir()
	t.Setenv("HOME", home)
	defer chdir(t, dir)()

	// Pre-create .worktree.yaml
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".worktree.yaml"), []byte("existing"), 0644))

	// Pre-create project config
	projDir := filepath.Join(home, ".config", "worktree-setup", "projects", "github.com-owner-repo")
	require.NoError(t, os.MkdirAll(projDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(projDir, "config.yaml"), []byte("main_worktree: /tmp/main\n"), 0644))

	// Non-interactive mode still overwrites without prompting
	out, _, err := executeCommand("init",
		"--main-worktree", "/tmp/other",
	)
	require.NoError(t, err)
	assert.Contains(t, out, "created")
}
```

- [ ] **Step 4: Add test for --no-save-vcs**

```go
func TestInitCmd_NoSaveVCS(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	initGitRepo(t, dir)
	runGitCmd(t, dir, "remote", "add", "origin", "https://github.com/owner/repo.git")
	home := t.TempDir()
	t.Setenv("HOME", home)
	defer chdir(t, dir)()

	out, _, err := executeCommand("init",
		"--main-worktree", "/tmp/main",
		"--no-save-vcs",
		"--post-create-run", "echo hello",
	)
	require.NoError(t, err)
	assert.Contains(t, out, "created")

	// .worktree.yaml should NOT exist
	assert.NoFileExists(t, filepath.Join(dir, ".worktree.yaml"))

	// Project config should have everything
	projCfg := filepath.Join(home, ".config", "worktree-setup", "projects", "github.com-owner-repo", "config.yaml")
	data, err := os.ReadFile(projCfg)
	require.NoError(t, err)
	assert.Contains(t, string(data), "main_worktree")
	assert.Contains(t, string(data), "echo hello")
}
```

- [ ] **Step 5: Add test for --post-create-run (repeatable)**

```go
func TestInitCmd_MultiplePostCreateRun(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	initGitRepo(t, dir)
	runGitCmd(t, dir, "remote", "add", "origin", "https://github.com/owner/repo.git")
	t.Setenv("HOME", t.TempDir())
	defer chdir(t, dir)()

	_, _, err := executeCommand("init",
		"--main-worktree", "/tmp/main",
		"--post-create-run", "make install",
		"--post-create-run", "npm install",
	)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, ".worktree.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "make install")
	assert.Contains(t, string(data), "npm install")
}
```

- [ ] **Step 6: Update TestInitCmd_NoGitRepo**

The test stays the same but requires the variables in resetFlags. No code changes needed beyond Step 1.

- [ ] **Step 7: Update TestInitCmd_FindMainWorktreeFallback**

```go
func TestInitCmd_FindMainWorktreeFallback(t *testing.T) {
	resetFlags()
	dir := t.TempDir()
	runGitCmd(t, dir, "init", "--bare")
	runGitCmd(t, dir, "remote", "add", "origin", "https://github.com/owner/repo.git")
	t.Setenv("HOME", t.TempDir())
	defer chdir(t, dir)()

	// Non-interactive mode with bare repo
	out, _, err := executeCommand("init",
		"--path-strategy", "sibling",
	)
	require.NoError(t, err)
	assert.Contains(t, out, "created")
}
```

- [ ] **Step 8: Run tests**

Run: `go test ./cmd/cli/ -run TestInit -v`
Expected: all PASS

- [ ] **Step 9: Commit**

```bash
git add cmd/cli/cli_test.go
git commit -m "test: update init CLI tests for interactive/non-interactive modes"
```

---

### Task 6: Integration verification

**Files:** None (verification only)

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -v`
Expected: all PASS

- [ ] **Step 2: Run golangci-lint**

Run: `golangci-lint run ./...`
Expected: no new issues

- [ ] **Step 3: Build the binary**

Run: `go build -o /tmp/wt ./cmd/cli/`
Expected: no errors, binary produced

- [ ] **Step 4: Manual smoke test (in a test repo)**

```bash
cd /tmp
mkdir wt-test && cd wt-test
git init -b main
git config user.email "test@test"
git config user.name "test"
git commit --allow-empty -m "init"
git remote add origin https://github.com/test/test.git
/tmp/wt init --main-worktree /tmp/wt-test --path-strategy sibling --post-create-run "echo hello"
cat .worktree.yaml
cat ~/.config/worktree-setup/projects/github.com-test-test/config.yaml
```

Expected: `.worktree.yaml` contains `echo hello`, project config contains `main_worktree` and `path_strategy`.

- [ ] **Step 5: Clean up and report**

```bash
rm -rf /tmp/wt-test /tmp/wt
rm -rf ~/.config/worktree-setup/projects/github.com-test-test
```
