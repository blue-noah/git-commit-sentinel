# git-commit-sentinel

A git `commit-msg` hook (Go, stdlib only) that validates commit messages
against [Conventional Commits](https://www.conventionalcommits.org/), with
independently configurable `off`/`warn`/`error` levels per rule.

## Install

macOS / Linux:

```sh
curl -fsSL https://raw.githubusercontent.com/blue-noah/git-commit-sentinel/main/scripts/install.sh | sh
```

Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/blue-noah/git-commit-sentinel/main/scripts/install.ps1 | iex
```

No Go toolchain, no `sudo`/admin required. Then: `git-commit-sentinel setup`.

- **Usage**: [docs/MANUAL.md](docs/MANUAL.md) — install, day-to-day
  behavior, configuration, troubleshooting.
- **Design decisions**: [docs/adr](docs/adr) — why it's built this way.
- **Docs site**: https://blue-noah.github.io/git-commit-sentinel/ — the same
  essentials, as a short static page.

## Build & test

```sh
go build ./...
go vet ./...
go test ./...
```

No external dependencies: only the Go standard library and the `git` CLI.

## License

[MIT](LICENSE)
