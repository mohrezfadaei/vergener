package engine_test

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mohrezfadaei/vergener/internal/config"
	"github.com/mohrezfadaei/vergener/internal/engine"
	"github.com/mohrezfadaei/vergener/internal/git"
)

func TestRenamedWorkflowRolesAndBranches(t *testing.T) {
	dir, _ := setup(t)
	_, file, _, _ := runtime.Caller(0)
	c, err := config.Load(filepath.Join(filepath.Dir(file), "../..", "examples", "trunk-integration.yml"), "")
	if err != nil {
		t.Fatal(err)
	}
	command(t, dir, "branch", "-m", "trunk")
	command(t, dir, "branch", "-m", "develop", "integration")
	evaluate(t, dir, c, "trunk", "1.0.0")
	evaluate(t, dir, c, "integration", "1.1.0-preview.0")
	command(t, dir, "switch", "integration")
	command(t, dir, "switch", "-c", "feat/renamed")
	commit(t, dir, "feature")
	command(t, dir, "switch", "integration")
	merge(t, dir, "feat/renamed")
	evaluate(t, dir, c, "integration", "1.1.0-preview.1")
	command(t, dir, "switch", "trunk")
	merge(t, dir, "integration")
	r := evaluate(t, dir, c, "trunk", "1.1.0")
	if len(r.Releases) != 1 || r.Releases[0].SourceRole != "integration" || r.Releases[0].DestinationRole != "stable" {
		t.Fatal(r.Releases)
	}
	evaluate(t, dir, c, "integration", "1.2.0-preview.0")
	s, err := git.Capture(context.Background(), dir, c, "integration")
	if err != nil {
		t.Fatal(err)
	}
	if s.StableBranch != "trunk" || s.IntegrationBranch != "integration" {
		t.Fatal(s.StableBranch, s.IntegrationBranch)
	}
	if _, err = engine.New(c, s).Evaluate("integration"); err != nil {
		t.Fatal(err)
	}
}
