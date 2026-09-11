package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"git-commit-sentinel/internal/config"
	"git-commit-sentinel/internal/conventionalcommit"
	"git-commit-sentinel/internal/gitcli"
	"git-commit-sentinel/internal/hookfile"
)

const setupUsage = `usage: git-commit-sentinel setup [flags]

Installs the commit-msg hook. By default it installs globally, so every
repository on the machine is covered without per-repository setup.

Flags:
  -scope global|local   where to install (default "global")
  -force                overwrite a hook/config not created by this tool
  -hooks-dir <path>     custom directory for the global hooks
                        (default $XDG_CONFIG_HOME/git-commit-sentinel/hooks)
  -types <csv>          set commitsentinel.types at the target scope
  -rule <name>=<level>  set a rule's level at the target scope (off|warn|error),
                        repeatable

Running setup again with the same flags is a no-op; running it with
different values updates the existing installation in place.
`

type overwritePolicy int

const (
	keepForeign overwritePolicy = iota
	forceOverwrite
)

func overwritePolicyFrom(force bool) overwritePolicy {
	if force {
		return forceOverwrite
	}
	return keepForeign
}

func abort(err error) int {
	fmt.Fprintf(os.Stderr, "git-commit-sentinel: %v\n", err)
	return 1
}

func runSetup(args []string) int {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, setupUsage) }

	scope := fs.String("scope", "global", "")
	force := fs.Bool("force", false, "")
	hooksDirFlag := fs.String("hooks-dir", "", "")
	types := fs.String("types", "", "")
	var rules ruleLevelFlags
	fs.Var(&rules, "rule", "")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *scope != "global" && *scope != "local" {
		fmt.Fprintf(os.Stderr, "git-commit-sentinel: invalid -scope %q: must be \"global\" or \"local\"\n", *scope)
		return 2
	}

	if err := preflightGitVersion(*scope); err != nil {
		return abort(err)
	}
	warnIfBinaryMissing()

	policy := overwritePolicyFrom(*force)
	gitScope, err := installForScope(*scope, *hooksDirFlag, policy)
	if err != nil {
		return abort(err)
	}

	if err := applyOverrides(gitScope, *types, rules); err != nil {
		return abort(err)
	}

	fmt.Println("\nsetup complete. Run `git-commit-sentinel doctor` to verify the installation.")
	return 0
}

func preflightGitVersion(scope string) error {
	ver, err := gitcli.DetectVersion()
	if err != nil {
		return err
	}
	fmt.Printf("git version: %s\n", ver)

	if scope == "global" && !ver.AtLeast(2, 9) {
		return fmt.Errorf(
			"git %s is too old for a global install: core.hooksPath requires git >= 2.9.0.\n"+
				"Upgrade git, or run `git-commit-sentinel setup -scope local` inside each repository instead",
			ver,
		)
	}
	return nil
}

func warnIfBinaryMissing() {
	if _, err := lookPath(hookfile.BinaryName); err != nil {
		fmt.Fprintf(os.Stderr,
			"warning: %q was not found on PATH; the installed hook will fail until it is.\n"+
				"Build and install it first, e.g. `go install ./cmd/git-commit-sentinel`.\n",
			hookfile.BinaryName)
	}
}

func installForScope(scope, hooksDirFlag string, policy overwritePolicy) (gitcli.Scope, error) {
	switch scope {
	case "global":
		return gitcli.ScopeGlobal, setupGlobal(hooksDirFlag, policy)
	case "local":
		return gitcli.ScopeLocal, setupLocal(policy)
	default:
		return "", fmt.Errorf("invalid scope %q", scope)
	}
}

