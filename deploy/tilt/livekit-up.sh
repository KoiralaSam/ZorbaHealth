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
OUTBOUND_TRUNK_NAME="${LIVEKIT_SIP_OUTBOUND_TRUNK_NAME:-voipms-outbound}"
OUTBOUND_ADDRESS="${LIVEKIT_SIP_OUTBOUND_ADDRESS:-dallas1.voip.ms}"
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
        # VoIP.ms POP servers currently resolve under 208.100.60.0/24.
        # Restricting sources stops open SIP scanners from tripping LiveKit flood protection (486 Busy).
        "allowedAddresses": ["208.100.60.0/24"],
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

ensure_outbound_trunk() {
  # CreateSIPParticipant (welfare / staff dial-out) requires an *outbound* trunk ID.
  # Voip.ms declines INVITEs (SIP 603) unless From is:
  #   <sip:{account}@{pop}.voip.ms>
  # not <sip:{DID}@{livekit-public-ip}>. See livekit FromHost + numbers=[sipUsername].
  #
  # voipms-credentials.username is often the portal/API email. SIP auth/From must be the
  # numeric account (or subaccount like 123456_trunk). Prefer sip_username when set.
  local api_user sip_user pass
  api_user="$(kubectl -n "$NS" get secret voipms-credentials -o jsonpath='{.data.username}' 2>/dev/null | base64 -d || true)"
  sip_user="$(kubectl -n "$NS" get secret voipms-credentials -o jsonpath='{.data.sip_username}' 2>/dev/null | base64 -d || true)"
  pass="$(kubectl -n "$NS" get secret voipms-credentials -o jsonpath='{.data.password}' 2>/dev/null | base64 -d || true)"
  if [[ -z "$sip_user" ]]; then
    sip_user="${LIVEKIT_SIP_AUTH_USERNAME:-}"
  fi
  if [[ -z "$sip_user" && -n "$api_user" && "$api_user" != *"@"* ]]; then
    sip_user="$api_user"
  fi
  # Voip.ms SIP accounts are numeric (or subaccount like 123456_trunk). Reject emails/placeholders.
  if [[ -n "$sip_user" ]] && ! [[ "$sip_user" =~ ^[0-9]+([_][A-Za-z0-9_-]+)?$ ]]; then
    log "Ignoring invalid Voip.ms SIP username (need numeric account/subaccount, got placeholder or email)."
    sip_user=""
  fi

  local json id needs_recreate="1"
  json="$(lk sip outbound list --json 2>/dev/null || echo '[]')"
  id="$(python3 - <<'PY' "$json" "$OUTBOUND_TRUNK_NAME" "$sip_user" "$OUTBOUND_ADDRESS"
import json, sys
raw, name, sip_user, address = sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4]
try:
    data = json.loads(raw)
except json.JSONDecodeError:
    data = []
items = data if isinstance(data, list) else data.get("items") or data.get("trunks") or []
for t in items:
    tname = t.get("name") or t.get("Name") or ""
    tid = t.get("sipTrunkId") or t.get("sip_trunk_id") or t.get("SipTrunkID") or t.get("id") or ""
    if tname != name or not tid:
        continue
    numbers = t.get("numbers") or t.get("Numbers") or []
    auth = t.get("authUsername") or t.get("auth_username") or ""
    from_host = t.get("fromHost") or t.get("from_host") or ""
    ok = (
        sip_user
        and numbers == [sip_user]
        and auth == sip_user
        and from_host == address
    )
    print(tid)
    print("ok" if ok else "stale")
    raise SystemExit(0)
print("")
print("missing")
PY
)"
  local trunk_state
  trunk_state="$(echo "$id" | sed -n '2p')"
  id="$(echo "$id" | sed -n '1p')"
  if [[ "$trunk_state" == "ok" ]]; then
    needs_recreate="0"
  fi

  if [[ -z "$pass" || -z "$sip_user" ]]; then
    if [[ -z "$pass" ]]; then
      log "voipms-credentials missing password; skip outbound trunk create/update"
    else
      log "Voip.ms SIP username missing."
      log "Set voipms-credentials.sip_username to your numeric account/subaccount (Main Menu → Account Settings)."
    fi
    if [[ -n "$id" ]]; then
      log "Keeping existing outbound trunk $id (SIP From not updated)"
      echo "$id"
    else
      echo ""
    fi
    return 0
  fi

  if [[ -n "$id" && "$needs_recreate" == "1" ]]; then
    log "Outbound trunk $id has stale From/auth (need $sip_user@$OUTBOUND_ADDRESS); recreating ..."
    lk sip outbound delete "$id" --yes >/dev/null 2>&1 || lk sip outbound delete --id "$id" --yes >/dev/null 2>&1 || true
    id=""
  fi

  if [[ -n "$id" && "$needs_recreate" == "0" ]]; then
    log "Outbound trunk already correct: $id (From=$sip_user@$OUTBOUND_ADDRESS)"
    echo "$id"
    return 0
  fi

  log "Creating outbound trunk $OUTBOUND_TRUNK_NAME → $OUTBOUND_ADDRESS (From=$sip_user@$OUTBOUND_ADDRESS) ..."
  local out
  # Create via JSON so fromHost is set at create time. Flag-based create cannot set
  # fromHost, and JSON update has been unreliable with this lk CLI version.
  out="$(
    python3 - <<'PY' "$OUTBOUND_TRUNK_NAME" "$OUTBOUND_ADDRESS" "$sip_user" "$pass" | lk sip outbound create - --yes 2>&1
