#!/usr/bin/env bash
# Start host Docker LiveKit + SIP and ensure inbound trunk + agent dispatch rule exist.
# Invoked by Tilt local_resource "livekit-docker" (auto_init).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPOSE_FILE="$ROOT/deploy/docker/livekit/docker-compose.yml"
LIVEKIT_YAML="$ROOT/deploy/docker/livekit/livekit.yaml"
NS="${K8S_NAMESPACE:-dev}"
export PATH="${HOME}/.local/bin:${PATH}"

TRUNK_NAME="${LIVEKIT_SIP_TRUNK_NAME:-voipms-inbound}"
RULE_NAME="${LIVEKIT_SIP_DISPATCH_RULE_NAME:-zorba-agent-individual}"
ROOM_PREFIX="${LIVEKIT_SIP_ROOM_PREFIX:-zorba-call-}"
AGENT_NAME="${LIVEKIT_AGENT_NAME:-zorba-health-voice}"
DID="${LIVEKIT_SIP_DID:-3185162690}"

log() { echo "[livekit-up] $*" >&2; }

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "[livekit-up] required command not found: $1" >&2
    exit 1
  }
}

need_cmd docker
need_cmd curl
need_cmd python3

# Prefer rootless docker; fall back to sudo when the socket is root-owned
# (common until the user re-logs after usermod -aG docker).
DOCKER=(docker)
if ! docker info >/dev/null 2>&1; then
  if sudo -n docker info >/dev/null 2>&1; then
    DOCKER=(sudo docker)
    log "Using sudo docker (user not in docker group yet — re-login after usermod -aG docker)"
  else
    echo "[livekit-up] Docker daemon is not reachable (is your user in the docker group?)" >&2
    exit 1
  fi
fi

compose() {
  "${DOCKER[@]}" compose -f "$COMPOSE_FILE" "$@"
}
ensure_lk() {
  if command -v lk >/dev/null 2>&1; then
    return 0
  fi
  log "Installing LiveKit CLI (lk) into ~/.local/bin ..."
  mkdir -p "${HOME}/.local/bin"
  local ver url
  ver="$(curl -fsSL https://api.github.com/repos/livekit/livekit-cli/releases/latest | python3 -c 'import sys,json; print(json.load(sys.stdin)["tag_name"].lstrip("v"))')"
  url="https://github.com/livekit/livekit-cli/releases/download/v${ver}/lk_${ver}_linux_amd64.tar.gz"
  curl -fsSL "$url" -o /tmp/lk.tar.gz
  tar -xzf /tmp/lk.tar.gz -C "${HOME}/.local/bin" lk
  chmod +x "${HOME}/.local/bin/lk"
  need_cmd lk
}

# Load API key/secret from livekit.yaml (keys: KEY: SECRET) so they stay in sync with the server.
load_credentials() {
  eval "$(python3 - <<'PY' "$LIVEKIT_YAML"
import re, sys
path = sys.argv[1]
text = open(path, encoding="utf-8").read()
# Find first entry under a top-level "keys:" mapping
m = re.search(r"(?m)^keys:\s*\n(?:[ \t]+\#.*\n)*[ \t]+([A-Za-z0-9]+):\s*(\S+)\s*$", text)
if not m:
    sys.stderr.write(f"[livekit-up] could not parse keys from {path}\n")
    sys.exit(1)
print(f"export LIVEKIT_API_KEY={m.group(1)!r}")
print(f"export LIVEKIT_API_SECRET={m.group(2)!r}")
PY
)"
  export LIVEKIT_URL="${LIVEKIT_URL:-http://127.0.0.1:7880}"
}

wait_livekit() {
  log "Waiting for LiveKit HTTP on 127.0.0.1:7880 ..."
  local i
  for i in $(seq 1 90); do
    if curl -fsS -o /dev/null "http://127.0.0.1:7880/" 2>/dev/null; then
      log "LiveKit server is up"
      return 0
    fi
    sleep 1
  done
  echo "[livekit-up] LiveKit did not become ready on :7880" >&2
  compose ps >&2 || true
  compose logs --tail=40 livekit-server livekit-sip >&2 || true
  exit 1
}

