# git-commit-sentinel

A git `commit-msg` hook, written in Go, that rejects commits whose message
does not follow the [Conventional Commits](https://www.conventionalcommits.org/)
specification.

## Format enforced

```
<type>(<scope>)!: <description>

<body>
```

- `type` must be one of: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`,
  `test`, `build`, `ci`, `chore`, `revert` (or a custom list, see below).
- `scope` is optional, parenthesized.
- `!` marks a breaking change and is optional.
- `description` must be non-empty and must not end with a period.
- if a body is present, it must be separated from the header by a blank line.

## Build

```sh
go build -o bin/git-commit-sentinel ./cmd/git-commit-sentinel
```

## Install as a git hook

From inside the target git repository:

```sh
/path/to/git-commit-sentinel/scripts/install-hook.sh
```

This builds the binary and wires it up as `.git/hooks/commit-msg`.

## Custom allowed types

The hook accepts additional arguments as the allowed type list. To customize
it, edit the generated `.git/hooks/commit-msg` script and pass your own types
after the message file path, e.g.:

```sh
exec "$(dirname "$0")/git-commit-sentinel" "$1" feat fix chore task
```

## Test

```sh
go test ./...
```
