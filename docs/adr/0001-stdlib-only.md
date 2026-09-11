# 1. No external Go dependencies

## Status

Accepted

## Context

git-commit-sentinel runs on every commit, on every developer machine. It
should build instantly and carry no dependency chain to audit, for a tool
whose entire job is checking a string against a small grammar.

## Decision

Use only the Go standard library. No third-party module is imported
anywhere in the module — including for configuration (see [ADR
0003](0003-git-config-as-configuration-store.md), which avoids needing a
config-file parser at all).

## Consequences

- Zero supply-chain surface: no `go.sum` to audit, no transitive CVEs.
- Trivial, fast, offline-capable builds.
- Slightly more code written by hand (e.g. CLI flag handling) instead of
  imported — acceptable given the small scope.
- Revisit if a future requirement needs something the stdlib doesn't do
  well (e.g. real YAML).
