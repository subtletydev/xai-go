#!/usr/bin/env bash
# Rewrites the upstream go_package import paths in the generated tree to this
# module's path.
#
# xAI's protos declare go_package = github.com/xai-org/xai-proto/..., but that
# module is not published, so the checked-in gen/ tree will not compile as
# generated. Run this after every regeneration.
#
# Only import statements are rewritten. The same path also appears inside the
# serialized file descriptors (rawDesc string literals), where it is preceded by
# a length prefix -- editing it there corrupts the descriptor and makes proto
# registration panic at init.

set -euo pipefail

UPSTREAM="github.com/xai-org/xai-proto"
GEN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODULE="$(cd "$GEN_DIR" && go list -m)"

changed=0
while IFS= read -r file; do
	# Match only gofmt'd import specs: a tab, an optional alias, then the
	# quoted path alone on the line.
	if perl -i -pe "
		BEGIN { \$c = 0 }
		\$c += s{^(\\t(?:[A-Za-z0-9_.]+ )?\")\\Q$UPSTREAM\\E/}{\$1$MODULE/};
		END { exit(\$c == 0) }
	" "$file"; then
		echo "rewrote $file"
		changed=$((changed + 1))
	fi
done < <(grep -rl --include="*.go" "$UPSTREAM" "$GEN_DIR/gen" || true)

if [ "$changed" -eq 0 ]; then
	echo "no import rewrites needed"
fi

# Guard: a bare upstream import surviving here means the pattern above missed a
# form protoc-gen-go emitted.
if grep -rnE "^\t([A-Za-z0-9_.]+ )?\"$UPSTREAM/" --include="*.go" "$GEN_DIR/gen"; then
	echo "error: upstream imports remain (see above)" >&2
	exit 1
fi
