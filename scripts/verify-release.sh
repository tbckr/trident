#!/usr/bin/env bash
set -euo pipefail

# verify-release.sh — verify a trident release artifact using cosign + checksums
#
# Usage:
#   ./scripts/verify-release.sh <VERSION> <ARCHIVE>
#
# Example:
#   ./scripts/verify-release.sh v0.5.0 trident_Linux_x86_64.tar.gz
#
# Requirements:
#   - cosign v2+ (https://docs.sigstore.dev/cosign/system_config/installation/)
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

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

CHECKSUMS="$TMPDIR/checksums.txt"
BUNDLE="$TMPDIR/checksums.txt.sigstore.json"
PROVENANCE_BUNDLE="$TMPDIR/checksums.txt.slsa-provenance.sigstore.json"

echo "==> Downloading checksums for ${VERSION}..."
curl -fsSL "${BASE_URL}/${VERSION}/checksums.txt" -o "$CHECKSUMS"
curl -fsSL "${BASE_URL}/${VERSION}/checksums.txt.sigstore.json" -o "$BUNDLE"
curl -fsSL "${BASE_URL}/${VERSION}/checksums.txt.slsa-provenance.sigstore.json" -o "$PROVENANCE_BUNDLE"

echo "==> Verifying SLSA provenance..."
cosign verify-blob-attestation \
  --bundle "$PROVENANCE_BUNDLE" \
  --type slsaprovenance1 \
  --certificate-identity "https://github.com/${REPO}/.github/workflows/release.yml@refs/tags/${VERSION}" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  "$CHECKSUMS"

echo "==> Verifying cosign signature..."
cosign verify-blob \
  --bundle "$BUNDLE" \
  --certificate-identity "https://github.com/${REPO}/.github/workflows/release.yml@refs/tags/${VERSION}" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  "$CHECKSUMS"

echo "==> Verifying archive checksum..."
if [[ ! -f "$ARCHIVE" ]]; then
  echo "Archive not found: $ARCHIVE"
  exit 1
fi

if command -v sha256sum &>/dev/null; then
  EXPECTED=$(grep "  ${ARCHIVE}$" "$CHECKSUMS" | awk '{print $1}')
  ACTUAL=$(sha256sum "$ARCHIVE" | awk '{print $1}')
else
  EXPECTED=$(grep "  ${ARCHIVE}$" "$CHECKSUMS" | awk '{print $1}')
  ACTUAL=$(shasum -a 256 "$ARCHIVE" | awk '{print $1}')
fi

if [[ "$EXPECTED" == "$ACTUAL" ]]; then
  echo "==> OK: ${ARCHIVE} verified successfully"
else
  echo "==> FAIL: checksum mismatch for ${ARCHIVE}"
  echo "    expected: $EXPECTED"
  echo "    actual:   $ACTUAL"
  exit 1
fi
