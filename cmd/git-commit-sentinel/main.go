// Command git-commit-sentinel implements a git commit-msg hook that
// validates commit messages against the Conventional Commits specification.
package main

import (
	"fmt"
	"os"
	"strings"

	"git-commit-sentinel/internal/conventionalcommit"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}

func run(args []string, stderr *os.File) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "usage: git-commit-sentinel <commit-msg-file> [allowed-type ...]")
		return 2
	}

	path := args[0]
	types := args[1:]

	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "git-commit-sentinel: cannot read commit message file %q: %v\n", path, err)
		return 2
	}

	msg := stripComments(string(raw))

	result := conventionalcommit.Validate(msg, types)
	if !result.Valid {
		fmt.Fprintln(stderr, "git-commit-sentinel: commit message is not Conventional Commits compliant:")
		for _, e := range result.Errors {
			fmt.Fprintf(stderr, "  - %s\n", e)
		}
		fmt.Fprintln(stderr, "\nExpected format: <type>(<scope>)!: <description>")
		fmt.Fprintf(stderr, "Allowed types: %s\n", strings.Join(defaultOrGiven(types), ", "))
		return 1
	}

	return 0
}

func defaultOrGiven(types []string) []string {
	if len(types) == 0 {
		return conventionalcommit.DefaultTypes
	}
	return types
}

// stripComments removes lines starting with '#', as git does before
// recording the final commit message.
func stripComments(msg string) string {
	lines := strings.Split(msg, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimLeft(line, " \t"), "#") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.TrimRight(strings.Join(kept, "\n"), "\n")
}
