#!/usr/bin/env bash
# Cross-compiles git-commit-sentinel for distribution. Run from any
# machine with Go installed (no target OS/arch toolchain needed — Go
# cross-compiles natively); intended to be run on macOS per this
# project's workflow, but has no macOS-specific step itself.
#
# Usage: scripts/build-release.sh [version]
#   version defaults to 0.0.1
#
# Produces, under dist/:
#   git-commit-sentinel-<version>-darwin-arm64
#   git-commit-sentinel-<version>-windows-amd64.exe
#   SHA256SUMS.txt
set -euo pipefail

VERSION="${1:-0.0.1}"
ROOT="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
PKG="$ROOT/cmd/git-commit-sentinel"
DIST="$ROOT/dist"
LDFLAGS="-s -w -X main.version=${VERSION}"

# Targets: "<GOOS> <GOARCH> <output filename>"
TARGETS=(
	"darwin  arm64 git-commit-sentinel-${VERSION}-darwin-arm64"
	"windows amd64 git-commit-sentinel-${VERSION}-windows-amd64.exe"
)

rm -rf "$DIST"
mkdir -p "$DIST"

for target in "${TARGETS[@]}"; do
	read -r goos goarch out <<<"$target"
	echo "==> ${goos}/${goarch} -> dist/${out}"
	CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
		go build -trimpath -ldflags "$LDFLAGS" -o "$DIST/$out" "$PKG"
done

echo
echo "==> checksums"
(cd "$DIST" && shasum -a 256 -- * | tee SHA256SUMS.txt)

echo
echo "==> done"
ls -lh "$DIST"
