package gitcli

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// requireGit skips the test if git isn't on PATH — these tests exercise
// the real exec* implementations against an actual git process.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found on PATH, skipping integration test")
	}
}

// tempRepo creates a scratch git repository and chdirs the test process
// into it, restoring the previous directory on cleanup — so execGet's
// "config --local" and friends operate on a throwaway repo, never on
// anything real.
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

func TestExecOutputRunsRealGit(t *testing.T) {
	requireGit(t)

	out, err := execOutput([]string{"--version"})
	if err != nil {
		t.Fatalf("execOutput([--version]) error = %v", err)
	}
	if !strings.HasPrefix(out, "git version") {
		t.Errorf("execOutput([--version]) = %q, want it to start with \"git version\"", out)
	}
}

func TestExecGetAndExecSetRoundTrip(t *testing.T) {
	tempRepo(t)

	// Unset before anything is written.
	_, ok, err := execGet([]string{"config", "--local", "--get", "commitsentinel.testkey"})
	if err != nil {
		t.Fatalf("execGet(unset key) error = %v", err)
	}
	if ok {
		t.Fatal("execGet(unset key) ok = true; want false before the key is ever set")
	}

	if err := execSet([]string{"config", "--local", "commitsentinel.testkey", "hello"}); err != nil {
		t.Fatalf("execSet() error = %v", err)
	}

	value, ok, err := execGet([]string{"config", "--local", "--get", "commitsentinel.testkey"})
	if err != nil {
		t.Fatalf("execGet(set key) error = %v", err)
	}
	if !ok || value != "hello" {
		t.Fatalf("execGet(set key) = %q, %v; want \"hello\", true", value, ok)
	}
}

func TestExecGetPropagatesUnexpectedFailure(t *testing.T) {
	dir := tempRepo(t)

	// A syntactically broken config file makes git exit 128 ("fatal: bad
	// config line ..."), not the exit-code-1 "unset key" case execGet
	// treats specially — this must come back as a real error, not ok=false.
	configPath := dir + "/.git/config"
	existing, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	broken := append(existing, []byte("[bad syntax\n")...)
	if err := os.WriteFile(configPath, broken, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := execGet([]string{"config", "--local", "--get", "core.hooksPath"}); err == nil {
		t.Fatal("execGet with a broken config file = nil error; want git's fatal error propagated")
	}
}

func TestExecOutputPropagatesFailure(t *testing.T) {
	requireGit(t)
	if _, err := execOutput([]string{"not-a-real-git-subcommand"}); err == nil {
		t.Fatal("execOutput(bogus subcommand) = nil error; want git's failure propagated")
	}
}

func TestExecSetPropagatesRealGitFailure(t *testing.T) {
	tempRepo(t)

	// A key with no section (no ".") is invalid syntax — git rejects it
	// without writing anything, confined to this scratch repo's own
	// .git/config since the scope is --local.
	if err := execSet([]string{"config", "--local", "not-a-valid-key", "x"}); err == nil {
		t.Fatal("execSet(invalid key) = nil error; want git itself to reject it")
	}
}
