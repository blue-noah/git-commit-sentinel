package config

import (
	"errors"
	"reflect"
	"testing"
)

// fakeSource is a minimal config.Source backed by a plain map, keyed
// directly by config key names. Because Load depends only on the Source
// interface, this package needs no knowledge of gitcli (or any other
// concrete backing store) to test itself.
type fakeSource map[string]string

func (f fakeSource) Get(key string) (string, bool, error) {
	v, ok := f[key]
	return v, ok, nil
}

// funcSource is a config.Source backed by a function, used where a test
// needs to simulate Get failing for a specific key — something a plain
// map can't express.
type funcSource func(key string) (string, bool, error)

func (f funcSource) Get(key string) (string, bool, error) { return f(key) }

func TestLoadReadsConfiguredValues(t *testing.T) {
	cfg, err := Load(fakeSource{
		KeyTypes:                      "feat,chore",
		KeyHeaderMaxLength:            "72",
		KeyRule("type"):               "off",
		KeyRule("description-period"): "error",
	}, []string{"type", "description-period", "format"})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !reflect.DeepEqual(cfg.Types, []string{"feat", "chore"}) {
		t.Errorf("Types = %v, want [feat chore]", cfg.Types)
	}
	if got := cfg.MaxHeaderLength(); got != 72 {
		t.Errorf("MaxHeaderLength() = %d, want 72", got)
	}
	if lvl, ok := cfg.RawLevel("type"); !ok || lvl != LevelOff {
		t.Errorf("RawLevel(type) = %q, %v; want off, true", lvl, ok)
	}
	if lvl, ok := cfg.RawLevel("description-period"); !ok || lvl != LevelError {
		t.Errorf("RawLevel(description-period) = %q, %v; want error, true", lvl, ok)
	}
	// Named in ruleNames but never set in src: reported as unset, not
	// defaulted — Load has no opinion on what a rule's default should be.
	if _, ok := cfg.RawLevel("format"); ok {
		t.Errorf("RawLevel(format) ok = true; want false (never configured)")
	}
}

func TestLoadOnlyReadsRequestedRuleNames(t *testing.T) {
	// A rule key present in the source but not passed in ruleNames must be
	// ignored: Load has no independent notion of which rules exist.
	cfg, err := Load(fakeSource{KeyRule("type"): "off"}, []string{"description-period"})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if _, ok := cfg.RawLevel("type"); ok {
		t.Error("RawLevel(type) ok = true; want false (type was not in ruleNames)")
	}
}

func TestLoadRejectsInvalidLevel(t *testing.T) {
	src := fakeSource{KeyRule("type"): "maybe"}
	if _, err := Load(src, []string{"type"}); err == nil {
		t.Fatal("Load() = nil error; want a validation error for an invalid level")
	}
}

func TestLoadRejectsInvalidHeaderMaxLength(t *testing.T) {
	src := fakeSource{KeyHeaderMaxLength: "not-a-number"}
	if _, err := Load(src, nil); err == nil {
		t.Fatal("Load() = nil error; want a validation error for a non-numeric headerMaxLength")
	}
}

func TestLoadPropagatesGetErrors(t *testing.T) {
	boom := errors.New("boom")

	tests := []struct {
		name      string
		src       funcSource
		ruleNames []string
	}{
		{
			name: "types",
			src: func(key string) (string, bool, error) {
				if key == KeyTypes {
					return "", false, boom
				}
				return "", false, nil
			},
		},
		{
			name: "headerMaxLength",
			src: func(key string) (string, bool, error) {
				if key == KeyHeaderMaxLength {
					return "", false, boom
				}
				return "", false, nil
			},
		},
		{
			name: "rule",
			src: func(key string) (string, bool, error) {
				if key == KeyRule("type") {
					return "", false, boom
				}
				return "", false, nil
			},
			ruleNames: []string{"type"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Load(tt.src, tt.ruleNames); err == nil {
				t.Fatalf("Load() = nil error; want the %s Get failure propagated", tt.name)
			}
		})
	}
}

func TestRawLevelUnsetByDefault(t *testing.T) {
	cfg := Config{Rules: map[string]Level{}}
	if _, ok := cfg.RawLevel("type"); ok {
		t.Error("RawLevel(type) ok = true; want false on an empty Config")
	}
}

func TestRawLevelHonorsOverride(t *testing.T) {
	cfg := Config{Rules: map[string]Level{"type": LevelOff}}
	if lvl, ok := cfg.RawLevel("type"); !ok || lvl != LevelOff {
		t.Errorf("RawLevel(type) = %q, %v; want off, true", lvl, ok)
	}
}

func TestAllowedTypesFallsBackToDefault(t *testing.T) {
	cfg := Config{}
	if got := cfg.AllowedTypes(); !reflect.DeepEqual(got, defaultTypes) {
		t.Errorf("AllowedTypes() = %v, want %v", got, defaultTypes)
	}
}

func TestAllowedTypesHonorsOverride(t *testing.T) {
	cfg := Config{Types: []string{"task"}}
	if got := cfg.AllowedTypes(); !reflect.DeepEqual(got, []string{"task"}) {
		t.Errorf("AllowedTypes() = %v, want [task]", got)
	}
}

func TestMaxHeaderLength(t *testing.T) {
	cfg := Config{}
	if got := cfg.MaxHeaderLength(); got != DefaultHeaderMaxLength {
		t.Errorf("MaxHeaderLength() = %d, want default %d", got, DefaultHeaderMaxLength)
	}

	custom := 72
	cfg.HeaderMaxLength = &custom
	if got := cfg.MaxHeaderLength(); got != 72 {
		t.Errorf("MaxHeaderLength() = %d, want 72", got)
	}
}

func TestSplitCSV(t *testing.T) {
	got := SplitCSV(" feat, fix ,,chore ")
	want := []string{"feat", "fix", "chore"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SplitCSV(...) = %v, want %v", got, want)
	}
}

func TestKeyRule(t *testing.T) {
	if got, want := KeyRule("type"), "commitsentinel.rules.type"; got != want {
		t.Errorf("KeyRule(type) = %q, want %q", got, want)
	}
}
