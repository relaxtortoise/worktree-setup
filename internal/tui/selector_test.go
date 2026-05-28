package tui

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/relaxtortoise/worktree-setup/internal/git"
)

func TestFormatTime(t *testing.T) {
	tests := []struct {
		name string
		t    time.Time
		want string
	}{
		{
			name: "zero time",
			t:    time.Time{},
			want: "unknown",
		},
		{
			name: "just now",
			t:    time.Now().Add(-30 * time.Second),
			want: "just now",
		},
		{
			name: "minutes ago",
			t:    time.Now().Add(-5 * time.Minute),
			want: "5m ago",
		},
		{
			name: "hours ago",
			t:    time.Now().Add(-3 * time.Hour),
			want: "3h ago",
		},
		{
			name: "days ago",
			t:    time.Now().Add(-48 * time.Hour),
			want: "2d ago",
		},
		{
			name: "weeks ago",
			t:    time.Now().Add(-14 * 24 * time.Hour),
			want: "2w ago",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTime(tt.t)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInitialModel(t *testing.T) {
	items := []SelectItem{
		{Name: "item1", Detail: "detail1"},
		{Name: "item2", Detail: "detail2"},
	}
	m := initialModel(items, SelectBranch)

	assert.Equal(t, len(items), len(m.items))
	assert.Equal(t, len(items), len(m.filtered))
	assert.Equal(t, SelectBranch, m.selType)
	assert.Equal(t, 0, m.cursor)
	assert.Equal(t, "Search...", m.textInput.Placeholder)
	assert.True(t, m.textInput.Focused())
}

func TestModel_Init(t *testing.T) {
	m := initialModel([]SelectItem{{Name: "test"}}, SelectBranch)
	cmd := m.Init()
	assert.NotNil(t, cmd)
}

// --- Quit / Cancel key messages ---

func TestModel_Update_Esc(t *testing.T) {
	m := initialModel([]SelectItem{{Name: "test"}}, SelectBranch)
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.True(t, newModel.(model).quitting)
	assert.NotNil(t, cmd)
}

func TestModel_Update_CtrlC(t *testing.T) {
	m := initialModel([]SelectItem{{Name: "test"}}, SelectBranch)
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	assert.True(t, newModel.(model).quitting)
	assert.NotNil(t, cmd)
}

func TestModel_Update_Enter(t *testing.T) {
	m := initialModel([]SelectItem{{Name: "test"}}, SelectBranch)
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotNil(t, cmd)
	assert.False(t, newModel.(model).quitting)
	assert.Equal(t, 0, newModel.(model).cursor)
}

// --- Cursor movement ---

func TestModel_Update_Down(t *testing.T) {
	items := []SelectItem{
		{Name: "a"}, {Name: "b"}, {Name: "c"},
	}
	m := initialModel(items, SelectBranch)

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m2.(model).cursor)

	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 2, m3.(model).cursor)

	// At bottom — should not go past last item
	m4, _ := m3.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 2, m4.(model).cursor)
}

func TestModel_Update_Up(t *testing.T) {
	items := []SelectItem{
		{Name: "a"}, {Name: "b"}, {Name: "c"},
	}
	m := initialModel(items, SelectBranch)
	m.cursor = 2

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 1, m2.(model).cursor)

	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 0, m3.(model).cursor)

	// At top — should not go above 0
	m4, _ := m3.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 0, m4.(model).cursor)
}

func TestModel_Update_Up_EmptyFiltered(t *testing.T) {
	m := initialModel([]SelectItem{}, SelectBranch)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 0, m2.(model).cursor)
}

func TestModel_Update_Down_EmptyFiltered(t *testing.T) {
	m := initialModel([]SelectItem{}, SelectBranch)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 0, m2.(model).cursor)
}

// --- Keyboard shortcuts (ctrl+k = up, ctrl+j = down) ---

func TestModel_Update_CtrlK(t *testing.T) {
	items := []SelectItem{
		{Name: "a"}, {Name: "b"}, {Name: "c"},
	}
	m := initialModel(items, SelectBranch)
	m.cursor = 2

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	assert.Equal(t, 1, m2.(model).cursor)
}

func TestModel_Update_CtrlJ(t *testing.T) {
	items := []SelectItem{
		{Name: "a"}, {Name: "b"}, {Name: "c"},
	}
	m := initialModel(items, SelectBranch)

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
	assert.Equal(t, 1, m2.(model).cursor)
}

// --- Filtering ---

func TestModel_Update_Filter(t *testing.T) {
	items := []SelectItem{
		{Name: "apple"},
		{Name: "banana"},
		{Name: "apricot"},
	}
	m := initialModel(items, SelectBranch)
	m.textInput.SetValue("ap")
	// send a benign message to trigger re-filtering
	m2, _ := m.Update(nil)
	assert.Equal(t, 2, len(m2.(model).filtered))
}

func TestModel_Update_FilterEmptyQuery(t *testing.T) {
	items := []SelectItem{
		{Name: "apple"},
		{Name: "banana"},
	}
	m := initialModel(items, SelectBranch)
	m.textInput.SetValue("")
	m2, _ := m.Update(nil)
	assert.Equal(t, 2, len(m2.(model).filtered))
}

