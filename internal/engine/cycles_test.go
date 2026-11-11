package engine_test

import (
	"fmt"
	"testing"
)

func TestTwoCompleteReleaseCycles(t *testing.T) {
	dir, c := setup(t)
	for cycle := 1; cycle <= 2; cycle++ {
		command(t, dir, "switch", "develop")
		branch := fmt.Sprintf("feat/cycle%d", cycle)
		command(t, dir, "switch", "-c", branch)
		commit(t, dir, fmt.Sprintf("cycle%d-file", cycle))
		command(t, dir, "switch", "develop")
		merge(t, dir, branch)
		if cycle == 1 {
			evaluate(t, dir, c, "develop", "1.0.1-beta.1")
		} else {
			evaluate(t, dir, c, "develop", "1.0.2-beta.1")
		}
		command(t, dir, "switch", "main")
		merge(t, dir, "develop")
		if cycle == 1 {
			evaluate(t, dir, c, "main", "1.0.1")
			evaluate(t, dir, c, "develop", "1.0.2-beta.0")
		} else {
			evaluate(t, dir, c, "main", "1.0.2")
			evaluate(t, dir, c, "develop", "1.0.3-beta.0")
		}
	}
}
