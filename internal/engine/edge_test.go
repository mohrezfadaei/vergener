package engine_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/mohrezfadaei/vergener/internal/engine"
	"github.com/mohrezfadaei/vergener/internal/git"
)

func TestNoOpMergeAndDeletedSource(t *testing.T) {
	dir, c := setup(t)
	command(t, dir, "switch", "develop")
	command(t, dir, "switch", "-c", "feat/one")
	commit(t, dir, "one")
	command(t, dir, "switch", "develop")
	merge(t, dir, "feat/one")
	command(t, dir, "branch", "-D", "feat/one")
	evaluate(t, dir, c, "develop", "1.0.1-beta.1")
	command(t, dir, "merge", "develop")
	evaluate(t, dir, c, "develop", "1.0.1-beta.1")
}
func TestUnknownMergeAndOctopus(t *testing.T) {
	dir, c := setup(t)
	command(t, dir, "switch", "develop")
	command(t, dir, "switch", "-c", "feat/a")
	commit(t, dir, "a")
	command(t, dir, "switch", "develop")
	command(t, dir, "merge", "--no-ff", "feat/a", "-m", "unidentifiable merge")
	s, err := git.Capture(context.Background(), dir, c, "develop")
	if err != nil {
		t.Fatal(err)
	}
	_, err = engine.New(c, s).Evaluate("develop")
	if err == nil || !strings.Contains(err.Error(), "unknown merge identity") {
		t.Fatalf("expected unknown identity, got %v", err)
	}
}
func TestAmbiguousRulePriority(t *testing.T) {
	dir, c := setup(t)
	b := c.Branches["develop"]
	rule := b.Prerelease.Counter.Rules[0]
	rule.ID = "same-priority"
	b.Prerelease.Counter.Rules = append(b.Prerelease.Counter.Rules, rule)
	c.Branches["develop"] = b
	command(t, dir, "switch", "develop")
	command(t, dir, "switch", "-c", "feat/a")
	commit(t, dir, "a")
	command(t, dir, "switch", "develop")
	merge(t, dir, "feat/a")
	s, err := git.Capture(context.Background(), dir, c, "develop")
	if err != nil {
		t.Fatal(err)
	}
	_, err = engine.New(c, s).Evaluate("develop")
	if err == nil || !strings.Contains(err.Error(), "ambiguous priority") {
		t.Fatalf("expected priority error, got %v", err)
	}
}
func TestLabelCollision(t *testing.T) {
	dir, c := setup(t)
	command(t, dir, "switch", "develop")
	command(t, dir, "branch", "feat/a_b")
	command(t, dir, "branch", "feat/a-b")
	s, err := git.Capture(context.Background(), dir, c, "feat/a_b")
	if err != nil {
		t.Fatal(err)
	}
	_, err = engine.New(c, s).Evaluate("feat/a_b")
	if err == nil || !strings.Contains(err.Error(), "label collision") {
		t.Fatalf("expected collision, got %v", err)
	}
}
func TestDirtyWorkingTree(t *testing.T) {
	dir, c := setup(t)
	command(t, dir, "switch", "develop")
	r := evaluate(t, dir, c, "develop", "1.0.1-beta.0")
	if r.Dirty {
		t.Fatal("clean tree marked dirty")
	}
	commit(t, dir, "tracked")
	r = evaluate(t, dir, c, "develop", "1.0.1-beta.0")
	if r.Dirty {
		t.Fatal("committed tree marked dirty")
	}
	command(t, dir, "status", "--short") // append without committing
	f := dir + "/tracked"
	if err := os.WriteFile(f, []byte("dirty"), 0600); err != nil {
		t.Fatal(err)
	}
	r = evaluate(t, dir, c, "develop", "1.0.1-beta.0")
	if !r.Dirty {
		t.Fatal("dirty tree not reported")
	}
}
