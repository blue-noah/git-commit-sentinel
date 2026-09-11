package conventionalcommit

import (
	"testing"

	"git-commit-sentinel/internal/config"
)

func hasRule(findings []Finding, rule string) bool {
	for _, f := range findings {
		if f.Rule == rule {
			return true
		}
	}
	return false
}

func TestValidateDefaults(t *testing.T) {
	cfg := config.Config{} // no overrides: every rule falls back to its own DefaultLevel

	tests := []struct {
		name       string
		msg        string
		wantErrors bool
		wantRule   string // rule expected to fire, "" if none
	}{
		{"valid feat", "feat: add login endpoint", false, ""},
		{"valid fix with scope", "fix(auth): handle expired tokens", false, ""},
		{"valid breaking change", "feat(api)!: remove deprecated field", false, ""},
		{"valid with body", "fix: correct off-by-one error\n\nThis fixes issue #42.", false, ""},
		{"missing colon", "feat add login endpoint", true, "format"},
		{"unknown type", "wip: work in progress", true, "type"},
		{"empty description", "feat: ", true, "description-empty"},
		{"trailing period", "fix: correct the bug.", false, "description-period"}, // warn, not error
		{"missing blank line before body", "fix: correct bug\nsee also #1", false, "body-blank-line"},
		{"empty message", "", true, "format"},

		// Edge cases: parses/matches correctly, but easy to get subtly wrong.
		{"uppercase type is rejected (case-sensitive match)", "Feat: add thing", true, "type"},
		{"empty scope is a format violation, not accepted as no scope", "feat(): add thing", true, "format"},
		{"breaking change without a scope is valid", "feat!: remove deprecated field", false, ""},
		{"CRLF line endings are normalized like LF", "feat: add thing\r\n\r\nbody line", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := Validate(tt.msg, cfg)
			if got := HasErrors(findings); got != tt.wantErrors {
				t.Errorf("HasErrors(%q) = %v, findings = %+v; want %v", tt.msg, got, findings, tt.wantErrors)
			}
			if tt.wantRule != "" && !hasRule(findings, tt.wantRule) {
				t.Errorf("Validate(%q) = %+v; want rule %q to fire", tt.msg, findings, tt.wantRule)
			}
		})
	}
}

func TestValidateCustomTypes(t *testing.T) {
	cfg := config.Config{Types: []string{"task"}}

	if findings := Validate("task: do something", cfg); HasErrors(findings) {
		t.Errorf("expected custom type to be allowed, got %+v", findings)
	}
	if findings := Validate("feat: add thing", cfg); !HasErrors(findings) {
		t.Errorf("expected default type to be rejected when custom types are set")
	}
}

func TestValidateRuleOff(t *testing.T) {
	cfg := config.Config{Rules: map[string]config.Level{"type": config.LevelOff}}

	findings := Validate("wip: work in progress", cfg)
	if hasRule(findings, "type") {
		t.Errorf("expected type rule to be silenced when off, got %+v", findings)
	}
}

func TestValidateRuleErrorEscalation(t *testing.T) {
	cfg := config.Config{Rules: map[string]config.Level{"description-period": config.LevelError}}

	findings := Validate("fix: correct the bug.", cfg)
	if !HasErrors(findings) {
		t.Errorf("expected description-period to be an error once escalated, got %+v", findings)
	}
}

func TestValidateHeaderLength(t *testing.T) {
	max := 20
	cfg := config.Config{HeaderMaxLength: &max}

	findings := Validate("feat: this description is definitely too long", cfg)
	if !hasRule(findings, "header-length") {
		t.Errorf("expected header-length rule to fire, got %+v", findings)
	}
}

func TestValidateHeaderLengthBoundary(t *testing.T) {
	max := len("feat: exactly twenty") // 21 characters — deliberately not a round number
	cfg := config.Config{HeaderMaxLength: &max}

	atLimit := "feat: exactly twenty"
	if hasRule(Validate(atLimit, cfg), "header-length") {
		t.Errorf("header of exactly %d characters triggered header-length; a header AT the limit should pass", max)
	}

	overLimit := atLimit + "1"
	if !hasRule(Validate(overLimit, cfg), "header-length") {
		t.Errorf("header of %d characters (one over the limit) did not trigger header-length", max+1)
	}
}

func TestValidateHeaderLengthCountsRunesNotBytes(t *testing.T) {
	// "feat: 🎉🎉🎉" is 9 runes but, since each emoji is a 4-byte UTF-8
	// sequence, 18 bytes. A byte-based length check would wrongly flag
	// this as exceeding a limit of, say, 10 — a rune-based one must not.
	header := "feat: 🎉🎉🎉"
	if runes := len([]rune(header)); runes != 9 {
		t.Fatalf("test fixture assumption broken: %q has %d runes, want 9", header, runes)
	}
	if bytes := len(header); bytes <= 9 {
		t.Fatalf("test fixture assumption broken: %q has %d bytes, want more than 9 (i.e. more than its rune count)", header, bytes)
	}

	max := 10
	cfg := config.Config{HeaderMaxLength: &max}

	if hasRule(Validate(header, cfg), "header-length") {
		t.Errorf("Validate(%q) with headerMaxLength=%d triggered header-length; "+
			"the header has 9 runes (under the limit) even though it has more than 9 bytes", header, max)
	}
}

func TestRuleRegistryIsPopulated(t *testing.T) {
	// A regression guard for the plugin registry itself: every rule file's
	// init() must actually have run and registered exactly one Rule.
	want := []string{"format", "type", "description-empty", "description-period", "header-length", "body-blank-line"}
	got := RuleNames()
	if len(got) != len(want) {
		t.Fatalf("RuleNames() = %v (%d rules); want %d rules", got, len(got), len(want))
	}
	for _, name := range want {
		found := false
		for _, g := range got {
			if g == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("RuleNames() = %v; missing %q", got, name)
		}
	}
}
