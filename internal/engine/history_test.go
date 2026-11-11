package engine_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mohrezfadaei/vergener/internal/git"
)

func TestShallowHistoryFails(t *testing.T) {
	dir, c := setup(t)
	command(t, dir, "switch", "develop")
	commit(t, dir, "new")
	clone := filepath.Join(t.TempDir(), "clone")
	cmd := exec.Command("git", "clone", "--quiet", "--depth", "1", "--branch", "develop", "file://"+dir, clone)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("clone: %v %s", err, out)
	}
	_, err := git.Capture(context.Background(), clone, c, "develop")
	if err == nil || !strings.Contains(err.Error(), "shallow") {
		t.Fatalf("expected shallow error: %v", err)
	}
}
func TestOctopusHistoryFails(t *testing.T) {
	dir, c := setup(t)
	command(t, dir, "switch", "develop")
	command(t, dir, "switch", "-c", "feat/a")
	commit(t, dir, "a")
	command(t, dir, "switch", "develop")
	command(t, dir, "switch", "-c", "feat/b")
	commit(t, dir, "b")
	command(t, dir, "switch", "develop")
	command(t, dir, "merge", "--no-ff", "feat/a", "feat/b", "-m", "octopus")
	_, err := git.Capture(context.Background(), dir, c, "develop")
	if err == nil || !strings.Contains(err.Error(), "octopus") {
		t.Fatalf("expected octopus error: %v", err)
	}
}
func TestMissingAnchorFails(t *testing.T) {
	dir, c := setup(t)
	c.Version.Anchor = "refs/tags/absent"
	_, err := git.Capture(context.Background(), dir, c, "develop")
	if err == nil || !strings.Contains(err.Error(), "anchor") {
		t.Fatalf("expected anchor error: %v", err)
	}
}
