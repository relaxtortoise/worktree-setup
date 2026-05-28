package worktree

import (
	"testing"

	"github.com/relaxtortoise/worktree-setup/internal/config"
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
