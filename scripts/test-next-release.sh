#!/usr/bin/env bash
# Tests scripts/next-release.sh against throwaway repositories.
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
script="$here/next-release.sh"
fail=0

check() { # name expected actual
  if [ "$2" = "$3" ]; then
    echo "ok   $1"
  else
    echo "FAIL $1: expected [$2], got [$3]"
    fail=1
  fi
}

# repo <tags...>: a repo with one commit and the given tags, and a CHANGELOG
# written by the caller on stdin.
repo() {
  d=$(mktemp -d)
  git -C "$d" init -q
  cat >"$d/CHANGELOG.md"
  git -C "$d" add CHANGELOG.md
  git -C "$d" -c user.email=t@example.com -c user.name=t -c commit.gpgsign=false commit -q -m init
  for t in "$@"; do git -C "$d" tag "$t"; done
  echo "$d"
}
run() { (cd "$1" && bash "$script" CHANGELOG.md 2>/dev/null); }

d=$(repo v1.2.0 <<'X'
# Changelog

## [Unreleased]

## [1.3.0] - 2026-10-20

### Added
- a thing

## [1.2.0] - 2026-10-19
X
)
check "a new version with no tag is released" "1.3.0" "$(run "$d")"

d=$(repo v1.2.0 v1.3.0 <<'X'
## [Unreleased]

## [1.3.0] - 2026-10-20
## [1.2.0] - 2026-10-19
X
)
check "a version that is already tagged is not released again" "" "$(run "$d")"

d=$(repo v1.3.0 <<'X'
## [Unreleased]
- notes that are not released yet
## [1.3.0] - 2026-10-20
X
)
check "an Unreleased section alone releases nothing" "" "$(run "$d")"

d=$(repo v1.3.0 <<'X'
## [1.2.0] - 2026-10-19
X
)
check "a heading older than the newest release never goes backwards" "" "$(run "$d")"

d=$(repo v1.9.0 <<'X'
## [1.10.0] - 2026-10-20
X
)
check "1.10.0 is newer than 1.9.0 (versions, not text)" "1.10.0" "$(run "$d")"

d=$(repo v1.3.0 <<'X'
## [1.4.0-rc.1] - 2026-10-20
X
)
check "a pre-release is released" "1.4.0-rc.1" "$(run "$d")"

d=$(repo <<'X'
## [0.1.0] - 2026-01-01
X
)
check "the first release in a repository with no tags" "0.1.0" "$(run "$d")"

d=$(repo <<'X'
# no versions here
X
)
check "no version heading at all releases nothing" "" "$(run "$d")"

exit "$fail"
