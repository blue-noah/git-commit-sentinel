package conventionalcommit

import (
	"regexp"
	"slices"
	"strings"

	"git-commit-sentinel/internal/config"
)

var headerPattern = regexp.MustCompile(`^(?P<type>[a-zA-Z]+)(\((?P<scope>[^)]+)\))?(?P<breaking>!)?: (?P<description>.*)$`)

// Context carries everything a Rule needs to evaluate one commit message.
// Each registered Rule gets its own copy (see Validate), including its own
// Lines, so one rule can never corrupt what another sees in the same call.
type Context struct {
	Header string
	Lines  []string
	Cfg    config.Config

	// Parsed is true when Header matched the Conventional Commits grammar.
	// Type, Scope, Breaking and Description are meaningful only when it is.
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
