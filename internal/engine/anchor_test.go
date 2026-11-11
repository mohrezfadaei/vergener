package engine_test

import (
	"context"
	"strings"
	"testing"

	"github.com/mohrezfadaei/vergener/internal/engine"
	"github.com/mohrezfadaei/vergener/internal/git"
)

func TestAmbiguousInitialDevelopBoundary(t *testing.T) {
	dir, c := setup(t)
	commit(t, dir, "main-only")
	command(t, dir, "tag", "checkpoint")
	c.Version.Anchor = "refs/tags/checkpoint"
	s, err := git.Capture(context.Background(), dir, c, "develop")
	if err != nil {
		t.Fatal(err)
	}
	_, err = engine.New(c, s).Evaluate("develop")
	if err == nil || !strings.Contains(err.Error(), "develop-anchor") {
		t.Fatalf("expected explicit boundary error, got %v", err)
	}
	c.Version.DevelopAnchor = "refs/tags/v1.0.0"
	s, err = git.Capture(context.Background(), dir, c, "develop")
	if err != nil {
		t.Fatal(err)
	}
	r, err := engine.New(c, s).Evaluate("develop")
	if err != nil || r.SemVer != "1.0.1-beta.0" {
		t.Fatalf("explicit boundary: %v %v", r, err)
	}
}
