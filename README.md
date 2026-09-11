# git-commit-sentinel

A git `commit-msg` hook (Go, stdlib only) that validates commit messages
against [Conventional Commits](https://www.conventionalcommits.org/), with
independently configurable `off`/`warn`/`error` levels per rule.

- **Usage**: [docs/MANUAL.md](docs/MANUAL.md) — install, day-to-day
  behavior, configuration, troubleshooting.
- **Design decisions**: [docs/adr](docs/adr) — why it's built this way.
- **Docs site**: https://diuis.github.io/git-commit-sentinel/ — the same
  essentials, as a short static page.

## Build & test

```sh
go build ./...
go vet ./...
go test ./...
```

No external dependencies: only the Go standard library and the `git` CLI.