func TestModel_Update_FilterNoMatch(t *testing.T) {
	items := []SelectItem{
		{Name: "apple"},
		{Name: "banana"},
	}
	m := initialModel(items, SelectBranch)
	m.textInput.SetValue("zzz_nomatch")
	m2, _ := m.Update(nil)
	assert.Equal(t, 0, len(m2.(model).filtered))
}

func TestModel_Update_FilterCursorClamp(t *testing.T) {
	items := []SelectItem{
		{Name: "apple"},
		{Name: "banana"},
		{Name: "apricot"},
	}
	m := initialModel(items, SelectBranch)
	m.cursor = 2 // points to "apricot" (idx 2 in original)
	m.textInput.SetValue("ap")
	m2, _ := m.Update(nil)
	// "ap" matches "apple" (idx 0) and "apricot" (idx 2) => filtered = [0, 2]
	// cursor was 2, len(filtered) = 2, so cursor = max(0, 1) = 1
	assert.Equal(t, 2, len(m2.(model).filtered))
	assert.Equal(t, 1, m2.(model).cursor)
}

// --- View ---

func TestModel_View(t *testing.T) {
	items := []SelectItem{
		{Name: "item1", Detail: "detail1"},
	}
	m := initialModel(items, SelectBranch)
	view := m.View()
	assert.Contains(t, view, "item1")
	assert.Contains(t, view, "detail1")
	assert.Contains(t, view, ">")
	assert.Contains(t, view, "1 matches")
}

func TestModel_View_EmptyFiltered(t *testing.T) {
	items := []SelectItem{
		{Name: "apple"},
	}
	m := initialModel(items, SelectBranch)
	m.textInput.SetValue("zzz_nomatch")
	m2, _ := m.Update(nil)
	view := m2.View()
	assert.Contains(t, view, "0 matches")
}

func TestModel_View_Quitting(t *testing.T) {
	m := initialModel([]SelectItem{}, SelectBranch)
	m.quitting = true
	view := m.View()
	assert.Equal(t, "", view)
}

// --- Integration-style tests using git.CmdFn mock ---

// mockGitCmd replaces git.CmdFn with a function that returns fake git output.
// Returns a restore function to be called via defer.
func mockGitCmd(handlers map[string]string) func() {
	old := git.CmdFn
	git.CmdFn = func(name string, args ...string) *exec.Cmd {
		key := strings.Join(args, " ")
		if out, ok := handlers[key]; ok {
			return exec.Command("echo", out)
		}
		// default: succeed with empty output
		return exec.Command("echo")
	}
	return func() { git.CmdFn = old }
}

func TestRunBranchSelector(t *testing.T) {
	old := git.CmdFn
	git.CmdFn = func(name string, args ...string) *exec.Cmd {
		cmd := strings.Join(args, " ")
		if strings.Contains(cmd, "for-each-ref") {
			// Use printf to produce binary output with \x00 separators
			return exec.Command("sh", "-c",
				"printf 'origin/my-feature\\0002023-01-15 10:00:00 -0700\\000Alice\\n'")
		}
		return exec.Command("echo")
	}
	defer func() { git.CmdFn = old }()

	_, err := RunBranchSelector(false)
	assert.ErrorContains(t, err, "could not open a new TTY")
}

func TestRunBranchSelector_AutoFetch(t *testing.T) {
	old := git.CmdFn
	git.CmdFn = func(name string, args ...string) *exec.Cmd {
		cmd := strings.Join(args, " ")
		if strings.Contains(cmd, "for-each-ref") {
			return exec.Command("sh", "-c",
				"printf 'origin/main\\0002023-06-01 12:00:00 -0700\\000Bob\\n'")
		}
		return exec.Command("echo")
	}
	defer func() { git.CmdFn = old }()

	_, err := RunBranchSelector(true)
	assert.ErrorContains(t, err, "could not open a new TTY")
}

func TestRunBranchSelector_ListError(t *testing.T) {
	old := git.CmdFn
	git.CmdFn = func(name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}
	defer func() { git.CmdFn = old }()

	_, err := RunBranchSelector(false)
	// Should fail during ListRemoteBranches, before TUI starts
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "TTY")
}

func TestRunWorktreeSelector(t *testing.T) {
	restore := mockGitCmd(map[string]string{
		"worktree list --porcelain": "worktree /home/user/proj\nHEAD a1b2c3d\nbranch refs/heads/feature-x\n\n",
	})
	defer restore()

	projects := []struct{ Name, Path string }{
		{Name: "myapp", Path: "/home/user/proj"},
	}
	_, err := RunWorktreeSelector(projects)
	assert.ErrorContains(t, err, "could not open a new TTY")
}

func TestRunWorktreeSelector_NoProjects(t *testing.T) {
	restore := mockGitCmd(nil)
	defer restore()

	_, err := RunWorktreeSelector(nil)
	assert.ErrorContains(t, err, "could not open a new TTY")
}

func TestRunWorktreeSelector_ListError(t *testing.T) {
	old := git.CmdFn
	git.CmdFn = func(name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}
	defer func() { git.CmdFn = old }()

	projects := []struct{ Name, Path string }{
		{Name: "myapp", Path: "/home/user/proj"},
	}
	// worktree list fails → project skipped → no items → empty model runs, TUI errors
	_, err := RunWorktreeSelector(projects)
	assert.ErrorContains(t, err, "could not open a new TTY")
}
