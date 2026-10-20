package hookfile

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRenderIsManaged(t *testing.T) {
	content := Render("/usr/local/bin/git-commit-sentinel")
	if !IsManaged(content) {
		t.Error("IsManaged(Render(...)) = false; want true")
	}
}

func TestIsManagedRejectsForeignHooks(t *testing.T) {
	foreign := "#!/bin/sh\nnpx husky run commit-msg \"$1\"\n"
	if IsManaged(foreign) {
		t.Error("IsManaged(foreign hook) = true; want false")
	}
}

func TestShellQuote(t *testing.T) {
	cases := map[string]string{
		"/usr/local/bin/x":      `'/usr/local/bin/x'`,
		"/a dir/x":              `'/a dir/x'`,
		"/it's/x":               `'/it'\''s/x'`,
		"C:/Users/me/bin/x.exe": `'C:/Users/me/bin/x.exe'`,
	}
	for in, want := range cases {
		if got := shellQuote(in); got != want {
			t.Errorf("shellQuote(%q) = %s; want %s", in, got, want)
		}
	}
}

// writeFakeBinary writes a script that prints its name and arguments,
// standing in for git-commit-sentinel.
func writeFakeBinary(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\necho \"ran $0 $*\"\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

// runHook executes content as the commit-msg hook with sh and PATH set to
// path, returning its combined output.
func runHook(t *testing.T, content, path string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("executes the hook with a POSIX sh and a shell-script fake binary")
	}
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not found on PATH")
	}
	hook := filepath.Join(t.TempDir(), "commit-msg")
	if err := os.WriteFile(hook, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(sh, hook, "MSGFILE")
	cmd.Env = []string{"PATH=" + path}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("hook failed: %v (%s)", err, out)
	}
	return string(out)
}

func TestRenderRunsBinaryByAbsolutePath(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "a dir", "it's here", BinaryName)
	writeFakeBinary(t, bin)

	out := runHook(t, Render(bin), "/usr/bin:/bin")
	want := "ran " + bin + " hook MSGFILE"
	if strings.TrimSpace(out) != want {
		t.Errorf("hook output = %q; want %q", out, want)
	}
}

func TestRenderFallsBackToPathWhenBinaryIsGone(t *testing.T) {
	dir := t.TempDir()
	writeFakeBinary(t, filepath.Join(dir, BinaryName))

	out := runHook(t, Render("/nonexistent/"+BinaryName), dir+":/usr/bin:/bin")
	want := "ran " + filepath.Join(dir, BinaryName) + " hook MSGFILE"
	if strings.TrimSpace(out) != want {
		t.Errorf("hook output = %q; want %q", out, want)
	}
}
