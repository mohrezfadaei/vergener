package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mohrezfadaei/vergener/internal/config"
	"github.com/mohrezfadaei/vergener/internal/engine"
	"github.com/mohrezfadaei/vergener/internal/git"
)

func main() {
	if err := execute(); err != nil {
		fmt.Fprintln(os.Stderr, "vergener:", err)
		os.Exit(1)
	}
}
func execute() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: vergener <version|explain|validate|config> [--config path] [--branch name] [--profile name] [--format json|semver]")
	}
	cmd := os.Args[1]
	if cmd != "version" && cmd != "explain" && cmd != "validate" && cmd != "config" {
		return fmt.Errorf("unknown command %q", cmd)
	}
	f := flag.NewFlagSet(cmd, flag.ContinueOnError)
	path := f.String("config", "", "configuration path (default repository root/Vergener.yml)")
	branch := f.String("branch", "", "target branch (default current branch)")
	profile := f.String("profile", "", "named configuration profile")
	format := f.String("format", "", "json or semver")
	effective := f.Bool("effective", false, "print effective configuration")
	schema := f.Bool("schema", false, "print JSON Schema")
	if err := f.Parse(os.Args[2:]); err != nil {
		return err
	}
	if cmd == "config" && *schema {
		fmt.Println(config.Schema)
		return nil
	}
	root, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return fmt.Errorf("not in a Git repository: %w", err)
	}
	dir := strings.TrimSpace(string(root))
	if *path == "" {
		*path = dir + "/Vergener.yml"
	}
	c, err := config.Load(*path, *profile)
	if err != nil {
		return err
	}
	if cmd == "validate" {
		fmt.Printf("valid configuration (sha256 %s)\n", c.Hash)
		return nil
	}
	if cmd == "config" {
		if !*effective {
			return fmt.Errorf("use config --effective or config --schema")
		}
		return json.NewEncoder(os.Stdout).Encode(c)
	}
	if *branch == "" {
		b, err := exec.Command("git", "symbolic-ref", "--quiet", "--short", "HEAD").Output()
		if err != nil {
			return fmt.Errorf("detached HEAD: specify --branch")
		}
		*branch = strings.TrimSpace(string(b))
	}
	s, err := git.Capture(context.Background(), dir, c, *branch)
	if err != nil {
		return err
	}
	r, err := engine.New(c, s).Evaluate(*branch)
	if err != nil {
		return err
	}
	if cmd == "explain" {
		fmt.Printf("%s at %s: %s\nconfig: %s\n%s: %s\n%s: %s\ntarget: %s\nanchor: %s\ncycle boundary: %s\n", r.Branch, r.SHA, r.SemVer, r.ConfigHash, s.StableBranch, s.Refs[s.StableBranch], s.IntegrationBranch, s.Refs[s.IntegrationBranch], r.SHA, s.Anchor, r.CycleBaseSHA)
		for _, d := range r.Decisions {
			fmt.Println(d)
		}
		return nil
	}
	out := *format
	if out == "" {
		out = c.Output.DefaultFormat
	}
	switch out {
	case "semver":
		fmt.Println(r.SemVer)
		return nil
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(r)
	default:
		return fmt.Errorf("unknown format %s", out)
	}
}
