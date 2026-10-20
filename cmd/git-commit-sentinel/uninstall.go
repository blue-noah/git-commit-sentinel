package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"git-commit-sentinel/internal/gitcli"
	"git-commit-sentinel/internal/hookfile"
)

const uninstallUsage = `usage: git-commit-sentinel uninstall [flags]

Removes what setup installed: the commit-msg hook it wrote, the global
core.hooksPath pointing at that hook's directory, and the commitsentinel
settings. Hooks and settings it didn't create are left alone. The binary
itself stays: delete it, or use the uninstall script, which also takes it
off PATH.

Flags:
  -scope global|local   what to remove (default "global"; "local" acts on
                        the current repository)
  -keep-config          keep the commitsentinel.* settings at that scope

Running it again when nothing is left to remove is a no-op.
`

// configSections are the git config sections setup and users write to:
// rule levels live in a subsection of their own.
var configSections = []string{"commitsentinel", "commitsentinel.rules"}

func runUninstall(args []string) int {
	fs := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, uninstallUsage) }

	scope := fs.String("scope", "global", "")
	keepConfig := fs.Bool("keep-config", false, "")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	var gitScope gitcli.Scope
	var err error
	switch *scope {
	case "global":
		gitScope, err = gitcli.ScopeGlobal, uninstallGlobal()
	case "local":
		gitScope, err = gitcli.ScopeLocal, uninstallLocal()
	default:
		fmt.Fprintf(os.Stderr, "git-commit-sentinel: invalid -scope %q: must be \"global\" or \"local\"\n", *scope)
		return 2
	}
	if err != nil {
		return abort(err)
	}

	if !*keepConfig {
		if err := removeConfig(gitScope); err != nil {
			return abort(err)
		}
	}

	fmt.Println("\nuninstall complete.")
	return 0
}

func uninstallGlobal() error {
	dir, isSet, err := gitcli.ConfigGet(gitcli.ScopeGlobal, "core.hooksPath")
	if err != nil {
		return err
	}
	if !isSet {
		fmt.Println("global core.hooksPath is not set; no global hook to remove")
		return nil
	}

	removed, err := removeManagedHook(filepath.Join(dir, "commit-msg"))
	if err != nil {
		return err
	}
	if !removed {
		fmt.Printf("global core.hooksPath = %s holds no git-commit-sentinel hook; leaving it as is\n", dir)
		return nil
	}

	// Other hooks in that directory only run while core.hooksPath points
	// at it, so it stays set until the directory is empty.
	if !removeIfEmpty(dir) {
		fmt.Printf("kept global core.hooksPath = %s: it still holds other hooks\n", dir)
		return nil
	}
	if filepath.Base(filepath.Dir(dir)) == hookfile.BinaryName {
		removeIfEmpty(filepath.Dir(dir))
	}
	if err := gitcli.ConfigUnset(gitcli.ScopeGlobal, "core.hooksPath"); err != nil {
		return err
	}
	fmt.Println("unset global core.hooksPath")
	return nil
}

func uninstallLocal() error {
	if !gitcli.InsideRepo() {
		return fmt.Errorf("not inside a git repository; run this from within the repository to clean up, or use -scope global")
	}
	gitDir, err := gitcli.GitDir()
	if err != nil {
		return err
	}
	removed, err := removeManagedHook(filepath.Join(gitDir, "hooks", "commit-msg"))
	if err != nil {
		return err
	}
	if !removed {
		fmt.Println("no git-commit-sentinel hook in this repository")
	}
	return nil
}

// removeManagedHook deletes the hook at path if git-commit-sentinel wrote
// it, and reports whether it did.
func removeManagedHook(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !hookfile.IsManaged(string(data)) {
		fmt.Printf("%s was not created by git-commit-sentinel; leaving it as is\n", path)
		return false, nil
	}
	if err := os.Remove(path); err != nil {
		return false, err
	}
	fmt.Printf("removed hook %s\n", path)
	return true, nil
}

// removeIfEmpty deletes dir if it has no entries, and reports whether dir
// is gone.
func removeIfEmpty(dir string) bool {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	if err != nil || len(entries) > 0 {
		return false
	}
	return os.Remove(dir) == nil
}

func removeConfig(scope gitcli.Scope) error {
	for _, section := range configSections {
		removed, err := gitcli.RemoveSection(scope, section)
		if err != nil {
			return err
		}
		if removed {
			fmt.Printf("removed [%s] from %s config\n", section, scopeName(scope))
		}
	}
	return nil
}

func scopeName(scope gitcli.Scope) string {
	if scope == gitcli.ScopeGlobal {
		return "global"
	}
	return "local"
}
