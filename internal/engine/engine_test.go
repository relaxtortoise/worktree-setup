package engine

import (
	"path/filepath"
	"testing"

	"github.com/relaxtortoise/worktree-setup/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	e := New("/home/me/app")
	assert.NotNil(t, e)
	assert.NotNil(t, e.Runner)
	assert.Equal(t, "/home/me/app", e.Runner.MainWorktree)
}

func TestRunPreCreate(t *testing.T) {
	dir := t.TempDir()
	e := New(dir)
	cfg := &config.Config{
		On: &config.Events{
			PreCreate: &config.Event{
				Run: []string{"echo precreate > " + filepath.Join(dir, "out.txt")},
			},
		},
	}
	err := e.RunPreCreate(cfg)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, "out.txt"))
}

func TestRunPreCreate_NilOn(t *testing.T) {
	e := New("/tmp")
	err := e.RunPreCreate(&config.Config{})
	require.NoError(t, err)
}

func TestRunPostCreate(t *testing.T) {
	dir := t.TempDir()
	e := New(dir)
	cfg := &config.Config{
		On: &config.Events{
			PostCreate: &config.Event{
				Run: []string{"echo postcreate > " + filepath.Join(dir, "out.txt")},
			},
		},
	}
	err := e.RunPostCreate(cfg, dir)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, "out.txt"))
}

func TestRunPostCreate_NilOn(t *testing.T) {
	e := New("/tmp")
	err := e.RunPostCreate(&config.Config{}, "/tmp")
	require.NoError(t, err)
}

func TestRunPostCheckout(t *testing.T) {
	dir := t.TempDir()
	e := New(dir)
	cfg := &config.Config{
		On: &config.Events{
			PostCheckout: &config.Event{
				Run: []string{"echo checkout > " + filepath.Join(dir, "out.txt")},
			},
		},
	}
	err := e.RunPostCheckout(cfg, dir)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, "out.txt"))
}

func TestRunPostCheckout_NilOn(t *testing.T) {
	e := New("/tmp")
	err := e.RunPostCheckout(&config.Config{}, "/tmp")
	require.NoError(t, err)
}

func TestRunPreDelete(t *testing.T) {
	dir := t.TempDir()
	e := New(dir)
	cfg := &config.Config{
		On: &config.Events{
			PreDelete: &config.Event{
				Run: []string{"echo predelete > " + filepath.Join(dir, "out.txt")},
			},
		},
	}
	err := e.RunPreDelete(cfg, dir)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, "out.txt"))
}

func TestRunPreDelete_NilOn(t *testing.T) {
	e := New("/tmp")
	err := e.RunPreDelete(&config.Config{}, "/tmp")
	require.NoError(t, err)
}

func TestRunPostDelete(t *testing.T) {
	dir := t.TempDir()
	e := New(dir)
	cfg := &config.Config{
		On: &config.Events{
			PostDelete: &config.Event{
				Run: []string{"echo postdelete > " + filepath.Join(dir, "out.txt")},
			},
		},
	}
	err := e.RunPostDelete(cfg)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, "out.txt"))
}

func TestRunPostDelete_NilOn(t *testing.T) {
	e := New("/tmp")
	err := e.RunPostDelete(&config.Config{})
	require.NoError(t, err)
}

func TestIsNewWorktree(t *testing.T) {
	tests := []struct {
		name     string
		prevHead string
		want     bool
	}{
		{
			name:     "all zeros long form",
			prevHead: "0000000000000000000000000000000000000000",
			want:     true,
		},
		{
			name:     "short zero form",
			prevHead: "0000",
			want:     false,
		},
		{
			name:     "real commit hash",
			prevHead: "abc123def456789012345678901234567890abcd",
			want:     false,
		},
		{
			name:     "empty string",
			prevHead: "",
			want:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNewWorktree(tt.prevHead)
			assert.Equal(t, tt.want, got)
		})
	}
}
