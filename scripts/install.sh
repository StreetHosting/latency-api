#!/usr/bin/env bash
# First-time install on a single Debian 13 VPS (bare metal).
# Run as root from the repo root, or use Ansible for fleet deploy.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

PROBE_HOSTNAME="${PROBE_HOSTNAME:?Set PROBE_HOSTNAME (e.g. latency-sp-games-1.streethosting.com.br)}"
CERTBOT_EMAIL="${CERTBOT_EMAIL:-noreply@streethosting.com.br}"
BINARY_SRC="${BINARY_SRC:-${ROOT}/dist/latency-probe-linux-amd64}"
ALLOWED_ORIGINS="${ALLOWED_ORIGINS:-http://localhost:3000}"
ALLOWED_ORIGIN_SUFFIXES="${ALLOWED_ORIGIN_SUFFIXES:-streethosting.com.br,strt.host,ruas.run}"

if [[ "$(id -u)" -ne 0 ]]; then
  echo "Run as root." >&2
  exit 1
fi

export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y nginx certbot python3-certbot-nginx gettext-base mtr-tiny libcap2-bin

if [[ -x /usr/bin/mtr ]]; then
  setcap cap_net_raw+ep /usr/bin/mtr 2>/dev/null || true
fi
install -m 0440 deploy/sudoers/latency-probe-mtr /etc/sudoers.d/latency-probe-mtr
visudo -c -f /etc/sudoers.d/latency-probe-mtr

id -u latency-probe &>/dev/null || useradd --system --no-create-home --shell /usr/sbin/nologin latency-probe
install -d -m 0750 /etc/latency-probe
install -d -m 0755 /var/lib/latency-probe

cat >/etc/latency-probe/probe.env <<EOF
LISTEN_ADDR=127.0.0.1:8080
ALLOWED_ORIGINS=${ALLOWED_ORIGINS}
ALLOWED_ORIGIN_SUFFIXES=${ALLOWED_ORIGIN_SUFFIXES}
MTR_ENABLED=true
MTR_BIN=/usr/bin/mtr
MTR_CYCLES=10
MTR_TIMEOUT=45s
MTR_MIN_INTERVAL=60s
MTR_USE_SUDO=true
EOF
chmod 0640 /etc/latency-probe/probe.env

install -m 0755 "${BINARY_SRC}" /usr/local/bin/latency-probe
install -m 0644 deploy/systemd/latency-probe.service /etc/systemd/system/latency-probe.service
systemctl daemon-reload
systemctl enable --now latency-probe

mkdir -p /var/www/certbot

# HTTP bootstrap for ACME + probe until TLS is ready
cat >/etc/nginx/sites-available/"${PROBE_HOSTNAME}".conf <<NGINX
server {
    listen 80;
    listen [::]:80;
    server_name ${PROBE_HOSTNAME};
    location /.well-known/acme-challenge/ { root /var/www/certbot; }
    location = /ping { proxy_pass http://127.0.0.1:8080; }
    location = /mtr  { proxy_pass http://127.0.0.1:8080; proxy_read_timeout 60s; }
    location / { return 404; }
}
NGINX
ln -sf /etc/nginx/sites-available/"${PROBE_HOSTNAME}".conf \
  /etc/nginx/sites-enabled/"${PROBE_HOSTNAME}".conf
rm -f /etc/nginx/sites-enabled/default
nginx -t && systemctl reload nginx

if [[ ! -f "/etc/letsencrypt/live/${PROBE_HOSTNAME}/fullchain.pem" ]]; then
  certbot certonly --webroot -w /var/www/certbot --non-interactive --agree-tos \
    --email "${CERTBOT_EMAIL}" -d "${PROBE_HOSTNAME}"
fi

export PROBE_HOSTNAME
envsubst '${PROBE_HOSTNAME}' <configs/nginx/latency-probe.conf.template \
  >/etc/nginx/sites-available/"${PROBE_HOSTNAME}".conf
nginx -t && systemctl reload nginx
echo "Installed probe for ${PROBE_HOSTNAME}"
