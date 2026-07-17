#!/usr/bin/env bash
# Tear down the Codespaces / DinD kind cluster created by kind-up.sh.
set -euo pipefail

CLUSTER_NAME="${KIND_CLUSTER_NAME:-zorba}"
export PATH="${HOME}/.local/bin:${PATH}"

command -v kind >/dev/null || { echo "kind required" >&2; exit 1; }

if kind get clusters 2>/dev/null | grep -qx "${CLUSTER_NAME}"; then
  kind delete cluster --name "${CLUSTER_NAME}"
  echo "Deleted kind cluster '${CLUSTER_NAME}'"
else
  echo "No kind cluster named '${CLUSTER_NAME}'"
fi
