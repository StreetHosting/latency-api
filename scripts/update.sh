#!/usr/bin/env bash
# Rolling update on one VPS: replace binary and restart (zero-downtime via systemd).
set -euo pipefail

BINARY_SRC="${BINARY_SRC:-./dist/latency-probe-linux-amd64}"

if [[ "$(id -u)" -ne 0 ]]; then
  echo "Run as root." >&2
  exit 1
fi

if [[ ! -f "${BINARY_SRC}" ]]; then
  echo "Missing binary: ${BINARY_SRC}" >&2
  exit 1
fi

install -m 0755 "${BINARY_SRC}" /usr/local/bin/latency-probe
systemctl restart latency-probe
systemctl is-active --quiet latency-probe
echo "latency-probe updated and running."
