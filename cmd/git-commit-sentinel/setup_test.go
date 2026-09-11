package main

import (
	"testing"

	"git-commit-sentinel/internal/config"
)

func TestOverwritePolicyFrom(t *testing.T) {
	if overwritePolicyFrom(true) != forceOverwrite {
		t.Error("overwritePolicyFrom(true) != forceOverwrite")
	}
	if overwritePolicyFrom(false) != keepForeign {
		t.Error("overwritePolicyFrom(false) != keepForeign")
	}
}

func TestRuleLevelFlagsSetValid(t *testing.T) {
	var flags ruleLevelFlags
	if err := flags.Set("type=warn"); err != nil {
		t.Fatalf("Set(\"type=warn\") error = %v", err)
	}
	want := ruleLevelFlag{name: "type", level: config.LevelWarn}
	if len(flags) != 1 || flags[0] != want {
		t.Errorf("flags = %+v, want [%+v]", flags, want)
	}
}

func TestRuleLevelFlagsSetRejectsUnknownRule(t *testing.T) {
	var flags ruleLevelFlags
	if err := flags.Set("not-a-rule=warn"); err == nil {
		t.Fatal("Set(unknown rule) = nil error; want an error")
	}
}

func TestRuleLevelFlagsSetRejectsInvalidLevel(t *testing.T) {
	var flags ruleLevelFlags
	if err := flags.Set("type=maybe"); err == nil {
		t.Fatal("Set(invalid level) = nil error; want an error")
	}
}

func TestRuleLevelFlagsSetRejectsMissingEquals(t *testing.T) {
	var flags ruleLevelFlags
	if err := flags.Set("type"); err == nil {
		t.Fatal("Set(no '=') = nil error; want an error")
	}
}

func TestDefaultHooksDirUsesXDGConfigHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/xdg-home")
	dir, err := defaultHooksDir()
	if err != nil {
		t.Fatalf("defaultHooksDir() error = %v", err)
	}
	if want := "/xdg-home/git-commit-sentinel/hooks"; dir != want {
		t.Errorf("defaultHooksDir() = %q, want %q", dir, want)
	}
}

func TestDefaultHooksDirFallsBackToHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/home-fallback")
	dir, err := defaultHooksDir()
	if err != nil {
		t.Fatalf("defaultHooksDir() error = %v", err)
	}
	if want := "/home-fallback/.config/git-commit-sentinel/hooks"; dir != want {
		t.Errorf("defaultHooksDir() = %q, want %q", dir, want)
	}
}

func TestInstallForScopeRejectsUnknownScope(t *testing.T) {
	if _, err := installForScope("bogus", "", keepForeign); err == nil {
		t.Fatal(`installForScope("bogus", ...) = nil error; want an error`)
	}
}
