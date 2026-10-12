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

echo "== attestation: skipped for a custom download base"
echo "$out" | grep -q "Skipped the attestation check (custom download base)" || { echo "FAIL: expected the custom-base note"; exit 1; }

# A fake gh: GH_MODE=ok verifies, bad fails verification, out is signed out.
mkdir -p "$WORK/fakebin"
cat > "$WORK/fakebin/gh" <<'GH'
#!/usr/bin/env bash
case "$1" in
  auth) [ "${GH_MODE:-ok}" != out ] ;;
  attestation) [ "${GH_MODE:-ok}" = ok ] ;;
  *) exit 1 ;;
esac
GH
chmod +x "$WORK/fakebin/gh"
attest() { # mode version dest [extra env...]
  mode="$1"; ver="$2"; dest="$3"; shift 3
  env GH_MODE="$mode" PATH="$WORK/fakebin:$PATH" HIKMA_DOWNLOAD_BASE="$BASE" HIKMA_VERIFY_ATTESTATION=1 \
    HIKMA_VERSION="$ver" HIKMA_INSTALL_DIR="$dest" "$@" bash "$ROOT/scripts/install.sh" 2>&1
}

echo "== attestation: releases before v0.3.0 are checksum only"
o="$(attest bad v0.2.0 "$WORK/bin-old")"
echo "$o" | grep -q "predates build attestations" || { echo "FAIL: old release should skip"; echo "$o"; exit 1; }
[ -x "$WORK/bin-old/$BIN" ] || { echo "FAIL: old release was not installed"; exit 1; }

echo "== attestation: verified"
o="$(attest ok v0.3.0 "$WORK/bin-ok")"
echo "$o" | grep -q "Attestation verified" || { echo "FAIL: expected a verified attestation"; echo "$o"; exit 1; }
[ -x "$WORK/bin-ok/$BIN" ] || { echo "FAIL: not installed after a good attestation"; exit 1; }

echo "== attestation: a pre-release of v0.3.0 is checked too"
o="$(attest ok v0.3.0-rc.1 "$WORK/bin-rc")"
echo "$o" | grep -q "Attestation verified" || { echo "FAIL: rc should be verified"; echo "$o"; exit 1; }

echo "== attestation: a failed check installs nothing"
if o="$(attest bad v0.3.0 "$WORK/bin-bad")"; then echo "FAIL: accepted a bad attestation"; echo "$o"; exit 1; fi
echo "$o" | grep -q "could not be verified" || { echo "FAIL: no failure message"; echo "$o"; exit 1; }
[ ! -e "$WORK/bin-bad/$BIN" ] || { echo "FAIL: installed after a failed attestation"; exit 1; }

echo "== attestation: signed-out gh skips, unless required"
o="$(attest out v0.3.0 "$WORK/bin-out")"
echo "$o" | grep -q "Skipped the attestation check (needs the gh CLI" || { echo "FAIL: expected a skip note"; echo "$o"; exit 1; }
if o="$(attest out v0.3.0 "$WORK/bin-req" HIKMA_REQUIRE_ATTESTATION=1)"; then echo "FAIL: REQUIRE did not fail"; echo "$o"; exit 1; fi
[ ! -e "$WORK/bin-req/$BIN" ] || { echo "FAIL: installed although attestation was required"; exit 1; }

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
