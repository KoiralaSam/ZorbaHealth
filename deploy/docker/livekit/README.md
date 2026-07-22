# LiveKit + LiveKit SIP (host Docker sidecar)

Runs **outside** Kubernetes next to the Tilt/minikube app stack. FreePBX is not required; VoIP.ms DIDs route straight to LiveKit SIP.

## First-time setup: credentials

`livekit.yaml` and `sip.yaml` hold real API keys and are **gitignored**. Create them from the templates and fill in the same key/secret pair in both:

```bash
cd deploy/docker/livekit
cp livekit.example.yaml livekit.yaml
cp sip.example.yaml sip.yaml
# generate a key pair:
docker run --rm livekit/livekit-server generate-keys
```

## Tilt (recommended)

`tilt up` auto-runs [`deploy/tilt/livekit-up.sh`](../../tilt/livekit-up.sh) as the `livekit-docker` resource. That script:

1. `docker compose up -d` for this stack
2. Waits until LiveKit (`:7880`) and SIP (`:5060`) respond
3. Installs the `lk` CLI into `~/.local/bin` if missing
4. Ensures an inbound SIP trunk for DID `3185162690` (override with `LIVEKIT_SIP_DID`)
5. Ensures a dispatch rule that wakes agent `zorba-health-voice`
6. Writes `sipTrunkId` into `deploy/kubernetes/development/secrets.yaml` and patches the live k8s secret when present

`voice-agent-service` and `patient-service` wait on `livekit-docker`.

## Manual start / stop

From the repository root:

```bash
./deploy/tilt/livekit-up.sh
docker compose -f deploy/docker/livekit/docker-compose.yml ps
docker compose -f deploy/docker/livekit/docker-compose.yml logs -f
docker compose -f deploy/docker/livekit/docker-compose.yml down
```

## Ports published on the host

| Port / range | Protocol | Purpose |
|---|---|---|
| `7880` | TCP | LiveKit HTTP / WebSocket API |
| `7881` | TCP | WebRTC ICE over TCP fallback |
| `50000–50100` | UDP | WebRTC media (RTC) |
| `5060` | UDP + TCP | SIP signaling |
| `10000–10100` | UDP | SIP RTP |

## Firewall / VoIP.ms (still manual)

Tilt cannot open your cloud firewall or change VoIP.ms. You still need:

1. **GCP firewall** (this is a GCP VM): allow ingress `udp/tcp:5060` and `udp:10000-10100` (and for WebRTC: `udp:50000-50100`, `tcp:7881`)
2. **VoIP.ms** DID voice routing to SIP URI: `3185162690@<public-ip>:5060`

SMS routing (notification-service) is separate and unchanged.

**Host networking:** `livekit-server` and `livekit-sip` use `network_mode: host` so ICE advertises the VM’s private + public IPs. Bridge networking alone advertises only the GCP public IP, which minikube pods cannot hairpin to (agent joins then `wait_pc_connection timed out`).

## How the k8s stack reaches LiveKit

Dev ConfigMap ([`app-config.yaml`](../../kubernetes/development/app-config.yaml)):

- Pods: `LIVEKIT_URL` / `LIVEKIT_WS_URL` = `ws://host.minikube.internal:7880`
- Browsers / join links: `LIVEKIT_PUBLIC_WS_URL` = host public or LAN IP on `:7880` (update if your IP changes)

API key/secret in `livekit.yaml` / `sip.yaml` must match `livekit-credentials` in Kubernetes secrets.

## Softphone / local test (no PSTN)

With the stack up (via Tilt or manual), call `sip:3185162690@127.0.0.1:5060` from a softphone on the host, or join a room over `ws://localhost:7880`.
