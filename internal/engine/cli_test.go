package engine_test

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCLIRealRepository(t *testing.T) {
	dir, _ := setup(t)
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "../..")
	binary := filepath.Join(t.TempDir(), "vergener")
	build := exec.Command("go", "build", "-o", binary, "./cmd/vergener")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, out)
	}
	cfg := filepath.Join(root, "Vergener.yml")
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(binary, args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("CLI %v: %v %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	check := func(branch, want string) {
		t.Helper()
		got := run("version", "--branch", branch, "--config", cfg, "--format", "semver")
		t.Logf("CLI: %s => %s (expected %s)", branch, got, want)
		if got != want {
			t.Fatalf("CLI: %s: got %s, want %s", branch, got, want)
		}
	}
	check("main", "1.0.0")
	check("develop", "1.0.1-beta.0")
	var result map[string]any
	if err := json.Unmarshal([]byte(run("version", "--branch", "develop", "--config", cfg)), &result); err != nil {
		t.Fatal(err)
	}
	if result["coreVersion"] != "1.0.1" || result["stableVersion"] != "1.0.0" {
		t.Fatal(result)
	}
	if !strings.Contains(run("explain", "--branch", "develop", "--config", cfg), "cycle boundary:") {
		t.Fatal("missing explanation")
	}
	command(t, dir, "switch", "develop")
	command(t, dir, "switch", "-c", "ci/build")
	for i := 0; i < 7; i++ {
		commit(t, dir, "ci")
	}
	check("ci/build", "1.0.1-ci-build.7")
	command(t, dir, "switch", "develop")
	command(t, dir, "switch", "-c", "feat/login")
	for i := 0; i < 20; i++ {
		commit(t, dir, "feature")
	}
	command(t, dir, "switch", "develop")
	merge(t, dir, "feat/login")
	check("develop", "1.0.1-beta.1")
	for i := 0; i < 3; i++ {
		commit(t, dir, "flux")
	}
	check("develop", "1.0.1-beta.1")
	merge(t, dir, "ci/build")
	check("develop", "1.0.1-beta.2")
	command(t, dir, "switch", "-c", "fix/example")
	commit(t, dir, "fix")
	command(t, dir, "switch", "develop")
	merge(t, dir, "fix/example")
	check("develop", "1.0.1-beta.3")
	command(t, dir, "switch", "main")
	merge(t, dir, "develop")
	check("main", "1.0.1")
	check("develop", "1.0.2-beta.0")
	command(t, dir, "switch", "develop")
	merge(t, dir, "main")
	check("develop", "1.0.2-beta.0")
	command(t, dir, "switch", "-c", "feat/next")
	commit(t, dir, "next")
	command(t, dir, "switch", "develop")
	merge(t, dir, "feat/next")
	check("develop", "1.0.2-beta.1")
	var final map[string]any
	if err := json.Unmarshal([]byte(run("version", "--branch", "develop", "--config", cfg)), &final); err != nil {
		t.Fatal(err)
	}
	if events, ok := final["countedEvents"].([]any); !ok || len(events) != 1 {
		t.Fatalf("next-cycle counted events: %v", final["countedEvents"])
	}
	if !strings.Contains(run("explain", "--branch", "develop", "--config", cfg), "BOUNDARY:") {
		t.Fatal("explain omits release boundary")
	}
}
