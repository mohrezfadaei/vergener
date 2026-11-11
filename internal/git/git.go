package git

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mohrezfadaei/vergener/internal/config"
)

type Commit struct {
	SHA     string
	Parents []string
	Message string
	Author  string
	Paths   []string
}
type Snapshot struct {
	Refs              map[string]string
	Tags              map[string]string
	StableBranch      string
	IntegrationBranch string
	Anchor            string
	DevelopAnchor     string
	Commits           map[string]Commit
	Dirty             bool
}

func run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	b, e := cmd.CombinedOutput()
	if e != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), e, strings.TrimSpace(string(b)))
	}
	return string(b), nil
}
func Capture(ctx context.Context, dir string, c *config.Config, branch string) (*Snapshot, error) {
	shallow, e := run(ctx, dir, "rev-parse", "--is-shallow-repository")
	if e != nil {
		return nil, e
	}
	if strings.TrimSpace(shallow) == "true" {
		return nil, fmt.Errorf("shallow repository unsupported: fetch full history")
	}
	s := &Snapshot{Refs: map[string]string{}, Tags: map[string]string{}, Commits: map[string]Commit{}}
	refs, e := run(ctx, dir, "for-each-ref", "--format=%(refname:short) %(objectname)", "refs/heads")
	if e != nil {
		return nil, e
	}
	for _, line := range strings.Split(strings.TrimSpace(refs), "\n") {
		f := strings.Fields(line)
		if len(f) == 2 {
			s.Refs[f[0]] = f[1]
		}
	}
	stableRole, integrationRole, err := c.WorkflowRoles()
	if err != nil {
		return nil, err
	}
	s.StableBranch, err = branchForRole(s.Refs, stableRole, c.Branches[stableRole].Match)
	if err != nil {
		return nil, err
	}
	s.IntegrationBranch, err = branchForRole(s.Refs, integrationRole, c.Branches[integrationRole].Match)
	if err != nil {
		return nil, err
	}
	if _, ok := s.Refs[branch]; !ok {
		return nil, fmt.Errorf("branch %q not found", branch)
	}
	var tagRefs string
	if c.Tags.Discovery == "matching" {
		tagRefs, e = run(ctx, dir, "for-each-ref", "--format=%(refname:short)%00%(objecttype)%00%(objectname)%00%(*objecttype)%00%(*objectname)", "refs/tags")
		if e != nil { return nil, e }
		for _, line := range strings.Split(strings.TrimSuffix(tagRefs, "\n"), "\n") {
			if line == "" { continue }
			parts := strings.Split(line, "\x00")
			if len(parts) != 5 { return nil, fmt.Errorf("cannot parse tag ref %q", line) }
			sha := ""
			if parts[1] == "commit" { sha = parts[2] } else if parts[1] == "tag" && parts[3] == "commit" { sha = parts[4] }
			s.Tags[parts[0]] = sha
		}
	}
	anchor, e := run(ctx, dir, "rev-parse", "--verify", c.Version.Anchor+"^{commit}")
	if e != nil {
		return nil, fmt.Errorf("initial anchor: %w", e)
	}
	s.Anchor = strings.TrimSpace(anchor)
	if c.Version.DevelopAnchor != "" {
		v, e := run(ctx, dir, "rev-parse", "--verify", c.Version.DevelopAnchor+"^{commit}")
		if e != nil {
			return nil, fmt.Errorf("develop anchor: %w", e)
		}
		s.DevelopAnchor = strings.TrimSpace(v)
	}
	raw, e := run(ctx, dir, "log", "--all", "--format=%H%x00%P%x00%an%x00%B%x00%x1e")
	if e != nil {
		return nil, e
	}
	for _, record := range strings.Split(raw, "\x1e") {
		record = strings.Trim(record, "\n\r")
		if record == "" {
			continue
		}
		fields := strings.SplitN(record, "\x00", 5)
		if len(fields) != 5 {
			return nil, fmt.Errorf("cannot parse git log record")
		}
		sha := fields[0]
		parents := strings.Fields(fields[1])
		if len(parents) > 2 {
			return nil, fmt.Errorf("octopus merge %s unsupported", sha)
		}
		s.Commits[sha] = Commit{SHA: sha, Parents: parents, Message: strings.TrimSpace(fields[3]), Author: fields[2]}
	}
	if needsPaths(c) {
		for sha, commit := range s.Commits {
			paths, err := run(ctx, dir, "diff-tree", "--root", "--first-parent", "-m", "--no-commit-id", "--name-only", "-r", sha)
			if err != nil {
				return nil, err
			}
			commit.Paths = strings.Fields(paths)
			s.Commits[sha] = commit
		}
	}
	for _, v := range s.Commits {
		for _, p := range v.Parents {
			if _, ok := s.Commits[p]; !ok {
				return nil, fmt.Errorf("missing parent %s of %s", p, v.SHA)
			}
		}
	}
	if _, ok := s.Commits[s.Anchor]; !ok {
		return nil, fmt.Errorf("anchor %s not in reachable history", s.Anchor)
	}
	status, e := run(ctx, dir, "status", "--porcelain")
	if e != nil {
		return nil, e
	}
	s.Dirty = strings.TrimSpace(status) != ""
	again, e := run(ctx, dir, "for-each-ref", "--format=%(refname:short) %(objectname)", "refs/heads")
	if e != nil {
		return nil, e
	}
	if again != refs {
		return nil, fmt.Errorf("refs changed during snapshot capture; retry")
	}
	if c.Tags.Discovery == "matching" {
		again, e = run(ctx, dir, "for-each-ref", "--format=%(refname:short)%00%(objecttype)%00%(objectname)%00%(*objecttype)%00%(*objectname)", "refs/tags")
		if e != nil || again != tagRefs { return nil, fmt.Errorf("tag refs changed during snapshot capture; retry") }
	}
	again, e = run(ctx, dir, "rev-parse", "--verify", c.Version.Anchor+"^{commit}")
	if e != nil || strings.TrimSpace(again) != s.Anchor {
		return nil, fmt.Errorf("anchor changed during snapshot capture; retry")
	}
	if c.Version.DevelopAnchor != "" {
		again, e = run(ctx, dir, "rev-parse", "--verify", c.Version.DevelopAnchor+"^{commit}")
		if e != nil || strings.TrimSpace(again) != s.DevelopAnchor {
			return nil, fmt.Errorf("develop anchor changed during snapshot capture; retry")
		}
	}
	return s, nil
}
func (s *Snapshot) Chain(tip string) ([]Commit, error) {
	seen := map[string]bool{}
	var out []Commit
	for tip != "" {
		if seen[tip] {
			return nil, fmt.Errorf("first-parent cycle at %s", tip)
		}
		seen[tip] = true
		c, ok := s.Commits[tip]
		if !ok {
			return nil, fmt.Errorf("commit %s missing", tip)
		}
		out = append(out, c)
		if len(c.Parents) == 0 {
			break
		}
		tip = c.Parents[0]
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}
func (s *Snapshot) OnChain(tip, sha string) bool {
	chain, e := s.Chain(tip)
	if e != nil {
		return false
	}
	for _, v := range chain {
		if v.SHA == sha {
			return true
		}
	}
	return false
}
func (s *Snapshot) Ancestor(old, tip string) bool {
	seen := map[string]bool{}
	var visit func(string) bool
	visit = func(sha string) bool {
		if sha == old {
			return true
		}
		if seen[sha] {
			return false
		}
		seen[sha] = true
		v, ok := s.Commits[sha]
		if !ok {
			return false
		}
		for _, p := range v.Parents {
			if visit(p) {
				return true
			}
		}
		return false
	}
	return visit(tip)
}

func branchForRole(refs map[string]string, role string, matcher config.Match) (string, error) {
	found := ""
	for name := range refs {
		match := false
		switch {
		case matcher.Exact != "":
			match = name == matcher.Exact
		case matcher.Glob != "":
			match, _ = filepath.Match(matcher.Glob, name)
		case matcher.Regex != "":
			match = regexp.MustCompile(matcher.Regex).MatchString(name)
		}
		if match {
			if found != "" {
				return "", fmt.Errorf("role %s matches multiple branches: %s and %s", role, found, name)
			}
			found = name
		}
	}
	if found == "" {
		return "", fmt.Errorf("required role %s has no matching branch", role)
	}
	return found, nil
}

func needsPaths(c *config.Config) bool {
	var condition func(config.Condition) bool
	condition = func(w config.Condition) bool {
		if len(w.Paths) > 0 {
			return true
		}
		for _, part := range append(w.All, w.Any...) {
			if condition(part) {
				return true
			}
		}
		return w.Not != nil && condition(*w.Not)
	}
	rules := append([]config.Rule{}, c.Rules.Shared...)
	for _, b := range c.Branches {
		rules = append(rules, b.Rules...)
		rules = append(rules, b.Prerelease.Counter.Rules...)
	}
	for _, set := range c.RuleSets {
		rules = append(rules, set...)
	}
	for _, r := range rules {
		if condition(r.When) {
			return true
		}
	}
	return false
}

var builtin = []*regexp.Regexp{
	regexp.MustCompile(`^Merge branch '(?P<source>[^']+)' into '(?P<target>[^']+)'`),
	regexp.MustCompile(`^Merge branch '(?P<source>[^']+)'$`),
	regexp.MustCompile(`^Merge remote-tracking branch '[^/]+/(?P<source>[^']+)' into (?P<target>\S+)`),
	regexp.MustCompile(`^Merge pull request #(?P<pullRequest>[0-9]+) from [^/]+/(?P<source>.+)$`),
}

func MergeIdentity(message string, formats map[string]string) (string, string, error) {
	first := strings.SplitN(message, "\n", 2)[0]
	var matches [][2]string
	keys := make([]string, 0, len(formats))
	for k := range formats {
		keys = append(keys, k)
	}
	sortStrings(keys)
	parsers := append([]*regexp.Regexp{}, builtin...)
	for _, k := range keys {
		parsers = append(parsers, regexp.MustCompile(formats[k]))
	}
	for _, r := range parsers {
		m := r.FindStringSubmatch(first)
		if m == nil {
			continue
		}
		src := m[r.SubexpIndex("source")]
		dst := ""
		if i := r.SubexpIndex("target"); i >= 0 {
			dst = m[i]
		}
		if src != "" {
			matches = append(matches, [2]string{src, dst})
		}
	}
	if len(matches) == 0 {
		return "", "", fmt.Errorf("unknown merge identity: %q", first)
	}
	for _, m := range matches[1:] {
		if m != matches[0] {
			return "", "", fmt.Errorf("ambiguous merge identity: %q", first)
		}
	}
	return matches[0][0], matches[0][1], nil
}
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
