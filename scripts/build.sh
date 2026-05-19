#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST="${ROOT}/dist"
VERSION="${VERSION:-$(git -C "${ROOT}" describe --tags --always --dirty 2>/dev/null || echo dev)}"

mkdir -p "${DIST}"

echo "Building latency-probe ${VERSION} for linux/amd64..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -trimpath \
  -ldflags="-s -w" \
  -o "${DIST}/latency-probe-linux-amd64" \
  "${ROOT}/cmd/probe"

echo "Binary: ${DIST}/latency-probe-linux-amd64"
ls -lh "${DIST}/latency-probe-linux-amd64"
