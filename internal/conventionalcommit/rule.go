package conventionalcommit

import (
	"slices"

	"git-commit-sentinel/internal/config"
)

// Rule is one independently pluggable Conventional Commits check. New
// rules are added by building a ruleSpec value and calling Register on it
// from an init() (see rule_format.go for the pattern) — Validate, the
// registry, and every existing rule stay untouched (Open/Closed: extension
// by addition, not by modification).
type Rule interface {
	// Name identifies the rule, and is the suffix of the git config key
	// (commitsentinel.rules.<Name>) that controls its level.
	Name() string
	// Description is a short, human-readable explanation shown by `doctor`.
	Description() string
	// DefaultLevel is used when the user has not configured this rule.
	DefaultLevel() config.Level
	// Check inspects ctx and reports whether the rule is violated, with a
	// human-readable message when it is.
	Check(ctx Context) (violated bool, message string)
}

// ruleSpec is the sole implementation of Rule: a plain value plus a check
// function, instead of one bespoke type per rule. Every rule_*.go file
// registers one ruleSpec literal — this is the only place the four Rule
// methods are implemented, so a new rule adds zero new methods.
type ruleSpec struct {
	name         string
	description  string
	defaultLevel config.Level
	check        func(ctx Context) (violated bool, message string)
}

func (r ruleSpec) Name() string                     { return r.name }
func (r ruleSpec) Description() string              { return r.description }
func (r ruleSpec) DefaultLevel() config.Level       { return r.defaultLevel }
func (r ruleSpec) Check(ctx Context) (bool, string) { return r.check(ctx) }

var registry []Rule

// Register adds a rule to the set Validate evaluates. Call it from an
// init() in the file that defines the rule.
func Register(r Rule) { registry = append(registry, r) }

// Rules returns every registered rule, in registration order. The result
// is a copy of the internal registry, so a caller can never corrupt the
// shared, process-wide rule set — e.g. by reassigning an element, or by
// appending past its capacity and racing a concurrent Register call —
// through the returned slice.
func Rules() []Rule { return slices.Clone(registry) }

// RuleNames returns the Name() of every registered rule. config.Load uses
// this to know which commitsentinel.rules.<name> keys to read, without
// this package's rule catalog ever needing to live in package config.
func RuleNames() []string {
	names := make([]string, len(registry))
	for i, r := range registry {
		names[i] = r.Name()
	}
	return names
}

// EffectiveLevel resolves rule's level from cfg, falling back to the
// rule's own default when the user has not configured it.
func EffectiveLevel(cfg config.Config, r Rule) config.Level {
	if lvl, ok := cfg.RawLevel(r.Name()); ok {
		return lvl
	}
	return r.DefaultLevel()
}
