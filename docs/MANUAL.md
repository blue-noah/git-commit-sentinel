# git-commit-sentinel — user manual

Validates commit messages against [Conventional
Commits](https://www.conventionalcommits.org/) as a git `commit-msg` hook.

## Install

```sh
go install ./cmd/git-commit-sentinel
```

Make sure the binary is on `PATH` (`go install` puts it in `$(go env GOBIN)`,
or `$(go env GOPATH)/bin` if `GOBIN` is unset).

## Activate

```sh
git-commit-sentinel setup
```

Installs the hook **globally** — every repository on the machine, no
per-repo step. Requires git ≥ 2.9.0.

For one repository only (different rules, or older git):

```sh
git-commit-sentinel setup -scope local
```

Safe to re-run: unchanged flags are a no-op, changed flags update the
install in place. Won't overwrite a hook it didn't create unless `-force`.

## On every commit

- **Valid** → proceeds silently.
- **`warn` violation** (e.g. trailing period) → printed, commit proceeds.
- **`error` violation** (e.g. wrong type) → commit **rejected**:

```
$ git commit -m "fixed a bug"
git-commit-sentinel: error: [format] header "fixed a bug" does not match
the expected format: <type>(<scope>)!: <description>

commit rejected — fix the errors above, or adjust a rule's level with:
  git config --global commitsentinel.rules.<rule> off|warn|error
```

Format: `<type>(<scope>)!: <description>` — scope and `!` (breaking
change) are optional. Example: `feat(auth)!: drop legacy tokens`.

## Diagnose problems

```sh
git-commit-sentinel doctor
```

Checks git version, binary on `PATH`, hook status (global + local), and
effective config. `[FAIL]` = broken; `[WARN]` = informational. Run this
first whenever the hook doesn't fire.

## Configuration

Lives in `git config`, section `commitsentinel` — no separate config file.
Global applies everywhere; local (inside a repo) overrides it there.

```sh
git config --global commitsentinel.types "feat,fix,docs,chore"
git config --global commitsentinel.headerMaxLength 72
git config --global commitsentinel.rules.<rule> off|warn|error
```

Or seed at setup time: `git-commit-sentinel setup -types "feat,fix" -rule
description-period=error`.

### Rules

| Rule | Default | Checks |
|---|---|---|
| `format` | error | header matches `<type>(<scope>)!: <description>` |
| `type` | error | type is in `commitsentinel.types` |
| `description-empty` | error | description is not empty |
| `description-period` | warn | description doesn't end with `.` |
| `header-length` | warn | header ≤ `commitsentinel.headerMaxLength` (100) |
| `body-blank-line` | warn | blank line separates header and body |

Default types: `feat fix docs style refactor perf test build ci chore
revert`.

## Troubleshooting

| Symptom | Fix |
|---|---|
| Hook doesn't run | `doctor`. If `core.hooksPath` isn't set, run `setup`. |
| `doctor`: binary not on PATH | `go install ./cmd/git-commit-sentinel`; check `$(go env GOBIN)` is on `PATH`. |
| `setup`: "too old for a global install" | git < 2.9.0 — upgrade, or use `-scope local` per repo. |
| `setup`: "a hook already exists ... not created by git-commit-sentinel" | Another tool owns that hook. Use `-force` only if replacing it is intended. |
| A repo's hook is silently ignored | Its local `core.hooksPath` may override the global one — `doctor` reports this. |
| Disable a check | `git config --global commitsentinel.rules.<rule> off` |
| macOS: "cannot be opened because the developer cannot be verified" | The release binaries aren't signed/notarized. Only affects binaries downloaded via a browser (Gatekeeper quarantines them); `curl`/`wget` downloads and `go install`-built binaries are unaffected. Fix: `xattr -d com.apple.quarantine ./git-commit-sentinel-*-darwin-arm64`, or right-click → Open once. |
