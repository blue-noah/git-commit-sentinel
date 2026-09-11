package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"git-commit-sentinel/internal/hookfile"
)

// withFakeGit prepends a directory containing a fake `git` executable to
// PATH for the duration of the test, so code that shells out to git sees
// whatever --version output versionOutput specifies. The fake only
// understands --version: a test using this should only exercise code
// paths that call that (e.g. preflightGitVersion, checkGit), not ones
// that need a real repository.
func withFakeGit(t *testing.T, versionOutput string) {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"--version\" ]; then\n" +
		"  echo '" + versionOutput + "'\n" +
		"  exit 0\n" +
		"fi\n" +
		"echo \"fake git: unsupported invocation: $@\" >&2\n" +
		"exit 1\n"
	path := filepath.Join(dir, "git")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func gitConfigLocal(t *testing.T, key, value string) {
	t.Helper()
	if out, err := exec.Command("git", "config", "--local", key, value).CombinedOutput(); err != nil {
		t.Fatalf("git config --local %s %s: %v (%s)", key, value, err, out)
	}
}

// --- git version handling ---

func TestCheckGitFailsWhenGitMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // nothing named "git" anywhere on PATH
	results := checkGit()
	if len(results) != 1 || results[0].Level != levelFail {
		t.Errorf("checkGit() with no git on PATH = %+v, want a single FAIL", results)
	}
}

func TestCheckGitWarnsOnOldVersion(t *testing.T) {
	withFakeGit(t, "git version 2.1.0")
	results := checkGit()
	sawWarn := false
	for _, r := range results {
		if r.Level == levelFail {
			t.Errorf("checkGit() with git 2.1.0 produced a FAIL: %+v", r)
		}
		if r.Level == levelWarn {
			sawWarn = true
		}
	}
	if !sawWarn {
		t.Errorf("checkGit() with git 2.1.0 = %+v, want a WARN", results)
	}
}

func TestPreflightGitVersionPropagatesDetectFailure(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if err := preflightGitVersion("global"); err == nil {
		t.Fatal("preflightGitVersion() with no git on PATH = nil error; want a failure")
	}
}

func TestPreflightGitVersionRejectsOldGitGlobally(t *testing.T) {
	withFakeGit(t, "git version 2.1.0")
	if err := preflightGitVersion("global"); err == nil {
		t.Fatal(`preflightGitVersion("global") with git 2.1.0 = nil error; want rejection`)
	}
}

func TestPreflightGitVersionAllowsOldGitLocally(t *testing.T) {
	withFakeGit(t, "git version 2.1.0")
	if err := preflightGitVersion("local"); err != nil {
		t.Errorf(`preflightGitVersion("local") with git 2.1.0 error = %v, want nil`, err)
	}
}

func TestRunSetupRejectsOldGitForGlobalScope(t *testing.T) {
	sandboxHome(t)
	withFakeGit(t, "git version 2.1.0")
	if code := runSetup([]string{"-scope", "global"}); code == 0 {
		t.Fatal("runSetup(-scope global) with git 2.1.0 = 0, want a rejected exit code")
	}
}

// --- binary-on-PATH handling ---

func TestWarnIfBinaryMissingDoesNotPanicWhenAbsent(t *testing.T) {
	original := lookPath
	lookPath = func(string) (string, error) { return "", errors.New("not found") }
	defer func() { lookPath = original }()
	warnIfBinaryMissing() // just must not panic; it only prints a warning
}

func TestRunDoctorReturnsNonZeroWhenUnhealthy(t *testing.T) {
	sandboxHome(t)
	original := lookPath
	lookPath = func(string) (string, error) { return "", errors.New("not found") }
	defer func() { lookPath = original }()
	tempRepo(t)

	if code := runDoctor(nil); code != 1 {
		t.Errorf("runDoctor() with the binary missing from PATH = %d, want 1", code)
	}
}

// --- usage-error branches ---

func TestRunDoctorRejectsUnknownFlag(t *testing.T) {
	if code := runDoctor([]string{"-bogus"}); code != 2 {
		t.Errorf("runDoctor([-bogus]) = %d, want 2", code)
	}
}

func TestRunSetupRejectsUnknownFlag(t *testing.T) {
	if code := runSetup([]string{"-bogus"}); code != 2 {
		t.Errorf("runSetup([-bogus]) = %d, want 2", code)
	}
}

