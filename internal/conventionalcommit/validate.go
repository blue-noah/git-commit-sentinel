// Package conventionalcommit validates commit messages against the
// Conventional Commits specification (https://www.conventionalcommits.org).
// Each check is a self-contained Rule (see rule.go); Validate itself only
// builds the shared Context and asks every registered rule to look at it.
package conventionalcommit

import "git-commit-sentinel/internal/config"

// Finding is a single rule violation.
type Finding struct {
	Rule    string
	Level   config.Level
	Message string
}

// HasErrors reports whether any finding is at error level.
func HasErrors(findings []Finding) bool {
	for _, f := range findings {
		if f.Level == config.LevelError {
			return true
		}
	}
	return false
}

// Validate checks msg against cfg and returns every triggered finding.
// A rule set to "off" is never even asked to Check; the caller decides
// what off/warn/error mean (see cmd/hook.go).
func Validate(msg string, cfg config.Config) []Finding {
	ctx := newContext(msg, cfg)

	var findings []Finding
	for _, r := range Rules() {
		level := EffectiveLevel(cfg, r)
		if level == config.LevelOff {
			continue
		}
		if violated, message := r.Check(ctx.cloneForRule()); violated {
			findings = append(findings, Finding{Rule: r.Name(), Level: level, Message: message})
		}
	}
	return findings
}
