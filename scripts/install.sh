#!/usr/bin/env bash
# Install the hikma CLI from a GitHub release.
#
#   curl -fsSL https://raw.githubusercontent.com/HasanKhatib/hikma-ai/main/scripts/install.sh | bash
#   gh api -H "Accept: application/vnd.github.raw+json" \
#     repos/HasanKhatib/hikma-ai/contents/scripts/install.sh | bash
#
# Pin a version:  ... | bash -s -- v0.1.0
#
# Environment:
#   HIKMA_VERSION       release tag to install (default: latest)
#   HIKMA_INSTALL_DIR   where to put the binary (default: ~/.local/bin)
#   HIKMA_REPO          GitHub repo to download from (default: HasanKhatib/hikma-ai)
#   HIKMA_DOWNLOAD_BASE base URL holding the archive and checksums.txt (testing)
#   HIKMA_VERIFY_ATTESTATION  set to 1 to also check the attestation when
#                       HIKMA_DOWNLOAD_BASE is set (testing)
#   HIKMA_REQUIRE_ATTESTATION set to 1 to fail instead of skipping the
#                       attestation check when gh is missing or signed out
#
# Works on macOS, Linux, and Windows (Git Bash). Uses curl, or gh when it is
# installed and signed in. The download is checked against checksums.txt, and,
# for releases that publish one (v0.3.0 and later), its build attestation when
# gh is available.
set -euo pipefail

REPO="${HIKMA_REPO:-HasanKhatib/hikma-ai}"
VERSION="${1:-${HIKMA_VERSION:-latest}}"
INSTALL_DIR="${HIKMA_INSTALL_DIR:-$HOME/.local/bin}"
FIRST_ATTESTED="0.3.0" # earlier releases have no build attestation

say() { printf '%s\n' "$*"; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }
have() { command -v "$1" >/dev/null 2>&1; }

# --- platform -------------------------------------------------------------
case "$(uname -s)" in
  Linux)                OS=linux;   EXT=tar.gz; BIN=hikma ;;
  Darwin)               OS=darwin;  EXT=tar.gz; BIN=hikma ;;
  MINGW*|MSYS*|CYGWIN*) OS=windows; EXT=zip;    BIN=hikma.exe ;;
  *) die "unsupported operating system: $(uname -s)" ;;
esac
case "$(uname -m)" in
  x86_64|amd64)  ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *) die "unsupported architecture: $(uname -m)" ;;
esac
if [ "$OS" = windows ] && [ "$ARCH" = arm64 ]; then
  die "no Windows arm64 build is published yet"
fi

ARCHIVE="hikma_${OS}_${ARCH}.${EXT}"
case "$VERSION" in
  latest) TAG="" ;;
  v*)     TAG="$VERSION" ;;
  *)      TAG="v$VERSION" ;;
esac

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

# --- download -------------------------------------------------------------
fetch_url() { # url dest
  curl -fsSL --retry 3 -o "$2" "$1"
}

download() {
  if [ -n "${HIKMA_DOWNLOAD_BASE:-}" ]; then
    fetch_url "$HIKMA_DOWNLOAD_BASE/$ARCHIVE" "$TMP/$ARCHIVE"
    fetch_url "$HIKMA_DOWNLOAD_BASE/checksums.txt" "$TMP/checksums.txt"
    return
  fi
  if have gh && gh auth status >/dev/null 2>&1; then
    if gh release download ${TAG:+"$TAG"} -R "$REPO" -p "$ARCHIVE" -p checksums.txt -D "$TMP" --clobber >/dev/null 2>&1; then
      return
    fi
    say "gh could not download the release; falling back to curl"
  fi
  have curl || die "curl is required (or install and sign in to the gh CLI)"
  if [ -z "$TAG" ]; then
    base="https://github.com/$REPO/releases/latest/download"
  else
    base="https://github.com/$REPO/releases/download/$TAG"
  fi
  fetch_url "$base/$ARCHIVE" "$TMP/$ARCHIVE" || die "could not download $ARCHIVE from $REPO (does the release exist, and is the repo public?)"
  fetch_url "$base/checksums.txt" "$TMP/checksums.txt" || die "could not download checksums.txt"
}

