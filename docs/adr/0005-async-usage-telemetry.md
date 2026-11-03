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
- **Natural retry, no daemon**: a failed flush puts its batch back in
  the queue for the next commit to retry. No background service, no
  timers.
- **Claim by rename to a unique name, no OS file lock**: the flush
  process renames the queue file to
  `telemetry-queue.<unix-nanos>.<pid>.sending` before reading it. The
  target name is unique, so the rename never depends on overwrite
  semantics (POSIX `rename` silently replaces an existing target, Windows
  refuses to). If the source is already gone, another flush claimed it
  first, and this one exits quietly. Avoids `flock`/`LockFileEx`, which
  don't behave the same across platforms.
  - On success the claimed file is deleted.
  - On failure (non-2xx, non-zero exit, timeout) its lines are appended
    back to the live queue, within the size cap below, and the claimed
    file is deleted. A permanently unreachable sink therefore never
    leaves a growing pile of `.sending` files.
  - **Stale claims**: a `.sending` file whose name-embedded timestamp is
    older than 10 minutes was left by a flush that crashed. The next
    flush claims it the same way (rename to a fresh unique name, skip on
    "source gone") and delivers it like any other batch. The age comes
    from the name, not the mtime, because a rename keeps the mtime of
    the last append.
- **Queue size cap, no age cap**: before appending, the hook checks the
  live queue's size; at 1 MiB or more it drops the new event instead of
  writing it (a few thousand events). Dropping the oldest would mean
  rewriting the file inside `git commit`. No separate age cap: every
  event carries `ts`, so the receiving side can discard what it
  considers too old.
- **Queue location is `$XDG_CONFIG_HOME/git-commit-sentinel/telemetry/`**
  (falling back to `~/.config`, the same resolution as the default
  hooks directory in [ADR 0002](0002-global-hook-via-core-hookspath.md)),
  holding `telemetry-queue.ndjson` and any `.sending` claims. A sibling
  of `hooks/`, not inside it: `uninstall` only unsets `core.hooksPath`
  once that directory is empty, and a `-scope local` install has no such
  directory at all. One queue per workstation, whatever the install
  scope. `uninstall` removes it.
- **Inactive until a sink is configured, then opt-out**: there is no
  built-in endpoint — this project runs no server, and the binary never
  phones home on its own. While `commitsentinel.telemetry.sink` is
  unset, the hook neither queues nor spawns anything. An org enables it
  by setting the sink in each workstation's global git config, and a
  developer can still opt out with `git config --global
  commitsentinel.telemetry off` — checked before any write, so `off`
  costs nothing.
- **Delivery is pluggable, via the same registry pattern as `Rule`**
  ([ADR 0004](0004-off-warn-error-levels.md)): a `Sink` interface
  (`Name() string`, `Send(events []Event) error`) that concrete sinks
  self-register from their own file's `init()`. Which one runs is chosen
  by `commitsentinel.telemetry.sink`. Two built-ins cover the general
  case without ever touching this codebase for a new backend. Both
  receive the batch as one JSON array of events, each valid against the
  schema, and both run under the flush's 10-second timeout:
  - `http` — POSTs the batch (`Content-Type: application/json`) to
    `commitsentinel.telemetry.http.endpoint`, which must be `https://`.
    Optional auth: `commitsentinel.telemetry.http.token`, sent as
    `Authorization: Bearer <token>`. That token lives in plain git
    config, so it is meant to be an ingest-only credential; anything
    stronger (mTLS, OAuth, a secret store) belongs in an `exec` sink.
    Any 2xx response means delivered.
  - `exec` — pipes the batch to stdin of
    `commitsentinel.telemetry.exec.path`, an external program the org
    provides. Exit code 0 means delivered. This is the actual
    extensibility escape hatch: a team wanting Splunk, Datadog, Kafka,
    or an internal service writes a small script/binary in *any*
    language and points `exec.path` at it — no Go code, no fork of this
    repo, no rebuild.

## Consequences

- The hook's own execution time is unaffected by network conditions —
  the only added cost is one local disk append.
- Delivery latency is "next time this workstation is online and someone
  commits" — acceptable for adoption tracking, not for real-time alerting.
- `repo.hash` is pseudonymous, not anonymous: whoever controls the
  telemetry server and already knows the org's repo URLs can reverse it.
- Delivery is best effort, which fits adoption tracking but not
  accounting. Events are dropped once the queue reaches its cap. On POSIX
  a hook that opened the queue just before a flush claimed it can append
  after the flush has read it, and that event is lost with the claimed
  file (on Windows the rename of an open file fails, and the flush just
  retries next time).
- The org owns the receiving side: endpoint, token issuance, storage and
  any retention policy. This project only defines what is sent and how.
- The `exec` sink itself is inherently cross-platform (`os/exec` is), but
  what an org points `exec.path` at is their own concern — a `.sh` script
  won't run unmodified on Windows and vice versa. Not this project's
  problem to solve, just to document in the manual once implemented.
