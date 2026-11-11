package git

import (
	"strings"
	"testing"
)

func TestMergeIdentity(t *testing.T) {
	for _, tc := range []struct{ msg, src, dst string }{{"Merge branch 'feat/login' into 'develop'", "feat/login", "develop"}, {"Merge branch 'main'", "main", ""}, {"Merge pull request #42 from org/ci/build", "ci/build", ""}, {"Merge remote-tracking branch 'origin/develop' into feat/sync", "develop", "feat/sync"}} {
		src, dst, err := MergeIdentity(tc.msg, nil)
		if err != nil || src != tc.src || dst != tc.dst {
			t.Fatalf("%q: %q %q %v", tc.msg, src, dst, err)
		}
	}
}
func TestMergeIdentityAmbiguity(t *testing.T) {
	patterns := map[string]string{"misleading": "^Merge branch '(?P<source>[^']+)' into '(?P<target>[^']+)'$", "conflict": "^Merge branch '[^']+' into '(?P<source>[^']+)'$"}
	_, _, err := MergeIdentity("Merge branch 'feat/login' into 'develop'", patterns)
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguity: %v", err)
	}
}
