#!/usr/bin/env bash
set -euo pipefail

OUT_DIR="${1:-deploy/observability/certs/dev-mtls}"
mkdir -p "$OUT_DIR"

openssl genrsa -out "$OUT_DIR/ca.key" 2048
openssl req -x509 -new -nodes -key "$OUT_DIR/ca.key" -sha256 -days 365 \
  -out "$OUT_DIR/ca.crt" -subj "/CN=zorba-dev-ca"

for name in audit-service health-records-service mcp-server patient-service; do
  openssl genrsa -out "$OUT_DIR/${name}.key" 2048
  openssl req -new -key "$OUT_DIR/${name}.key" \
    -out "$OUT_DIR/${name}.csr" -subj "/CN=${name}"
  openssl x509 -req -in "$OUT_DIR/${name}.csr" -CA "$OUT_DIR/ca.crt" -CAkey "$OUT_DIR/ca.key" \
    -CAcreateserial -out "$OUT_DIR/${name}.crt" -days 365 -sha256
done

echo "Generated development mTLS certificates in $OUT_DIR"
