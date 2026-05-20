#!/usr/bin/env bash
#
# Bootstrap completo para Debian 13 minimal:
#   - instala dependências (git, make, sudo, go, nginx, certbot, ...)
#   - clona ou atualiza https://github.com/StreetHosting/latency-api
#   - compila o binário latency-probe
#   - (opcional) instala nginx + systemd se PROBE_HOSTNAME estiver definido
#
# Uso como root na VPS (primeira vez):
#
#   export PROBE_HOSTNAME=latency-sp-games-1.streethosting.com.br
#   export CERTBOT_EMAIL=noreply@streethosting.com.br
#   curl -fsSL https://raw.githubusercontent.com/StreetHosting/latency-api/main/scripts/bootstrap-debian.sh | bash
#
# Ou, com o script já no disco:
#   bash scripts/bootstrap-debian.sh
#
set -euo pipefail

REPO_URL="${REPO_URL:-https://github.com/StreetHosting/latency-api.git}"
REPO_BRANCH="${REPO_BRANCH:-main}"
INSTALL_DIR="${INSTALL_DIR:-/opt/latency-api}"
GO_VERSION="${GO_VERSION:-1.23.4}"
SKIP_INSTALL="${SKIP_INSTALL:-0}"

log() { printf '\033[1;34m[bootstrap]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[bootstrap]\033[0m %s\n' "$*"; }
die() { printf '\033[1;31m[bootstrap]\033[0m %s\n' "$*" >&2; exit 1; }

require_root() {
  if [[ "$(id -u)" -ne 0 ]]; then
    die "Execute como root: su -  ou  curl ... | sudo bash"
  fi
}

detect_debian() {
  if [[ ! -f /etc/debian_version ]]; then
    die "Este script é apenas para Debian."
  fi
  local ver
  ver="$(. /etc/os-release && echo "${VERSION_ID:-}")"
  log "Debian detectado (VERSION_ID=${ver:-desconhecido})"
}

install_base_packages() {
  log "Atualizando índice apt e instalando pacotes base..."
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -qq
  apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    git \
    make \
    sudo \
    gnupg \
    nginx \
    certbot \
    python3-certbot-nginx \
    gettext-base \
    openssl \
    mtr-tiny \
    libcap2-bin

  if [[ -x /usr/bin/mtr ]]; then
    setcap cap_net_raw+ep /usr/bin/mtr 2>/dev/null || warn "setcap em /usr/bin/mtr falhou"
  fi
}

detect_probe_hostname() {
  if [[ -n "${PROBE_HOSTNAME:-}" ]]; then
    return 0
  fi
  local conf name
  for conf in /etc/nginx/sites-enabled/*.conf; do
    [[ -f "${conf}" ]] || continue
    name="$(grep -m1 '^\s*server_name\s' "${conf}" | sed -E 's/^\s*server_name\s+([^;]+);.*/\1/' | awk '{print $1}')"
    if [[ -n "${name}" && "${name}" != "_" ]]; then
      PROBE_HOSTNAME="${name}"
      log "PROBE_HOSTNAME detectado via nginx: ${PROBE_HOSTNAME}"
      export PROBE_HOSTNAME
      return 0
    fi
  done
  return 1
}

go_version_ok() {
  command -v go >/dev/null 2>&1 || return 1
  local v
  v="$(go env GOVERSION 2>/dev/null | sed 's/^go//')"
  [[ -n "${v}" ]] || return 1
  # Precisa >= 1.23
  printf '%s\n' "1.23.0" "${v}" | sort -V -C 2>/dev/null
}

install_go_toolchain() {
  if go_version_ok; then
    log "Go já instalado: $(go version)"
    return
  fi

  if dpkg -s golang-go &>/dev/null 2>&1; then
    log "Removendo golang-go do apt (versão antiga) para instalar Go ${GO_VERSION}..."
    apt-get remove -y golang-go 2>/dev/null || true
  fi

  log "Instalando Go ${GO_VERSION} em /usr/local/go ..."
  local arch tarball url
  case "$(uname -m)" in
    x86_64|amd64) arch=amd64 ;;
    aarch64|arm64) arch=arm64 ;;
    *) die "Arquitetura não suportada: $(uname -m)" ;;
  esac
  tarball="go${GO_VERSION}.linux-${arch}.tar.gz"
  url="https://go.dev/dl/${tarball}"

  curl -fsSL "${url}" -o "/tmp/${tarball}"
  rm -rf /usr/local/go
  tar -C /usr/local -xzf "/tmp/${tarball}"
  rm -f "/tmp/${tarball}"

  cat >/etc/profile.d/golang.sh <<'EOF'
