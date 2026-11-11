package engine_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mohrezfadaei/vergener/internal/config"
	"github.com/mohrezfadaei/vergener/internal/engine"
	"github.com/mohrezfadaei/vergener/internal/git"
)

func command(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Tester", "GIT_AUTHOR_EMAIL=test@example.invalid", "GIT_COMMITTER_NAME=Tester", "GIT_COMMITTER_EMAIL=test@example.invalid", "GIT_AUTHOR_DATE=2001-01-01T00:00:00Z", "GIT_COMMITTER_DATE=2001-01-01T00:00:00Z")
	b, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("git %v: %v: %s", args, e, b)
	}
	return strings.TrimSpace(string(b))
}
func commit(t *testing.T, dir, filename string) {
	t.Helper()
	path := filepath.Join(dir, filename)
	f, e := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if e != nil {
		t.Fatal(e)
	}
	fmt.Fprintln(f, "change")
	f.Close()
	command(t, dir, "add", filename)
	command(t, dir, "commit", "-m", "change "+filename)
}
func setup(t *testing.T) (string, *config.Config) {
	t.Helper()
	dir := t.TempDir()
	command(t, dir, "init", "-b", "main")
	commit(t, dir, "base")
	command(t, dir, "tag", "v1.0.0")
	command(t, dir, "branch", "develop")
	_, file, _, _ := runtime.Caller(0)
	c, e := config.Load(filepath.Join(filepath.Dir(file), "../..", "Vergener.yml"), "")
	if e != nil {
		t.Fatal(e)
	}
	return dir, c
}
func evaluate(t *testing.T, dir string, c *config.Config, branch, want string) *engine.Result {
	t.Helper()
	s, e := git.Capture(context.Background(), dir, c, branch)
	if e != nil {
		t.Fatal(e)
	}
	r, e := engine.New(c, s).Evaluate(branch)
	if e != nil {
		t.Fatal(e)
	}
	if r.SemVer != want {
		t.Fatalf("%s: got %s, want %s; decisions: %v", branch, r.SemVer, want, r.Decisions)
	}
	t.Logf("%s @ %s => %s (expected %s)", branch, r.SHA[:12], r.SemVer, want)
	return r
}
func merge(t *testing.T, dir, source string) {
	t.Helper()
	command(t, dir, "merge", "--no-ff", source, "-m", fmt.Sprintf("Merge branch '%s' into '%s'", source, command(t, dir, "branch", "--show-current")))
	parents := strings.Fields(command(t, dir, "rev-list", "--parents", "-n", "1", "HEAD"))
	if len(parents) != 3 {
		t.Fatalf("expected two-parent merge, got %v", parents)
	}
}
func TestAcceptanceScenario(t *testing.T) {
	dir, c := setup(t)
	evaluate(t, dir, c, "main", "1.0.0")
	evaluate(t, dir, c, "develop", "1.0.1-beta.0")
	command(t, dir, "switch", "develop")
	command(t, dir, "switch", "-c", "ci/build")
	for i := 0; i < 7; i++ {
		commit(t, dir, "ci")
	}
	evaluate(t, dir, c, "ci/build", "1.0.1-ci-build.7")
	command(t, dir, "switch", "develop")
	command(t, dir, "switch", "-c", "feat/login")
	for i := 0; i < 20; i++ {
		commit(t, dir, "feature")
	}
	command(t, dir, "switch", "develop")
	merge(t, dir, "feat/login")
	evaluate(t, dir, c, "develop", "1.0.1-beta.1")
	for i := 0; i < 3; i++ {
		commit(t, dir, "flux")
	}
	evaluate(t, dir, c, "develop", "1.0.1-beta.1")
	merge(t, dir, "ci/build")
	r := evaluate(t, dir, c, "develop", "1.0.1-beta.2")
	if len(r.CountedEvents) != 2 {
		t.Fatalf("counted %v", r.CountedEvents)
	}
	command(t, dir, "switch", "-c", "fix/example")
	commit(t, dir, "fix")
	command(t, dir, "switch", "develop")
	merge(t, dir, "fix/example")
	evaluate(t, dir, c, "develop", "1.0.1-beta.3")
	command(t, dir, "switch", "main")
	merge(t, dir, "develop")
	release := evaluate(t, dir, c, "main", "1.0.1")
	if len(release.Releases) != 1 || release.Releases[0].SourceTip == "" {
		t.Fatalf("missing release evidence: %+v", release.Releases)
	}
	evaluate(t, dir, c, "develop", "1.0.2-beta.0")
	command(t, dir, "switch", "develop")
	merge(t, dir, "main")
	evaluate(t, dir, c, "develop", "1.0.2-beta.0")
	command(t, dir, "switch", "-c", "feat/next")
	commit(t, dir, "next")
	command(t, dir, "switch", "develop")
	merge(t, dir, "feat/next")
	r = evaluate(t, dir, c, "develop", "1.0.2-beta.1")
	if len(r.CountedEvents) != 1 {
		t.Fatalf("next cycle counted %v", r.CountedEvents)
	}
	again := evaluate(t, dir, c, "develop", "1.0.2-beta.1")
	if r.ConfigHash != again.ConfigHash || r.SHA != again.SHA {
		t.Fatal("unstable fixed snapshot")
	}
}
func TestPendingMergeAfterReleaseSourceTip(t *testing.T) {
	dir, c := setup(t)
	command(t, dir, "switch", "develop")
	command(t, dir, "switch", "-c", "feat/first")
	commit(t, dir, "first")
	command(t, dir, "switch", "develop")
	merge(t, dir, "feat/first")
	sourceTip := command(t, dir, "rev-parse", "HEAD")
	command(t, dir, "switch", "main")
	merge(t, dir, "develop")
	command(t, dir, "switch", "develop")
	command(t, dir, "switch", "-c", "feat/pending")
	commit(t, dir, "pending")
	command(t, dir, "switch", "develop")
	merge(t, dir, "feat/pending")
	r := evaluate(t, dir, c, "develop", "1.0.2-beta.1")
	if r.CycleBaseSHA != sourceTip {
		t.Fatalf("boundary %s != %s", r.CycleBaseSHA, sourceTip)
	}
}
func TestWorkSyncDoesNotReset(t *testing.T) {
	dir, c := setup(t)
	command(t, dir, "switch", "develop")
	command(t, dir, "switch", "-c", "feat/sync")
	commit(t, dir, "feature")
	command(t, dir, "switch", "develop")
	commit(t, dir, "direct")
	command(t, dir, "switch", "feat/sync")
	merge(t, dir, "develop")
	commit(t, dir, "feature")
	evaluate(t, dir, c, "feat/sync", "1.0.1-feat-sync.2")
}
