# 2. Global install by default, via core.hooksPath

## Status

Accepted

## Context

Git hooks are normally per-repository (`.git/hooks/<name>`) — not copied
by `git clone`, reinstalled by hand in every repo. We want validation to
be the default everywhere, not an opt-in per repository.

Git's `core.hooksPath` config (settable globally) points at a directory
used *instead of* `.git/hooks`, for every repo that doesn't override it
locally. Introduced in **git 2.9.0** (2016).

## Decision

`setup` installs globally by default: writes `commit-msg` into
`$XDG_CONFIG_HOME/git-commit-sentinel/hooks` and points `git config
--global core.hooksPath` there. Both `setup` and `doctor` detect the git
version and refuse/warn below 2.9.0.

`setup -scope local` covers the two cases the global path doesn't fit: git
< 2.9, or a repo needing different rules — writes directly into that
repo's `.git/hooks/commit-msg`.

## Consequences

- One `setup` run covers the whole machine, including future clones.
- git consults only *one* hooksPath per repo: a local `core.hooksPath`
  silently shadows the global one. `doctor` detects and reports this.
- Conflicts with any other tool that also claims `core.hooksPath`
  globally — `setup` refuses to repoint an existing value without
  `-force`.
- The installed script just `exec`s `git-commit-sentinel hook "$1"` via
  `PATH`, not a copied binary — upgrading the binary needs no re-setup.
