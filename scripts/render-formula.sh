#!/usr/bin/env bash
# Print the Homebrew formula for a release.
#
#   scripts/render-formula.sh <version> <checksums.txt> [owner/repo]
#
# The formula installs the prebuilt release archives, so Homebrew never marks
# the binary as quarantined and no notarization or install hook is needed.
set -euo pipefail

VERSION="${1:?usage: render-formula.sh <version> <checksums.txt> [owner/repo]}"
CHECKSUMS="${2:?usage: render-formula.sh <version> <checksums.txt> [owner/repo]}"
REPO="${3:-HasanKhatib/hikma-ai}"
VERSION="${VERSION#v}"

sha_for() { # archive name
  local sum
  sum="$(awk -v f="$1" '$2 == f || $2 == "*" f { print $1 }' "$CHECKSUMS" | head -n 1)"
  [ -n "$sum" ] || { echo "error: $1 is not listed in $CHECKSUMS" >&2; exit 1; }
  printf '%s' "$sum"
}

block() { # os arch hardware-block
  local archive="hikma_$1_$2.tar.gz" sum
  sum="$(sha_for "$archive")" || exit 1
  printf '      url "https://github.com/%s/releases/download/v%s/%s"\n' "$REPO" "$VERSION" "$archive"
  printf '      sha256 "%s"\n' "$sum"
}

# Resolve every checksum first: a failure inside the heredoc would not stop the script.
DARWIN_ARM="$(block darwin arm64)" || exit 1
DARWIN_INTEL="$(block darwin amd64)" || exit 1
LINUX_ARM="$(block linux arm64)" || exit 1
LINUX_INTEL="$(block linux amd64)" || exit 1

cat <<EOF
class Hikma < Formula
  desc "Install, create, and publish reusable AI-agent skills from any repo"
  homepage "https://github.com/${REPO}"
  version "${VERSION}"
  license "MIT"

  on_macos do
    on_arm do
${DARWIN_ARM}
    end
    on_intel do
${DARWIN_INTEL}
    end
  end

  on_linux do
    on_arm do
${LINUX_ARM}
    end
    on_intel do
${LINUX_INTEL}
    end
  end

  def install
    bin.install "hikma"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/hikma --version")
  end
end
EOF