say "Installing hikma (${TAG:-latest}) for ${OS}/${ARCH}..."
download

# --- verify ---------------------------------------------------------------
sha256_of() {
  if have sha256sum; then sha256sum "$1" | cut -d' ' -f1
  elif have shasum; then shasum -a 256 "$1" | cut -d' ' -f1
  else die "sha256sum or shasum is required to verify the download"
  fi
}
want="$(awk -v f="$ARCHIVE" '$2 == f || $2 == "*" f { print $1 }' "$TMP/checksums.txt" | head -n 1)"
[ -n "$want" ] || die "$ARCHIVE is not listed in checksums.txt"
got="$(sha256_of "$TMP/$ARCHIVE")"
[ "$want" = "$got" ] || die "checksum mismatch for $ARCHIVE (expected $want, got $got)"
say "Checksum verified."

# --- attestation ----------------------------------------------------------
# A checksum only proves the download matches checksums.txt, which comes from
# the same place. The attestation proves the archive was built by this repo's
# release workflow.
resolve_tag() { # prints the release tag, or nothing when it cannot be found
  if [ -n "$TAG" ]; then printf '%s' "$TAG"; return; fi
  have curl || return 0
  curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/$REPO/releases/latest" 2>/dev/null | sed -n 's#.*/tag/##p'
}
older_than_attested() { # version without v or suffix
  v="${1%%-*}"
  [ "$(printf '%s\n%s\n' "$v" "$FIRST_ATTESTED" | sort -V | head -n 1)" != "$FIRST_ATTESTED" ]
}
verify_attestation() {
  if [ -n "${HIKMA_DOWNLOAD_BASE:-}" ] && [ "${HIKMA_VERIFY_ATTESTATION:-}" != 1 ]; then
    say "Skipped the attestation check (custom download base)."
    return
  fi
  resolved="$(resolve_tag)"
  if [ -n "$resolved" ] && older_than_attested "${resolved#v}"; then
    say "Release $resolved predates build attestations; checksum only."
    return
  fi
  if have gh && gh auth status >/dev/null 2>&1; then
    if gh attestation verify "$TMP/$ARCHIVE" -R "$REPO" --signer-workflow "$REPO/.github/workflows/release.yml" >"$TMP/attest.log" 2>&1; then
      say "Attestation verified."
    else
      cat "$TMP/attest.log" >&2
      die "the build attestation for $ARCHIVE could not be verified; nothing was installed"
    fi
  elif [ "${HIKMA_REQUIRE_ATTESTATION:-}" = 1 ]; then
    die "HIKMA_REQUIRE_ATTESTATION=1 but the gh CLI is missing or not signed in"
  else
    say "Skipped the attestation check (needs the gh CLI, signed in). To check it yourself:"
    say "  gh attestation verify <archive> -R $REPO"
  fi
}
verify_attestation

# --- extract and install --------------------------------------------------
mkdir -p "$TMP/out"
if [ "$EXT" = zip ]; then
  if have unzip; then
    unzip -q -o "$TMP/$ARCHIVE" -d "$TMP/out"
  elif have powershell.exe; then
    (cd "$TMP" && powershell.exe -NoProfile -Command "Expand-Archive -Force -Path '$ARCHIVE' -DestinationPath 'out'")
  else
    tar -xf "$TMP/$ARCHIVE" -C "$TMP/out"
  fi
else
  tar -xzf "$TMP/$ARCHIVE" -C "$TMP/out"
fi
[ -f "$TMP/out/$BIN" ] || die "$BIN not found in $ARCHIVE"

mkdir -p "$INSTALL_DIR"
cp "$TMP/out/$BIN" "$INSTALL_DIR/$BIN"
chmod +x "$INSTALL_DIR/$BIN"
say "Installed $INSTALL_DIR/$BIN"

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *) say ""
     say "Add $INSTALL_DIR to your PATH, for example:"
     say "  echo 'export PATH=\"$INSTALL_DIR:\$PATH\"' >> ~/.bashrc" ;;
esac
"$INSTALL_DIR/$BIN" --version || true
