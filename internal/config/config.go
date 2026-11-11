package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/mohrezfadaei/vergener/internal/semver"
)

type Config struct {
	SchemaVersion int `yaml:"schema-version" json:"schema-version"`
	Version       struct {
		Initial       string `yaml:"initial" json:"initial"`
		Anchor        string `yaml:"anchor" json:"anchor"`
		DevelopAnchor string `yaml:"develop-anchor" json:"develop-anchor,omitempty"`
	} `yaml:"version" json:"version"`
	Defaults struct {
		Increment     string `yaml:"increment" json:"increment"`
		UnknownCommit string `yaml:"unknown-commit" json:"unknown-commit"`
		UnknownMerge  string `yaml:"unknown-merge" json:"unknown-merge"`
	} `yaml:"defaults" json:"defaults"`
	Git struct {
		Traversal           string            `yaml:"traversal" json:"traversal"`
		SourceIdentity      string            `yaml:"source-identity" json:"source-identity"`
		MergeMessageFormats map[string]string `yaml:"merge-message-formats" json:"merge-message-formats"`
		History             struct {
			Shallow       string `yaml:"shallow" json:"shallow"`
			MissingParent string `yaml:"missing-parent" json:"missing-parent"`
			Octopus       string `yaml:"octopus" json:"octopus"`
			Squash        string `yaml:"squash" json:"squash"`
			Rebase        string `yaml:"rebase" json:"rebase"`
		} `yaml:"history" json:"history"`
	} `yaml:"git" json:"git"`
	Tags struct {
		Discovery string `yaml:"discovery" json:"discovery"`
		Create    bool   `yaml:"create" json:"create"`
	} `yaml:"tags" json:"tags"`
	Branches map[string]Branch `yaml:"branches" json:"branches"`
	RuleSets map[string][]Rule `yaml:"rule-sets" json:"rule-sets,omitempty"`
	Rules    struct {
		Evaluation string   `yaml:"evaluation" json:"evaluation"`
		ScopeOrder []string `yaml:"scope-order" json:"scope-order"`
		NoMatch    struct {
			Commit          string `yaml:"commit" json:"commit"`
			RecognizedMerge string `yaml:"recognized-merge" json:"recognized-merge"`
		} `yaml:"no-match" json:"no-match"`
		Shared []Rule `yaml:"shared" json:"shared"`
	} `yaml:"rules" json:"rules"`
	Output struct {
		DefaultFormat string   `yaml:"default-format" json:"default-format"`
		Metadata      []string `yaml:"metadata" json:"metadata"`
	} `yaml:"output" json:"output"`
	Validation struct {
		UnknownKeys           string `yaml:"unknown-keys" json:"unknown-keys"`
		AmbiguousBranchMatch  string `yaml:"ambiguous-branch-match" json:"ambiguous-branch-match"`
		AmbiguousRulePriority string `yaml:"ambiguous-rule-priority" json:"ambiguous-rule-priority"`
		InheritanceCycle      string `yaml:"inheritance-cycle" json:"inheritance-cycle"`
		LabelCollision        string `yaml:"label-collision" json:"label-collision"`
	} `yaml:"validation" json:"validation"`
	Hash string `yaml:"-" json:"-"`
}
type Branch struct {
	Match       Match      `yaml:"match" json:"match"`
	Priority    int        `yaml:"priority" json:"priority"`
	Inherits    string     `yaml:"inherits" json:"inherits,omitempty"`
	Increment   string     `yaml:"increment" json:"increment"`
	Core        Core       `yaml:"core" json:"core"`
	Prerelease  Prerelease `yaml:"prerelease" json:"prerelease"`
	Rules       []Rule     `yaml:"rules" json:"rules"`
	UseRuleSets []string   `yaml:"use-rule-sets" json:"use-rule-sets,omitempty"`
}
type Match struct {
	Exact string `yaml:"exact" json:"exact,omitempty"`
	Glob  string `yaml:"glob" json:"glob,omitempty"`
	Regex string `yaml:"regex" json:"regex,omitempty"`
}
type Core struct {
	Source    string `yaml:"source" json:"source"`
	Of        string `yaml:"of" json:"of,omitempty"`
	Increment string `yaml:"increment" json:"increment,omitempty"`
	Value     string `yaml:"value" json:"value,omitempty"`
	Pattern   string `yaml:"pattern" json:"pattern,omitempty"`
}
type Prerelease struct {
	Enabled bool    `yaml:"enabled" json:"enabled"`
	Label   string  `yaml:"label" json:"label,omitempty"`
	Counter Counter `yaml:"counter" json:"counter"`
}
type Counter struct {
	Initial     int      `yaml:"initial" json:"initial"`
	Origin      Origin   `yaml:"origin" json:"origin"`
	Rules       []Rule   `yaml:"rules" json:"rules"`
	UseRuleSets []string `yaml:"use-rule-sets" json:"use-rule-sets,omitempty"`
	Reset       Reset    `yaml:"reset" json:"reset"`
}
type Origin struct {
	Kind       string `yaml:"kind" json:"kind"`
	Branch     string `yaml:"branch" json:"branch,omitempty"`
	Projection string `yaml:"projection" json:"projection,omitempty"`
	Algorithm  string `yaml:"algorithm" json:"algorithm,omitempty"`
	Ref        string `yaml:"ref" json:"ref,omitempty"`
	Pattern    string `yaml:"pattern" json:"pattern,omitempty"`
}
type Reset struct {
	On                    string `yaml:"on" json:"on"`
	Branch                string `yaml:"branch" json:"branch"`
	RequiresSourceRole    string `yaml:"requires-source-role" json:"requires-source-role"`
	Value                 int    `yaml:"value" json:"value"`
	PreservePendingEvents bool   `yaml:"preserve-pending-events" json:"preserve-pending-events"`
}
type Rule struct {
	ID       string    `yaml:"id" json:"id"`
	Priority int       `yaml:"priority" json:"priority"`
	When     Condition `yaml:"when" json:"when"`
	Actions  []Action  `yaml:"actions" json:"actions"`
}
type Condition struct {
	Event    string            `yaml:"event" json:"event,omitempty"`
	Source   Selector          `yaml:"source" json:"source,omitempty"`
	Target   Selector          `yaml:"target" json:"target,omitempty"`
	Message  string            `yaml:"message" json:"message,omitempty"`
	Author   string            `yaml:"author" json:"author,omitempty"`
	Paths    []string          `yaml:"paths" json:"paths,omitempty"`
	Trailers map[string]string `yaml:"trailers" json:"trailers,omitempty"`
	All      []Condition       `yaml:"all" json:"all,omitempty"`
	Any      []Condition       `yaml:"any" json:"any,omitempty"`
	Not      *Condition        `yaml:"not" json:"not,omitempty"`
}
type Selector struct {
	Roles   []string `yaml:"roles" json:"roles,omitempty"`
	Pattern string   `yaml:"pattern" json:"pattern,omitempty"`
}
type Action struct {
	Release  *Release `yaml:"release" json:"release,omitempty"`
	BumpCore *struct {
		Increment string `yaml:"increment" json:"increment"`
	} `yaml:"bump-core" json:"bump-core,omitempty"`
	IncrementCounter *struct {
		By int `yaml:"by" json:"by"`
	} `yaml:"increment-counter" json:"increment-counter,omitempty"`
	SetCounter *struct {
		Value int `yaml:"value" json:"value"`
	} `yaml:"set-counter" json:"set-counter,omitempty"`
	Ignore *struct{} `yaml:"ignore" json:"ignore,omitempty"`
	Fail   *struct {
		Message string `yaml:"message" json:"message"`
	} `yaml:"fail" json:"fail,omitempty"`
}
type Release struct {
	Increment string `yaml:"increment" json:"increment"`
	Boundary  string `yaml:"boundary" json:"boundary"`
}

