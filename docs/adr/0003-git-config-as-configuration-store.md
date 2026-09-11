# 3. git config as the configuration store

## Status

Accepted

## Context

Rules need per-feature settings (types, header length, an off/warn/error
level each — [ADR 0004](0004-off-warn-error-levels.md)), at two scopes
(global, per-repo). [ADR 0001](0001-stdlib-only.md) rules out a
third-party parser, leaving: a hand-rolled file format (JSON, or custom
`KEY=VALUE`) with our own merge logic, or `git config` itself.

## Decision

Store everything as `git config` keys under `commitsentinel`:
`commitsentinel.types`, `commitsentinel.headerMaxLength`,
`commitsentinel.rules.<name>`.

## Consequences

- Global/local layering is git's own — reading with no explicit scope
  returns the merged value, exactly as git resolves it elsewhere. No
  custom merge logic to write or test.
- No file format to invent or parse: `internal/gitcli` shells out to
  `git config --get`/`git config <scope> <key> <value>`. Users already
  know how to inspect (`git config --get-regexp ^commitsentinel\.`) and
  edit it.
- Editing works through every existing git-config surface (CLI, GUI
  clients, direct file edits) — no bespoke `config` subcommand needed.
- Every read/write is a `git` subprocess call, not in-process file I/O —
  a handful of milliseconds, imperceptible in a commit-msg hook.
- `doctor` can inspect one scope in isolation (`--global` vs `--local`) to
  report *where* a setting comes from.
