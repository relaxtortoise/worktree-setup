package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/relaxtortoise/worktree-setup/internal/git"
)

type SelectItem struct {
	Name    string
	Detail  string
	Payload any
}

type SelectType int

const (
	SelectBranch SelectType = iota
	SelectWorktree
)

type model struct {
	textInput textinput.Model
	items     []SelectItem
	filtered  []int
	cursor    int
	selType   SelectType
	quitting  bool
}

func RunBranchSelector(autoFetch bool) (string, error) {
	if autoFetch {
		_ = git.FetchOrigin()
	}
	branches, err := git.ListRemoteBranches()
	if err != nil {
		return "", err
	}
	var items []SelectItem
	for _, b := range branches {
		detail := fmt.Sprintf("%s  %s", formatTime(b.LastCommit), b.Author)
		items = append(items, SelectItem{
			Name:    "origin/" + b.Name,
			Detail:  detail,
			Payload: b,
		})
	}
	p := tea.NewProgram(initialModel(items, SelectBranch))
	m, err := p.Run()
	if err != nil {
		return "", err
	}
	if m, ok := m.(model); ok && !m.quitting {
		if len(m.filtered) > 0 {
			return m.items[m.filtered[m.cursor]].Payload.(git.Branch).Name, nil
		}
	}
	return "", fmt.Errorf("cancelled")
}

func RunWorktreeSelector(knownProjects []struct{ Name, Path string }) (string, error) {
	var items []SelectItem
	for _, proj := range knownProjects {
		wts, err := git.ListWorktrees()
		if err != nil {
			continue
		}
		for _, wt := range wts {
			if wt.Bare {
				continue
			}
			name := proj.Name + "/" + strings.TrimPrefix(wt.Branch, "refs/heads/")
			items = append(items, SelectItem{
				Name:    name,
				Detail:  wt.Path,
				Payload: wt,
			})
		}
	}
	p := tea.NewProgram(initialModel(items, SelectWorktree))
	m, err := p.Run()
	if err != nil {
		return "", err
	}
	if m, ok := m.(model); ok && !m.quitting {
		if len(m.filtered) > 0 {
			return m.items[m.filtered[m.cursor]].Payload.(git.Worktree).Path, nil
		}
	}
	return "", fmt.Errorf("cancelled")
}

func initialModel(items []SelectItem, selType SelectType) model {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.Focus()
	ti.CharLimit = 80

	filtered := make([]int, len(items))
	for i := range items {
		filtered[i] = i
	}

	return model{
		textInput: ti,
		items:     items,
		filtered:  filtered,
		selType:   selType,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			return m, tea.Quit
		case "up", "ctrl+k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "ctrl+j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)

	query := strings.ToLower(m.textInput.Value())
	m.filtered = nil
	for i, item := range m.items {
		if query == "" || strings.Contains(strings.ToLower(item.Name), query) {
			m.filtered = append(m.filtered, i)
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}

	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	s := m.textInput.View() + "\n\n"
	for i, idx := range m.filtered {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}
		item := m.items[idx]
		s += fmt.Sprintf("%s %s  %s\n", cursor, item.Name, item.Detail)
	}

	s += fmt.Sprintf("\n%d matches | enter select | esc quit", len(m.filtered))
	return s
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%dw ago", int(d.Hours()/24/7))
	}
}
