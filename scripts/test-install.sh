#!/usr/bin/env bash
# Tests scripts/install.sh against a local fake release (no network).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

case "$(uname -s)" in
  Linux)                OS=linux;   EXT=tar.gz; BIN=hikma ;;
  Darwin)               OS=darwin;  EXT=tar.gz; BIN=hikma ;;
  MINGW*|MSYS*|CYGWIN*) OS=windows; EXT=zip;    BIN=hikma.exe ;;
esac
case "$(uname -m)" in
  x86_64|amd64)  ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
esac
ARCHIVE="hikma_${OS}_${ARCH}.${EXT}"

echo "== build a host binary"
mkdir -p "$WORK/release" "$WORK/pkg"
(cd "$ROOT" && go build -ldflags "-X github.com/hasankhatib/hikma-ai/cmd/hikma/commands.version=9.9.9" -o "$WORK/pkg/$BIN" ./cmd/hikma)

echo "== package it like GoReleaser does"
if [ "$EXT" = zip ]; then
  (cd "$WORK/pkg" && powershell.exe -NoProfile -Command "Compress-Archive -Force -Path '$BIN' -DestinationPath '../release/$ARCHIVE'")
else
  tar -czf "$WORK/release/$ARCHIVE" -C "$WORK/pkg" "$BIN"
fi
sum() { if command -v sha256sum >/dev/null; then sha256sum "$1"; else shasum -a 256 "$1"; fi; }
(cd "$WORK/release" && sum "$ARCHIVE" | awk -v f="$ARCHIVE" '{print $1 "  " f}' > checksums.txt)

# Windows curl cannot read MSYS paths like /tmp/...; use a mixed C:/... path there.
RELEASE_DIR="$WORK/release"
if command -v cygpath >/dev/null 2>&1; then RELEASE_DIR="$(cygpath -m "$RELEASE_DIR")"; fi
case "$RELEASE_DIR" in
  /*) BASE="file://$RELEASE_DIR" ;;
  *)  BASE="file:///$RELEASE_DIR" ;;
esac

echo "== install"
DEST="$WORK/bin"
out="$(HIKMA_DOWNLOAD_BASE="$BASE" HIKMA_INSTALL_DIR="$DEST" bash "$ROOT/scripts/install.sh")"
echo "$out"
echo "$out" | grep -q "Checksum verified" || { echo "FAIL: checksum was not verified"; exit 1; }
[ -x "$DEST/$BIN" ] || { echo "FAIL: binary not installed"; exit 1; }
"$DEST/$BIN" --version | grep -q "9.9.9" || { echo "FAIL: wrong version"; exit 1; }

echo "== a tampered download is rejected"
printf 'tampered' >> "$WORK/release/$ARCHIVE"
if HIKMA_DOWNLOAD_BASE="$BASE" HIKMA_INSTALL_DIR="$WORK/bin2" bash "$ROOT/scripts/install.sh" >"$WORK/tamper.log" 2>&1; then
  echo "FAIL: tampered archive was accepted"; exit 1
fi
grep -q "checksum mismatch" "$WORK/tamper.log" || { echo "FAIL: no mismatch message"; cat "$WORK/tamper.log"; exit 1; }
[ ! -e "$WORK/bin2/$BIN" ] || { echo "FAIL: binary installed after a mismatch"; exit 1; }

echo "== a missing entry in checksums.txt is rejected"
: > "$WORK/release/checksums.txt"
if HIKMA_DOWNLOAD_BASE="$BASE" HIKMA_INSTALL_DIR="$WORK/bin3" bash "$ROOT/scripts/install.sh" >/dev/null 2>&1; then
  echo "FAIL: accepted an archive missing from checksums.txt"; exit 1
fi

echo "OK"
