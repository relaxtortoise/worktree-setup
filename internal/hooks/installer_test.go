package hooks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstall(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(dir string) // prepare the repo directory
		wantErr string           // substring of expected error, empty means success
		want    []string         // expected installed hooks list
	}{
		{
			name: "successfully installs post-checkout hook",
			setup: func(dir string) {
				_ = os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0755)
			},
			wantErr: "",
			want:    []string{"post-checkout"},
		},
		{
			name: "fails when hooks directory does not exist",
			setup: func(dir string) {
				// Only create .git without hooks subdirectory
				_ = os.MkdirAll(filepath.Join(dir, ".git"), 0755)
			},
			wantErr: "write",
			want:    nil,
		},
		{
			name:    "fails when .git directory does not exist",
			setup:   func(dir string) {},
			wantErr: "write",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoDir := t.TempDir()
			if tt.setup != nil {
				tt.setup(repoDir)
			}

			got, err := Install(repoDir)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)

				// Verify the hook file was written correctly
				hookPath := filepath.Join(repoDir, ".git", "hooks", "post-checkout")
				data, err := os.ReadFile(hookPath)
				require.NoError(t, err)
				assert.Contains(t, string(data), "Installed by wt")
				assert.Contains(t, string(data), "#!/bin/sh")

				// Verify execute permission is set
				info, err := os.Stat(hookPath)
				require.NoError(t, err)
				assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
			}
		})
	}
}

func TestIsInstalled(t *testing.T) {
	tests := []struct {
		name  string
		setup func(dir string) // prepare the repo directory
		want  bool
	}{
		{
			name: "returns true when hook contains the wt marker",
			setup: func(dir string) {
				hooksDir := filepath.Join(dir, ".git", "hooks")
				_ = os.MkdirAll(hooksDir, 0755)
				_, err := Install(dir)
				require.New(t).NoError(err)
			},
			want: true,
		},
		{
			name: "returns false when hook file does not exist",
			setup: func(dir string) {
				_ = os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0755)
			},
			want: false,
		},
		{
			name: "returns false when hook file lacks the wt marker",
			setup: func(dir string) {
				hooksDir := filepath.Join(dir, ".git", "hooks")
				_ = os.MkdirAll(hooksDir, 0755)
				err := os.WriteFile(filepath.Join(hooksDir, "post-checkout"), []byte("#!/bin/sh\necho hello"), 0755)
				require.New(t).NoError(err)
			},
			want: false,
		},
		{
			name: "returns false when .git directory does not exist",
			setup: func(dir string) {
				// No .git directory at all
			},
			want: false,
		},
		{
			name: "returns false when hooks directory does not exist",
			setup: func(dir string) {
				_ = os.MkdirAll(filepath.Join(dir, ".git"), 0755)
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoDir := t.TempDir()
			if tt.setup != nil {
				tt.setup(repoDir)
			}

			got := IsInstalled(repoDir)
			assert.Equal(t, tt.want, got)
		})
	}
}