func merge(dst, src map[string]any) {
	for k, v := range src {
		if a, ok := dst[k].(map[string]any); ok {
			if b, ok := v.(map[string]any); ok {
				merge(a, b)
				continue
			}
		}
		dst[k] = v
	}
}
func load(path string, stack map[string]bool) (map[string]any, error) {
	abs, e := filepath.Abs(path)
	if e != nil {
		return nil, e
	}
	if stack[abs] {
		return nil, fmt.Errorf("extends cycle at %s", abs)
	}
	stack[abs] = true
	defer delete(stack, abs)
	raw, e := os.ReadFile(abs)
	if e != nil {
		return nil, e
	}
	var data map[string]any
	if e = yaml.Unmarshal(raw, &data); e != nil {
		return nil, e
	}
	if data == nil {
		return nil, fmt.Errorf("empty config %s", abs)
	}
	out := map[string]any{}
	if ext, ok := data["extends"]; ok {
		list, ok := ext.([]any)
		if !ok {
			return nil, fmt.Errorf("extends must be a list")
		}
		for _, p := range list {
			s, ok := p.(string)
			if !ok {
				return nil, fmt.Errorf("extends entry must be a path")
			}
			parent, e := load(filepath.Join(filepath.Dir(abs), s), stack)
			if e != nil {
				return nil, e
			}
			merge(out, parent)
		}
	}
	delete(data, "extends")
	merge(out, data)
	return out, nil
}
func Load(path, profile string) (*Config, error) {
	m, e := load(path, map[string]bool{})
	if e != nil {
		return nil, e
	}
	if profile != "" {
		p, ok := m["profiles"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("no profiles configured")
		}
		v, ok := p[profile].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("unknown profile %q", profile)
		}
		merge(m, v)
	}
	delete(m, "profiles")
	raw, e := yaml.Marshal(m)
	if e != nil {
		return nil, e
	}
	h := sha256.Sum256(raw)
	var c Config
	dec := yaml.NewDecoder(strings.NewReader(string(raw)))
	dec.KnownFields(true)
	if e = dec.Decode(&c); e != nil {
		return nil, fmt.Errorf("configuration: %w", e)
	}
	c.Hash = hex.EncodeToString(h[:])
	if e = c.Validate(); e != nil {
		return nil, e
	}
	return &c, nil
}
func validIncrement(s string) bool {
	return s == "Major" || s == "Minor" || s == "Patch" || s == "None" || s == "Inherit"
}
func validateTagPattern(role, pattern string, version bool) error {
	if pattern == "" { return fmt.Errorf("role %s: matching tag requires a regex pattern", role) }
	r, err := regexp.Compile(pattern)
	if err != nil { return fmt.Errorf("role %s: matching-tag pattern: %w",role,err) }
	if version && r.SubexpIndex("version") < 0 { return fmt.Errorf("role %s: core matching-tag pattern requires a named version group",role) }
	return nil
}

