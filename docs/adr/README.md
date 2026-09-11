# Architecture Decision Records

One file per significant decision, in the standard [ADR
format](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions)
(Nygard): Status, Context, Decision, Consequences. Never edited after
acceptance — a changed decision gets a new ADR that supersedes the old one.

| ADR | Decision |
|---|---|
| [0001](0001-stdlib-only.md) | No external Go dependencies |
| [0002](0002-global-hook-via-core-hookspath.md) | Global install by default, via `core.hooksPath` |
| [0003](0003-git-config-as-configuration-store.md) | `git config` as the configuration store |
| [0004](0004-off-warn-error-levels.md) | Three-level, per-rule enforcement (off/warn/error) |
| [0005](0005-async-usage-telemetry.md) | Async, offline-safe usage telemetry (proposed) |
