#!/usr/bin/env bash
# Tests scripts/render-formula.sh with a fake checksums.txt (no network).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

{
  echo "1111111111111111111111111111111111111111111111111111111111111111  hikma_darwin_arm64.tar.gz"
  echo "2222222222222222222222222222222222222222222222222222222222222222  hikma_darwin_amd64.tar.gz"
  echo "3333333333333333333333333333333333333333333333333333333333333333  hikma_linux_arm64.tar.gz"
  echo "4444444444444444444444444444444444444444444444444444444444444444  hikma_linux_amd64.tar.gz"
  echo "5555555555555555555555555555555555555555555555555555555555555555  hikma_windows_amd64.zip"
} > "$WORK/checksums.txt"

echo "== render a formula"
"$ROOT/scripts/render-formula.sh" v1.2.3 "$WORK/checksums.txt" Example/hikma > "$WORK/hikma.rb"
for want in \
  'class Hikma < Formula' \
  'version "1.2.3"' \
  'https://github.com/Example/hikma/releases/download/v1.2.3/hikma_darwin_arm64.tar.gz' \
  'sha256 "1111111111111111111111111111111111111111111111111111111111111111"' \
  'https://github.com/Example/hikma/releases/download/v1.2.3/hikma_darwin_amd64.tar.gz' \
  'sha256 "2222222222222222222222222222222222222222222222222222222222222222"' \
  'hikma_linux_arm64.tar.gz' 'sha256 "3333333333333333333333333333333333333333333333333333333333333333"' \
  'hikma_linux_amd64.tar.gz' 'sha256 "4444444444444444444444444444444444444444444444444444444444444444"' \
  'bin.install "hikma"' 'generate_completions_from_executable'; do
  grep -qF "$want" "$WORK/hikma.rb" || { echo "FAIL: formula is missing: $want"; cat "$WORK/hikma.rb"; exit 1; }
done
if grep -q 'quarantine\|xattr\|postflight' "$WORK/hikma.rb"; then
  echo "FAIL: the formula must not need a quarantine workaround"; exit 1
fi
if command -v ruby >/dev/null 2>&1; then
  ruby -c "$WORK/hikma.rb" >/dev/null || { echo "FAIL: formula is not valid Ruby"; exit 1; }
fi

echo "== a version without the v prefix renders the same"
"$ROOT/scripts/render-formula.sh" 1.2.3 "$WORK/checksums.txt" Example/hikma | cmp - "$WORK/hikma.rb" \
  || { echo "FAIL: v prefix changed the output"; exit 1; }

echo "== a missing archive is an error"
grep -v linux_arm64 "$WORK/checksums.txt" > "$WORK/short.txt"
if "$ROOT/scripts/render-formula.sh" v1.2.3 "$WORK/short.txt" >/dev/null 2>&1; then
  echo "FAIL: rendered a formula with a missing checksum"; exit 1
fi

echo "OK"
