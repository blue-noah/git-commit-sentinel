package conventionalcommit

import (
	"regexp"
	"slices"
	"strings"

	"git-commit-sentinel/internal/config"
)

var headerPattern = regexp.MustCompile(`^(?P<type>[a-zA-Z]+)(\((?P<scope>[^)]+)\))?(?P<breaking>!)?: (?P<description>.*)$`)

// Context carries everything a Rule needs to evaluate one commit message.
// Validate builds it once (including the header parse) and passes it to
// every registered Rule by value: since Rule is an exported interface that
// third-party code could implement and Register, nothing here trusts a
// Check implementation not to mutate its ctx — passing a copy means it
// wouldn't matter if one did, since no other rule in the same Validate
// call shares that copy.
type Context struct {
	Header string
	Lines  []string
	Cfg    config.Config

	// Parsed is true when Header matched the Conventional Commits grammar.
	// Type, Scope, Breaking and Description are only meaningful when it is;
	// rules that depend on them should return "not violated" otherwise,
	// leaving the report to the format rule.
	Parsed      bool
	Type        string
	Scope       string
	Breaking    bool
	Description string
}

func newContext(msg string, cfg config.Config) Context {
	lines := strings.Split(strings.ReplaceAll(msg, "\r\n", "\n"), "\n")
	ctx := Context{Header: lines[0], Lines: lines, Cfg: cfg}

	match := headerPattern.FindStringSubmatch(ctx.Header)
	if match == nil {
		return ctx
	}

	groups := namedGroups(headerPattern, match)
	ctx.Parsed = true
	ctx.Type = groups["type"]
	ctx.Scope = groups["scope"]
	ctx.Breaking = groups["breaking"] == "!"
	ctx.Description = strings.TrimSpace(groups["description"])
	return ctx
}

// cloneForRule returns an independent copy of c, safe to hand to a single
// Rule.Check call. Passing Context by value already isolates every scalar
// field, but Lines is a slice — reference semantics mean a copy of the
// struct still shares the same backing array as the original, so a rule
// that (incorrectly) writes into ctx.Lines by index could otherwise
// corrupt what every other rule in the same Validate call sees.
func (c Context) cloneForRule() Context {
	c.Lines = slices.Clone(c.Lines)
	return c
}

func namedGroups(re *regexp.Regexp, match []string) map[string]string {
	groups := make(map[string]string, len(match))
	for i, name := range re.SubexpNames() {
		if i == 0 || name == "" {
			continue
		}
		groups[name] = match[i]
	}
	return groups
}
