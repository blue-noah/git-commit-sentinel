#!/usr/bin/env bash
# Cross-compiles git-commit-sentinel for distribution. Works on any OS
# with Go installed — no target toolchain needed, Go cross-compiles
# natively — and is used both for local releases and by
# .github/workflows/release.yml, so there is one definition of "what we
# ship", not two.
#
# Usage: scripts/build-release.sh [version]
#   version defaults to 0.0.1
#
# Produces, under dist/:
#   git-commit-sentinel-<version>-darwin-arm64
#   git-commit-sentinel-<version>-windows-amd64.exe
#   git-commit-sentinel-<version>-linux-amd64
#   git-commit-sentinel-<version>-linux-arm64
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
	"linux   amd64 git-commit-sentinel-${VERSION}-linux-amd64"
	"linux   arm64 git-commit-sentinel-${VERSION}-linux-arm64"
)

rm -rf "$DIST"
mkdir -p "$DIST"

for target in "${TARGETS[@]}"; do
	read -r goos goarch out <<<"$target"
	echo "==> ${goos}/${goarch} -> dist/${out}"
	CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
		go build -trimpath -ldflags "$LDFLAGS" -o "$DIST/$out" "$PKG"
done

# Prefer sha256sum (Linux, most CI images); fall back to shasum -a 256
# (macOS, and anywhere sha256sum isn't installed).
if command -v sha256sum >/dev/null 2>&1; then
	CHECKSUM_CMD=(sha256sum)
else
	CHECKSUM_CMD=(shasum -a 256)
fi

echo
echo "==> checksums"
(cd "$DIST" && "${CHECKSUM_CMD[@]}" -- * | tee SHA256SUMS.txt)

echo
echo "==> done"
ls -lh "$DIST"
