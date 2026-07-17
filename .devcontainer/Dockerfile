# Prebuilt Codespaces / VS Code Dev Containers image for Zorba Health.
# Published to GHCR by .github/workflows/devcontainer-ghcr.yml
FROM mcr.microsoft.com/devcontainers/base:ubuntu-24.04

LABEL org.opencontainers.image.source="https://github.com/KoiralaSam/ZorbaHealth"
LABEL org.opencontainers.image.description="Zorba Health Codespaces base (DinD features applied at runtime)"
LABEL org.opencontainers.image.licenses="MIT"

ARG KIND_VERSION=v0.27.0
ARG MIGRATE_VERSION=v4.18.1
ARG TARGETARCH

USER root

# golang-migrate, kind, tilt — baked so post-create stays fast
RUN set -eux; \
  arch="${TARGETARCH:-amd64}"; \
  case "$arch" in \
    amd64|arm64) ;; \
    *) echo "unsupported TARGETARCH=$arch" >&2; exit 1 ;; \
  esac; \
  curl -fsSL "https://github.com/golang-migrate/migrate/releases/download/${MIGRATE_VERSION}/migrate.linux-${arch}.tar.gz" \
    | tar -xz -C /usr/local/bin migrate; \
  chmod +x /usr/local/bin/migrate; \
  curl -fsSL -o /usr/local/bin/kind "https://kind.sigs.k8s.io/dl/${KIND_VERSION}/kind-linux-${arch}"; \
  chmod +x /usr/local/bin/kind; \
  curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash

USER vscode
