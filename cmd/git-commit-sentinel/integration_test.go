package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"git-commit-sentinel/internal/gitcli"
)

// requireGit skips the test if git isn't on PATH.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found on PATH, skipping integration test")
	}
}

// sandboxHome points HOME and XDG_CONFIG_HOME at a fresh temp directory
// for the duration of the test (t.Setenv restores both on cleanup), so a
// "-scope global" setup can never write to the developer's real
// ~/.gitconfig or ~/.config.
func sandboxHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	return home
}

// tempRepo creates a scratch git repository and chdirs the test process
// into it, restoring the previous directory on cleanup.
func tempRepo(t *testing.T) string {
	t.Helper()
	requireGit(t)

	dir := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v (%s)", err, out)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	return dir
}

// fakeBinaryOnPath makes checkBinaryOnPath/warnIfBinaryMissing believe
// git-commit-sentinel is installed, isolating these tests from whether
// `go install` has actually been run in this environment.
func fakeBinaryOnPath(t *testing.T) {
	t.Helper()
	original := lookPath
	lookPath = func(string) (string, error) { return "/fake/bin/git-commit-sentinel", nil }
	t.Cleanup(func() { lookPath = original })
}

func writeCommitMsg(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "commit-msg")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSetupLocalThenDoctorThenHook(t *testing.T) {
	sandboxHome(t)
	fakeBinaryOnPath(t)
	repo := tempRepo(t)

	if code := runSetup([]string{"-scope", "local"}); code != 0 {
		t.Fatalf("runSetup(-scope local) = %d, want 0", code)
	}

	hookPath := filepath.Join(repo, ".git", "hooks", "commit-msg")
	if _, err := os.Stat(hookPath); err != nil {
		t.Fatalf("commit-msg hook not installed: %v", err)
	}

	// Idempotent: running it again with the same flags changes nothing.
	if code := runSetup([]string{"-scope", "local"}); code != 0 {
		t.Fatalf("second runSetup(-scope local) = %d, want 0", code)
	}

	if code := runDoctor(nil); code != 0 {
		t.Fatalf("runDoctor() = %d, want 0 after a local setup", code)
	}

	if code := runHook([]string{writeCommitMsg(t, "feat: add thing\n")}); code != 0 {
		t.Errorf("runHook(valid message) = %d, want 0", code)
	}
	if code := runHook([]string{writeCommitMsg(t, "not conventional\n")}); code != 1 {
		t.Errorf("runHook(invalid message) = %d, want 1", code)
	}
}

func TestSetupLocalRefusesForeignHookWithoutForce(t *testing.T) {
	sandboxHome(t)
	fakeBinaryOnPath(t)
	repo := tempRepo(t)

	hookPath := filepath.Join(repo, ".git", "hooks", "commit-msg")
	if err := os.MkdirAll(filepath.Dir(hookPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hookPath, []byte("#!/bin/sh\necho foreign\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	if code := runSetup([]string{"-scope", "local"}); code == 0 {
		t.Fatal("runSetup(-scope local) over a foreign hook = 0, want a rejected (non-zero) exit code")
	}

	if code := runSetup([]string{"-scope", "local", "-force"}); code != 0 {
		t.Fatalf("runSetup(-scope local, -force) = %d, want 0", code)
	}
}

func TestSetupGlobal(t *testing.T) {
	home := sandboxHome(t)
	fakeBinaryOnPath(t)
	tempRepo(t)

	if code := runSetup([]string{"-scope", "global"}); code != 0 {
		t.Fatalf("runSetup(-scope global) = %d, want 0", code)
	}

	hookPath := filepath.Join(home, ".config", "git-commit-sentinel", "hooks", "commit-msg")
	if _, err := os.Stat(hookPath); err != nil {
		t.Fatalf("global commit-msg hook not installed at %s: %v", hookPath, err)
	}

	if code := runDoctor(nil); code != 0 {
		t.Fatalf("runDoctor() = %d, want 0 after a global setup", code)
	}
}

func TestSetupWithRuleOverride(t *testing.T) {
	sandboxHome(t)
	fakeBinaryOnPath(t)
	tempRepo(t)

	if code := runSetup([]string{"-scope", "local", "-rule", "type=off"}); code != 0 {
		t.Fatalf("runSetup(-rule type=off) = %d, want 0", code)
	}

	if code := runHook([]string{writeCommitMsg(t, "wip: anything\n")}); code != 0 {
		t.Errorf("runHook() with type rule off = %d, want 0 (an unknown type should no longer be rejected)", code)
	}
}

func TestSetupRejectsInvalidScope(t *testing.T) {
	sandboxHome(t)
	if code := runSetup([]string{"-scope", "nonsense"}); code != 2 {
		t.Errorf("runSetup(-scope nonsense) = %d, want 2 (usage error)", code)
	}
}

func TestSetupWithTypesOverride(t *testing.T) {
	sandboxHome(t)
	fakeBinaryOnPath(t)
	tempRepo(t)

	if code := runSetup([]string{"-scope", "local", "-types", "task, chore"}); code != 0 {
		t.Fatalf("runSetup(-types ...) = %d, want 0", code)
	}

	if code := runHook([]string{writeCommitMsg(t, "task: anything\n")}); code != 0 {
		t.Errorf("runHook() with a custom -types list = %d, want 0 (task should now be allowed)", code)
	}
	if code := runHook([]string{writeCommitMsg(t, "feat: anything\n")}); code != 1 {
		t.Errorf("runHook() with a custom -types list = %d, want 1 (feat is no longer in the allowed list)", code)
	}
}

func TestSetupLocalOutsideRepo(t *testing.T) {
	sandboxHome(t)
	fakeBinaryOnPath(t)
	// Deliberately no tempRepo(t): run from a plain, non-repo directory.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	if code := runSetup([]string{"-scope", "local"}); code == 0 {
		t.Fatal("runSetup(-scope local) outside a git repository = 0, want a rejected (non-zero) exit code")
	}
}

func TestSetupGlobalRefusesToRepointWithoutForce(t *testing.T) {
	sandboxHome(t)
	fakeBinaryOnPath(t)
	tempRepo(t)

	firstDir := t.TempDir()
	if code := runSetup([]string{"-scope", "global", "-hooks-dir", firstDir}); code != 0 {
		t.Fatalf("first runSetup(-hooks-dir %s) = %d, want 0", firstDir, code)
	}

	secondDir := t.TempDir()
	if code := runSetup([]string{"-scope", "global", "-hooks-dir", secondDir}); code == 0 {
		t.Fatal("runSetup(-hooks-dir <different dir>) without -force = 0, want a rejected (non-zero) exit code")
	}

	if code := runSetup([]string{"-scope", "global", "-hooks-dir", secondDir, "-force"}); code != 0 {
		t.Fatalf("runSetup(-hooks-dir <different dir>, -force) = %d, want 0", code)
	}

	value, ok, err := gitcli.ConfigGet(gitcli.ScopeGlobal, "core.hooksPath")
	if err != nil || !ok || value != secondDir {
		t.Errorf("global core.hooksPath = %q, %v, %v; want %q, true, nil", value, ok, err, secondDir)
	}
}
