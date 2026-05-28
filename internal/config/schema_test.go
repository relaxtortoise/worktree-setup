package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPathStrategy_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want PathStrategy
	}{
		{
			name: "sibling string",
			yaml: "sibling",
			want: PathStrategy{Name: "sibling"},
		},
		{
			name: "nested string",
			yaml: "nested",
			want: PathStrategy{Name: "nested"},
		},
		{
			name: "home string",
			yaml: "home",
			want: PathStrategy{Name: "home"},
		},
		{
			name: "custom template",
			yaml: "template: /data/worktrees/{project_name}/{branch}",
			want: PathStrategy{Template: "/data/worktrees/{project_name}/{branch}"},
		},
		{
			name: "empty template",
			yaml: "template: ''",
			want: PathStrategy{Template: ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ps PathStrategy
			err := yaml.Unmarshal([]byte(tt.yaml), &ps)
			require.NoError(t, err)
			assert.Equal(t, tt.want, ps)
		})
	}
}

func TestStep_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want Step
	}{
		{
			name: "string becomes implicit run",
			yaml: `"echo hello"`,
			want: Step{Run: "echo hello"},
		},
		{
			name: "object with run",
			yaml: `run: "echo hello"`,
			want: Step{Run: "echo hello"},
		},
		{
			name: "object with copy",
			yaml: `copy:
  "a.txt": "b.txt"`,
			want: Step{Copy: &CopyItems{Items: []CopyAction{{From: "a.txt", To: "b.txt"}}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s Step
			err := yaml.Unmarshal([]byte(tt.yaml), &s)
			require.NoError(t, err)
			assert.Equal(t, tt.want, s)
		})
	}
}

func TestCopyItems_UnmarshalYAML_EmptyList(t *testing.T) {
	var ci CopyItems
	err := yaml.Unmarshal([]byte("[]"), &ci)
	require.NoError(t, err)
	assert.Nil(t, ci.Items)
}

func TestCopyItems_UnmarshalYAML_InvalidElement(t *testing.T) {
	var ci CopyItems
	// [123] is valid YAML; the int element doesn't match string or map
	// so it is silently skipped, leaving Items nil.
	err := yaml.Unmarshal([]byte("[123]"), &ci)
	require.NoError(t, err)
	assert.Nil(t, ci.Items)
}

func TestCopyItems_UnmarshalYAML_MapForm(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want []CopyAction
	}{
		{
			name: "simple map",
			yaml: `".env.example": ".env"`,
			want: []CopyAction{{From: ".env.example", To: ".env"}},
		},
		{
			name: "same from and to",
			yaml: `"go.mod": ""`,
			want: []CopyAction{{From: "go.mod", To: "go.mod"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ci CopyItems
			err := yaml.Unmarshal([]byte(tt.yaml), &ci)
			require.NoError(t, err)
			assert.Equal(t, tt.want, ci.Items)
		})
	}
}

func TestEvent_StepsOrLegacy(t *testing.T) {
	tests := []struct {
		name  string
		event *Event
		want  int // number of steps expected
	}{
		{
			name:  "empty event",
			event: &Event{},
			want:  0,
		},
		{
			name: "steps take priority",
			event: &Event{
				Steps: []Step{{Run: "step1"}},
				Run:   []string{"legacy"},
			},
			want: 1,
		},
		{
			name: "legacy copy only",
			event: &Event{
				Copy: &CopyItems{Items: []CopyAction{{From: "a", To: "b"}}},
			},
			want: 1,
		},
		{
			name: "legacy symlink only",
			event: &Event{
				Symlink: &CopyItems{Items: []CopyAction{{From: "a", To: "b"}}},
			},
			want: 1,
		},
		{
			name: "legacy run only",
			event: &Event{
				Run: []string{"cmd1", "cmd2"},
			},
			want: 1,
		},
		{
			name: "legacy all three",
			event: &Event{
				Copy:    &CopyItems{Items: []CopyAction{{From: "a", To: "b"}}},
				Symlink: &CopyItems{Items: []CopyAction{{From: "c", To: "d"}}},
				Run:     []string{"cmd1"},
			},
			want: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.event.StepsOrLegacy()
			assert.Len(t, got, tt.want)
		})
	}
}

func TestParseColonShorthand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantFrom string
		wantTo   string
	}{
		{
			name:     "with colon",
			input:    ".env.example:.env",
			wantFrom: ".env.example",
			wantTo:   ".env",
		},
		{
			name:     "without colon",
			input:    "go.mod",
			wantFrom: "go.mod",
			wantTo:   "go.mod",
		},
		{
			name:     "path with multiple colons",
			input:    "a:b:c",
			wantFrom: "a",
			wantTo:   "b:c",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, to := parseColonShorthand(tt.input)
			assert.Equal(t, tt.wantFrom, from)
			assert.Equal(t, tt.wantTo, to)
		})
	}
}
