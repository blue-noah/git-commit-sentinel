#!/bin/sh
# Used by .github/workflows/install-smoke.yml once git-commit-sentinel is
# on PATH: installs the hook globally, then checks git runs it on a real
# commit — a non-conventional message is rejected by the hook, a
# conventional one goes through.
set -eu

git config --global user.name "install-smoke"
git config --global user.email "install-smoke@example.invalid"

git-commit-sentinel setup

repo="$(mktemp -d)"
cd "$repo"
git init -q

if out="$(git commit --allow-empty -m "fixed a bug" 2>&1)"; then
	echo "error: the hook accepted a non-conventional commit message" >&2
	exit 1
fi
echo "$out"
case "$out" in
	*"git-commit-sentinel: error"*) ;;
	*)
		echo "error: the commit failed, but not because of the hook" >&2
		exit 1
		;;
esac

git commit -q --allow-empty -m "feat: smoke test"
echo "==> hook rejected the bad message and accepted the good one"

# Last, so a doctor false positive can't hide whether the hook works.
git-commit-sentinel doctor
