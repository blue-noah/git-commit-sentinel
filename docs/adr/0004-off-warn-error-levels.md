# 4. Three-level, per-rule enforcement (off / warn / error)

## Status

Accepted

## Context

Teams reasonably disagree on strictness, often rule by rule — a missing
type may be worth blocking, a trailing period usually isn't. One global
strict/lenient switch, or a fixed rule set, can't express that.

## Decision

Validation is independent rules (`format`, `type`, `description-empty`,
`description-period`, `header-length`, `body-blank-line` — see
[MANUAL.md](../MANUAL.md)), each with its own level:

- `off` — never checked.
- `warn` — reported, commit proceeds.
- `error` — reported, commit rejected.

Read per rule from `commitsentinel.rules.<name>`, each defaulting
independently (`format`/`type`/`description-empty` → error; the more
stylistic ones → warn).

## Consequences

- Gradual adoption: start everything at `warn`, promote to `error` once
  the noise is acceptable — no all-or-nothing switch.
- A new rule ships with its own default level and is backward compatible:
  it only affects repos that haven't already configured it.
- Exit code depends only on whether *any* finding is `error`-level; `warn`
  findings are always printed but never block.
