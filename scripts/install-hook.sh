#!/usr/bin/env sh
# Builds git-commit-sentinel and installs it as the commit-msg hook
# for the current git repository.
set -eu

repo_root=$(git rev-parse --show-toplevel)
hooks_dir="$repo_root/.git/hooks"
bin_name="git-commit-sentinel"
project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

go build -C "$project_root" -o "$hooks_dir/$bin_name" ./cmd/git-commit-sentinel

cat > "$hooks_dir/commit-msg" <<EOF
#!/usr/bin/env sh
exec "\$(dirname "\$0")/$bin_name" "\$1"
EOF
chmod +x "$hooks_dir/commit-msg" "$hooks_dir/$bin_name"

echo "Installed commit-msg hook at $hooks_dir/commit-msg"
