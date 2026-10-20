#!/usr/bin/env bash
# Print the version to release, or nothing.
#
# The version is the newest "## [X.Y.Z]" heading in CHANGELOG.md. Nothing is
# printed when that version already has a tag, or when it is older than the
# newest release (a heading someone edited back), so running this twice never
# releases twice and never goes backwards.
#
# usage: next-release.sh [CHANGELOG.md]   (run inside the git repository)
set -euo pipefail

changelog="${1:-CHANGELOG.md}"

top=$(sed -n -E 's/^## \[([0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?)\].*/\1/p' "$changelog" | head -n 1)
[ -n "$top" ] || exit 0

if git rev-parse -q --verify "refs/tags/v$top" >/dev/null; then
  exit 0
fi

# A pre-release (1.4.0-rc.1) is compared with nothing; a release must be newer than
# every release tag already there.
case "$top" in
  *-*) ;;
  *)
    latest=$(git tag -l 'v[0-9]*.[0-9]*.[0-9]*' | grep -v -- '-' | sed 's/^v//' | sort -V | tail -n 1 || true)
    if [ -n "$latest" ] && [ "$(printf '%s\n%s\n' "$latest" "$top" | sort -V | tail -n 1)" != "$top" ]; then
      echo "not releasing $top: the newest release is already $latest" >&2
      exit 0
    fi
    ;;
esac

printf '%s\n' "$top"
