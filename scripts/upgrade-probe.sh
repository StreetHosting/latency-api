#!/usr/bin/env bash
# Atualiza binário, MTR, probe.env e nginx num nó já instalado.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

if [[ "$(id -u)" -ne 0 ]]; then
  echo "Execute como root." >&2
  exit 1
fi

detect_hostname() {
  if [[ -n "${PROBE_HOSTNAME:-}" ]]; then
    echo "${PROBE_HOSTNAME}"
    return
  fi
  local conf name
  for conf in /etc/nginx/sites-enabled/*.conf; do
    [[ -f "${conf}" ]] || continue
    name="$(grep -m1 '^\s*server_name\s' "${conf}" | sed -E 's/^\s*server_name\s+([^;]+);.*/\1/' | awk '{print $1}')"
    if [[ -n "${name}" && "${name}" != "_" ]]; then
      echo "${name}"
      return
    fi
  done
  return 1
}

export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y mtr-tiny libcap2-bin gettext-base sudo

short="$(hostname -s 2>/dev/null || true)"
if [[ -n "${short}" ]] && ! grep -qE "[[:space:]]${short}([[:space:]]|$)" /etc/hosts; then
  echo "127.0.1.1 ${short}" >>/etc/hosts
fi

if [[ -x /usr/bin/mtr ]]; then
  setcap cap_net_raw+ep /usr/bin/mtr 2>/dev/null || true
fi
install -m 0440 "${ROOT}/deploy/sudoers/latency-probe-mtr" /etc/sudoers.d/latency-probe-mtr
visudo -c -f /etc/sudoers.d/latency-probe-mtr

PROBE_HOSTNAME="$(detect_hostname)" || {
  echo "Defina PROBE_HOSTNAME ou configure nginx sites-enabled." >&2
  exit 1
}
echo "[upgrade] hostname: ${PROBE_HOSTNAME}"

install -d -m 0750 /etc/latency-probe
ENV_FILE=/etc/latency-probe/probe.env
if [[ ! -f "${ENV_FILE}" ]]; then
  cat >"${ENV_FILE}" <<EOF
LISTEN_ADDR=127.0.0.1:8080
ALLOWED_ORIGINS=http://localhost:3000
ALLOWED_ORIGIN_SUFFIXES=streethosting.com.br,strt.host,ruas.run
EOF
fi
grep -q '^MTR_ENABLED=' "${ENV_FILE}" 2>/dev/null || echo 'MTR_ENABLED=true' >>"${ENV_FILE}"
grep -q '^MTR_BIN=' "${ENV_FILE}" 2>/dev/null || echo 'MTR_BIN=/usr/bin/mtr' >>"${ENV_FILE}"
grep -q '^MTR_CYCLES=' "${ENV_FILE}" 2>/dev/null || echo 'MTR_CYCLES=10' >>"${ENV_FILE}"
grep -q '^MTR_TIMEOUT=' "${ENV_FILE}" 2>/dev/null || echo 'MTR_TIMEOUT=45s' >>"${ENV_FILE}"
grep -q '^MTR_MIN_INTERVAL=' "${ENV_FILE}" 2>/dev/null || echo 'MTR_MIN_INTERVAL=60s' >>"${ENV_FILE}"
grep -q '^MTR_USE_SUDO=' "${ENV_FILE}" 2>/dev/null || echo 'MTR_USE_SUDO=true' >>"${ENV_FILE}"
chmod 0640 "${ENV_FILE}"

install -m 0644 "${ROOT}/deploy/systemd/latency-probe.service" /etc/systemd/system/latency-probe.service

BINARY_SRC="${ROOT}/dist/latency-probe-linux-amd64"
[[ -x "${BINARY_SRC}" ]] || { echo "Rode scripts/build.sh antes." >&2; exit 1; }
install -m 0755 "${BINARY_SRC}" /usr/local/bin/latency-probe

export PROBE_HOSTNAME
envsubst '${PROBE_HOSTNAME}' <configs/nginx/latency-probe.conf.template \
  >/etc/nginx/sites-available/"${PROBE_HOSTNAME}.conf"
ln -sf /etc/nginx/sites-available/"${PROBE_HOSTNAME}.conf" \
  /etc/nginx/sites-enabled/"${PROBE_HOSTNAME}.conf"

nginx -t
systemctl daemon-reload
systemctl restart latency-probe
systemctl reload nginx

echo "[upgrade] OK — teste:"
echo "  curl -sS http://127.0.0.1:8080/ping -H 'Origin: https://streethosting.com.br' -o /dev/null -w 'ping HTTP %{http_code}\n'"
echo "  curl -sS http://127.0.0.1:8080/mtr -H 'Origin: https://streethosting.com.br' -H 'X-Real-IP: 1.1.1.1' | head"
echo "  curl -sS https://${PROBE_HOSTNAME}/mtr -H 'Origin: https://streethosting.com.br' | head"