wait_sip() {
  log "Waiting for LiveKit SIP on 127.0.0.1:5060/udp ..."
  local i
  for i in $(seq 1 60); do
    if python3 - <<'PY'
import socket
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
s.settimeout(1)
msg = (
    b"OPTIONS sip:ping@127.0.0.1 SIP/2.0\r\n"
    b"Via: SIP/2.0/UDP 127.0.0.1:5099;branch=z9hG4bKtilt\r\n"
    b"From: <sip:tilt@127.0.0.1>;tag=tilt\r\n"
    b"To: <sip:ping@127.0.0.1>\r\n"
    b"Call-ID: tilt-livekit-up\r\n"
    b"CSeq: 1 OPTIONS\r\n"
    b"Content-Length: 0\r\n\r\n"
)
try:
    s.sendto(msg, ("127.0.0.1", 5060))
    data, _ = s.recvfrom(2048)
    raise SystemExit(0 if data.startswith(b"SIP/2.0") else 1)
except Exception:
    raise SystemExit(1)
PY
    then
      log "LiveKit SIP is up"
      return 0
    fi
    sleep 1
  done
  echo "[livekit-up] LiveKit SIP did not become ready on :5060" >&2
  compose logs --tail=40 livekit-sip >&2 || true
  exit 1
}

ensure_trunk() {
  local json id
  json="$(lk sip inbound list --json 2>/dev/null || echo '[]')"
  id="$(python3 - <<'PY' "$json" "$TRUNK_NAME" "$DID"
import json, sys
raw, name, did = sys.argv[1], sys.argv[2], sys.argv[3]
try:
    data = json.loads(raw)
except json.JSONDecodeError:
    data = []
items = data if isinstance(data, list) else data.get("items") or data.get("trunks") or []
for t in items:
    # CLI JSON shapes vary; accept common fields
    tname = t.get("name") or t.get("Name") or ""
    tid = t.get("sipTrunkId") or t.get("sip_trunk_id") or t.get("SipTrunkID") or t.get("id") or ""
    numbers = t.get("numbers") or t.get("Numbers") or []
    if tname == name or did in numbers or f"+1{did}" in numbers or f"1{did}" in numbers:
        print(tid)
        raise SystemExit(0)
print("")
PY
)"
  if [[ -n "$id" ]]; then
    log "Inbound trunk already present: $id"
    echo "$id"
    return 0
  fi
  log "Creating inbound trunk $TRUNK_NAME for DID $DID ..."
  local out
  out="$(
    python3 - <<'PY' "$TRUNK_NAME" "$DID" | lk sip inbound create - --yes
import json, sys
name, did = sys.argv[1], sys.argv[2]
print(json.dumps({
    "trunk": {
        "name": name,
        "numbers": [did, f"1{did}", f"+1{did}"],
    }
}))
PY
  )"
  id="$(echo "$out" | python3 -c 'import sys,re; m=re.search(r"ST_\w+", sys.stdin.read()); print(m.group(0) if m else "")')"
  if [[ -z "$id" ]]; then
    echo "[livekit-up] failed to create inbound trunk: $out" >&2
    exit 1
  fi
  log "Created inbound trunk: $id"
  echo "$id"
}

ensure_dispatch() {
  local trunk_id="$1"
  local json id
  json="$(lk sip dispatch list --json 2>/dev/null || echo '[]')"
  id="$(python3 - <<'PY' "$json" "$RULE_NAME" "$trunk_id" "$AGENT_NAME"
import json, sys
raw, name, trunk_id, agent = sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4]
try:
    data = json.loads(raw)
except json.JSONDecodeError:
    data = []
