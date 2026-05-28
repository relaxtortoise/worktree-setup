package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWorktreeYAML_MapForm(t *testing.T) {
	yaml := `
main_worktree: /home/me/projects/myapp
path_strategy: sibling
on:
  post-create:
    run:
      - "go mod download"
    copy:
      ".env.example": ".env"
    symlink:
      "../main/node_modules": "node_modules"
`
	dir := t.TempDir()
	path := filepath.Join(dir, ".worktree.yaml")
	_ = os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MainWorktree != "/home/me/projects/myapp" {
		t.Errorf("MainWorktree = %q", cfg.MainWorktree)
	}
	if cfg.On.PostCreate == nil {
		t.Fatal("post-create is nil")
	}
	if len(cfg.On.PostCreate.Run) != 1 {
		t.Errorf("run count = %d", len(cfg.On.PostCreate.Run))
	}
	if len(cfg.On.PostCreate.Copy.Items) != 1 {
		t.Errorf("copy items = %d", len(cfg.On.PostCreate.Copy.Items))
	}
	if len(cfg.On.PostCreate.Symlink.Items) != 1 {
		t.Errorf("symlink items = %d", len(cfg.On.PostCreate.Symlink.Items))
	}
}

func TestParseWorktreeYAML_StepsWithImplicitRun(t *testing.T) {
	yaml := `
on:
  post-create:
    steps:
      - "make generate"
      - copy:
          "output/bundle.js": "public/bundle.js"
      - symlink:
          "../main/vendor": "vendor"
      - "go build ./..."
`
	dir := t.TempDir()
	path := filepath.Join(dir, ".worktree.yaml")
	_ = os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	steps := cfg.On.PostCreate.Steps
	if len(steps) != 4 {
		t.Fatalf("expected 4 steps, got %d", len(steps))
	}
	if steps[0].Run != "make generate" {
		t.Errorf("step 0 Run = %q", steps[0].Run)
	}
	if steps[1].Copy == nil || len(steps[1].Copy.Items) != 1 {
		t.Error("step 1 copy missing")
	}
	if steps[2].Symlink == nil || len(steps[2].Symlink.Items) != 1 {
		t.Error("step 2 symlink missing")
	}
	if steps[3].Run != "go build ./..." {
		t.Errorf("step 3 Run = %q", steps[3].Run)
	}
}

func TestParseWorktreeYAML_ListForm(t *testing.T) {
	yaml := `
on:
  post-create:
    copy:
      - "go.mod"
      - ".env.example:.env"
      - from: "scripts/hooks.sh"
        to: ".git/hooks/pre-commit"
`
	dir := t.TempDir()
	path := filepath.Join(dir, ".worktree.yaml")
	_ = os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	items := cfg.On.PostCreate.Copy.Items
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[0].From != "go.mod" || items[0].To != "go.mod" {
		t.Errorf("string item: %+v", items[0])
	}
	if items[1].From != ".env.example" || items[1].To != ".env" {
		t.Errorf("colon item: %+v", items[1])
	}
	if items[2].From != "scripts/hooks.sh" || items[2].To != ".git/hooks/pre-commit" {
		t.Errorf("object item: %+v", items[2])
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    *Config
		wantErr bool
	}{
		{
			name:  "valid yaml",
			input: []byte("main_worktree: /home/me/projects/app\npath_strategy: sibling\n"),
			want: &Config{
				MainWorktree: "/home/me/projects/app",
				PathStrategy: &PathStrategy{Name: "sibling"},
			},
		},
		{
			name:    "invalid yaml",
			input:   []byte("{{{invalid"),
			wantErr: true,
		},
		{
			name:  "empty input",
			input: []byte{},
			want:  &Config{},
		},
		{
			name: "only on section",
			input: []byte(`on:
  post-create:
    run:
      - "echo hello"
`),
			want: &Config{
				On: &Events{
					PostCreate: &Event{
						Run: []string{"echo hello"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseFile_ErrorPaths(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr bool
	}{
		{
			name: "file not found",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent.yaml")
			},
			wantErr: true,
		},
		{
			name: "invalid yaml content",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, ".worktree.yaml")
				os.WriteFile(path, []byte("{{{bad"), 0644)
				return path
			},
			wantErr: true,
		},
		{
			name: "empty file is valid",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, ".worktree.yaml")
				os.WriteFile(path, []byte{}, 0644)
				return path
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			_, err := ParseFile(path)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