func TestRunHookRejectsUnknownFlag(t *testing.T) {
	if code := runHook([]string{"-bogus", "/tmp/x"}); code != 2 {
		t.Errorf("runHook([-bogus, ...]) = %d, want 2", code)
	}
}

func TestRunHookMissingFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	if code := runHook([]string{missing}); code != 2 {
		t.Errorf("runHook(missing file) = %d, want 2", code)
	}
}

// --- invalid configuration handling ---

func TestRunHookInvalidConfiguration(t *testing.T) {
	sandboxHome(t)
	tempRepo(t)
	gitConfigLocal(t, "commitsentinel.rules.type", "not-a-level")

	if code := runHook([]string{writeCommitMsg(t, "feat: x\n")}); code != 2 {
		t.Errorf("runHook() with an invalid configured level = %d, want 2", code)
	}
}

func TestRunDoctorInvalidConfiguration(t *testing.T) {
	sandboxHome(t)
	fakeBinaryOnPath(t)
	tempRepo(t)
	gitConfigLocal(t, "commitsentinel.rules.type", "not-a-level")

	if code := runDoctor(nil); code != 1 {
		t.Errorf("runDoctor() with an invalid configured level = %d, want 1", code)
	}
}

// --- doctor state scenarios ---

func TestDoctorOutsideRepo(t *testing.T) {
	sandboxHome(t)
	fakeBinaryOnPath(t)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	if code := runDoctor(nil); code != 0 {
		t.Errorf("runDoctor() outside a repository = %d, want 0", code)
	}
}

func TestDoctorPristineRepoNoHooksInstalled(t *testing.T) {
	sandboxHome(t)
	fakeBinaryOnPath(t)
	tempRepo(t)

	if code := runDoctor(nil); code != 0 {
		t.Errorf("runDoctor() in a repo with nothing installed = %d, want 0 (WARN only, no FAIL)", code)
	}
}

func TestDoctorLocalHookIgnoredWhenGlobalSet(t *testing.T) {
	sandboxHome(t)
	fakeBinaryOnPath(t)
	tempRepo(t)

	// global first, so setupLocal below sees it already set (exercising
	// its "note: global core.hooksPath is set..." branch too).
	if code := runSetup([]string{"-scope", "global"}); code != 0 {
		t.Fatalf("runSetup(-scope global) = %d, want 0", code)
	}
	if code := runSetup([]string{"-scope", "local"}); code != 0 {
		t.Fatalf("runSetup(-scope local) = %d, want 0", code)
	}

	if code := runDoctor(nil); code != 0 {
		t.Errorf("runDoctor() = %d, want 0 (local-shadowed-by-global is a WARN, not a FAIL)", code)
	}
}

func TestDoctorLocalOverrideRelativePath(t *testing.T) {
	sandboxHome(t)
	fakeBinaryOnPath(t)
	repo := tempRepo(t)

	relDir := "myhooks"
	absDir := filepath.Join(repo, relDir)
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(absDir, "commit-msg"), []byte(hookfile.Render()), 0o755); err != nil {
		t.Fatal(err)
	}
	gitConfigLocal(t, "core.hooksPath", relDir)

	if code := runDoctor(nil); code != 0 {
		t.Errorf("runDoctor() with a relative local core.hooksPath = %d, want 0", code)
	}
}

func TestSetupLocalWarnsWhenLocalOverrideDiffers(t *testing.T) {
	sandboxHome(t)
	fakeBinaryOnPath(t)
	tempRepo(t)

	gitConfigLocal(t, "core.hooksPath", "/somewhere/else")

	if code := runSetup([]string{"-scope", "local"}); code != 0 {
		t.Errorf("runSetup(-scope local) with a differing local override = %d, want 0 (a warning, not a failure)", code)
	}
}

func TestSetupGlobalRepeatWithSameDirIsUnchanged(t *testing.T) {
	sandboxHome(t)
	fakeBinaryOnPath(t)
	tempRepo(t)

	dir := t.TempDir()
	if code := runSetup([]string{"-scope", "global", "-hooks-dir", dir}); code != 0 {
		t.Fatalf("first runSetup(-hooks-dir %s) = %d, want 0", dir, code)
	}
	if code := runSetup([]string{"-scope", "global", "-hooks-dir", dir}); code != 0 {
		t.Errorf("repeat runSetup(-hooks-dir %s) = %d, want 0 (already set to the same value)", dir, code)
	}
}

