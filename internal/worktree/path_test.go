package worktree

import (
	"testing"

	"github.com/relaxtortoise/worktree-setup/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestComputePath_Sibling(t *testing.T) {
	ps := &config.PathStrategy{Name: "sibling"}
	path := ComputePath("/home/me/projects/myapp", "feature-x", "github.com-owner-myapp", ps)
	expected := "/home/me/projects/myapp@feature-x"
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestComputePath_CustomTemplate(t *testing.T) {
	ps := &config.PathStrategy{Template: "/data/worktrees/{project_name}/{branch}"}
	path := ComputePath("/home/me/projects/myapp", "feature-x", "myapp", ps)
	expected := "/data/worktrees/myapp/feature-x"
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestComputePath_DefaultSibling(t *testing.T) {
	path := ComputePath("/home/me/projects/myapp", "feature-x", "proj", nil)
	expected := "/home/me/projects/myapp@feature-x"
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestComputePath_HomeStrategy(t *testing.T) {
	ps := &config.PathStrategy{Name: "home"}
	path := ComputePath("/home/me/projects/myapp", "feature-x", "github.com-owner-myapp", ps)
	assert.Contains(t, path, "worktrees")
	assert.Contains(t, path, "feature-x")
	assert.Contains(t, path, "github.com-owner-myapp")
}

func TestComputePath_NestedStrategy(t *testing.T) {
	ps := &config.PathStrategy{Name: "nested"}
	path := ComputePath("/home/me/projects/myapp", "feature-x", "proj", ps)
	assert.Contains(t, path, ".worktrees")
	assert.Contains(t, path, "feature-x")
}

func TestSanitizeBranch_ComputePath(t *testing.T) {
	path := ComputePath("/home/me/projects/myapp", "feature/slash", "proj", nil)
	assert.Contains(t, path, "feature-slash")
	assert.NotContains(t, path, "feature/slash")
}

func TestSanitizeBranch(t *testing.T) {
	tests := []struct {
		name   string
		branch string
		want   string
	}{
		{"no slash", "feature-x", "feature-x"},
		{"with slash", "feature/x", "feature-x"},
		{"multiple slashes", "a/b/c", "a-b-c"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeBranch(tt.branch)
			assert.Equal(t, tt.want, got)
		})
	}
}
