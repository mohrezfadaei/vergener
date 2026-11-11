package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func base(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "../..", "Vergener.yml")
}
func TestLoadAndInheritance(t *testing.T) {
	c, e := Load(base(t), "")
	if e != nil {
		t.Fatal(e)
	}
	for _, role := range []string{"main", "develop", "work"} {
		inc, e := c.Increment(role)
		if e != nil || inc != "Patch" {
			t.Fatalf("%s: %s %v", role, inc, e)
		}
	}
}
func TestCompositionAndProfile(t *testing.T) {
	dir := t.TempDir()
	original, e := os.ReadFile(base(t))
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(dir, "base.yml"), original, 0600); e != nil {
		t.Fatal(e)
	}
	override := "extends: [base.yml]\nprofiles:\n  minor:\n    branches:\n      main:\n        increment: Minor\n"
	path := filepath.Join(dir, "override.yml")
	if e = os.WriteFile(path, []byte(override), 0600); e != nil {
		t.Fatal(e)
	}
	c, e := Load(path, "minor")
	if e != nil {
		t.Fatal(e)
	}
	v, _ := c.Increment("work")
	if v != "Minor" {
		t.Fatal(v)
	}
	if _, e = Load(path, "missing"); e == nil {
		t.Fatal("unknown profile accepted")
	}
}
func TestObviousPriorityAmbiguity(t *testing.T) {
	c, err := Load(base(t), "")
	if err != nil {
		t.Fatal(err)
	}
	b := c.Branches["develop"]
	r := b.Prerelease.Counter.Rules[0]
	r.ID = "duplicate-condition"
	b.Prerelease.Counter.Rules = append(b.Prerelease.Counter.Rules, r)
	c.Branches["develop"] = b
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "identical conditions") {
		t.Fatalf("expected obvious ambiguity: %v", err)
	}
}

func TestUnknownRuleRole(t *testing.T) {
	c, err := Load(base(t), "")
	if err != nil {
		t.Fatal(err)
	}
	b := c.Branches["develop"]
	b.Prerelease.Counter.Rules[0].When.Source.Roles = []string{"missing"}
	c.Branches["develop"] = b
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "unknown condition role") {
		t.Fatalf("expected unresolved role: %v", err)
	}
}

func TestInvalidConfig(t *testing.T) {
	original, e := os.ReadFile(base(t))
	if e != nil {
		t.Fatal(e)
	}
	for name, change := range map[string]func(string) string{"unknown-key": func(s string) string { return s + "\nunsupported: true\n" }, "inheritance-cycle": func(s string) string { return strings.Replace(s, "inherits: main", "inherits: work", 1) }, "core-cycle": func(s string) string {
		return strings.Replace(s, "core: {source: releases}", "core: {source: branch-core, of: work}", 1)
	}, "invalid-increment": func(s string) string { return strings.Replace(s, "increment: Patch", "increment: Explode", 1) }} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yml")
			if e := os.WriteFile(path, []byte(change(string(original))), 0600); e != nil {
				t.Fatal(e)
			}
			if _, e := Load(path, ""); e == nil {
				t.Fatal("accepted invalid configuration")
			}
		})
	}
}
