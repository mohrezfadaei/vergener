package engine_test

import (
	"fmt"
	"testing"
)

func TestRequestedBranchWorkflow(t *testing.T) {
	dir, cfg := setup(t)
	evaluate(t, dir, cfg, "main", "1.0.0")
	evaluate(t, dir, cfg, "develop", "1.0.1-beta.0")

	command(t, dir, "switch", "develop")
	for i, branch := range []string{"feat/mjt-1", "fix/mjt-2", "feature/mjt-3", "features/mjt-4", "featues/mjt-5"} {
		command(t, dir, "switch", "-c", branch)
		commit(t, dir, fmt.Sprintf("work-%d", i))
		if i == 0 {
			evaluate(t, dir, cfg, branch, "1.0.1-feat-mjt-1.1")
			commit(t, dir, "work-0")
			evaluate(t, dir, cfg, branch, "1.0.1-feat-mjt-1.2")
		}
		command(t, dir, "switch", "develop")
		merge(t, dir, branch)
		command(t, dir, "branch", "-d", branch)
		evaluate(t, dir, cfg, "develop", fmt.Sprintf("1.0.1-beta.%d", i+1))
	}

	command(t, dir, "switch", "main")
	merge(t, dir, "develop")
	evaluate(t, dir, cfg, "main", "1.0.1")
	evaluate(t, dir, cfg, "develop", "1.0.2-beta.0")
	command(t, dir, "switch", "develop")
	command(t, dir, "switch", "-c", "feat/mjt-6")
	commit(t, dir, "work-6")
	command(t, dir, "switch", "develop")
	merge(t, dir, "feat/mjt-6")
	evaluate(t, dir, cfg, "develop", "1.0.2-beta.1")
}
