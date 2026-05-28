package config

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestPrintValue(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		key  string
		want string
	}{
		{
			name: "main worktree",
			cfg:  &Config{MainWorktree: "/home/me/projects/app"},
			key:  "main_worktree",
			want: "/home/me/projects/app\n",
		},
		{
			name: "path strategy name",
			cfg:  &Config{PathStrategy: &PathStrategy{Name: "sibling"}},
			key:  "path_strategy",
			want: "sibling\n",
		},
		{
			name: "path strategy template",
			cfg:  &Config{PathStrategy: &PathStrategy{Template: "/data/{branch}"}},
			key:  "path_strategy",
			want: "template: /data/{branch}\n",
		},
		{
			name: "unknown key",
			cfg:  &Config{},
			key:  "unknown",
			want: "",
		},
		{
			name: "nil path strategy",
			cfg:  &Config{},
			key:  "path_strategy",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := captureStdout(func() { PrintValue(tt.cfg, tt.key) })
			assert.Equal(t, tt.want, out)
		})
	}
}

func TestSetValue(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
		want  *Config
	}{
		{
			name:  "main worktree",
			key:   "main_worktree",
			value: "/home/me/app",
			want:  &Config{MainWorktree: "/home/me/app"},
		},
		{
			name:  "path strategy",
			key:   "path_strategy",
			value: "nested",
			want:  &Config{PathStrategy: &PathStrategy{Name: "nested"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{}
			SetValue(cfg, tt.key, tt.value)
			assert.Equal(t, tt.want, cfg)
		})
	}
}

func TestPrintFile(t *testing.T) {
	t.Run("existing file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")
		os.WriteFile(path, []byte("main_worktree: /app\n"), 0644)

		out := captureStdout(func() { PrintFile(path) })
		assert.Equal(t, "main_worktree: /app\n", out)
	})

	t.Run("missing file", func(t *testing.T) {
		out := captureStdout(func() {
			PrintFile("/nonexistent/path/config.yaml")
		})
		assert.Equal(t, "(no config)\n", out)
	})
}

func TestWriteFile(t *testing.T) {
	t.Run("write and read back", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")

		cfg := &Config{
			MainWorktree: "/home/me/app",
			PathStrategy: &PathStrategy{Name: "sibling"},
		}

		err := WriteFile(path, cfg)
		require.NoError(t, err)

		// Read back to verify
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Contains(t, string(data), "main_worktree: /home/me/app")
		// Parse back and verify round-trip (PathStrategy is lossy due to yaml:"-" tags)
		got := &Config{}
		err = yaml.Unmarshal(data, got)
		require.NoError(t, err)
		assert.Equal(t, "/home/me/app", got.MainWorktree)
	})

	t.Run("overwrite existing", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")
		os.WriteFile(path, []byte("old: true\n"), 0644)

		cfg := &Config{MainWorktree: "/new"}
		err := WriteFile(path, cfg)
		require.NoError(t, err)

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Contains(t, string(data), "main_worktree: /new")
		assert.NotContains(t, string(data), "old")
	})
}