items = data if isinstance(data, list) else data.get("items") or data.get("rules") or []
for r in items:
    rname = r.get("name") or r.get("Name") or ""
    rid = r.get("sipDispatchRuleId") or r.get("sip_dispatch_rule_id") or r.get("SipDispatchRuleID") or r.get("id") or ""
    agents = r.get("agents") or r.get("Agents") or []
    # agents may be list of names or objects
    agent_names = []
    for a in agents:
        if isinstance(a, str):
            agent_names.append(a)
        elif isinstance(a, dict):
            agent_names.append(a.get("agentName") or a.get("agent_name") or "")
    if rname == name and (not agent_names or agent in agent_names):
        print(rid)
        raise SystemExit(0)
print("")
PY
)"
  if [[ -n "$id" ]]; then
    log "Dispatch rule already present: $id"
    echo "$id"
    return 0
  fi
  log "Creating dispatch rule $RULE_NAME (agent=$AGENT_NAME) ..."
  local out
  out="$(
    python3 - <<'PY' "$RULE_NAME" "$trunk_id" "$ROOM_PREFIX" "$AGENT_NAME" | lk sip dispatch create - --yes
import json, sys
name, trunk_id, prefix, agent = sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4]
print(json.dumps({
    "dispatch_rule": {
        "name": name,
        "trunk_ids": [trunk_id],
        "rule": {"dispatchRuleIndividual": {"roomPrefix": prefix}},
        "room_config": {"agents": [{"agent_name": agent}]},
        "metadata": '{"language":"en"}',
    }
}))
PY
  )"
  id="$(echo "$out" | python3 -c 'import sys,re; m=re.search(r"SDR_\w+", sys.stdin.read()); print(m.group(0) if m else "")')"
  if [[ -z "$id" ]]; then
    echo "[livekit-up] failed to create dispatch rule: $out" >&2
    exit 1
  fi
  log "Created dispatch rule: $id"
  echo "$id"
}

sync_secret_trunk_id() {
  local trunk_id="$1"
  local secrets_file="$ROOT/deploy/kubernetes/development/secrets.yaml"
  if [[ -f "$secrets_file" ]]; then
    python3 - <<'PY' "$secrets_file" "$trunk_id"
import pathlib, re, sys
path, trunk_id = pathlib.Path(sys.argv[1]), sys.argv[2]
text = path.read_text(encoding="utf-8")
# Only touch the livekit-credentials sipTrunkId field.
pat = re.compile(
    r"(name:\s*livekit-credentials[\s\S]*?sipTrunkId:\s*)(\"[^\"]*\"|[^\n]*)",
    re.M,
)
new, n = pat.subn(rf'\1"{trunk_id}"', text, count=1)
if n:
    path.write_text(new, encoding="utf-8")
    print(f"[livekit-up] Updated {path} sipTrunkId={trunk_id}")
else:
    print(f"[livekit-up] Could not find sipTrunkId under livekit-credentials in {path}", file=sys.stderr)
PY
  fi

  if command -v kubectl >/dev/null 2>&1 && kubectl get secret livekit-credentials -n "$NS" >/dev/null 2>&1; then
    kubectl -n "$NS" patch secret livekit-credentials --type merge \
      -p "{\"stringData\":{\"sipTrunkId\":\"${trunk_id}\"}}" >/dev/null
    log "Patched k8s secret livekit-credentials.sipTrunkId=$trunk_id"
  else
    log "k8s secret livekit-credentials not ready yet; file update only (Tilt will apply secrets.yaml)"
  fi
}

log "Starting LiveKit compose stack ..."
compose up -d

ensure_lk
load_credentials
wait_livekit
wait_sip

trunk_id="$(ensure_trunk)"
dispatch_id="$(ensure_dispatch "$trunk_id")"
sync_secret_trunk_id "$trunk_id"

log "Ready. trunk=$trunk_id dispatch=$dispatch_id agent=$AGENT_NAME"
log "Reminder: VoIP.ms DID voice → ${DID}@<public-ip>:5060 ; open GCP firewall udp/tcp 5060 + udp 10000-10100"