func (c *Config) WorkflowRoles() (string, string, error) {
	stable, integration := "", ""
	for role, b := range c.Branches {
		if b.Core.Source == "releases" {
			if stable != "" {
				return "", "", fmt.Errorf("multiple release-history roles: %s and %s", stable, role)
			}
			stable = role
		}
		if b.Prerelease.Enabled && b.Prerelease.Counter.Origin.Kind == "release-boundary" {
			if integration != "" {
				return "", "", fmt.Errorf("multiple release-boundary roles: %s and %s", integration, role)
			}
			integration = role
		}
	}
	if stable == "" || integration == "" {
		return "", "", fmt.Errorf("one release-history role and one release-boundary role are required")
	}
	if c.Branches[integration].Prerelease.Counter.Origin.Branch != stable {
		return "", "", fmt.Errorf("integration role %s must use release-history role %s", integration, stable)
	}
	return stable, integration, nil
}

func (c *Config) Increment(role string) (string, error) {
	seen := map[string]bool{}
	for role != "" {
		if seen[role] {
			return "", fmt.Errorf("increment inheritance cycle at %s", role)
		}
		seen[role] = true
		b, ok := c.Branches[role]
		if !ok {
			return "", fmt.Errorf("unknown inherited role %s", role)
		}
		if b.Increment != "" && b.Increment != "Inherit" {
			return b.Increment, nil
		}
		role = b.Inherits
	}
	return c.Defaults.Increment, nil
}
func (c *Config) Validate() error {
	if c.SchemaVersion != 1 {
		return fmt.Errorf("schema-version must be 1")
	}
	if c.Version.Initial == "" || c.Version.Anchor == "" {
		return fmt.Errorf("version.initial and version.anchor required")
	}
	if _, err := semver.Parse(c.Version.Initial); err != nil {
		return err
	}
	if strings.HasPrefix(c.Version.Anchor, "refs/tags/v") && strings.TrimPrefix(c.Version.Anchor, "refs/tags/v") != c.Version.Initial {
		return fmt.Errorf("anchor tag version does not match version.initial %s", c.Version.Initial)
	}
	if c.Validation.UnknownKeys != "error" || c.Validation.AmbiguousBranchMatch != "error" || c.Validation.AmbiguousRulePriority != "error" || c.Validation.InheritanceCycle != "error" || c.Validation.LabelCollision != "error" {
		return fmt.Errorf("validation policies must be error")
	}
	if !validIncrement(c.Defaults.Increment) || c.Defaults.Increment == "Inherit" {
		return fmt.Errorf("defaults.increment must be a concrete increment")
	}
	if c.Git.Traversal != "first-parent" || c.Git.SourceIdentity != "merge-message" {
		return fmt.Errorf("only first-parent traversal and merge-message source identity supported")
	}
	if c.Git.History.Shallow != "error" || c.Git.History.MissingParent != "error" || c.Git.History.Octopus != "error" || c.Git.History.Squash != "error" || c.Git.History.Rebase != "error" {
		return fmt.Errorf("history policies must be error (squash/rebase cannot be detected reliably)")
	}
	if (c.Tags.Discovery != "anchor-only" && c.Tags.Discovery != "matching") || c.Tags.Create {
		return fmt.Errorf("tags.discovery must be anchor-only or matching; tags.create must be false")
	}
	if c.Defaults.UnknownMerge != "error" && c.Defaults.UnknownMerge != "ignore" {
		return fmt.Errorf("unknown-merge must be error or ignore")
	}
	if c.Defaults.UnknownCommit != "ignore" && c.Defaults.UnknownCommit != "error" {
		return fmt.Errorf("unknown-commit must be error or ignore")
	}
	if c.Rules.Evaluation != "first-match" || len(c.Rules.ScopeOrder) != 2 || c.Rules.ScopeOrder[0] != "branch" || c.Rules.ScopeOrder[1] != "shared" {
		return fmt.Errorf("only first-match with scope-order [branch, shared] supported")
	}
	if c.Rules.NoMatch.Commit != "ignore" || c.Rules.NoMatch.RecognizedMerge != "ignore" {
		return fmt.Errorf("no-match policies must be ignore")
	}
	if c.Output.DefaultFormat != "json" && c.Output.DefaultFormat != "semver" {
		return fmt.Errorf("output.default-format must be json or semver")
	}
	if len(c.Output.Metadata) > 0 {
		return fmt.Errorf("output.metadata is not yet supported: use []")
	}
	for name, pattern := range c.Git.MergeMessageFormats {
		r, e := regexp.Compile(pattern)
		if e != nil {
			return fmt.Errorf("merge parser %s: %w", name, e)
		}
		if r.SubexpIndex("source") < 0 {
			return fmt.Errorf("merge parser %s needs named source group", name)
		}
	}
	if len(c.Git.MergeMessageFormats) == 0 {
		return fmt.Errorf("at least one merge parser required")
	}
	stableRole, integrationRole, workflowErr := c.WorkflowRoles()
	if workflowErr != nil {
		return workflowErr
	}
	for leftRole, left := range c.Branches {
		for rightRole, right := range c.Branches {
			if leftRole < rightRole && left.Priority == right.Priority && left.Match.Exact != "" && left.Match.Exact == right.Match.Exact {
				return fmt.Errorf("roles %s and %s match the same exact branch at priority %d", leftRole, rightRole, left.Priority)
			}
		}
	}
	for role, b := range c.Branches {
		n := 0
		for _, s := range []string{b.Match.Exact, b.Match.Glob, b.Match.Regex} {
			if s != "" {
				n++
			}
		}
		if n != 1 {
			return fmt.Errorf("role %s: exactly one matcher required", role)
		}
		if b.Match.Regex != "" {
			if _, e := regexp.Compile(b.Match.Regex); e != nil {
				return e
			}
		}
		if b.Increment != "" && !validIncrement(b.Increment) {
			return fmt.Errorf("role %s: invalid increment", role)
		}
		if _, e := c.Increment(role); e != nil {
			return e
		}
		switch b.Core.Source {
		case "releases":
		case "next-release", "branch-core", "matching-tag":
			if _, ok := c.Branches[b.Core.Of]; !ok { return fmt.Errorf("role %s: unknown core dependency %s", role, b.Core.Of) }
			if b.Core.Source == "matching-tag" {
				if c.Tags.Discovery != "matching" { return fmt.Errorf("role %s: matching-tag core requires tags.discovery: matching",role) }
				if err := validateTagPattern(role,b.Core.Pattern,true); err != nil { return err }
			}
		case "fixed":
			if _, err := semver.Parse(b.Core.Value); err != nil {
				return fmt.Errorf("role %s: %w", role, err)
			}
		default:
			return fmt.Errorf("role %s: unsupported core source %s", role, b.Core.Source)
		}
		if b.Core.Source != "next-release" && b.Core.Increment != "" {
			return fmt.Errorf("role %s: core.increment requires next-release", role)
		}
		if b.Core.Source != "fixed" && b.Core.Value != "" {
			return fmt.Errorf("role %s: core.value requires fixed", role)
		}
		if b.Core.Source != "next-release" && b.Core.Source != "branch-core" && b.Core.Source != "matching-tag" && b.Core.Of != "" {
			return fmt.Errorf("role %s: core.of is unused", role)
		}
		if b.Core.Source != "matching-tag" && b.Core.Pattern != "" { return fmt.Errorf("role %s: core.pattern requires matching-tag",role) }
		if b.Core.Increment != "" && !validIncrement(b.Core.Increment) {
			return fmt.Errorf("role %s: invalid core increment", role)
		}
		if !b.Prerelease.Enabled && (b.Prerelease.Label != "" || b.Prerelease.Counter.Origin.Kind != "" || len(b.Prerelease.Counter.Rules) > 0 || len(b.Prerelease.Counter.UseRuleSets) > 0 || b.Prerelease.Counter.Initial != 0 || b.Prerelease.Counter.Reset.On != "") {
			return fmt.Errorf("role %s: disabled prerelease cannot define counter or label", role)
		}
		if b.Prerelease.Enabled {
			if b.Prerelease.Label == "" {
				return fmt.Errorf("role %s: missing label", role)
			}
			o := b.Prerelease.Counter.Origin
			switch o.Kind {
			case "fork":
				if o.Algorithm != "common-first-parent" || o.Projection != "" || o.Ref != "" || o.Pattern != "" {
					return fmt.Errorf("role %s: unsupported fork algorithm", role)
				}
			case "release-boundary":
				if o.Projection != "released-source-tip" || o.Algorithm != "" || o.Ref != "" || o.Pattern != "" {
					return fmt.Errorf("role %s: unsupported projection", role)
				}
			case "anchor":
				if o.Ref != c.Version.Anchor || o.Branch != "" || o.Projection != "" || o.Algorithm != "" || o.Pattern != "" {
					return fmt.Errorf("role %s: anchor origin needs ref", role)
				}
			case "matching-tag":
				if c.Tags.Discovery != "matching" { return fmt.Errorf("role %s: matching-tag counter requires tags.discovery: matching",role) }
				if o.Ref != "" || o.Projection != "" || o.Algorithm != "" { return fmt.Errorf("role %s: unsupported matching-tag origin options",role) }
				if err := validateTagPattern(role,o.Pattern,false); err != nil { return err }
			default:
				return fmt.Errorf("role %s: unsupported counter origin %s", role, o.Kind)
			}
			if _, ok := c.Branches[o.Branch]; o.Kind != "anchor" && !ok {
				return fmt.Errorf("role %s: unknown origin branch %s", role, o.Branch)
			}
			r := b.Prerelease.Counter.Reset
			if r.On == "" && (r.Branch != "" || r.RequiresSourceRole != "" || r.Value != 0 || r.PreservePendingEvents) {
				return fmt.Errorf("role %s: reset fields require reset.on", role)
			}
			if r.On != "" && (o.Kind != "release-boundary" || r.On != "release" || r.Branch != o.Branch || r.RequiresSourceRole != role || !r.PreservePendingEvents || r.Value != b.Prerelease.Counter.Initial) {
				return fmt.Errorf("role %s: unsupported reset", role)
			}
			if o.Kind == "release-boundary" && (role != integrationRole || o.Branch != stableRole || r.On == "") {
				return fmt.Errorf("role %s: unsupported release boundary", role)
			}
			if o.Kind == "fork" && o.Branch != integrationRole {
				return fmt.Errorf("role %s: fork must use develop", role)
			}
			if b.Prerelease.Counter.Initial < 0 {
				return fmt.Errorf("role %s: negative initial counter", role)
			}
		}
		if role != stableRole && (len(b.Rules) > 0 || len(b.UseRuleSets) > 0) {
			return fmt.Errorf("role %s: branch event rules are currently supported only on main", role)
		}
		if e := c.validateRules(b.Rules, "branch"); e != nil {
			return e
		}
		if e := c.validateRules(b.Prerelease.Counter.Rules, "counter"); e != nil {
			return e
		}
		for _, set := range b.UseRuleSets {
			rules, ok := c.RuleSets[set]
			if !ok {
				return fmt.Errorf("unknown rule set %s", set)
			}
			if err := c.validateRules(rules, "branch"); err != nil {
				return err
			}
		}
		for _, set := range b.Prerelease.Counter.UseRuleSets {
			rules, ok := c.RuleSets[set]
			if !ok {
				return fmt.Errorf("unknown rule set %s", set)
			}
			if err := c.validateRules(rules, "counter"); err != nil {
				return err
			}
		}
	}
	if e := c.validateRules(c.Rules.Shared, "branch"); e != nil {
		return e
	}
	for name, rules := range c.RuleSets {
		used := false
		for _, branch := range c.Branches {
			for _, ref := range append(branch.UseRuleSets, branch.Prerelease.Counter.UseRuleSets...) {
				if ref == name {
					used = true
				}
			}
		}
		if !used {
			return fmt.Errorf("unused rule set %s", name)
		}
		_ = rules
	}
	visited := map[string]int{}
	var visit func(string) error
	visit = func(role string) error {
		if visited[role] == 1 {
			return fmt.Errorf("core dependency cycle at %s", role)
		}
		if visited[role] == 2 {
			return nil
		}
		visited[role] = 1
		b := c.Branches[role]
		if b.Core.Of != "" {
			if e := visit(b.Core.Of); e != nil {
				return e
			}
		}
		visited[role] = 2
		return nil
	}
	for role := range c.Branches {
		if e := visit(role); e != nil {
			return e
		}
	}
	return nil
}
func (c *Config) validateRoleReferences(w Condition) error {
	for _, role := range append(append([]string{}, w.Source.Roles...), w.Target.Roles...) {
		if _, ok := c.Branches[role]; !ok {
			return fmt.Errorf("unknown condition role %s", role)
		}
	}
	for _, child := range append(w.All, w.Any...) {
		if err := c.validateRoleReferences(child); err != nil {
			return err
		}
	}
	if w.Not != nil {
		return c.validateRoleReferences(*w.Not)
	}
	return nil
}

