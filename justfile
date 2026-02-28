# Release: tag next version with svu and push
release:
    #!/usr/bin/env bash
    set -euo pipefail
    next=$(svu next)
    git tag "${next}"
    git push
    git push --tags
    @echo "Released ${next}"

# Update Nix flake inputs
flake-update:
    nix flake update
