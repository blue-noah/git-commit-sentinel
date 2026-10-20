package main

import (
	"os"
	"path/filepath"
	"testing"

	"git-commit-sentinel/internal/gitcli"
)

func requireUnset(t *testing.T, scope gitcli.Scope, key string) {
	t.Helper()
	if v, isSet, err := gitcli.ConfigGet(scope, key); err != nil || isSet {
		t.Errorf("%s %s = %q, %v (err %v); want unset", scope, key, v, isSet, err)
	}
}

func requireGone(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("%s still exists (stat err %v)", path, err)
	}
}

func TestUninstallGlobalRemovesWhatSetupCreated(t *testing.T) {
	home := sandboxHome(t)
	fakeBinaryOnPath(t)
	tempRepo(t)

	if code := runSetup([]string{"-types", "feat,fix", "-rule", "type=warn"}); code != 0 {
		t.Fatalf("runSetup() = %d, want 0", code)
	}
	if code := runUninstall(nil); code != 0 {
		t.Fatalf("runUninstall() = %d, want 0", code)
	}

	requireUnset(t, gitcli.ScopeGlobal, "core.hooksPath")
	requireUnset(t, gitcli.ScopeGlobal, "commitsentinel.types")
	requireUnset(t, gitcli.ScopeGlobal, "commitsentinel.rules.type")
	requireGone(t, filepath.Join(home, ".config", "git-commit-sentinel"))

	if code := runUninstall(nil); code != 0 {
		t.Errorf("second runUninstall() = %d, want 0 when nothing is left", code)
	}
}

func TestUninstallGlobalKeepConfig(t *testing.T) {
	sandboxHome(t)
	fakeBinaryOnPath(t)
	tempRepo(t)

	if code := runSetup([]string{"-types", "feat,fix"}); code != 0 {
		t.Fatalf("runSetup() = %d, want 0", code)
	}
	if code := runUninstall([]string{"-keep-config"}); code != 0 {
		t.Fatalf("runUninstall(-keep-config) = %d, want 0", code)
	}

	requireUnset(t, gitcli.ScopeGlobal, "core.hooksPath")
	if v, _, _ := gitcli.ConfigGet(gitcli.ScopeGlobal, "commitsentinel.types"); v != "feat,fix" {
		t.Errorf("commitsentinel.types = %q after -keep-config, want feat,fix", v)
	}
}

func TestUninstallGlobalLeavesForeignHooksPath(t *testing.T) {
	sandboxHome(t)
	tempRepo(t)

	dir := t.TempDir()
	foreign := filepath.Join(dir, "commit-msg")
	if err := os.WriteFile(foreign, []byte("#!/bin/sh\nnpx husky run commit-msg \"$1\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := gitcli.ConfigSet(gitcli.ScopeGlobal, "core.hooksPath", dir); err != nil {
		t.Fatal(err)
	}

	if code := runUninstall(nil); code != 0 {
		t.Fatalf("runUninstall() = %d, want 0", code)
	}
	if v, _, _ := gitcli.ConfigGet(gitcli.ScopeGlobal, "core.hooksPath"); v != dir {
		t.Errorf("core.hooksPath = %q, want it left at %q", v, dir)
	}
	if _, err := os.Stat(foreign); err != nil {
		t.Errorf("foreign hook was removed: %v", err)
	}
}

func TestUninstallGlobalKeepsHooksPathWhileOtherHooksRemain(t *testing.T) {
	home := sandboxHome(t)
	fakeBinaryOnPath(t)
	tempRepo(t)

	if code := runSetup(nil); code != 0 {
		t.Fatalf("runSetup() = %d, want 0", code)
	}
	dir := filepath.Join(home, ".config", "git-commit-sentinel", "hooks")
	if err := os.WriteFile(filepath.Join(dir, "pre-commit"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	if code := runUninstall(nil); code != 0 {
		t.Fatalf("runUninstall() = %d, want 0", code)
	}
	requireGone(t, filepath.Join(dir, "commit-msg"))
	if v, _, _ := gitcli.ConfigGet(gitcli.ScopeGlobal, "core.hooksPath"); v != dir {
		t.Errorf("core.hooksPath = %q, want it kept at %q for the remaining hook", v, dir)
	}
}

func TestUninstallLocal(t *testing.T) {
	sandboxHome(t)
	fakeBinaryOnPath(t)
	repo := tempRepo(t)

	if code := runSetup([]string{"-scope", "local", "-types", "feat"}); code != 0 {
		t.Fatalf("runSetup(-scope local) = %d, want 0", code)
	}
	if err := gitcli.ConfigSet(gitcli.ScopeGlobal, "commitsentinel.types", "fix"); err != nil {
		t.Fatal(err)
	}

	if code := runUninstall([]string{"-scope", "local"}); code != 0 {
		t.Fatalf("runUninstall(-scope local) = %d, want 0", code)
	}
	requireGone(t, filepath.Join(repo, ".git", "hooks", "commit-msg"))
	requireUnset(t, gitcli.ScopeLocal, "commitsentinel.types")
	if v, _, _ := gitcli.ConfigGet(gitcli.ScopeGlobal, "commitsentinel.types"); v != "fix" {
		t.Errorf("global commitsentinel.types = %q, want it untouched by a local uninstall", v)
	}
}

func TestUninstallLocalLeavesForeignHook(t *testing.T) {
	sandboxHome(t)
	repo := tempRepo(t)

	hook := filepath.Join(repo, ".git", "hooks", "commit-msg")
	if err := os.MkdirAll(filepath.Dir(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hook, []byte("#!/bin/sh\necho mine\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	if code := runUninstall([]string{"-scope", "local"}); code != 0 {
		t.Fatalf("runUninstall(-scope local) = %d, want 0", code)
	}
	if _, err := os.Stat(hook); err != nil {
		t.Errorf("foreign local hook was removed: %v", err)
	}
}

func TestUninstallLocalOutsideRepo(t *testing.T) {
	sandboxHome(t)
	requireGit(t)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	if code := runUninstall([]string{"-scope", "local"}); code != 1 {
		t.Errorf("runUninstall(-scope local) outside a repo = %d, want 1", code)
	}
}

func TestUninstallRejectsBadArguments(t *testing.T) {
	if code := runUninstall([]string{"-scope", "system"}); code != 2 {
		t.Errorf("runUninstall(-scope system) = %d, want 2", code)
	}
	if code := runUninstall([]string{"-bogus"}); code != 2 {
		t.Errorf("runUninstall(-bogus) = %d, want 2", code)
	}
}
