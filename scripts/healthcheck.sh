#!/usr/bin/env bash
# Local healthcheck for systemd or monitoring (run on the VPS).
set -euo pipefail

ADDR="${LISTEN_ADDR:-127.0.0.1:8080}"
URL="http://${ADDR}/ping"

code=$(curl -sS -o /dev/null -w "%{http_code}" -H "Origin: https://streethosting.com.br" "${URL}")
[[ "${code}" == "204" ]]
