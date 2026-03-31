#!/usr/bin/env bash
set -euo pipefail

# verify-release.sh — verify a trident release artifact using GitHub attestation + checksums
#
# Usage:
#   ./scripts/verify-release.sh <VERSION> <ARCHIVE>
#
# Example:
#   ./scripts/verify-release.sh v0.5.0 trident_Linux_x86_64.tar.gz
#
# Requirements:
#   - gh CLI 2.49+ (https://cli.github.com/)
#   - curl

REPO="tbckr/trident"
BASE_URL="https://github.com/${REPO}/releases/download"

VERSION="${1:-}"
ARCHIVE="${2:-}"

if [[ -z "$VERSION" || -z "$ARCHIVE" ]]; then
  echo "Usage: $0 <VERSION> <ARCHIVE>"
  echo "Example: $0 v0.5.0 trident_Linux_x86_64.tar.gz"
  exit 1
fi

if [[ ! -f "$ARCHIVE" ]]; then
  echo "Archive not found: $ARCHIVE"
  exit 1
fi

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

CHECKSUMS="$TMPDIR/checksums.txt"

echo "==> Downloading checksums for ${VERSION}..."
curl -fsSL "${BASE_URL}/${VERSION}/checksums.txt" -o "$CHECKSUMS"

echo "==> Verifying GitHub attestation..."
gh attestation verify "$ARCHIVE" --repo "$REPO"

echo "==> Verifying archive checksum..."
BASENAME=$(basename "$ARCHIVE")

EXPECTED=$(grep "  ${BASENAME}$" "$CHECKSUMS" | awk '{print $1}')
if [[ -z "$EXPECTED" ]]; then
  echo "==> FAIL: ${BASENAME} not found in checksums.txt for ${VERSION}"
  exit 1
fi

if command -v sha256sum &>/dev/null; then
  ACTUAL=$(sha256sum "$ARCHIVE" | awk '{print $1}')
else
  ACTUAL=$(shasum -a 256 "$ARCHIVE" | awk '{print $1}')
fi

if [[ "$EXPECTED" != "$ACTUAL" ]]; then
  echo "==> FAIL: checksum mismatch for ${BASENAME}"
  echo "    expected: $EXPECTED"
  echo "    actual:   $ACTUAL"
  exit 1
fi

echo "==> OK: ${BASENAME} verified successfully"
