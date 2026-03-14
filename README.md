# Zorba Health

AI-powered voice health assistant — backend microservices (Go), web app (Next.js), and voice/SIP integration.

## Project overview

Zorba Health is a health-assistant platform with patient registration, email verification, and voice/call flows. The repo contains:

- **Go microservices:** API gateway, auth, patient, notification (email + planned SMS OTP)
- **Web app:** Next.js frontend (registration, login, verify-email)
- **Infrastructure:** Kubernetes manifests for local (Tilt) and production, PostgreSQL, Redis, RabbitMQ

Voice calls use **FreePBX**, **LiveKit**, and **LiveKit SIP** on a **separate server** (see below).

---

## Project architecture and flow

![Zorba Health architecture and flow](src/docs/mermaid_project_plan.svg)

Detailed architecture and flows for Zorba Health are in the repo:

- **RabbitMQ event flow** — exchanges, queues, and service interactions: `src/docs/architecture/rabbitmq-flow-v1.md`
- **Voice call flow** — end-to-end call with LiveKit, agent worker, patient/RAG/medical records, and events: `src/docs/architecture/voice-call-flow-v1.md`

---

## Prerequisites

- **Docker** (Desktop or Engine)
- **Go** (1.21+)
- **Tilt** — [install](https://docs.tilt.dev/install.html)
- **Local Kubernetes** — Docker Desktop Kubernetes, Minikube, or similar
- **kubectl** — [install](https://kubernetes.io/docs/tasks/tools/)

### macOS

```bash
brew install go
# Install Docker Desktop, Tilt, and kubectl per the links above.
```

### Windows (WSL)

Install WSL, Docker Desktop for Windows, then inside WSL install Go, Tilt, and kubectl. Use the same versions as above.

---

## Project setup

### 1. Clone and enter the repo

```bash
git clone https://github.com/KoiralaSam/ZorbaHealth.git
cd ZorbaHealth
```

### 2. Kubernetes secrets (required for Tilt)

Secrets are not committed. Create them from the example file:

```bash
cp src/infra/development/k8s/secrets.example.yaml src/infra/development/k8s/secrets.yaml
```

Edit `src/infra/development/k8s/secrets.yaml` and replace every placeholder:

- **postgres-secret:** `POSTGRES_PASSWORD` (use the same value inside `app-secrets.DATABASE_URL`)
- **sendgrid-credentials:** `apiKey` (use a verified SendGrid API key)
- **voipms-credentials:** `username`, `password` (VoIP.ms API login; used for OTP SMS when implemented)
- **app-secrets:** `DATABASE_URL` (full Postgres URL), `AUTH_SERVICE_JWT_SECRET`, `PATIENT_SERVICE_JWT_SECRET`

Set `SENDGRID_FROM_EMAIL` and `SENDGRID_FROM_NAME` in `app-config.yaml` (or override in ConfigMap) to your verified sender address and name.

See `src/infra/development/k8s/secrets.example.yaml` for the exact keys.

### 3. Start the cluster and run the app

From the **repository root**:

```bash
cd src
tilt up
```

Tilt will:

- Apply Kubernetes config (ConfigMap, Secrets, PostgreSQL, Redis, RabbitMQ)
- Build and deploy the API gateway, auth-service, patient-service, notification-service, and web app
- Open a UI in the browser (default: http://localhost:10350)

### 4. Access the app

- **Web app:** http://localhost:3000 (port-forward from the `web` deployment)
- **API gateway:** http://localhost:8081

Use the Tilt UI to see logs, port-forwards, and resource status. To stop: `Ctrl+C` in the terminal where `tilt up` is running, or run `tilt down` in `src/`.

### 5. Useful commands

```bash
# From repo root
cd src

# List running pods
kubectl get pods

# Optional: Kubernetes dashboard (if using Minikube)
minikube dashboard
```

### Troubleshooting

- **`couldn't find key fromEmail in Secret sendgrid-credentials`** — Fixed: the notification-service now reads `SENDGRID_FROM_EMAIL` and `SENDGRID_FROM_NAME` from the ConfigMap (`app-config`). Set them in `src/infra/development/k8s/app-config.yaml` to your verified sender (e.g. `your-verified-sender@example.com` and `ZorbaHealth`).
- **`secret "app-secrets" not found`** — Your `secrets.yaml` is missing the `app-secrets` (and possibly `postgres-secret`) block. Copy the full `secrets.example.yaml` to `secrets.yaml`, then replace every placeholder. If you already have a `secrets.yaml`, add the `app-secrets` and `postgres-secret` sections from `secrets.example.yaml`. Re-run `tilt up` or run `kubectl apply -f src/infra/development/k8s/secrets.yaml`.

---

## FreePBX, LiveKit, and LiveKit SIP (separate server)

Voice and SIP are **not** run inside this repo’s Tilt/Kubernetes stack. They run on a **separate server** that you configure and operate yourself.

### Role of each component

- **FreePBX** — PBX for phone numbers, trunks, and call routing.
- **LiveKit** — Real-time media (voice/video) and room-based sessions.
- **LiveKit SIP** — Bridges SIP (FreePBX) to LiveKit so incoming/outgoing calls become LiveKit rooms your app can join.

### Typical setup on the separate server

1. **Install and configure FreePBX**
   - Set up trunks, DID(s), and dialplan so calls can be sent to LiveKit SIP (e.g. via a SIP trunk or endpoint).

2. **Install and run LiveKit**
   - Run the [LiveKit server](https://docs.livekit.io/deploy/) (single binary or Docker) and expose the required ports.

3. **Install and run LiveKit SIP**
   - Follow [LiveKit SIP](https://docs.livekit.io/realtime-voice/sip/) to run the SIP dispatcher/connector that registers or receives SIP from FreePBX and creates LiveKit rooms.

4. **Point this app at LiveKit**
   - Configure the Zorba Health backend (e.g. API gateway or the service that handles calls) with the LiveKit API URL and credentials so it can create/join rooms and handle call logic.

Exact steps depend on your host (VM, cloud, or on-prem). Use the official docs for [LiveKit](https://docs.livekit.io/) and [LiveKit SIP](https://docs.livekit.io/realtime-voice/sip/) and your FreePBX distribution.

---

## Running with Tilt and Kubernetes (summary)

| Step | Action                                                                                                             |
| ---- | ------------------------------------------------------------------------------------------------------------------ |
| 1    | Clone repo, `cd ZorbaHealth`                                                                                       |
| 2    | `cp src/infra/development/k8s/secrets.example.yaml src/infra/development/k8s/secrets.yaml` and fill in real values |
| 3    | `cd src && tilt up`                                                                                                |
| 4    | Open web app at http://localhost:3000, API at http://localhost:8081                                                |

Tilt uses the manifests under `src/infra/development/k8s/` (ConfigMap, Secrets, PostgreSQL, Redis, RabbitMQ, and each service deployment). Ensure your Kubernetes context points to your local cluster (e.g. Docker Desktop or Minikube) before running `tilt up`.

---

## Deployment (production)

For production you would:

1. Build and push Docker images for each service (and the web app) to your registry.
2. Apply the production Kubernetes manifests (or use Helm/another orchestrator) in order: config and secrets first, then infrastructure (Postgres, Redis, RabbitMQ), then services.
3. Keep FreePBX, LiveKit, and LiveKit SIP on their separate server and configure the production backend with the correct LiveKit URL and credentials.

The production manifests in this repo are under `src/infra/production/`; adapt them to your environment and CI/CD.

---

## Adding HTTPS to your API (GCP example)

See the original deployment section in git history or your runbooks for reserving a static IP, adding an Ingress, switching the API gateway to ClusterIP, and (optionally) using a Google-managed certificate. The same pattern applies to other clouds with an Ingress controller and TLS.
