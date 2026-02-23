#!/usr/bin/env bash
set -euo pipefail

# Upgrade all direct dependencies (no transitive deps — mirrors latest-deps.yml)
DIRECT=$(grep -E '^\s+\S+ v[0-9]' go.mod | grep -v '// indirect' | awk '{print $1}')

if [ -z "$DIRECT" ]; then
  echo "No direct dependencies found in go.mod"
  exit 0
fi

echo "Upgrading direct dependencies..."
# shellcheck disable=SC2046
go get $(echo "$DIRECT" | awk '{print $1 "@latest"}')

echo "Tidying module..."
go mod tidy

echo "Done. Run 'go test ./...' to verify."
