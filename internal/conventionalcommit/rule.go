package conventionalcommit

import (
	"slices"

	"git-commit-sentinel/internal/config"
)

// Rule is one independently pluggable Conventional Commits check. New
// rules register themselves from an init() (see rule_format.go) without
// touching Validate, the registry, or any existing rule.
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
// is a copy, so a caller can't corrupt the shared registry through it.
func Rules() []Rule { return slices.Clone(registry) }

// RuleNames returns the Name() of every registered rule.
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
