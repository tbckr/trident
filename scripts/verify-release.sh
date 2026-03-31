#!/usr/bin/env bash
set -euo pipefail

# verify-release.sh — verify a trident release artifact using GitHub Artifact Attestation
#
# Usage:
#   ./scripts/verify-release.sh <ARCHIVE>
#
# Example:
#   ./scripts/verify-release.sh trident_Linux_x86_64.tar.gz
#
# Requirements:
#   - gh CLI 2.49+ (https://cli.github.com/)

REPO="tbckr/trident"

ARCHIVE="${1:-}"

if [[ -z "$ARCHIVE" ]]; then
  echo "Usage: $0 <ARCHIVE>"
  echo "Example: $0 trident_Linux_x86_64.tar.gz"
  exit 1
fi

if [[ ! -f "$ARCHIVE" ]]; then
  echo "Archive not found: $ARCHIVE"
  exit 1
fi

echo "==> Verifying GitHub attestation..."
gh attestation verify "$ARCHIVE" --repo "$REPO"

BASENAME=$(basename "$ARCHIVE")
echo "==> OK: ${BASENAME} verified successfully"
