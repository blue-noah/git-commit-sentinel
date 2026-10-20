#!/bin/sh
# Uninstalls git-commit-sentinel on macOS or Linux, undoing install.sh and
# setup: runs `git-commit-sentinel uninstall` to remove the global hook
# and settings, deletes the binary, and removes the PATH line install.sh
# added once the install directory is empty.
#
# Usage:
#   curl --proto '=https' --tlsv1.2 -fsSL https://raw.githubusercontent.com/blue-noah/git-commit-sentinel/main/scripts/uninstall.sh | sh
#
# Env overrides (set them on `sh`, after the pipe):
#   GIT_COMMIT_SENTINEL_INSTALL_DIR=/some/dir where it was installed (default: $HOME/.local/bin)
#   GIT_COMMIT_SENTINEL_KEEP_CONFIG=1         keep the commitsentinel.* git settings
set -eu

INSTALL_DIR="${GIT_COMMIT_SENTINEL_INSTALL_DIR:-$HOME/.local/bin}"
bin="$INSTALL_DIR/git-commit-sentinel"

if [ -f "$bin" ]; then
	set -- uninstall
	if [ "${GIT_COMMIT_SENTINEL_KEEP_CONFIG:-}" = "1" ]; then
		set -- "$@" -keep-config
	fi
	if ! "$bin" "$@"; then
		cat >&2 <<-EOF
			error: \`git-commit-sentinel uninstall\` failed, so nothing was deleted.
			It needs v0.0.4 or later: if it reported an unknown command, upgrade
			by running install.sh, then run this script again.
		EOF
		exit 1
	fi
	rm -f "$bin"
	echo "==> removed $bin"
else
	echo "==> no binary at $bin (already removed? set GIT_COMMIT_SENTINEL_INSTALL_DIR if you installed elsewhere)"
fi

# The PATH line also serves anything else in that directory: it goes only
# when the directory is empty, and the empty directory goes with it.
if [ -d "$INSTALL_DIR" ] && [ -n "$(ls -A "$INSTALL_DIR")" ]; then
	echo "==> kept $INSTALL_DIR on PATH: it holds other files"
else
	if [ -d "$INSTALL_DIR" ]; then
		rmdir "$INSTALL_DIR"
	fi
	sh_line="export PATH=\"$INSTALL_DIR:\$PATH\""
	fish_line="fish_add_path $INSTALL_DIR"
	for rc in "$HOME/.zshrc" "$HOME/.bashrc" "$HOME/.bash_profile" "$HOME/.bash_login" \
		"$HOME/.profile" "$HOME/.config/fish/config.fish"; do
		if [ -f "$rc" ] && grep -qxF -e "$sh_line" -e "$fish_line" "$rc"; then
			# Drops the exact lines install.sh wrote, with the blank line it
			# put before each. Rewritten in place, so a symlinked rc file
			# stays a symlink.
			tmp="$(mktemp)"
			L1="$sh_line" L2="$fish_line" awk '
				$0 == ENVIRON["L1"] || $0 == ENVIRON["L2"] { blank = 0; next }
				{ if (blank) print "" }
				{ blank = ($0 == "") }
				!blank { print }
				END { if (blank) print "" }
			' "$rc" >"$tmp"
			cat "$tmp" >"$rc"
			rm -f "$tmp"
			echo "==> removed $INSTALL_DIR from PATH in $rc"
		fi
	done
fi

cat <<-'EOF'

	==> git-commit-sentinel is uninstalled. Open a new terminal for the PATH change.

	Repositories set up with `setup -scope local` keep their own hook, which now
	fails every commit with "git-commit-sentinel: not found". Delete
	.git/hooks/commit-msg in each of them.
EOF
