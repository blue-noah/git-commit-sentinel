package gitcli

import (
	"errors"
	"strings"
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		raw                             string
		wantMajor, wantMinor, wantPatch int
		wantErr                         bool
	}{
		{"git version 2.43.0", 2, 43, 0, false},
		{"git version 2.9.0", 2, 9, 0, false},
		{"git version 2.43.0.windows.1", 2, 43, 0, false},
		{"git version 2.39.3 (Apple Git-145)", 2, 39, 3, false},
		{"not a version string", 0, 0, 0, true},
		{"", 0, 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			v, err := parseVersion(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseVersion(%q) = %+v, nil; want error", tt.raw, v)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseVersion(%q) unexpected error: %v", tt.raw, err)
			}
			if v.Major != tt.wantMajor || v.Minor != tt.wantMinor || v.Patch != tt.wantPatch {
				t.Errorf("parseVersion(%q) = %+v, want %d.%d.%d", tt.raw, v, tt.wantMajor, tt.wantMinor, tt.wantPatch)
			}
		})
	}
}

func TestVersionAtLeast(t *testing.T) {
	tests := []struct {
		v            Version
		major, minor int
		want         bool
	}{
		{Version{Major: 2, Minor: 9}, 2, 9, true},
		{Version{Major: 2, Minor: 10}, 2, 9, true},
		{Version{Major: 3, Minor: 0}, 2, 9, true},
		{Version{Major: 2, Minor: 8}, 2, 9, false},
		{Version{Major: 1, Minor: 99}, 2, 9, false},
	}
	for _, tt := range tests {
		if got := tt.v.AtLeast(tt.major, tt.minor); got != tt.want {
			t.Errorf("%+v.AtLeast(%d,%d) = %v, want %v", tt.v, tt.major, tt.minor, got, tt.want)
		}
	}
}

// Each test below fakes only the single seam (gitGet/gitSet/gitOutput) the
// function under test actually calls — not a bundled Runner — matching
// Interface Segregation: a test for ConfigSet has no reason to know
// anything about "get" or "output".

func withGet(t *testing.T, f getFunc) {
	t.Helper()
	original := gitGet
	gitGet = f
	t.Cleanup(func() { gitGet = original })
}

func withSet(t *testing.T, f setFunc) {
	t.Helper()
	original := gitSet
	gitSet = f
	t.Cleanup(func() { gitSet = original })
}

func withOutput(t *testing.T, f outputFunc) {
	t.Helper()
	original := gitOutput
	gitOutput = f
	t.Cleanup(func() { gitOutput = original })
}

func TestConfigGet(t *testing.T) {
	withGet(t, func(args []string) (string, bool, error) {
		if strings.Join(args, " ") == "config --get commitsentinel.types" {
			return "feat,fix", true, nil
		}
		return "", false, nil
	})

	value, ok, err := ConfigGet(ScopeEffective, "commitsentinel.types")
	if err != nil || !ok || value != "feat,fix" {
		t.Fatalf("ConfigGet = %q, %v, %v; want \"feat,fix\", true, nil", value, ok, err)
	}

	_, ok, err = ConfigGet(ScopeEffective, "commitsentinel.unset")
	if err != nil || ok {
		t.Fatalf("ConfigGet(unset) = _, %v, %v; want false, nil", ok, err)
	}
}

func TestConfigGetScoped(t *testing.T) {
	withGet(t, func(args []string) (string, bool, error) {
		switch strings.Join(args, " ") {
		case "config --global --get core.hooksPath":
			return "/global/hooks", true, nil
		case "config --local --get core.hooksPath":
			return "/local/hooks", true, nil
		}
		return "", false, nil
	})

	if v, ok, _ := ConfigGet(ScopeGlobal, "core.hooksPath"); !ok || v != "/global/hooks" {
		t.Errorf("global ConfigGet = %q, %v; want /global/hooks, true", v, ok)
	}
	if v, ok, _ := ConfigGet(ScopeLocal, "core.hooksPath"); !ok || v != "/local/hooks" {
		t.Errorf("local ConfigGet = %q, %v; want /local/hooks, true", v, ok)
	}
}

func TestConfigGetPropagatesFailure(t *testing.T) {
	withGet(t, func([]string) (string, bool, error) { return "", false, errors.New("boom") })
	if _, _, err := ConfigGet(ScopeEffective, "commitsentinel.types"); err == nil {
		t.Fatal("ConfigGet = nil error; want the underlying failure wrapped and returned")
	}
}

