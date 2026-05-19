#!/usr/bin/env bash
# Verify probe contract (CORS + 204) for one or all nodes from inventory.
set -euo pipefail

ORIGIN="${ORIGIN:-https://streethosting.com.br}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

probe_one() {
  local url="$1"
  echo "==> ${url}"

  local opts_out opts_code get_out get_code
  opts_out=$(curl -sS -o /tmp/probe-opts-body -w "%{http_code}" -X OPTIONS "${url}" \
    -H "Origin: ${ORIGIN}" \
    -H "Access-Control-Request-Method: GET")
  opts_code="${opts_out}"

  get_out=$(curl -sS -o /tmp/probe-get-body -w "%{http_code}" "${url}" \
    -H "Origin: ${ORIGIN}" \
    -H "Cache-Control: no-store")
  get_code="${get_out}"

  local cors cache
  cors=$(curl -sSI "${url}" -H "Origin: ${ORIGIN}" | tr -d '\r' | grep -i '^access-control-allow-origin:' || true)
  cache=$(curl -sSI "${url}" -H "Origin: ${ORIGIN}" | tr -d '\r' | grep -i '^cache-control:' || true)

  [[ "${opts_code}" == "204" ]] || { echo "OPTIONS failed: ${opts_code}"; return 1; }
  [[ "${get_code}" == "204" ]] || { echo "GET failed: ${get_code}"; return 1; }
  [[ -n "${cors}" ]] || { echo "Missing Access-Control-Allow-Origin"; return 1; }
  [[ "${cache}" == *"no-store"* ]] || { echo "Missing Cache-Control: no-store (${cache})"; return 1; }

  echo "OK (OPTIONS/GET 204, CORS, no-store)"
}

if [[ $# -ge 1 ]]; then
  for host in "$@"; do
    probe_one "https://${host}/ping"
  done
  exit 0
fi

# Default: all hostnames from example inventory
HOSTS=(
  latency-sp-games-1.streethosting.com.br
  latency-sp-games-2.streethosting.com.br
  latency-sp-empresa-1.streethosting.com.br
  latency-sp-empresa-2.streethosting.com.br
  latency-sp-raw-1.streethosting.com.br
  latency-sp-raw-2.streethosting.com.br
)

failed=0
for h in "${HOSTS[@]}"; do
  probe_one "https://${h}/ping" || failed=$((failed + 1))
done

exit "${failed}"