// --- filesystem failure injection ---

func TestSetupGlobalMkdirAllFailure(t *testing.T) {
	sandboxHome(t)
	fakeBinaryOnPath(t)
	tempRepo(t)

	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	badDir := filepath.Join(blocker, "hooks") // blocker is a file, not a dir

	if code := runSetup([]string{"-scope", "global", "-hooks-dir", badDir}); code == 0 {
		t.Fatal("runSetup(-hooks-dir under a regular file) = 0, want a rejected exit code")
	}
}

func TestSetupLocalMkdirAllFailure(t *testing.T) {
	sandboxHome(t)
	fakeBinaryOnPath(t)
	repo := tempRepo(t)

	hooksDir := filepath.Join(repo, ".git", "hooks")
	if err := os.RemoveAll(hooksDir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hooksDir, []byte("blocker"), 0o644); err != nil {
		t.Fatal(err)
	}

	if code := runSetup([]string{"-scope", "local"}); code == 0 {
		t.Fatal("runSetup(-scope local) with .git/hooks blocked by a file = 0, want a rejected exit code")
	}
}

func TestInstallHookPathIsDirectory(t *testing.T) {
	dir := t.TempDir() // installHook's path argument, itself a directory
	if err := installHook(dir, keepForeign); err == nil {
		t.Fatal("installHook(path=directory) = nil error; want a failure")
	}
}

func TestInstallHookWriteFailure(t *testing.T) {
	// Parent directory doesn't exist, so the read reports "not exist"
	// (installHook proceeds to write) but the write itself then fails.
	path := filepath.Join(t.TempDir(), "missing-parent", "commit-msg")
	if err := installHook(path, keepForeign); err == nil {
		t.Fatal("installHook() with a missing parent directory = nil error; want a failure")
	}
}

// makeGitDirReadOnly removes write permission from repo's .git directory,
// restored on cleanup. `git config` writes via a lockfile it creates as a
// sibling of .git/config, so this — not chmod'ing the config file itself,
// which git's rename-based write survives — is what makes a write fail
// with "could not lock config file: Permission denied".
func makeGitDirReadOnly(t *testing.T, repo string) {
	t.Helper()
	gitDir := filepath.Join(repo, ".git")
	if err := os.Chmod(gitDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(gitDir, 0o755) })
}

func TestApplyOverridesTypesFailure(t *testing.T) {
	sandboxHome(t)
	fakeBinaryOnPath(t)
	repo := tempRepo(t)

	if code := runSetup([]string{"-scope", "local"}); code != 0 {
		t.Fatalf("runSetup(-scope local) = %d, want 0", code)
	}

	makeGitDirReadOnly(t, repo)

	if code := runSetup([]string{"-scope", "local", "-types", "task"}); code == 0 {
		t.Fatal("runSetup(-types ...) with a read-only .git directory = 0, want a rejected exit code")
	}
}

func TestApplyOverridesRuleFailure(t *testing.T) {
	sandboxHome(t)
	fakeBinaryOnPath(t)
	repo := tempRepo(t)

	if code := runSetup([]string{"-scope", "local"}); code != 0 {
		t.Fatalf("runSetup(-scope local) = %d, want 0", code)
	}

	makeGitDirReadOnly(t, repo)

	if code := runSetup([]string{"-scope", "local", "-rule", "type=off"}); code == 0 {
		t.Fatal("runSetup(-rule ...) with a read-only .git directory = 0, want a rejected exit code")
	}
}

func TestDefaultHooksDirPropagatesHomeDirFailure(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	if _, err := defaultHooksDir(); err == nil {
		t.Fatal("defaultHooksDir() with no HOME and no XDG_CONFIG_HOME = nil error; want a failure")
	}
}

func TestSetupGlobalPropagatesDefaultHooksDirFailure(t *testing.T) {
	fakeBinaryOnPath(t)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")

	// No -hooks-dir given, so setupGlobal must fall through to
	// defaultHooksDir(), which fails with neither XDG_CONFIG_HOME nor HOME
	// resolvable.
	if code := runSetup([]string{"-scope", "global"}); code == 0 {
		t.Fatal("runSetup(-scope global) with no HOME and no XDG_CONFIG_HOME = 0, want a rejected exit code")
	}
}