func TestConfigSetRequiresExplicitScope(t *testing.T) {
	if err := ConfigSet(ScopeEffective, "commitsentinel.types", "feat"); err == nil {
		t.Fatal("ConfigSet(ScopeEffective, ...) = nil; want an error")
	}
}

func TestConfigSetSuccess(t *testing.T) {
	var gotArgs []string
	withSet(t, func(args []string) error {
		gotArgs = args
		return nil
	})

	if err := ConfigSet(ScopeGlobal, "commitsentinel.types", "feat"); err != nil {
		t.Fatalf("ConfigSet() error = %v", err)
	}
	want := []string{"config", "--global", "commitsentinel.types", "feat"}
	if strings.Join(gotArgs, " ") != strings.Join(want, " ") {
		t.Errorf("ConfigSet() invoked git with %v, want %v", gotArgs, want)
	}
}

func TestConfigSetPropagatesFailure(t *testing.T) {
	withSet(t, func([]string) error { return errors.New("boom") })
	if err := ConfigSet(ScopeGlobal, "commitsentinel.types", "feat"); err == nil {
		t.Fatal("ConfigSet = nil; want an error")
	}
}

func TestInsideRepo(t *testing.T) {
	withOutput(t, func([]string) (string, error) { return "true", nil })
	if !InsideRepo() {
		t.Error("InsideRepo() = false; want true")
	}
}

func TestRepoRootAndGitDir(t *testing.T) {
	withOutput(t, func(args []string) (string, error) {
		switch strings.Join(args, " ") {
		case "rev-parse --show-toplevel":
			return "/repo", nil
		case "rev-parse --absolute-git-dir":
			return "/repo/.git", nil
		}
		return "", errors.New("unexpected args")
	})

	root, err := RepoRoot()
	if err != nil || root != "/repo" {
		t.Errorf("RepoRoot() = %q, %v; want /repo, nil", root, err)
	}
	dir, err := GitDir()
	if err != nil || dir != "/repo/.git" {
		t.Errorf("GitDir() = %q, %v; want /repo/.git, nil", dir, err)
	}
}

func TestRepoRootPropagatesFailure(t *testing.T) {
	withOutput(t, func([]string) (string, error) { return "", errors.New("not a repo") })
	if _, err := RepoRoot(); err == nil {
		t.Fatal("RepoRoot() = nil error; want the underlying failure wrapped and returned")
	}
}

func TestGitDirPropagatesFailure(t *testing.T) {
	withOutput(t, func([]string) (string, error) { return "", errors.New("not a repo") })
	if _, err := GitDir(); err == nil {
		t.Fatal("GitDir() = nil error; want the underlying failure wrapped and returned")
	}
}

func TestDetectVersion(t *testing.T) {
	withOutput(t, func([]string) (string, error) { return "git version 2.43.0", nil })
	v, err := DetectVersion()
	if err != nil {
		t.Fatalf("DetectVersion() error = %v", err)
	}
	if v.Major != 2 || v.Minor != 43 || v.Patch != 0 {
		t.Errorf("DetectVersion() = %+v, want 2.43.0", v)
	}
}

func TestDetectVersionPropagatesFailure(t *testing.T) {
	withOutput(t, func([]string) (string, error) { return "", errors.New("git not found") })
	if _, err := DetectVersion(); err == nil {
		t.Fatal("DetectVersion() = nil error; want the underlying failure wrapped and returned")
	}
}

func TestVersionString(t *testing.T) {
	v := Version{Raw: "git version 2.43.0"}
	if v.String() != "git version 2.43.0" {
		t.Errorf("Version.String() = %q, want %q", v.String(), "git version 2.43.0")
	}
}

func TestConfigSource(t *testing.T) {
	withGet(t, func(args []string) (string, bool, error) {
		if strings.Join(args, " ") == "config --local --get commitsentinel.types" {
			return "feat", true, nil
		}
		return "", false, nil
	})

	src := ConfigSource{Scope: ScopeLocal}
	v, ok, err := src.Get("commitsentinel.types")
	if err != nil || !ok || v != "feat" {
		t.Errorf("ConfigSource.Get = %q, %v, %v; want \"feat\", true, nil", v, ok, err)
	}
}
