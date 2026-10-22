#!/usr/bin/env bash
# Tests scripts/ci-changes.sh: which CI jobs a set of changed files turns on.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
fail=0

# check "description" "files (newline separated)" "expected go release scripts docs"
check() {
  local what="$1" files="$2" want="$3" got
  got="$(printf '%s' "$files" | bash "$ROOT/scripts/ci-changes.sh" | sed 's/.*=//' | tr '\n' ' ' | sed 's/ $//')"
  if [ "$got" != "$want" ]; then
    echo "FAIL $what: go release scripts docs = $got, want $want"
    fail=1
  else
    echo "ok   $what"
  fi
}

check "docs page only"        "docs/site/commands/upgrade.md"            "false false false true"
check "board only"            "docs/board/next.json"                     "false false false false"
check "readme only"           "README.md"                                "false false false false"
check "go code"               "cmd/hikma/commands/upgrade.go"            "true true false false"
check "a test file"           "internal/ui/color_test.go"                "true true false false"
check "go.mod"                "go.mod"                                   "true true false false"
check "goreleaser config"     ".goreleaser.yml"                          "true true true false"
check "install script"        "scripts/install.sh"                       "false true true false"
check "release workflow"      ".github/workflows/release.yml"            "false true false false"
check "pages workflow"        ".github/workflows/pages.yml"              "false false false true"
check "ci workflow"           ".github/workflows/ci.yml"                 "true true true true"
check "this script"           "scripts/ci-changes.sh"                    "true true true true"
check "go and docs together"  $'cmd/hikma/main.go\ndocs/site/index.md'   "true true false true"
check "a rename keeps both"   $'docs/site/a.md\ncmd/x.go'                "true true false true"
check "nothing listed"        ""                                         "true true true true"
check "a long list is unknown" "$(seq -f 'docs/board/f%g.json' 1 300)"   "true true true true"
check "just under the limit"  "$(seq -f 'docs/board/f%g.json' 1 299)"    "false false false false"

exit "$fail"