import json, sys
name, address, sip_user, password = sys.argv[1:5]
print(json.dumps({
    "trunk": {
        "name": name,
        "address": address,
        "numbers": [sip_user],
        "authUsername": sip_user,
        "authPassword": password,
        "fromHost": address,
        "destinationCountry": "US",
    }
}))
PY
  )" || true
  id="$(echo "$out" | python3 -c 'import sys,re; m=re.search(r"ST_\w+", sys.stdin.read()); print(m.group(0) if m else "")')"
  if [[ -z "$id" ]]; then
    # Fallback to flags (fromHost may be empty → Voip.ms 603).
    out="$(
      lk sip outbound create \
        --name "$OUTBOUND_TRUNK_NAME" \
        --address "$OUTBOUND_ADDRESS" \
        --numbers "$sip_user" \
        --auth-user "$sip_user" \
        --auth-pass "$pass" \
        --destination-country US \
        --yes 2>&1
    )" || true
    id="$(echo "$out" | python3 -c 'import sys,re; m=re.search(r"ST_\w+", sys.stdin.read()); print(m.group(0) if m else "")')"
  fi
  if [[ -z "$id" ]]; then
    echo "[livekit-up] failed to create outbound trunk: $out" >&2
    exit 1
  fi

  # Best-effort patch if create omitted fromHost (flag fallback path).
  local from_host_now
  from_host_now="$(
    lk sip outbound list --json 2>/dev/null | python3 -c '
import json, sys
want = sys.argv[1]
try:
    data = json.load(sys.stdin)
except json.JSONDecodeError:
    data = []
items = data if isinstance(data, list) else data.get("items") or data.get("trunks") or []
for t in items:
    tid = t.get("sipTrunkId") or t.get("sip_trunk_id") or ""
    if tid == want:
        print(t.get("fromHost") or t.get("from_host") or "")
        break
' "$id"
  )"
  if [[ -z "$from_host_now" ]]; then
    log "Patching fromHost=$OUTBOUND_ADDRESS on trunk $id ..."
    upd_out="$(
      python3 - <<'PY' "$id" "$OUTBOUND_ADDRESS" | lk sip outbound update - --yes 2>&1
import json, sys
trunk_id, address = sys.argv[1], sys.argv[2]
print(json.dumps({
    "sipTrunkId": trunk_id,
    "update": {"fromHost": address},
}))
PY
    )" || true
    if ! echo "$upd_out" | grep -q "ST_"; then
      log "WARN: could not set fromHost on $id (Voip.ms may 603). update output: $upd_out"
    fi
  fi

  log "Created outbound trunk: $id"
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
outbound_id="$(ensure_outbound_trunk)"
# patient-service dial-out reads livekit-credentials.sipTrunkId — must be outbound.
if [[ -n "$outbound_id" ]]; then
  sync_secret_trunk_id "$outbound_id"
else
  log "No outbound trunk; leaving sipTrunkId unchanged (inbound-only)"
fi

public_ip="$(curl -4 -fsS --max-time 3 ifconfig.me 2>/dev/null || true)"
log "Ready. inbound=$trunk_id outbound=${outbound_id:-none} dispatch=$dispatch_id agent=$AGENT_NAME"
log "Reminder: open GCP firewall for LiveKit WebRTC (Cloud Shell): bash deploy/tilt/open-livekit-firewall.sh"
if [[ -n "$public_ip" ]]; then
  log "Reminder: VoIP.ms DID ($DID) voice routing MUST be SIP URI → ${DID}@${public_ip}:5060"
  log "  (Manage DIDs → Routing → SIP URI / IP. If the VM public IP changed, update this or inbound calls never reach LiveKit.)"
else
  log "Reminder: VoIP.ms DID voice → ${DID}@<public-ip>:5060 ; SIP ports udp/tcp 5060 + udp 10000-10100"
fi