func setupGlobal(hooksDirFlag string, policy overwritePolicy) error {
	dir := hooksDirFlag
	if dir == "" {
		var err error
		dir, err = defaultHooksDir()
		if err != nil {
			return err
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating hooks directory %s: %w", dir, err)
	}

	hookPath := filepath.Join(dir, "commit-msg")
	if err := installHook(hookPath, policy); err != nil {
		return err
	}

	existing, isSet, err := gitcli.ConfigGet(gitcli.ScopeGlobal, "core.hooksPath")
	if err != nil {
		return err
	}
	if isSet && existing != dir && policy != forceOverwrite {
		return fmt.Errorf(
			"global core.hooksPath is already set to %q; rerun with -force to point it at %q instead",
			existing, dir,
		)
	}
	if !isSet || existing != dir {
		if err := gitcli.ConfigSet(gitcli.ScopeGlobal, "core.hooksPath", dir); err != nil {
			return err
		}
		fmt.Printf("set global core.hooksPath = %s\n", dir)
	} else {
		fmt.Printf("global core.hooksPath already = %s (unchanged)\n", dir)
	}
	return nil
}

func setupLocal(policy overwritePolicy) error {
	if !gitcli.InsideRepo() {
		return fmt.Errorf("not inside a git repository; run this from within the repository you want to configure, or use -scope global")
	}

	gitDir, err := gitcli.GitDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(gitDir, "hooks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating hooks directory %s: %w", dir, err)
	}

	hookPath := filepath.Join(dir, "commit-msg")
	if err := installHook(hookPath, policy); err != nil {
		return err
	}

	if override, isSet, _ := gitcli.ConfigGet(gitcli.ScopeLocal, "core.hooksPath"); isSet && override != dir {
		fmt.Printf(
			"warning: local core.hooksPath is set to %q, which overrides %q; the hook just installed will not run until that is unset or repointed here\n",
			override, dir,
		)
	} else if globalOverride, isSet, _ := gitcli.ConfigGet(gitcli.ScopeGlobal, "core.hooksPath"); isSet {
		fmt.Printf(
			"note: global core.hooksPath is set to %q; git only consults one hooksPath, so this repository's hooks in .git/hooks will be ignored unless its local core.hooksPath is unset\n",
			globalOverride,
		)
	}
	return nil
}

func installHook(path string, policy overwritePolicy) error {
	content := hookfile.Render()

	existing, err := os.ReadFile(path)
	switch {
	case err == nil:
		if string(existing) == content {
			fmt.Printf("hook already up to date at %s\n", path)
			return nil
		}
		if !hookfile.IsManaged(string(existing)) && policy != forceOverwrite {
			return fmt.Errorf("a hook already exists at %s and was not created by git-commit-sentinel; rerun with -force to overwrite it", path)
		}
	case !os.IsNotExist(err):
		return err
	}

	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		return err
	}
	fmt.Printf("installed hook at %s\n", path)
	return nil
}

func applyOverrides(scope gitcli.Scope, types string, rules ruleLevelFlags) error {
	if types != "" {
		list := strings.Join(config.SplitCSV(types), ",")
		if err := gitcli.ConfigSet(scope, config.KeyTypes, list); err != nil {
			return err
		}
		fmt.Printf("set %s = %s\n", config.KeyTypes, list)
	}
	for _, rl := range rules {
		key := config.KeyRule(rl.name)
		if err := gitcli.ConfigSet(scope, key, string(rl.level)); err != nil {
			return err
		}
		fmt.Printf("set %s = %s\n", key, rl.level)
	}
	return nil
}

func defaultHooksDir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolving home directory: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "git-commit-sentinel", "hooks"), nil
}

type ruleLevelFlags []ruleLevelFlag

type ruleLevelFlag struct {
	name  string
	level config.Level
}

func (r *ruleLevelFlags) String() string { return "" }

func (r *ruleLevelFlags) Set(s string) error {
	name, levelStr, found := strings.Cut(s, "=")
	if !found {
		return fmt.Errorf("expected name=level, got %q", s)
	}
	name = strings.TrimSpace(name)

	known := false
	for _, rn := range conventionalcommit.RuleNames() {
		if rn == name {
			known = true
			break
		}
	}
	if !known {
		return fmt.Errorf("unknown rule %q (known rules: %s)", name, strings.Join(conventionalcommit.RuleNames(), ", "))
	}

	level := config.Level(strings.ToLower(strings.TrimSpace(levelStr)))
	if !level.Valid() {
		return fmt.Errorf("invalid level %q for rule %q: must be off, warn or error", levelStr, name)
	}

	*r = append(*r, ruleLevelFlag{name: name, level: level})
	return nil
}
