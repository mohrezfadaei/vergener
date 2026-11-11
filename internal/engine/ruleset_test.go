package engine_test

import (
	"testing"

	"github.com/mohrezfadaei/vergener/internal/config"
)

func TestReusableCounterRuleSet(t *testing.T) {
	dir, c := setup(t)
	b := c.Branches["work"]
	c.RuleSets = map[string][]config.Rule{"direct-work": b.Prerelease.Counter.Rules}
	b.Prerelease.Counter.Rules = nil
	b.Prerelease.Counter.UseRuleSets = []string{"direct-work"}
	c.Branches["work"] = b
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	command(t, dir, "switch", "develop")
	command(t, dir, "switch", "-c", "feat/shared")
	commit(t, dir, "work")
	evaluate(t, dir, c, "feat/shared", "1.0.1-feat-shared.1")
}
