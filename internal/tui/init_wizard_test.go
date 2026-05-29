package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// --- Step progression ---

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
	assert.Equal(t, false, m.saveWithVCS)
}

func TestWizard_StepProgression_ReviewQuits(t *testing.T) {
	m := newWizardModel("/home/user/proj")
	m.step = StepReview
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotNil(t, cmd)
}

// --- ESC cancellation ---

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

	m.step = StepCustomTemplate
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, StepPathStrategy, m.step)

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

// --- Event toggling ---

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
