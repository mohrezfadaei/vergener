package engine_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/mohrezfadaei/vergener/internal/engine"
	"github.com/mohrezfadaei/vergener/internal/git"
)

func TestReleaseIncrementNoneStillRecordsBoundary(t *testing.T) {
	dir, c := setup(t)
	main := c.Branches["main"]
	main.Increment = "None"
	c.Branches["main"] = main
	command(t, dir, "switch", "develop")
	command(t, dir, "switch", "-c", "feat/one")
	commit(t, dir, "one")
	command(t, dir, "switch", "develop")
	merge(t, dir, "feat/one")
	evaluate(t, dir, c, "develop", "1.0.0-beta.1")
	command(t, dir, "switch", "main")
	merge(t, dir, "develop")
	r := evaluate(t, dir, c, "main", "1.0.0")
	if len(r.Releases) != 1 {
		t.Fatal("None suppressed release boundary")
	}
	evaluate(t, dir, c, "develop", "1.0.0-beta.0")
}
func TestMajorMinorPolicy(t *testing.T) {
	for _, tc := range []struct{ inc, want string }{{"Major", "2.0.0"}, {"Minor", "1.1.0"}} {
		t.Run(tc.inc, func(t *testing.T) {
			dir, c := setup(t)
			main := c.Branches["main"]
			main.Increment = tc.inc
			c.Branches["main"] = main
			command(t, dir, "switch", "develop")
			commit(t, dir, "change")
			command(t, dir, "switch", "main")
			merge(t, dir, "develop")
			evaluate(t, dir, c, "main", tc.want)
		})
	}
}
func TestRepeatedEvaluationDoesNotDuplicateEvents(t *testing.T) {
	dir, c := setup(t)
	command(t, dir, "switch", "develop")
	commit(t, dir, "change")
	command(t, dir, "switch", "main")
	merge(t, dir, "develop")
	s, err := git.Capture(context.Background(), dir, c, "develop")
	if err != nil {
		t.Fatal(err)
	}
	e := engine.New(c, s)
	var wg sync.WaitGroup
	failures := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := e.Evaluate("develop")
			if err != nil {
				failures <- err
				return
			}
			if r.SemVer != "1.0.2-beta.0" || len(r.Releases) != 1 {
				failures <- fmt.Errorf("nondeterministic replay: %+v", r)
			}
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
}

func TestUnsupportedWorkToWorkMerge(t *testing.T) {
	dir, c := setup(t)
	command(t, dir, "switch", "develop")
	command(t, dir, "switch", "-c", "feat/a")
	commit(t, dir, "a")
	command(t, dir, "switch", "develop")
	command(t, dir, "switch", "-c", "feat/b")
	commit(t, dir, "b")
	merge(t, dir, "feat/a")
	s, err := git.Capture(context.Background(), dir, c, "feat/b")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = engine.New(c, s).Evaluate("feat/b"); err == nil {
		t.Fatal("work-to-work merge silently accepted")
	}
}
