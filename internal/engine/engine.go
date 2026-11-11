package engine

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/mohrezfadaei/vergener/internal/config"
	"github.com/mohrezfadaei/vergener/internal/git"
	"github.com/mohrezfadaei/vergener/internal/semver"
)

type Release struct {
	SHA               string `json:"sha"`
	DestinationParent string `json:"destinationParent"`
	SourceTip         string `json:"sourceTip"`
	SourceRole        string `json:"sourceRole"`
	DestinationRole   string `json:"destinationRole"`
	Version           string `json:"version"`
}
type Result struct {
	SchemaVersion    int               `json:"schemaVersion"`
	Branch           string            `json:"branch"`
	SHA              string            `json:"sha"`
	CoreVersion      string            `json:"coreVersion"`
	Major            int               `json:"major"`
	Minor            int               `json:"minor"`
	Patch            int               `json:"patch"`
	PreReleaseLabel  string            `json:"preReleaseLabel,omitempty"`
	PreReleaseNumber *int              `json:"preReleaseNumber,omitempty"`
	SemVer           string            `json:"semVer"`
	StableVersion    string            `json:"stableVersion"`
	CycleBaseSHA     string            `json:"cycleBaseSha"`
	CountedEvents    []string          `json:"countedEvents"`
	Snapshot         map[string]string `json:"snapshot"`
	ConfigHash       string            `json:"configHash"`
	Dirty            bool              `json:"dirty"`
	Releases         []Release         `json:"releases,omitempty"`
	Decisions        []string          `json:"decisions,omitempty"`
}
type Engine struct {
	C               *config.Config
	S               *git.Snapshot
	releases        []Release
	stable          semver.Core
	stableRole      string
	integrationRole string
}

