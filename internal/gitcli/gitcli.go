// Package gitcli wraps the subset of the git CLI that
// git-commit-sentinel needs: version detection, repository discovery, and
// reading/writing git config values.
package gitcli

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// The three git invocation seams below are function types rather than one
// multi-method interface: ConfigGet only ever needs the "get" operation,
// ConfigSet only "set", DetectVersion/InsideRepo/RepoRoot/GitDir only
// "output". A fat interface would force every caller (and every test
// fake) to depend on operations it never uses; a single-operation function
// type cannot have that problem (Interface Segregation).
type getFunc func(args []string) (out string, ok bool, err error)
type setFunc func(args []string) error
type outputFunc func(args []string) (string, error)

// gitGet, gitSet and gitOutput are the seams used by this package's
// functions. Tests may replace any subset of them; production code should
// leave them as the defaults below.
var (
	gitGet    getFunc    = execGet
	gitSet    setFunc    = execSet
	gitOutput outputFunc = execOutput
)

// execGet runs a config-style "get" command: ok is false when the queried
// key is unset (git's exit code 1), err is reserved for unexpected
// failures.
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

// execSet runs a command expected to have no useful stdout, returning any
// failure with its combined output attached for context.
func execSet(args []string) error {
	out, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// execOutput runs a command and returns its trimmed stdout.
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

// parseVersion is the pure part of DetectVersion, split out so it can be
// unit tested against sample output without spawning git.
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

	// Sscanf stops as soon as the "%d.%d.%d" pattern is satisfied, so a
	// platform suffix (e.g. "2.43.0.windows.1") is simply left unread, and
	// a missing trailing part (e.g. "2.9") just leaves Patch at its zero
	// value — matching this function's previous field-by-field parsing,
	// whose error this deliberately still discards.
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
	// global overrides system), exactly as git itself sees it. It cannot be
	// used for writes.
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

// ConfigSource adapts a git config scope to the config.Source interface
// (Get(key string) (string, bool, error)), so that package config can read
// git config without ever importing this package. Callers wire the two
// together, e.g.:
//
//	cfg, err := config.Load(gitcli.ConfigSource{Scope: gitcli.ScopeEffective})
type ConfigSource struct {
	Scope Scope
}

func (s ConfigSource) Get(key string) (value string, ok bool, err error) {
	return ConfigGet(s.Scope, key)
}
