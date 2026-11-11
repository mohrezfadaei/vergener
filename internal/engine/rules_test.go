package engine_test

import (
	"context"
	"testing"

	"github.com/mohrezfadaei/vergener/internal/config"
	"github.com/mohrezfadaei/vergener/internal/engine"
	"github.com/mohrezfadaei/vergener/internal/git"
)

func TestMainMessageRuleBumpsCoreWithoutRelease(t *testing.T) {
	dir, c := setup(t)
	b := c.Branches["main"]
	b.Rules = append(b.Rules, config.Rule{ID: "minor-message", Priority: 200, When: config.Condition{Event: "commit", Message: "^change api", Paths: []string{"api*"}}, Actions: []config.Action{{BumpCore: &struct {
		Increment string `yaml:"increment" json:"increment"`
	}{Increment: "Minor"}}}})
	c.Branches["main"] = b
	commit(t, dir, "api")
	r := evaluate(t, dir, c, "main", "1.1.0")
	if len(r.Releases) != 0 {
		t.Fatal("bump-core registered a release")
	}
	s, err := git.Capture(context.Background(), dir, c, "main")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = engine.New(c, s).Evaluate("main"); err != nil {
		t.Fatal(err)
	}
}
func TestMergePathsUseDestinationDiff(t *testing.T) {
	dir, c := setup(t)
	main := c.Branches["main"]
	main.Rules[0].When.Paths = []string{"feature"}
	c.Branches["main"] = main
	command(t, dir, "switch", "develop")
	commit(t, dir, "feature")
	command(t, dir, "switch", "main")
	merge(t, dir, "develop")
	evaluate(t, dir, c, "main", "1.0.1")
}

func TestMainMessageRulePrecedence(t *testing.T) {
	dir, c := setup(t)
	b := c.Branches["main"]
	b.Rules = []config.Rule{{ID: "generic", Priority: 1, When: config.Condition{Event: "commit"}, Actions: []config.Action{{Ignore: &struct{}{}}}}, {ID: "specific", Priority: 2, When: config.Condition{Event: "commit", Message: "^change api"}, Actions: []config.Action{{BumpCore: &struct {
		Increment string `yaml:"increment" json:"increment"`
	}{Increment: "Patch"}}}}}
	c.Branches["main"] = b
	commit(t, dir, "api")
	evaluate(t, dir, c, "main", "1.0.1")
}
