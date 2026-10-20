// Package gitcli wraps the subset of the git CLI that
// git-commit-sentinel needs: version detection, repository discovery, and
// reading/writing git config values.
package gitcli

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

type getFunc func(args []string) (out string, ok bool, err error)
type setFunc func(args []string) error
type outputFunc func(args []string) (string, error)

var (
	gitGet    getFunc    = execGet
	gitSet    setFunc    = execSet
	gitOutput outputFunc = execOutput
)

func execGet(args []string) (string, bool, error) {
	out, err := exec.Command("git", args...).Output()
	if err == nil {
		return strings.TrimSpace(string(out)), true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return "", false, nil
	}
	return "", false, err
}

func execSet(args []string) error {
	out, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func execOutput(args []string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// Version is a parsed "git --version" result.
type Version struct {
	Major, Minor, Patch int
	Raw                 string
}

// AtLeast reports whether v is greater than or equal to major.minor.
func (v Version) AtLeast(major, minor int) bool {
	if v.Major != major {
		return v.Major > major
	}
	return v.Minor >= minor
}

func (v Version) String() string { return v.Raw }

// DetectVersion runs "git --version" and parses the result. It fails if
// git is not available or the output cannot be parsed.
func DetectVersion() (Version, error) {
	raw, err := gitOutput([]string{"--version"})
	if err != nil {
		return Version{}, fmt.Errorf("git is not available: %w", err)
	}
	return parseVersion(raw)
}

func parseVersion(raw string) (Version, error) {
	raw = strings.TrimSpace(raw)

	var numeric string
	for _, f := range strings.Fields(raw) {
		if f != "" && f[0] >= '0' && f[0] <= '9' {
			numeric = f
			break
		}
	}
	if numeric == "" {
		return Version{}, fmt.Errorf("could not parse git version from %q", raw)
	}

	v := Version{Raw: raw}
	fmt.Sscanf(numeric, "%d.%d.%d", &v.Major, &v.Minor, &v.Patch)
	return v, nil
}

// InsideRepo reports whether the current directory is inside a git working
// tree.
func InsideRepo() bool {
	out, err := gitOutput([]string{"rev-parse", "--is-inside-work-tree"})
	return err == nil && out == "true"
}

// RepoRoot returns the absolute path to the top level of the current
// repository's working tree.
func RepoRoot() (string, error) {
	out, err := gitOutput([]string{"rev-parse", "--show-toplevel"})
	if err != nil {
		return "", fmt.Errorf("not inside a git repository: %w", err)
	}
	return out, nil
}

// GitDir returns the absolute path to the current repository's .git
// directory (which may live outside the working tree, e.g. worktrees and
// submodules).
func GitDir() (string, error) {
	out, err := gitOutput([]string{"rev-parse", "--absolute-git-dir"})
	if err != nil {
		return "", fmt.Errorf("not inside a git repository: %w", err)
	}
	return out, nil
}

// Scope selects which config file git reads or writes.
type Scope string

const (
	// ScopeEffective reads the fully merged configuration (local overrides
	// global overrides system); it cannot be used for writes.
	ScopeEffective Scope = ""
	ScopeGlobal    Scope = "--global"
	ScopeLocal     Scope = "--local"
)

// ConfigGet reads a single-valued config key. ok is false when the key is
// unset; err is non-nil only for unexpected failures (e.g. malformed repo,
// git missing).
func ConfigGet(scope Scope, key string) (value string, ok bool, err error) {
	args := []string{"config"}
	if scope != ScopeEffective {
		args = append(args, string(scope))
	}
	args = append(args, "--get", key)

	out, ok, err := gitGet(args)
	if err != nil {
		return "", false, fmt.Errorf("git config --get %s: %w", key, err)
	}
	return out, ok, nil
}

// ConfigSet writes a single-valued config key at the given scope. scope
// must be ScopeGlobal or ScopeLocal.
func ConfigSet(scope Scope, key, value string) error {
	if scope == ScopeEffective {
		return fmt.Errorf("ConfigSet requires an explicit scope")
	}
	if err := gitSet([]string{"config", string(scope), key, value}); err != nil {
		return fmt.Errorf("git config %s %s: %w", scope, key, err)
	}
	return nil
}

// ConfigUnset removes a single-valued config key at the given scope.
// Unsetting a key that isn't set is not an error.
func ConfigUnset(scope Scope, key string) error {
	if scope == ScopeEffective {
		return fmt.Errorf("ConfigUnset requires an explicit scope")
	}
	if _, isSet, err := ConfigGet(scope, key); err != nil || !isSet {
		return err
	}
	if err := gitSet([]string{"config", string(scope), "--unset", key}); err != nil {
		return fmt.Errorf("git config %s --unset %s: %w", scope, key, err)
	}
	return nil
}

// RemoveSection removes the config section name (e.g. "commitsentinel",
// or "commitsentinel.rules" for its subsection) at the given scope, and
// reports whether it held any key. Its existence is checked first because
// git reports a missing section only through a localized message.
func RemoveSection(scope Scope, name string) (bool, error) {
	if scope == ScopeEffective {
		return false, fmt.Errorf("RemoveSection requires an explicit scope")
	}
	pattern := "^" + regexp.QuoteMeta(name) + `\.[^.]+$`
	_, found, err := gitGet([]string{"config", string(scope), "--get-regexp", pattern})
	if err != nil {
		return false, fmt.Errorf("git config %s --get-regexp %s: %w", scope, pattern, err)
	}
	if !found {
		return false, nil
	}
	if err := gitSet([]string{"config", string(scope), "--remove-section", name}); err != nil {
		return false, fmt.Errorf("git config %s --remove-section %s: %w", scope, name, err)
	}
	return true, nil
}

// ConfigSource adapts a Scope to the config.Source interface:
//
//	cfg, err := config.Load(gitcli.ConfigSource{Scope: gitcli.ScopeEffective})
type ConfigSource struct {
	Scope Scope
}

func (s ConfigSource) Get(key string) (value string, ok bool, err error) {
	return ConfigGet(s.Scope, key)
}
