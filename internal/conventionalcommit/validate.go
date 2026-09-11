// Package conventionalcommit validates commit messages against the
// Conventional Commits specification (https://www.conventionalcommits.org).
package conventionalcommit

import (
	"fmt"
	"regexp"
	"strings"
)

// DefaultTypes are the commit types allowed when no custom list is provided.
var DefaultTypes = []string{
	"feat", "fix", "docs", "style", "refactor",
	"perf", "test", "build", "ci", "chore", "revert",
}

var headerPattern = regexp.MustCompile(`^(?P<type>[a-zA-Z]+)(?P<scope>\([^)]+\))?(?P<breaking>!)?: (?P<description>.+)$`)

// Result carries the outcome of validating a commit message.
type Result struct {
	Valid  bool
	Errors []string
}

// Validate checks msg against the Conventional Commits format using the
// given set of allowed types. If types is empty, DefaultTypes is used.
func Validate(msg string, types []string) Result {
	if len(types) == 0 {
		types = DefaultTypes
	}

	lines := strings.Split(strings.ReplaceAll(msg, "\r\n", "\n"), "\n")
	header := strings.TrimRight(lines[0], " \t")

	var errs []string

	if header == "" {
		errs = append(errs, "commit message header is empty")
		return Result{Valid: false, Errors: errs}
	}

	match := headerPattern.FindStringSubmatch(header)
	if match == nil {
		errs = append(errs, fmt.Sprintf(
			"header %q does not match Conventional Commits format: <type>(<scope>)!: <description>",
			header,
		))
		return Result{Valid: false, Errors: errs}
	}

	groups := namedGroups(headerPattern, match)

	commitType := groups["type"]
	if !contains(types, commitType) {
		errs = append(errs, fmt.Sprintf(
			"type %q is not allowed; allowed types: %s",
			commitType, strings.Join(types, ", "),
		))
	}

	description := strings.TrimSpace(groups["description"])
	if description == "" {
		errs = append(errs, "description must not be empty")
	}
	if strings.HasSuffix(description, ".") {
		errs = append(errs, "description must not end with a period")
	}

	if len(lines) > 1 && strings.TrimSpace(lines[1]) != "" {
		errs = append(errs, "second line must be blank to separate header from body")
	}

	return Result{Valid: len(errs) == 0, Errors: errs}
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

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
