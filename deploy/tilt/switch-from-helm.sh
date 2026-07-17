#!/usr/bin/env bash
# One-time: remove Helm releases so Tilt owns namespace dev (no duplicate Deployments/PVCs).
set -euo pipefail
NS="${K8S_NAMESPACE:-dev}"

if helm list -n "$NS" 2>/dev/null | grep -qE 'zorbahealth|zorba-infra'; then
  echo "Uninstalling Helm releases in namespace $NS..."
  helm uninstall zorbahealth -n "$NS" 2>/dev/null || true
  helm uninstall zorba-infra -n "$NS" 2>/dev/null || true
else
  echo "No zorbahealth/zorba-infra Helm release in $NS"
fi

echo "Done. Start the local stack with: ./deploy/tilt/preflight.sh && tilt up"
