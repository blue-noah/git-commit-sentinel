// Package config defines the user-facing configuration for
// git-commit-sentinel: the allowed commit types, the header length limit,
// and the enforcement level of each validation rule.
//
// This package knows nothing about git: it depends only on the Source
// interface below. The concrete storage mechanism — git config, under the
// "commitsentinel" section — lives in internal/gitcli, which adapts itself
// to Source. Wiring the two together is the caller's job (see
// gitcli.ConfigSource), keeping the dependency pointed from the
// infrastructure detail toward this package's policy, never the reverse.
//
//	git config --global commitsentinel.types "feat,fix,chore"
//	git config --global commitsentinel.headerMaxLength 72
//	git config --global commitsentinel.rules.description-period error
//	git config --local  commitsentinel.rules.header-length off
package config

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Source reads a single configuration value by key. ok is false when the
// key is unset. Implementations decide what "unset" and scope mean (e.g.
// internal/gitcli's ConfigSource backs this with `git config --get`); this
// package only ever calls Get.
type Source interface {
	Get(key string) (value string, ok bool, err error)
}

// Level is the enforcement level of a single rule.
type Level string

const (
	LevelOff   Level = "off"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

func (l Level) Valid() bool {
	switch l {
	case LevelOff, LevelWarn, LevelError:
		return true
	}
	return false
}

// defaultTypes are the commit types allowed when commitsentinel.types is
// not configured. Unexported deliberately: the only way to read it is
// AllowedTypes, which always hands back a copy — so no package can ever
// reach (and corrupt) this shared slice, not even by convention, because
// there is no exported name to reach it through.
var defaultTypes = []string{
	"feat", "fix", "docs", "style", "refactor",
	"perf", "test", "build", "ci", "chore", "revert",
}

// DefaultHeaderMaxLength is used when commitsentinel.headerMaxLength is not
// configured.
const DefaultHeaderMaxLength = 100

// This package deliberately has no notion of which rules exist or what
// their default levels are: that catalog is owned by whatever package
// defines the rules (internal/conventionalcommit), so that adding a new
// rule there never requires touching this one (Open/Closed at the package
// boundary, not just within a single function). Load takes the rule name
// list as a parameter for exactly this reason.

// section is the git config section under which every key lives.
const section = "commitsentinel"

// KeyTypes, KeyHeaderMaxLength and KeyRule build the git config key names
// used to read and write configuration. They are exported so `setup` and
// `doctor` can operate on the same keys without duplicating the section
// name.
const KeyTypes = section + ".types"
const KeyHeaderMaxLength = section + ".headerMaxLength"

func KeyRule(rule string) string { return section + ".rules." + rule }

// Config is an in-memory snapshot of the enforcement configuration. Every
// field is optional (nil/empty means "not set"); use RawLevel,
// AllowedTypes and MaxHeaderLength to read it back.
type Config struct {
	Types           []string
	HeaderMaxLength *int
	Rules           map[string]Level
}

// Load reads configuration from src for each rule named in ruleNames.
// Callers choose what src reads (e.g. a specific git config scope) and
// which rules exist (this package has no built-in rule catalog — see the
// note above) — Load has no opinion on either.
//
// Unset keys are left as zero values (nil/empty map) rather than filled
// with defaults, so callers can distinguish "not configured" from
// "configured to the same value as some default". Use RawLevel,
// AllowedTypes and MaxHeaderLength to read the result.
func Load(src Source, ruleNames []string) (Config, error) {
	cfg := Config{Rules: map[string]Level{}}

	if raw, ok, err := src.Get(KeyTypes); err != nil {
		return Config{}, err
	} else if ok {
		cfg.Types = SplitCSV(raw)
	}

	if raw, ok, err := src.Get(KeyHeaderMaxLength); err != nil {
		return Config{}, err
	} else if ok {
		n, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil || n <= 0 {
			return Config{}, fmt.Errorf("%s must be a positive integer, got %q", KeyHeaderMaxLength, raw)
		}
		cfg.HeaderMaxLength = &n
	}

	for _, rule := range ruleNames {
		key := KeyRule(rule)
		raw, ok, err := src.Get(key)
		if err != nil {
			return Config{}, err
		}
		if !ok {
			continue
		}
		lvl := Level(strings.ToLower(strings.TrimSpace(raw)))
		if !lvl.Valid() {
			return Config{}, fmt.Errorf("%s is %q, must be one of off, warn, error", key, raw)
		}
		cfg.Rules[rule] = lvl
	}

	return cfg, nil
}

// SplitCSV parses a comma-separated list, trimming whitespace and dropping
// empty entries. It is exported because it defines the same format used
// both to parse commitsentinel.types out of git config (Load, above) and
// to normalize a -types flag value before writing it back (see
// cmd/git-commit-sentinel's setup command) — one function, not two
// copies agreeing on the format by convention.
func SplitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// RawLevel returns the level explicitly configured for rule, and whether
// one was set at all. It never guesses a default: a rule's default level
// is a property of the rule itself (see conventionalcommit.Rule), not of
// this package, so callers combine the two (see
// conventionalcommit.EffectiveLevel).
func (c Config) RawLevel(rule string) (Level, bool) {
	lvl, ok := c.Rules[rule]
	return lvl, ok
}

// AllowedTypes returns the configured commit types, or the built-in
// defaults if unset. The result is always a fresh copy: Config's Types
// field (and the package's internal default list) are never handed out
// directly, so a caller can never mutate this package's internal state
// through the returned slice — Go gives slices reference semantics, so
// returning the field itself would not have been a safe read-only view.
func (c Config) AllowedTypes() []string {
	if c.Types != nil {
		return slices.Clone(c.Types)
	}
	return slices.Clone(defaultTypes)
}

// MaxHeaderLength returns the configured header length limit, or
// DefaultHeaderMaxLength if unset.
func (c Config) MaxHeaderLength() int {
	if c.HeaderMaxLength != nil {
		return *c.HeaderMaxLength
	}
	return DefaultHeaderMaxLength
}
