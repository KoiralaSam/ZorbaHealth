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

1. **GCP firewall** (this is a GCP VM): allow ingress `tcp:7880` (LiveKit signaling / join WebSocket), `tcp:7881`, `udp:50000-50100` (WebRTC), plus SIP `udp/tcp:5060` and `udp:10000-10100`. Without `tcp:7880`, phones and browsers cannot reach `LIVEKIT_PUBLIC_WS_URL` and fail with `could not establish signal connection: Abort handler called`.
   - From **Cloud Shell** (the VM SA lacks compute scopes): `bash deploy/tilt/open-livekit-firewall.sh`
   - Targets existing instance tag `livekit-sip`.
2. **VoIP.ms** DID voice routing to SIP URI: `3185162690@<public-ip>:5060`
   - Current VM public IP example: `3185162690@34.30.50.212:5060`
   - If the GCP external IP changes, update the SIP URI in VoIP.ms or inbound PSTN calls never reach LiveKit (agent/trunk can be healthy while dials still fail).
   - Portal: **DID Numbers → Manage DID → Routing → SIP URI** (create/edit URI host=`<public-ip>`, port=`5060`, then point the DID at that URI).
3. **Outbound From header:** Voip.ms requires `From: <sip:{account}@{pop}.voip.ms>` (not `DID@livekit-ip`). `livekit-up.sh` configures the outbound trunk with `numbers=[sip_username]` and `fromHost=dallas1.voip.ms`. Put your **numeric SIP account/subaccount** in `voipms-credentials.sip_username` (portal email in `username` is for the REST API only). Set the account’s default Caller ID to your DID in the Voip.ms portal so callees still see `3185162690`.
4. **Outbound SIP 603 Declined after a GCP IP change:** LiveKit can still send a correct INVITE (`527931@dallas1.voip.ms` → patient number) and Voip.ms will decline it if the VM’s **current public IP** is not on the account allowlist. In the portal: **Main Menu → Account Settings → Security / Allowed IPs** — add the current external IP (check with `curl -4 ifconfig.me`). Prefer a **static/reserved** GCP external IP so this does not break again. Auth username/password alone is not enough when IP restrictions are enabled.

SMS routing (notification-service) is separate and unchanged.

**Host networking:** `livekit-server` and `livekit-sip` use `network_mode: host` so ICE advertises the VM’s private + public IPs. Bridge networking alone advertises only the GCP public IP, which minikube pods cannot hairpin to (agent joins then `wait_pc_connection timed out`).

## How the k8s stack reaches LiveKit

Dev ConfigMap ([`app-config.yaml`](../../kubernetes/development/app-config.yaml)):

- Pods: `LIVEKIT_URL` / `LIVEKIT_WS_URL` = `ws://host.minikube.internal:7880`
- Browsers / join links: `LIVEKIT_PUBLIC_WS_URL` = host public or LAN IP on `:7880` (update if your IP changes)

API key/secret in `livekit.yaml` / `sip.yaml` must match `livekit-credentials` in Kubernetes secrets.

## Softphone / local test (no PSTN)

With the stack up (via Tilt or manual), call `sip:3185162690@127.0.0.1:5060` from a softphone on the host, or join a room over `ws://localhost:7880`.
