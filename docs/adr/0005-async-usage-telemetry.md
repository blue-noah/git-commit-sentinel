# 5. Async, offline-safe usage telemetry (proposed)

## Status

Proposed — design only, not implemented.

## Context

The org wants to know who uses the hook, when, and how, across developer
workstations, aggregated centrally, to track adoption. Two hard
constraints: it must never add latency to `git commit`, and it must never
block (or error) when the workstation is offline.

## Decision

- **Event, metadata only, grouped — not a flat bag of fields**:
  who/where/what/emitted-by/run-against as separate nested objects, so a
  `Sink` (below) can consume or forward one group without parsing keys
  apart by naming convention. `environment.git_version` is there for a
  concrete reason: [ADR 0002](0002-global-hook-via-core-hookspath.md)'s
  global install needs git ≥ 2.9.0, so knowing the git-version spread
  across the fleet answers "how many workstations can't go global yet"
  without a separate inventory pass. Never the commit message content.
  Formally specified in
  [telemetry-event.schema.json](../telemetry-event.schema.json) (JSON
  Schema draft 2020-12) — language-agnostic on purpose, since an `exec`
  sink may be written in anything, not just Go; both the Go producer and
  any third-party sink validate against the same file, not a copy of it.

  ```json
  {
    "schema_version": 1,
    "ts": "2026-09-11T10:00:00Z",
    "actor": { "user": "dgallist", "host": "macbook-01" },
    "repo": { "hash": "a1b2c3d4e5f6" },
    "result": {
      "outcome": "rejected",
      "findings": [
        { "rule": "type", "level": "error" },
        { "rule": "format", "level": "error" }
      ]
    },
    "tool": { "name": "git-commit-sentinel", "version": "0.3.0" },
    "environment": { "git_version": "2.43.0" }
  }
  ```
- **Local queue, no network in the hook's critical path**: the hook
  appends one JSON line to a local queue file (location below) — a
  local disk write, sub-millisecond, works offline.
- **Detached flush, fire-and-forget, on both macOS and Windows**: after
  queuing, the hook spawns a separate `git-commit-sentinel
  telemetry-flush` process (`Start()`, never `Wait()`) and returns
  immediately. That detached process does the actual delivery, with its
  own short timeout; it can hang or fail without the hook — or `git
  commit` — ever waiting on it. Detaching so it outlives the hook needs
  OS-specific process attributes — the only platform-specific code this
  feature needs, isolated to two small files by build tag:
  - Unix (`syscall.SysProcAttr{Setsid: true}`): the child gets its own
    session, so it isn't killed when the hook's process group exits.
  - Windows (`syscall.SysProcAttr{CreationFlags: CREATE_NEW_PROCESS_GROUP
    | CREATE_NO_WINDOW}`): no `Setsid` there; this is the equivalent for
    surviving the parent and not flashing a console window.
- **Natural retry, no daemon**: a failed flush just leaves the queue
  file for the next commit to retry. No background service, no timers.
- **Atomic swap on flush, no OS file lock**: the flush process renames
  the queue file aside (`os.Rename`, atomic on POSIX and on Windows when
  the target doesn't already exist) before sending. If the rename fails,
  it assumes another flush is already in flight and exits quietly —
  avoids depending on `flock`/`LockFileEx`, which don't behave the same
  across platforms.
- **Queue location reuses the hooks config directory** from [ADR
  0002](0002-global-hook-via-core-hookspath.md) (already resolved
  identically on macOS and Windows by this project's convention) instead
  of inventing a second "state dir" path — e.g.
  `<that dir>/telemetry-queue.ndjson`.
- **Opt-out**: `git config --global commitsentinel.telemetry off` — checked
  before any write, so `off` costs nothing.
- **Delivery is pluggable, via the same registry pattern as `Rule`**
  ([ADR 0004](0004-off-warn-error-levels.md)): a `Sink` interface
  (`Name() string`, `Send(events []Event) error`) that concrete sinks
  self-register from their own file's `init()`. Which one runs is chosen
  by `commitsentinel.telemetry.sink`. Two built-ins cover the general
  case without ever touching this codebase for a new backend:
  - `http` — POSTs the batch as JSON to `commitsentinel.telemetry.http.endpoint`.
  - `exec` — pipes the batch as JSON to stdin of
    `commitsentinel.telemetry.exec.path`, an external program the org
    provides. This is the actual extensibility escape hatch: a team
    wanting Splunk, Datadog, Kafka, or an internal service writes a small
    script/binary in *any* language and points `exec.path` at it — no Go
    code, no fork of this repo, no rebuild.

## Consequences

- The hook's own execution time is unaffected by network conditions —
  the only added cost is one local disk append.
- Delivery latency is "next time this workstation is online and someone
  commits" — acceptable for adoption tracking, not for real-time alerting.
- `repo.hash` is pseudonymous, not anonymous: whoever controls the
  telemetry server and already knows the org's repo URLs can reverse it.
- Still open: exact server endpoint/auth, queue retention/size cap,
  whether to also cap by event age, and the recovery rule for a stale
  `.sending` file left behind by a flush that crashed mid-swap (rare on
  either OS, but must be defined — e.g. "sending" older than N minutes is
  folded back into the live queue). To resolve before implementation.
- The `exec` sink itself is inherently cross-platform (`os/exec` is), but
  what an org points `exec.path` at is their own concern — a `.sh` script
  won't run unmodified on Windows and vice versa. Not this project's
  problem to solve, just to document in the manual once implemented.
