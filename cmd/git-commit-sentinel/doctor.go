package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"git-commit-sentinel/internal/config"
	"git-commit-sentinel/internal/conventionalcommit"
	"git-commit-sentinel/internal/gitcli"
	"git-commit-sentinel/internal/hookfile"
)

var lookPath = exec.LookPath

type checkLevel int

const (
	levelOK checkLevel = iota
	levelWarn
	levelFail
)

func (l checkLevel) label() string {
	switch l {
	case levelOK:
		return "[OK]  "
	case levelWarn:
		return "[WARN]"
	default:
		return "[FAIL]"
	}
}

type checkResult struct {
	Level   checkLevel
	Message string
}

func ok(format string, a ...any) checkResult { return checkResult{levelOK, fmt.Sprintf(format, a...)} }
func warn(format string, a ...any) checkResult {
	return checkResult{levelWarn, fmt.Sprintf(format, a...)}
}
func fail(format string, a ...any) checkResult {
	return checkResult{levelFail, fmt.Sprintf(format, a...)}
}

type section struct {
	Title   string
	Results []checkResult
}

func runDoctor(args []string) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprintln(os.Stderr, "usage: git-commit-sentinel doctor") }
	if err := fs.Parse(args); err != nil {
		return 2
	}

	sections := []section{
		{"git", checkGit()},
		{"binary", checkBinaryOnPath()},
		{"global hook", checkGlobalHook()},
		{"local repository", checkLocalRepoSection()},
		{"effective configuration", checkConfig()},
	}

	healthy := printReport(os.Stdout, sections)

	fmt.Println()
	if !healthy {
		fmt.Println("some checks failed — see [FAIL] lines above.")
		return 1
	}
	fmt.Println("all checks passed.")
	return 0
}

func printReport(w io.Writer, sections []section) (healthy bool) {
	healthy = true
	for _, s := range sections {
		fmt.Fprintln(w, s.Title)
		for _, r := range s.Results {
			fmt.Fprintf(w, "%s %s\n", r.Level.label(), r.Message)
			if r.Level == levelFail {
				healthy = false
			}
		}
		fmt.Fprintln(w)
	}
	return healthy
}

func checkGit() []checkResult {
	ver, err := gitcli.DetectVersion()
	if err != nil {
		return []checkResult{fail("%v", err)}
	}
	results := []checkResult{ok("version %s", ver)}
	if ver.AtLeast(2, 9) {
		results = append(results, ok("supports core.hooksPath (global installs available)"))
	} else {
		results = append(results, warn("git < 2.9.0: global installs via core.hooksPath are not supported; use `setup -scope local` in each repository"))
	}
	return results
}

func checkBinaryOnPath() []checkResult {
	p, err := lookPath(hookfile.BinaryName)
	if err != nil {
		return []checkResult{fail(
			"%q not found on PATH; the installed hook will fail to run (build/install it, e.g. `go install ./cmd/git-commit-sentinel`)",
			hookfile.BinaryName,
		)}
	}
	return []checkResult{ok("found at %s", p)}
}

func checkGlobalHook() []checkResult {
	hooksPath, isSet, err := gitcli.ConfigGet(gitcli.ScopeGlobal, "core.hooksPath")
	if err != nil {
		return []checkResult{fail("%v", err)}
	}
	if !isSet {
		return []checkResult{warn("core.hooksPath is not set globally; run `git-commit-sentinel setup` to install it")}
	}
	return []checkResult{
		ok("core.hooksPath = %s", hooksPath),
		checkHookFile(filepath.Join(hooksPath, "commit-msg")),
	}
}

func checkLocalRepoSection() []checkResult {
	if !gitcli.InsideRepo() {
		return []checkResult{ok("not inside one, skipping")}
	}
	return checkLocalRepo()
}

func checkLocalRepo() []checkResult {
	root, err := gitcli.RepoRoot()
	if err != nil {
		return []checkResult{fail("%v", err)}
	}
	results := []checkResult{ok("repository: %s", root)}

	localOverride, hasLocalOverride, err := gitcli.ConfigGet(gitcli.ScopeLocal, "core.hooksPath")
	if err != nil {
		return append(results, fail("%v", err))
	}
	if hasLocalOverride {
		resolved := localOverride
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(root, resolved)
		}
		return append(results,
			ok("local core.hooksPath = %s (overrides global)", localOverride),
			checkHookFile(filepath.Join(resolved, "commit-msg")),
		)
	}

	_, hasGlobal, err := gitcli.ConfigGet(gitcli.ScopeGlobal, "core.hooksPath")
	if err != nil {
		return append(results, fail("%v", err))
	}
	gitDir, err := gitcli.GitDir()
	if err != nil {
		return append(results, fail("%v", err))
	}

	localHook := filepath.Join(gitDir, "hooks", "commit-msg")
	switch _, statErr := os.Stat(localHook); {
	case statErr == nil:
		results = append(results, checkHookFile(localHook))
		if hasGlobal {
			results = append(results, warn(".git/hooks/commit-msg is ignored because global core.hooksPath is set; git only consults one hooksPath"))
		}
	case hasGlobal:
		results = append(results, ok("no local hook; relying on the global install"))
	default:
		results = append(results, warn("no local or global hook installed; commits in this repository are not validated"))
	}
	return results
}

func checkConfig() []checkResult {
	cfg, err := config.Load(gitcli.ConfigSource{Scope: gitcli.ScopeEffective}, conventionalcommit.RuleNames())
	if err != nil {
		return []checkResult{fail("%v", err)}
	}
	rules := conventionalcommit.Rules()
	results := make([]checkResult, 0, len(rules)+2)
	for _, r := range rules {
		level := conventionalcommit.EffectiveLevel(cfg, r)
		results = append(results, ok("%-20s %-6s (%s)", r.Name(), level, r.Description()))
	}
	results = append(results,
		ok("%-20s %v", "types", cfg.AllowedTypes()),
		ok("%-20s %d", "headerMaxLength", cfg.MaxHeaderLength()),
	)
	return results
}

func checkHookFile(path string) checkResult {
	data, err := os.ReadFile(path)
	if err != nil {
		return fail("%s: %v", path, err)
	}
	if !hookfile.IsManaged(string(data)) {
		return warn("%s exists but was not created by git-commit-sentinel", path)
	}
	if info, err := os.Stat(path); err == nil && info.Mode()&0o111 == 0 {
		return fail("%s is not executable", path)
	}
	return ok("%s is installed and managed", path)
}
