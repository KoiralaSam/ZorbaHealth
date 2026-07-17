#!/usr/bin/env bash
# Install kind, Tilt, and golang-migrate when missing (Codespaces / local DinD).
set -euo pipefail

export PATH="${HOME}/.local/bin:/usr/local/bin:${PATH}"

KIND_VERSION="${KIND_VERSION:-v0.27.0}"
MIGRATE_VERSION="${MIGRATE_VERSION:-v4.18.1}"

arch="$(uname -m)"
case "${arch}" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) echo "unsupported arch: ${arch}" >&2; exit 1 ;;
esac

place_bin() {
  local src="$1"
  local name="$2"
  chmod +x "${src}"
  if [[ -w /usr/local/bin ]] || [[ -w /usr/local/bin/${name} ]]; then
    mv "${src}" "/usr/local/bin/${name}"
  elif command -v sudo >/dev/null; then
    sudo mv "${src}" "/usr/local/bin/${name}"
  else
    mkdir -p "${HOME}/.local/bin"
    mv "${src}" "${HOME}/.local/bin/${name}"
  fi
}

if ! command -v migrate >/dev/null; then
  echo "Installing golang-migrate ${MIGRATE_VERSION}…"
  tmp="$(mktemp -d)"
  curl -fsSL "https://github.com/golang-migrate/migrate/releases/download/${MIGRATE_VERSION}/migrate.linux-${arch}.tar.gz" \
    | tar -xz -C "${tmp}" migrate
  place_bin "${tmp}/migrate" migrate
  rm -rf "${tmp}"
fi

if ! command -v kind >/dev/null; then
  echo "Installing kind ${KIND_VERSION}…"
  tmp="$(mktemp)"
  curl -fsSL -o "${tmp}" "https://kind.sigs.k8s.io/dl/${KIND_VERSION}/kind-linux-${arch}"
  place_bin "${tmp}" kind
fi

if ! command -v tilt >/dev/null; then
  echo "Installing Tilt…"
  curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash
fi

hash -r 2>/dev/null || true
export PATH="${HOME}/.local/bin:/usr/local/bin:${PATH}"

echo "OK: migrate=$(command -v migrate) kind=$(command -v kind) tilt=$(command -v tilt)"
migrate -version
kind version
tilt version