func New(c *config.Config, s *git.Snapshot) *Engine {
	stableRole, integrationRole, _ := c.WorkflowRoles() // Load validates uniqueness.
	return &Engine{C: c, S: s, stableRole: stableRole, integrationRole: integrationRole}
}
func (e *Engine) role(name string) (string, map[string]string, error) {
	var found string
	vars := map[string]string{}
	best := -1 << 30
	for role, b := range e.C.Branches {
		m := map[string]string{}
		ok := false
		switch {
		case b.Match.Exact != "":
			ok = name == b.Match.Exact
		case b.Match.Glob != "":
			ok, _ = filepath.Match(b.Match.Glob, name)
		case b.Match.Regex != "":
			r := regexp.MustCompile(b.Match.Regex)
			match := r.FindStringSubmatch(name)
			if match != nil {
				ok = true
				for i, k := range r.SubexpNames() {
					if i > 0 && k != "" {
						m[k] = match[i]
					}
				}
			}
		}
		if ok {
			if b.Priority == best {
				return "", nil, fmt.Errorf("ambiguous branch match for %q: %s and %s", name, found, role)
			}
			if b.Priority > best {
				found = role
				vars = m
				best = b.Priority
			}
		}
	}
	if found == "" {
		return "", nil, fmt.Errorf("no branch role matches %q", name)
	}
	return found, vars, nil
}
func (e *Engine) identity(c git.Commit, destination string) (string, string, error) {
	src, target, err := git.MergeIdentity(c.Message, e.C.Git.MergeMessageFormats)
	if err != nil {
		return "", "", err
	}
	if target != "" && target != destination {
		return "", "", fmt.Errorf("merge %s says target %q, expected %q", c.SHA, target, destination)
	}
	role, _, err := e.role(src)
	return src, role, err
}
func selector(s config.Selector, role, name string) bool {
	if len(s.Roles) > 0 {
		ok := false
		for _, r := range s.Roles {
			if r == role {
				ok = true
			}
		}
		if !ok {
			return false
		}
	}
	if s.Pattern != "" && !regexp.MustCompile(s.Pattern).MatchString(name) {
		return false
	}
	return true
}
func match(w config.Condition, ev, source, sourceRole, target, targetRole string, c git.Commit) bool {
	if w.Event != "" && w.Event != ev {
		return false
	}
	if !selector(w.Source, sourceRole, source) || !selector(w.Target, targetRole, target) {
		return false
	}
	if w.Message != "" && !regexp.MustCompile(w.Message).MatchString(c.Message) {
		return false
	}
	if w.Author != "" && !regexp.MustCompile(w.Author).MatchString(c.Author) {
		return false
	}
	if len(w.Paths) > 0 {
		found := false
		for _, pattern := range w.Paths {
			for _, path := range c.Paths {
				matched, _ := filepath.Match(pattern, path)
				if matched {
					found = true
				}
			}
		}
		if !found {
			return false
		}
	}
	for key, p := range w.Trailers {
		value := ""
		for _, line := range strings.Split(c.Message, "\n") {
			if strings.HasPrefix(strings.ToLower(line), strings.ToLower(key)+":") {
				value = strings.TrimSpace(line[len(key)+1:])
			}
		}
		if !regexp.MustCompile(p).MatchString(value) {
			return false
		}
	}
	for _, x := range w.All {
		if !match(x, ev, source, sourceRole, target, targetRole, c) {
			return false
		}
	}
	if len(w.Any) > 0 {
		any := false
		for _, x := range w.Any {
			if match(x, ev, source, sourceRole, target, targetRole, c) {
				any = true
			}
		}
		if !any {
			return false
		}
	}
	if w.Not != nil && match(*w.Not, ev, source, sourceRole, target, targetRole, c) {
		return false
	}
	return true
}
func (e *Engine) choose(local []config.Rule, sets []string, ev, src, srcRole, target, targetRole string, commit git.Commit) (*config.Rule, error) {
	scope := append([]config.Rule{}, local...)
	for _, name := range sets {
		scope = append(scope, e.C.RuleSets[name]...)
	}
	for _, rules := range [][]config.Rule{scope, e.C.Rules.Shared} {
		var chosen *config.Rule
		for _, r := range config.Sorted(rules) {
			if !match(r.When, ev, src, srcRole, target, targetRole, commit) {
				continue
			}
			if chosen != nil {
				if chosen.Priority == r.Priority {
					return nil, fmt.Errorf("ambiguous priority %d: %s and %s at %s", r.Priority, chosen.ID, r.ID, commit.SHA)
				}
				break
			}
			copy := r
			chosen = &copy
		}
		if chosen != nil {
			return chosen, nil
		}
	}
	return nil, nil
}
func (e *Engine) mainHistory(anchor string) error {
	initial, err := semver.Parse(e.C.Version.Initial)
	if err != nil {
		return err
	}
	e.stable = initial
	chain, err := e.S.Chain(e.S.Refs[e.S.StableBranch])
	if err != nil {
		return err
	}
	index := -1
	for i, c := range chain {
		if c.SHA == anchor {
			index = i
			break
		}
	}
	if index < 0 {
		return fmt.Errorf("anchor %s not on %s first-parent chain", anchor, e.S.StableBranch)
	}
	for _, commit := range chain[index+1:] {
		ev, src, srcRole := "commit", "", ""
		if len(commit.Parents) == 2 {
			ev = "merge"
			src, srcRole, err = e.identity(commit, e.S.StableBranch)
			if err != nil {
				if e.C.Defaults.UnknownMerge == "ignore" {
					continue
				}
				return fmt.Errorf("stable merge %s: %w", commit.SHA, err)
			}
		}
		rule, err := e.choose(e.C.Branches[e.stableRole].Rules, e.C.Branches[e.stableRole].UseRuleSets, ev, src, srcRole, e.S.StableBranch, e.stableRole, commit)
		if err != nil {
			return err
		}
		if rule == nil {
			continue
		}
		for _, action := range rule.Actions {
			if action.Fail != nil {
				return fmt.Errorf("rule %s: %s", rule.ID, action.Fail.Message)
			}
			if action.Release != nil {
				if len(commit.Parents) != 2 {
					return fmt.Errorf("rule %s: release requires a two-parent merge", rule.ID)
				}
				kind := action.Release.Increment
				if kind == "Inherit" {
					kind, err = e.C.Increment(e.stableRole)
					if err != nil {
						return err
					}
				}
				e.stable, err = e.stable.Bump(kind)
				if err != nil {
					return err
				}
				e.releases = append(e.releases, Release{commit.SHA, commit.Parents[0], commit.Parents[1], srcRole, e.stableRole, e.stable.String()})
			} else if action.BumpCore != nil {
				kind := action.BumpCore.Increment
				if kind == "Inherit" {
					kind, err = e.C.Increment(e.stableRole)
					if err != nil {
						return err
					}
				}
				e.stable, err = e.stable.Bump(kind)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
func (e *Engine) branchForRole(role string) (string, error) {
	found := ""
	for name := range e.S.Refs {
		resolved, _, err := e.role(name)
		if err != nil || resolved != role { continue }
		if found != "" { return "", fmt.Errorf("role %s has multiple matching refs: %s and %s",role,found,name) }
		found = name
	}
	if found == "" { return "", fmt.Errorf("role %s has no matching branch ref",role) }
	return found,nil
}

func (e *Engine) matchingTag(role, pattern string) (string, string, string, error) {
	branch, err := e.branchForRole(role)
	if err != nil { return "","","",err }
	chain, err := e.S.Chain(e.S.Refs[branch])
	if err != nil { return "","","",err }
	matches := map[string][]string{}
	values := map[string]string{}
	re := regexp.MustCompile(pattern)
	for name, sha := range e.S.Tags {
		parts := re.FindStringSubmatch(name)
		if parts == nil { continue }
		if sha == "" { return "","","",fmt.Errorf("matching tag %s does not refer to a commit",name) }
		matches[sha] = append(matches[sha], name)
		if i := re.SubexpIndex("version"); i >= 0 { values[name] = parts[i] }
	}
	for i := len(chain)-1; i >= 0; i-- {
		commit := chain[i]
		names := matches[commit.SHA]
		if len(names) == 0 { continue }
		if len(names)>1 { return "","","",fmt.Errorf("ambiguous matching tags at %s: %v",commit.SHA,names) }
		return names[0],commit.SHA,values[names[0]],nil
	}
	return "","","",fmt.Errorf("no matching tag on %s first-parent history",branch)
}

func (e *Engine) core(role string, seen map[string]bool) (semver.Core, error) {
	if seen[role] {
		return semver.Core{}, fmt.Errorf("core dependency cycle at %s", role)
	}
	seen[role] = true
	defer delete(seen, role)
	b := e.C.Branches[role]
	switch b.Core.Source {
	case "releases":
		return e.stable, nil
	case "fixed":
		return semver.Parse(b.Core.Value)
	case "branch-core":
		return e.core(b.Core.Of, seen)
	case "next-release":
		v, err := e.core(b.Core.Of, seen)
		if err != nil {
			return v, err
		}
		kind := b.Core.Increment
		if kind == "Inherit" || kind == "" {
			kind, err = e.C.Increment(b.Core.Of)
			if err != nil {
				return v, err
			}
		}
		return v.Bump(kind)
	}
	return semver.Core{}, fmt.Errorf("invalid core source %s", b.Core.Source)
}
func (e *Engine) initialBoundary() (string, error) {
	if e.S.OnChain(e.S.Refs[e.S.IntegrationBranch], e.S.Anchor) {
		return e.S.Anchor, nil
	}
	a := e.S.Commits[e.S.Anchor]
	if len(a.Parents) == 2 {
		_, role, err := e.identity(a, e.S.StableBranch)
		if err == nil && role == e.integrationRole && e.S.OnChain(e.S.Refs[e.S.IntegrationBranch], a.Parents[1]) {
			return a.Parents[1], nil
		}
	}
	if e.S.DevelopAnchor != "" && e.S.OnChain(e.S.Refs[e.S.IntegrationBranch], e.S.DevelopAnchor) {
		return e.S.DevelopAnchor, nil
	}
	return "", fmt.Errorf("initial anchor is not on %s first-parent chain or a recognized %s release; set version.develop-anchor", e.S.IntegrationBranch, e.integrationRole)
}
func (e *Engine) developBoundary() (string, error) {
	base, err := e.initialBoundary()
	if err != nil {
		return "", err
	}
	for _, r := range e.releases {
		if r.SourceRole == e.integrationRole && e.S.OnChain(e.S.Refs[e.S.IntegrationBranch], r.SourceTip) {
			base = r.SourceTip
		}
	}
	return base, nil
}
func (e *Engine) workBoundary(tip string) (string, error) {
	work, err := e.S.Chain(tip)
	if err != nil {
		return "", err
	}
	dev, err := e.S.Chain(e.S.Refs[e.S.IntegrationBranch])
	if err != nil {
		return "", err
	}
	shared := map[string]bool{}
	for _, c := range dev {
		shared[c.SHA] = true
	}
	for i := len(work) - 1; i >= 0; i-- {
		if shared[work[i].SHA] {
			return work[i].SHA, nil
		}
	}
	return "", fmt.Errorf("no common first-parent fork with develop")
}
func label(template string, vars map[string]string) (string, error) {
	for k, v := range vars {
		template = strings.ReplaceAll(template, "{"+k+"}", v)
	}
	if strings.ContainsAny(template, "{}") {
		return "", fmt.Errorf("unresolved prerelease label %q", template)
	}
	template = strings.ToLower(template)
	var out strings.Builder
	dash := false
	for _, r := range template {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			out.WriteRune(r)
			dash = false
		} else if !dash && out.Len() > 0 {
			out.WriteByte('-')
			dash = true
		}
	}
	s := strings.Trim(out.String(), "-")
	if s == "" {
		return "", fmt.Errorf("empty prerelease label")
	}
	return s, nil
}
func (e *Engine) Evaluate(branch string) (*Result, error) {
	replay := &Engine{C: e.C, S: e.S, stableRole: e.stableRole, integrationRole: e.integrationRole}
	return replay.evaluate(branch)
}

func (e *Engine) evaluate(branch string) (*Result, error) {
	role, vars, err := e.role(branch)
	if err != nil {
		return nil, err
	}
	if err = e.mainHistory(e.S.Anchor); err != nil {
		return nil, err
	}
	tip := e.S.Refs[branch]
	core, err := e.core(role, map[string]bool{})
	if err != nil {
		return nil, err
	}
	r := &Result{SchemaVersion: 1, Branch: branch, SHA: tip, CoreVersion: core.String(), Major: core.Major, Minor: core.Minor, Patch: core.Patch, SemVer: core.String(), StableVersion: e.stable.String(), Snapshot: map[string]string{e.S.StableBranch: e.S.Refs[e.S.StableBranch], e.S.IntegrationBranch: e.S.Refs[e.S.IntegrationBranch], "target": tip}, ConfigHash: e.C.Hash, Dirty: e.S.Dirty, CountedEvents: []string{}, Releases: e.releases, Decisions: []string{}}
	for name, sha := range e.S.Refs {
		if name != e.S.StableBranch && name != e.S.IntegrationBranch && name != branch {
			r.Snapshot[name] = sha
		}
	}
	b := e.C.Branches[role]
	if !b.Prerelease.Enabled {
		return r, nil
	}
	r.PreReleaseLabel, err = label(b.Prerelease.Label, vars)
	if err != nil {
		return nil, err
	}
	for other := range e.S.Refs {
		if other == branch {
			continue
		}
		otherRole, otherVars, matchErr := e.role(other)
		if matchErr != nil || !e.C.Branches[otherRole].Prerelease.Enabled {
			continue
		}
		otherLabel, labelErr := label(e.C.Branches[otherRole].Prerelease.Label, otherVars)
		if labelErr != nil {
			return nil, labelErr
		}
		if otherLabel == r.PreReleaseLabel {
			return nil, fmt.Errorf("prerelease label collision: %s and %s both use %s", branch, other, r.PreReleaseLabel)
		}
	}
	counter := b.Prerelease.Counter
	num := counter.Initial
	r.PreReleaseNumber = &num
	var boundary string
	switch counter.Origin.Kind {
	case "release-boundary":
		boundary, err = e.developBoundary()
		if err == nil && len(e.releases) > 0 {
			r.Decisions = append(r.Decisions, "BOUNDARY: release consumed develop through "+boundary)
		}
	case "fork":
		boundary, err = e.workBoundary(tip)
	case "anchor":
		boundary = e.S.Anchor
		if counter.Origin.Ref != "" && counter.Origin.Ref != e.C.Version.Anchor {
			return nil, fmt.Errorf("counter anchor %s differs from version.anchor", counter.Origin.Ref)
		}
	}
	if err != nil {
		return nil, err
	}
	r.CycleBaseSHA = boundary
	chain, err := e.S.Chain(tip)
	if err != nil {
		return nil, err
	}
	start := -1
	for i, c := range chain {
		if c.SHA == boundary {
			start = i
			break
		}
	}
	if start < 0 {
		return nil, fmt.Errorf("counter boundary %s not on %s first-parent chain", boundary, branch)
	}
	for _, commit := range chain[start+1:] {
		ev := "commit"
		src := ""
		srcRole := ""
		if len(commit.Parents) == 2 {
			ev = "merge"
			src, srcRole, err = e.identity(commit, branch)
			if err != nil {
				if e.C.Defaults.UnknownMerge == "ignore" {
					r.Decisions = append(r.Decisions, "IGNORED: unknown merge "+commit.SHA)
					continue
				}
				return nil, fmt.Errorf("merge %s: %w", commit.SHA, err)
			}
			if counter.Origin.Kind == "fork" {
				if srcRole == e.integrationRole {
					r.Decisions = append(r.Decisions, "IGNORED: integration synchronization merge "+commit.SHA)
					continue
				}
				return nil, fmt.Errorf("unsupported history: %s merges %s into work branch", commit.SHA, src)
			}
		}
		rule, err := e.choose(counter.Rules, counter.UseRuleSets, ev, src, srcRole, branch, role, commit)
		if err != nil {
			return nil, err
		}
		if rule == nil {
			if ev == "commit" && e.C.Defaults.UnknownCommit == "error" {
				return nil, fmt.Errorf("no rule for commit %s", commit.SHA)
			}
			r.Decisions = append(r.Decisions, "IGNORED: "+ev+" "+commit.SHA)
			continue
		}
		for _, action := range rule.Actions {
			if action.IncrementCounter != nil {
				if num > int(^uint(0)>>1)-action.IncrementCounter.By {
					return nil, fmt.Errorf("counter overflow at %s", commit.SHA)
				}
				old := num
				num += action.IncrementCounter.By
				r.CountedEvents = append(r.CountedEvents, commit.SHA)
				r.Decisions = append(r.Decisions, fmt.Sprintf("COUNTED: %s %s, %s %d → %d", rule.ID, commit.SHA, r.PreReleaseLabel, old, num))
			} else if action.SetCounter != nil {
				num = action.SetCounter.Value
				r.Decisions = append(r.Decisions, fmt.Sprintf("SET: %s %d", rule.ID, num))
			} else if action.Fail != nil {
				return nil, fmt.Errorf("rule %s: %s", rule.ID, action.Fail.Message)
			} else if action.Ignore != nil {
				r.Decisions = append(r.Decisions, "IGNORED: "+rule.ID+" "+commit.SHA)
			} else {
				return nil, fmt.Errorf("rule %s: invalid counter action", rule.ID)
			}
		}
	}
	if len(e.releases) > 0 && role == e.integrationRole {
		r.Decisions = append(r.Decisions, fmt.Sprintf("RESET: current release cycle starts at %s %d", r.PreReleaseLabel, counter.Initial))
	}
	r.PreReleaseNumber = &num
	r.SemVer = fmt.Sprintf("%s-%s.%d", core, r.PreReleaseLabel, num)
	return r, nil
}
func SortedRoles(c *config.Config) []string {
	roles := make([]string, 0, len(c.Branches))
	for k := range c.Branches {
		roles = append(roles, k)
	}
	sort.Strings(roles)
	return roles
}
