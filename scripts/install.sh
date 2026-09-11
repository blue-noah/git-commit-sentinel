#!/bin/sh
# Installs git-commit-sentinel on macOS or Linux: detects OS/arch,
# downloads the matching release binary, verifies its checksum, and puts
# it on PATH. No Go toolchain, no gh CLI, no sudo required.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/diuis/git-commit-sentinel/main/scripts/install.sh | sh
#
# Env overrides:
#   VERSION=v0.0.1        install a specific release (default: latest)
#   INSTALL_DIR=/some/dir install location (default: $HOME/.local/bin)
set -eu

REPO="diuis/git-commit-sentinel"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

os="$(uname -s)"
arch="$(uname -m)"

case "$os" in
	Darwin) goos="darwin" ;;
	Linux) goos="linux" ;;
	*)
		echo "error: unsupported OS: $os (macOS and Linux only; for Windows use install.ps1)" >&2
		exit 1
		;;
esac

case "$arch" in
	arm64 | aarch64) goarch="arm64" ;;
	x86_64 | amd64) goarch="amd64" ;;
	*)
		echo "error: unsupported architecture: $arch" >&2
		exit 1
		;;
esac

if [ "$goos" = "darwin" ] && [ "$goarch" = "amd64" ]; then
	echo "error: no darwin/amd64 build is published (only darwin/arm64, i.e. Apple Silicon)" >&2
	exit 1
fi

if [ -z "${VERSION:-}" ]; then
	VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" |
		grep -m1 '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')"
fi
if [ -z "${VERSION:-}" ]; then
	echo "error: could not resolve the latest release version" >&2
	exit 1
fi

version_num="${VERSION#v}"
file="git-commit-sentinel-${version_num}-${goos}-${goarch}"
base_url="https://github.com/${REPO}/releases/download/${VERSION}"

echo "==> installing git-commit-sentinel ${VERSION} (${goos}/${goarch})"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

curl -fsSL -o "$tmp/$file" "$base_url/$file"
curl -fsSL -o "$tmp/SHA256SUMS.txt" "$base_url/SHA256SUMS.txt"

echo "==> verifying checksum"
(
	cd "$tmp"
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum --ignore-missing -c SHA256SUMS.txt
	else
		shasum -a 256 --ignore-missing -c SHA256SUMS.txt
	fi
)

mkdir -p "$INSTALL_DIR"
chmod +x "$tmp/$file"
mv "$tmp/$file" "$INSTALL_DIR/git-commit-sentinel"
echo "==> installed to $INSTALL_DIR/git-commit-sentinel"

# Idempotent PATH check: do nothing if already on PATH; otherwise add it
# to the detected shell's rc file, but only if that file doesn't already
# reference INSTALL_DIR (so re-running this script never duplicates it).
case ":$PATH:" in
	*":$INSTALL_DIR:"*)
		echo "==> $INSTALL_DIR is already on PATH"
		;;
	*)
		shell_name="$(basename "${SHELL:-}")"
		rc=""
		line=""
		case "$shell_name" in
			zsh)
				rc="$HOME/.zshrc"
				line="export PATH=\"$INSTALL_DIR:\$PATH\""
				;;
			bash)
				rc="$HOME/.bashrc"
				line="export PATH=\"$INSTALL_DIR:\$PATH\""
				;;
			fish)
				rc="$HOME/.config/fish/config.fish"
				line="fish_add_path $INSTALL_DIR"
				;;
		esac

		if [ -n "$rc" ]; then
			if [ -f "$rc" ] && grep -qF "$INSTALL_DIR" "$rc" 2>/dev/null; then
				echo "==> $rc already references $INSTALL_DIR, leaving it as is"
			else
				mkdir -p "$(dirname "$rc")"
				printf '\n%s\n' "$line" >>"$rc"
				echo "==> added $INSTALL_DIR to PATH in $rc — restart your shell, or run: source $rc"
			fi
		else
			echo "==> add $INSTALL_DIR to your PATH manually (couldn't detect your shell)"
		fi
		;;
esac

echo
echo "==> next: git-commit-sentinel setup"
