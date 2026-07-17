#!/usr/bin/env bash
set -euo pipefail

# Refresh Codespaces public URLs when the environment is a GitHub Codespace.
if [[ -n "${CODESPACE_NAME:-}" ]]; then
  ./scripts/codespaces/prepare-env.sh || true
fi
