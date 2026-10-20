# git-commit-sentinel — user manual

Validates commit messages against [Conventional
Commits](https://www.conventionalcommits.org/) as a git `commit-msg` hook.

## Install

No Go toolchain needed — these scripts download the right release binary,
verify its checksum, and put it on `PATH` (no `sudo`/admin required).

Supported: macOS on Apple Silicon, Linux (x86-64 and arm64), Windows x64.
Intel Macs and Windows on ARM are **not supported** — the scripts stop
with an error there.

**macOS / Linux:**

```sh
curl --proto '=https' --tlsv1.2 -fsSL https://raw.githubusercontent.com/blue-noah/git-commit-sentinel/main/scripts/install.sh | sh
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/blue-noah/git-commit-sentinel/main/scripts/install.ps1 | iex
```

Installs the latest release to `~/.local/bin` (macOS/Linux) or
`%USERPROFILE%\bin` (Windows). Safe to re-run — won't duplicate a `PATH`
entry that's already there.

**Upgrade** by running the same command again: it replaces the binary in
place, and the hook keeps using it. Coming from v0.0.2 or earlier, also
re-run `git-commit-sentinel setup` once (`doctor` points this out).

The `PATH` change reaches **new** terminals only:

- **macOS / Linux**: open a new terminal before running
  `git-commit-sentinel` — or reload the shell file the installer names,
  e.g. `source ~/.zshrc`.
- **Windows**: the PowerShell window you installed from is updated right
  away; other open terminals and IDEs need a restart.

To pick a specific release or install location, set
`GIT_COMMIT_SENTINEL_VERSION` and/or `GIT_COMMIT_SENTINEL_INSTALL_DIR`. On
macOS/Linux set them on `sh`, after the pipe — before `curl` they would
only reach `curl`:

```sh
curl --proto '=https' --tlsv1.2 -fsSL https://raw.githubusercontent.com/blue-noah/git-commit-sentinel/main/scripts/install.sh | GIT_COMMIT_SENTINEL_VERSION=v0.0.4 GIT_COMMIT_SENTINEL_INSTALL_DIR="$HOME/bin" sh
```

```powershell
$env:GIT_COMMIT_SENTINEL_VERSION = "v0.0.4"; $env:GIT_COMMIT_SENTINEL_INSTALL_DIR = "C:\tools\bin"
irm https://raw.githubusercontent.com/blue-noah/git-commit-sentinel/main/scripts/install.ps1 | iex
```

<details>
<summary>Manual download, or building from source</summary>

Download a binary directly from the
[releases page](https://github.com/blue-noah/git-commit-sentinel/releases) —
pick `-darwin-arm64`, `-linux-amd64`, `-linux-arm64` or
`-windows-amd64.exe` — then `chmod +x` it (macOS/Linux) and place it on
`PATH` yourself. `SHA256SUMS.txt` in the same release lets you verify it:
`shasum -a 256 -c SHA256SUMS.txt` (macOS) or `sha256sum -c SHA256SUMS.txt`
(Linux).

With a Go toolchain instead, from a clone of this repository (the module
path isn't `github.com/...`, so `go install ...@latest` doesn't work):

```sh
go install ./cmd/git-commit-sentinel
```

(puts it in `$(go env GOBIN)`, or `$(go env GOPATH)/bin` if `GOBIN` is unset).

</details>

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
effective config. `[FAIL]` = broken; `[WARN]` = worth a look, but commits
are still checked. Each line says how to fix it. Run this first whenever
the hook doesn't fire.

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

## Uninstall

1. **Hook and settings** — remove what `setup` created:

   ```sh
   git-commit-sentinel uninstall
   ```

   It deletes the global hook only if git-commit-sentinel wrote it, unsets
   `core.hooksPath` once that directory holds no other hooks, and removes
   the global `commitsentinel.*` settings (`-keep-config` keeps them). In
   each repository set up with `-scope local`, run
   `git-commit-sentinel uninstall -scope local` as well. Hooks and
   settings it didn't create are left alone, and running it again is a
   no-op. Needs v0.0.4 or later: if `uninstall` is an unknown command,
   upgrade first (see [Install](#install)).

2. **Binary** — delete `~/.local/bin/git-commit-sentinel` (macOS/Linux) or
   `%USERPROFILE%\bin\git-commit-sentinel.exe` (Windows), or wherever
   `GIT_COMMIT_SENTINEL_INSTALL_DIR` pointed.

3. **`PATH`** — only if nothing else lives in that directory: remove the
   line the installer added to your shell file (`~/.zshrc`, `~/.bashrc`,
   `~/.bash_profile`, `~/.profile` or fish's `config.fish`), or on Windows
   the entry in the user `PATH` (Settings → System → About → Advanced
   system settings → Environment Variables).

## Troubleshooting

| Symptom | Fix |
|---|---|
| Hook doesn't run | `doctor`. If `core.hooksPath` isn't set, run `setup`. |
| `git-commit-sentinel: command not found` right after installing | The `PATH` change reaches new terminals only — open one (see [Install](#install)). On Windows, restart terminals and IDEs that were already open. |
| `git-commit-sentinel: not found` when committing from an IDE or GUI git client | Re-run `git-commit-sentinel setup`. Hooks it writes run the binary by its full path, independent of the app's `PATH`; hooks from v0.0.2 or earlier look it up on `PATH`. |
| `doctor`: binary not on PATH | Hooks written by `setup` still work, since they run the binary by full path. To run it by name, open a new terminal or re-run the install script (see [Install](#install)). |
| `doctor`: hook "was written by another git-commit-sentinel binary or version" | Re-run `git-commit-sentinel setup`, with `-scope local` for a repository's own hook. |
| A repo's hook is silently ignored | Its local `core.hooksPath` may override the global one — `doctor` reports this. |
| `setup`: "too old for a global install" | git < 2.9.0 — upgrade, or use `-scope local` per repo. |
| `setup`: "a hook already exists ... not created by git-commit-sentinel" | Another tool owns that hook. Use `-force` only if replacing it is intended. |
| Disable a check | `git config --global commitsentinel.rules.<rule> off` |
| Windows: SmartScreen or antivirus warns about the binary | The release binaries aren't code-signed. The install script verifies the SHA-256 checksum and its download isn't marked by SmartScreen. For a browser download, check it against `SHA256SUMS.txt` (`Get-FileHash <file>`), then `Unblock-File <file>`. |
| macOS: "cannot be opened because the developer cannot be verified" | The release binaries aren't signed/notarized. Only affects binaries downloaded via a browser (Gatekeeper quarantines them); `curl`/`wget` downloads and `go install`-built binaries are unaffected. Fix: `xattr -d com.apple.quarantine ./git-commit-sentinel-*-darwin-arm64`, or right-click → Open once. |
