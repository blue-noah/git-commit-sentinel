// Command git-commit-sentinel is a git commit-msg hook that validates
// commit messages against the Conventional Commits specification, plus the
// setup/doctor tooling to install and diagnose it.
package main

import (
	"fmt"
	"os"
)

var version = "dev"

const usage = `git-commit-sentinel — a Conventional Commits git hook

Usage:
  git-commit-sentinel setup [flags]   install the commit-msg hook (see -h)
  git-commit-sentinel doctor          diagnose the installation and config
  git-commit-sentinel hook <file>     validate a commit message (called by git itself)
  git-commit-sentinel version         print the version

Run "git-commit-sentinel <command> -h" for flags specific to a command.

Configuration lives in git config, under the "commitsentinel" section:
  git config --global commitsentinel.types "feat,fix,chore"
  git config --global commitsentinel.headerMaxLength 72
  git config --global commitsentinel.rules.<rule> off|warn|error

See README.md for the full list of rules.
`

func main() {
	os.Exit(realMain(os.Args[1:]))
}

func realMain(args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}

	switch args[0] {
	case "hook":
		return runHook(args[1:])
	case "setup":
		return runSetup(args[1:])
	case "doctor":
		return runDoctor(args[1:])
	case "-h", "--help", "help":
		fmt.Fprint(os.Stdout, usage)
		return 0
	case "-v", "--version", "version":
		fmt.Println("git-commit-sentinel " + version)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "git-commit-sentinel: unknown command %q\n\n", args[0])
		fmt.Fprint(os.Stderr, usage)
		return 2
	}
}
