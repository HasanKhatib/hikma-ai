#!/usr/bin/env bash
# Decides which CI jobs a change needs. Reads the changed file paths on stdin, one
# per line, and prints name=true or name=false lines for $GITHUB_OUTPUT.
#
#   go       the Go test matrix (all three systems)
#   release  the GoReleaser dry run
#   scripts  the install script and Homebrew formula tests
#   docs     the docs build
#
# When it cannot tell what changed (no paths, or GitHub's list is cut at 300 files)
# it says everything changed, so a job is never skipped by mistake. A change to
# the workflow itself, or to this script, runs every job it controls.
set -euo pipefail

files="$(cat)"
count=0
[ -n "$files" ] && count="$(printf '%s\n' "$files" | grep -c .)"

touches() { [ "$all" = true ] || printf '%s\n' "$files" | grep -qE "$1"; }

all=false
if [ "$count" -eq 0 ] || [ "$count" -ge 300 ]; then
  all=true
fi

ci='\.github/workflows/ci\.yml|scripts/ci-changes\.sh'

out() { if touches "$2"; then echo "$1=true"; else echo "$1=false"; fi; }

out go      "^(cmd|internal|e2e)/|^go\.(mod|sum)$|^\.goreleaser\.yml$|^($ci)$"
out release "^(cmd|internal)/|^go\.(mod|sum)$|^\.goreleaser\.yml$|^scripts/|^\.github/workflows/release\.yml$|^($ci)$"
out scripts "^scripts/|^\.goreleaser\.yml$|^($ci)$"
out docs    "^docs/site/|^\.github/workflows/pages\.yml$|^($ci)$"
