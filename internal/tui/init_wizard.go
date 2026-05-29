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

// RunInitWizard starts the interactive init wizard and returns the collected result.
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
		vcsOptions: []string{
			"Yes — events → .worktree.yaml, personal settings → user config",
			"No  — everything saved to user config only",
		},
	}

	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return WizardResult{Cancelled: true}
	}
	fm, ok := final.(*wizardModel)
	if !ok || fm.cancelled {
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

// newWizardModel creates a wizardModel for testing.
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
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			m2, cmd2 := m.handleKey(keyMsg)
			return m2, tea.Batch(cmd, cmd2)
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
	case StepPathStrategy:
		m.step = StepMainWorktree
		m.cursor = 0
	case StepCustomTemplate:
		m.step = StepPathStrategy
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
		m.mainWorktree = strings.TrimSpace(m.textInput.Value())
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