func (c *Config) validateRules(rules []Rule, scope string) error {
	ids := map[string]bool{}
	for i, r := range rules {
		for _, prev := range rules[:i] {
			if prev.Priority == r.Priority && reflect.DeepEqual(prev.When, r.When) {
				return fmt.Errorf("%s: rules %s and %s have identical conditions at priority %d", scope, prev.ID, r.ID, r.Priority)
			}
		}
		if r.ID == "" || ids[r.ID] {
			return fmt.Errorf("%s: missing or duplicate rule id %q", scope, r.ID)
		}
		ids[r.ID] = true
		if len(r.Actions) == 0 {
			return fmt.Errorf("rule %s: no actions", r.ID)
		}
		if e := validateCondition(r.When); e != nil {
			return fmt.Errorf("rule %s: %w", r.ID, e)
		}
		if e := c.validateRoleReferences(r.When); e != nil {
			return fmt.Errorf("rule %s: %w", r.ID, e)
		}
		for _, a := range r.Actions {
			n := 0
			if a.Release != nil {
				n++
				if scope == "counter" || a.Release.Boundary != "source-tip" || !validIncrement(a.Release.Increment) {
					return fmt.Errorf("rule %s: invalid release action", r.ID)
				}
			}
			if a.BumpCore != nil {
				n++
				if scope == "counter" || !validIncrement(a.BumpCore.Increment) {
					return fmt.Errorf("rule %s: invalid bump-core", r.ID)
				}
			}
			if a.IncrementCounter != nil {
				n++
				if scope != "counter" || a.IncrementCounter.By < 1 {
					return fmt.Errorf("rule %s: invalid increment-counter", r.ID)
				}
			}
			if a.SetCounter != nil {
				n++
				if scope != "counter" || a.SetCounter.Value < 0 {
					return fmt.Errorf("rule %s: invalid set-counter", r.ID)
				}
			}
			if a.Ignore != nil {
				n++
			}
			if a.Fail != nil {
				n++
			}
			if n != 1 {
				return fmt.Errorf("rule %s: action must specify exactly one operation", r.ID)
			}
		}
	}
	return nil
}
func validateCondition(w Condition) error {
	if w.Event != "" && w.Event != "merge" && w.Event != "commit" {
		return fmt.Errorf("unsupported event %s", w.Event)
	}
	for _, p := range []string{w.Message, w.Author, w.Source.Pattern, w.Target.Pattern} {
		if p != "" {
			if _, e := regexp.Compile(p); e != nil {
				return e
			}
		}
	}
	for _, p := range w.Paths {
		if _, e := filepath.Match(p, ""); e != nil {
			return e
		}
	}
	for _, p := range w.Trailers {
		if _, e := regexp.Compile(p); e != nil {
			return e
		}
	}
	for _, x := range append(w.All, w.Any...) {
		if e := validateCondition(x); e != nil {
			return e
		}
	}
	if w.Not != nil {
		return validateCondition(*w.Not)
	}
	return nil
}
func Sorted(r []Rule) []Rule {
	out := append([]Rule(nil), r...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Priority == out[j].Priority {
			return out[i].ID < out[j].ID
		}
		return out[i].Priority > out[j].Priority
	})
	return out
}
