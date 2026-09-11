package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"git-commit-sentinel/internal/config"
	"git-commit-sentinel/internal/conventionalcommit"
	"git-commit-sentinel/internal/gitcli"
)

// runHook is invoked by git itself as the commit-msg hook: it receives the
// path to the file holding the candidate commit message.
func runHook(args []string) int {
	fs := flag.NewFlagSet("hook", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	rest := fs.Args()
	if len(rest) < 1 {
		fmt.Fprintln(os.Stderr, "usage: git-commit-sentinel hook <commit-msg-file>")
		return 2
	}
	path := rest[0]

	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "git-commit-sentinel: cannot read commit message file %q: %v\n", path, err)
		return 2
	}

	cfg, err := config.Load(gitcli.ConfigSource{Scope: gitcli.ScopeEffective}, conventionalcommit.RuleNames())
	if err != nil {
		fmt.Fprintf(os.Stderr, "git-commit-sentinel: invalid configuration: %v\n", err)
		fmt.Fprintln(os.Stderr, "Run `git-commit-sentinel doctor` for details.")
		return 2
	}

	msg := stripComments(string(raw))
	findings := conventionalcommit.Validate(msg, cfg)

	for _, f := range findings {
		label := "warning"
		if f.Level == config.LevelError {
			label = "error"
		}
		fmt.Fprintf(os.Stderr, "git-commit-sentinel: %s: [%s] %s\n", label, f.Rule, f.Message)
	}

	if conventionalcommit.HasErrors(findings) {
		fmt.Fprintln(os.Stderr, "\ncommit rejected — fix the errors above, or adjust a rule's level with:")
		fmt.Fprintln(os.Stderr, "  git config --global commitsentinel.rules.<rule> off|warn|error")
		return 1
	}

	return 0
}

// stripComments removes lines starting with '#', matching how git itself
// strips them before recording the final commit message.
func stripComments(msg string) string {
	lines := slices.DeleteFunc(strings.Split(msg, "\n"), func(line string) bool {
		return strings.HasPrefix(strings.TrimLeft(line, " \t"), "#")
	})
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}
