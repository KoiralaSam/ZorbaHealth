#!/usr/bin/env bash
# Open GCP ingress for LiveKit WebRTC (signaling + media).
#
# Do NOT run this on the Zorba VM itself. That instance uses the default compute
# service account without cloud-platform/compute scopes, so gcloud returns:
#   Request had insufficient authentication scopes.
#
# Run from Google Cloud Shell (console.cloud.google.com → Cloud Shell), or any
# laptop after: gcloud auth login && gcloud config set project PROJECT
#
# Without tcp:7880, mobile/web clients fail with:
#   could not establish signal connection: Abort handler called
set -euo pipefail

PROJECT="${GCP_PROJECT:-project-c3d7ae5f-c73e-4fb8-828}"
NETWORK="${GCP_NETWORK:-default}"
TAG="${LIVEKIT_FIREWALL_TAG:-livekit-sip}"
RULE_NAME="${LIVEKIT_FIREWALL_RULE:-zorba-livekit-webrtc}"

account="$(gcloud config get-value account 2>/dev/null || true)"
if [[ "${account}" == *"gserviceaccount.com" ]]; then
  cat >&2 <<EOF
Refusing to run as ${account}.

This VM service account cannot manage firewall rules (insufficient scopes).

Use Cloud Shell instead:
  1. Open https://console.cloud.google.com/home/dashboard?project=${PROJECT}
  2. Click the Cloud Shell icon (>_)
  3. Paste:

gcloud compute firewall-rules create ${RULE_NAME} \\
  --project=${PROJECT} \\
  --network=${NETWORK} \\
  --direction=INGRESS \\
  --priority=1000 \\
  --action=ALLOW \\
  --rules=tcp:7880,tcp:7881,udp:50000-50100 \\
  --source-ranges=0.0.0.0/0 \\
  --target-tags=${TAG} \\
  --description='Zorba Health LiveKit signaling + WebRTC'

Or create the same rule in the Console:
  VPC network → Firewall → Create firewall rule
  Targets: Specified target tags → ${TAG}
  Source IPv4 ranges: 0.0.0.0/0
  Protocols/ports: tcp:7880,7881 and udp:50000-50100
EOF
  exit 1
fi

echo "Creating/updating firewall rule ${RULE_NAME} (project=${PROJECT}, tag=${TAG})..."

if gcloud compute firewall-rules describe "${RULE_NAME}" --project="${PROJECT}" >/dev/null 2>&1; then
  gcloud compute firewall-rules update "${RULE_NAME}" \
    --project="${PROJECT}" \
    --allow=tcp:7880,tcp:7881,udp:50000-50100 \
    --source-ranges=0.0.0.0/0
else
  gcloud compute firewall-rules create "${RULE_NAME}" \
    --project="${PROJECT}" \
    --network="${NETWORK}" \
    --direction=INGRESS \
    --priority=1000 \
    --action=ALLOW \
    --rules=tcp:7880,tcp:7881,udp:50000-50100 \
    --source-ranges=0.0.0.0/0 \
    --target-tags="${TAG}" \
    --description="Zorba Health LiveKit signaling (7880) + WebRTC ICE (7881, UDP 50000-50100)"
fi

echo "Done. From your laptop (not this VM), verify:"
echo "  curl -fsS -o /dev/null -w '%{http_code}\\n' http://34.30.50.212:7880/"
echo "Expected: 200. Then rejoin the in-app meeting."