// corruptGlobalGitConfig writes syntactically invalid content to the
// sandboxed $HOME/.gitconfig, so any subsequent `git config --global`
// call — read or write — fails with git's own parse error, exercising the
// "unexpected failure" branches that a merely-unset key doesn't reach.
func corruptGlobalGitConfig(t *testing.T, home string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte("[bad syntax\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCheckGlobalHookFailsOnBrokenGlobalConfig(t *testing.T) {
	home := sandboxHome(t)
	corruptGlobalGitConfig(t, home)

	results := checkGlobalHook()
	if len(results) != 1 || results[0].Level != levelFail {
		t.Errorf("checkGlobalHook() with a broken global config = %+v, want a single FAIL", results)
	}
}

func TestSetupGlobalPropagatesConfigGetFailure(t *testing.T) {
	home := sandboxHome(t)
	fakeBinaryOnPath(t)
	corruptGlobalGitConfig(t, home)

	// installHook (writing under XDG_CONFIG_HOME) succeeds regardless;
	// it's the subsequent ConfigGet("core.hooksPath") that must fail.
	if code := runSetup([]string{"-scope", "global"}); code == 0 {
		t.Fatal("runSetup(-scope global) with a broken global config = 0, want a rejected exit code")
	}
}

func TestCheckLocalRepoFailsOnBrokenLocalConfig(t *testing.T) {
	sandboxHome(t)
	repo := tempRepo(t)
	if err := os.WriteFile(filepath.Join(repo, ".git", "config"), []byte("[bad syntax\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	results := checkLocalRepo()
	found := false
	for _, r := range results {
		if r.Level == levelFail {
			found = true
		}
	}
	if !found {
		t.Errorf("checkLocalRepo() with a broken local config = %+v, want a FAIL", results)
	}
}

func TestCheckLocalRepoFailsOnBrokenGlobalConfig(t *testing.T) {
	home := sandboxHome(t)
	tempRepo(t) // no local core.hooksPath override, so checkLocalRepo falls through to reading the global one
	corruptGlobalGitConfig(t, home)

	results := checkLocalRepo()
	found := false
	for _, r := range results {
		if r.Level == levelFail {
			found = true
		}
	}
	if !found {
		t.Errorf("checkLocalRepo() with a broken global config = %+v, want a FAIL", results)
	}
}

func TestSetupGlobalRefusesForeignHookWithoutForce(t *testing.T) {
	home := sandboxHome(t)
	fakeBinaryOnPath(t)
	tempRepo(t)

	hooksDir := filepath.Join(home, ".config", "git-commit-sentinel", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooksDir, "commit-msg"), []byte("#!/bin/sh\necho foreign\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	if code := runSetup([]string{"-scope", "global"}); code == 0 {
		t.Fatal("runSetup(-scope global) over a foreign hook = 0, want a rejected exit code")
	}
	if code := runSetup([]string{"-scope", "global", "-force"}); code != 0 {
		t.Fatalf("runSetup(-scope global, -force) = %d, want 0", code)
	}
}

// withReadOnlyGitConfigGlobal points GIT_CONFIG_GLOBAL at a fresh, valid
// (empty) config file inside a directory with write permission removed —
// so a read succeeds (nothing configured yet) but any write fails, since
// `git config --global` writes via a same-directory lockfile it can no
// longer create. Requires git >= 2.32 (GIT_CONFIG_GLOBAL support).
func withReadOnlyGitConfigGlobal(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	configFile := filepath.Join(dir, "gitconfig")
	if err := os.WriteFile(configFile, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	t.Setenv("GIT_CONFIG_GLOBAL", configFile)
}

func TestSetupGlobalPropagatesConfigSetFailure(t *testing.T) {
	sandboxHome(t)
	fakeBinaryOnPath(t)
	tempRepo(t)
	withReadOnlyGitConfigGlobal(t)

	if code := runSetup([]string{"-scope", "global"}); code == 0 {
		t.Fatal("runSetup(-scope global) unable to write core.hooksPath = 0, want a rejected exit code")
	}
}

func TestRealMainDispatchesSetupAndDoctor(t *testing.T) {
	if code := realMain([]string{"setup", "-bogus"}); code != 2 {
		t.Errorf(`realMain(["setup", "-bogus"]) = %d, want 2 (runSetup's own usage error)`, code)
	}
	if code := realMain([]string{"doctor", "-bogus"}); code != 2 {
		t.Errorf(`realMain(["doctor", "-bogus"]) = %d, want 2 (runDoctor's own usage error)`, code)
	}
}