export PATH=/usr/local/go/bin:$PATH
EOF
  export PATH="/usr/local/go/bin:${PATH}"

  go version || die "Falha ao instalar Go"
}

clone_or_update_repo() {
  if [[ -d "${INSTALL_DIR}/.git" ]]; then
    log "Atualizando repositório em ${INSTALL_DIR} ..."
    git -C "${INSTALL_DIR}" fetch origin "${REPO_BRANCH}"
    if [[ -n "$(git -C "${INSTALL_DIR}" status --porcelain 2>/dev/null)" ]]; then
      warn "Alterações locais detectadas — sincronizando com origin/${REPO_BRANCH} (descarte intencional em VPS de deploy)."
    fi
    git -C "${INSTALL_DIR}" checkout -B "${REPO_BRANCH}" "origin/${REPO_BRANCH}"
    git -C "${INSTALL_DIR}" reset --hard "origin/${REPO_BRANCH}"
    git -C "${INSTALL_DIR}" clean -fd
  else
    log "Clonando ${REPO_URL} → ${INSTALL_DIR} ..."
    mkdir -p "$(dirname "${INSTALL_DIR}")"
    git clone --depth 1 --branch "${REPO_BRANCH}" "${REPO_URL}" "${INSTALL_DIR}"
  fi
}

build_probe() {
  log "Compilando latency-probe..."
  export PATH="/usr/local/go/bin:${PATH}"
  cd "${INSTALL_DIR}"
  bash scripts/build.sh
  test -x "${INSTALL_DIR}/dist/latency-probe-linux-amd64" \
    || die "Binário não gerado em dist/latency-probe-linux-amd64"
  log "Build OK: ${INSTALL_DIR}/dist/latency-probe-linux-amd64"
}

run_install() {
  if [[ "${SKIP_INSTALL}" == "1" ]]; then
    warn "SKIP_INSTALL=1 — pulando install/upgrade"
    return
  fi

  if detect_probe_hostname || [[ -n "${PROBE_HOSTNAME:-}" ]]; then
    log "Atualizando instalação existente (${PROBE_HOSTNAME}) ..."
    export PROBE_HOSTNAME
    bash "${INSTALL_DIR}/scripts/upgrade-probe.sh"
    return
  fi

  if systemctl is-enabled latency-probe &>/dev/null; then
    log "Serviço latency-probe encontrado — executando upgrade-probe ..."
    bash "${INSTALL_DIR}/scripts/upgrade-probe.sh"
    return
  fi

  warn "PROBE_HOSTNAME não definido — apenas build concluído."
  warn "Primeira instalação:"
  warn "  export PROBE_HOSTNAME=latency-sp-games-1.streethosting.com.br"
  warn "  export CERTBOT_EMAIL=noreply@streethosting.com.br"
  warn "  bash ${INSTALL_DIR}/scripts/install.sh"
  warn "Nó já com nginx (sem variável): bash ${INSTALL_DIR}/scripts/upgrade-probe.sh"
}

print_summary() {
  cat <<EOF

================================================================================
 Bootstrap concluído
================================================================================
 Repositório : ${INSTALL_DIR}
 Binário     : ${INSTALL_DIR}/dist/latency-probe-linux-amd64

 Teste local (sem TLS):
   curl -i http://127.0.0.1:8080/ping -H "Origin: https://streethosting.com.br"

 Atualizar código no futuro:
   bash ${INSTALL_DIR}/scripts/bootstrap-debian.sh

 Só trocar binário (sem apt/git):
   bash ${INSTALL_DIR}/scripts/update.sh

 Teste MTR local (substitua o IP de teste):
   curl -sS http://127.0.0.1:8080/mtr -H "Origin: https://streethosting.com.br" -H "X-Real-IP: 1.1.1.1"

 HTTPS (use o hostname real, não o placeholder SEU-HOSTNAME):
   curl -sS "https://\${PROBE_HOSTNAME:-latency-sp-games-1.streethosting.com.br}/mtr" \\
     -H "Origin: https://streethosting.com.br"
================================================================================
EOF
}

main() {
  require_root
  detect_debian
  install_base_packages
  install_go_toolchain
  clone_or_update_repo
  build_probe
  run_install
  print_summary
}

main "$@"
