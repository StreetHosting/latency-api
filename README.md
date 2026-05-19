# StreetHosting Latency Probe API

HTTP(S) probe endpoints for the connectivity test page (`/ferramentas/teste-conectividade`). Implements the contract in [connectivity-probe-api.md](./connectivity-probe-api.md).

## Architecture

```
Browser (streethosting.com.br)
    │  GET /ping ×6 (CORS)
    ▼
nginx (TLS, rate limit, keepalive)
    ▼
latency-probe (Go, 127.0.0.1:8080)
    └── 204 No Content + CORS allowlist
```

Each **VPS** runs one probe node (one hostname). Six VPS instances cover three networks × two nodes.

## Quick start (development)

```bash
make dev
curl -i http://127.0.0.1:8080/ping -H "Origin: https://streethosting.com.br"
```

## Build

```bash
make build    # → dist/latency-probe-linux-amd64
make test
```

CI uploads the Linux binary as an artifact on every push.

## Deploy on Debian 13 VPS

### Debian 13 minimal (bootstrap automático)

Em uma VPS **minimal** sem `git`, `make`, `sudo` ou `go`, execute **como root** (substitua o hostname):

```bash
export PROBE_HOSTNAME=latency-sp-games-1.streethosting.com.br
export CERTBOT_EMAIL=noreply@streethosting.com.br

curl -fsSL https://raw.githubusercontent.com/StreetHosting/latency-api/main/scripts/bootstrap-debian.sh | bash
```

O script instala dependências via `apt`, clona [StreetHosting/latency-api](https://github.com/StreetHosting/latency-api) em `/opt/latency-api`, compila o binário e roda `install.sh` (nginx, certbot, systemd).

**Só build** (sem instalar nginx/TLS ainda):

```bash
curl -fsSL https://raw.githubusercontent.com/StreetHosting/latency-api/main/scripts/bootstrap-debian.sh | SKIP_INSTALL=1 bash
```

**Reexecutar** no mesmo servidor (atualiza repo + recompila + reinstala se `PROBE_HOSTNAME` estiver setado):

```bash
export PROBE_HOSTNAME=latency-sp-games-1.streethosting.com.br
bash /opt/latency-api/scripts/bootstrap-debian.sh
```

Antes do bootstrap: DNS do `PROBE_HOSTNAME` deve apontar para o IP desta VPS.

### Fleet (recommended): Ansible

1. Copy inventory and edit IPs/hostnames:

   ```bash
   cp deploy/inventory/hosts.example.yml deploy/inventory/hosts.yml
   ```

2. Ensure DNS `A`/`AAAA` for each `probe_hostname` points to that VPS.

3. Build and deploy all nodes:

   ```bash
   make deploy
   ```

4. Verify from your workstation:

   ```bash
   make verify
   # or: ./scripts/verify.sh latency-sp-games-1.streethosting.com.br
   ```

### Single node (manual)

On the VPS as root, copy the repo (or `make package` bundle), set hostname, install:

```bash
export PROBE_HOSTNAME=latency-sp-games-1.streethosting.com.br
export CERTBOT_EMAIL=noreply@streethosting.com.br
make build
sudo bash scripts/install.sh
```

### Update binary (rolling)

One host:

```bash
sudo BINARY_SRC=./dist/latency-probe-linux-amd64 bash scripts/update.sh
```

All hosts:

```bash
make update-fleet
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `LISTEN_ADDR` | `127.0.0.1:8080` | Bind address (behind nginx) |
| `ALLOWED_ORIGINS` | streethosting + localhost | Comma-separated CORS allowlist |

File on VPS: `/etc/latency-probe/probe.env` (see [configs/probe.env.example](./configs/probe.env.example)).

## Operations

| Script | Purpose |
|--------|---------|
| `scripts/bootstrap-debian.sh` | VPS minimal: apt + clone + build (+ install opcional) |
| `scripts/healthcheck.sh` | systemd/monitoring: local `GET /ping` |
| `scripts/verify.sh` | Contract check (OPTIONS + GET, CORS, cache) |
| `scripts/update.sh` | Replace binary + restart |
| `make update-fleet` | Ansible rolling restart on all nodes |

**Rate limit:** nginx `60 req/min` per IP with burst 20 (see nginx template).

**Logs:** probe uses JSON stdout; nginx access log disabled on `/ping` to avoid disk churn.

## Probe nodes (inventory)

| Network | Hostname | Example IP |
|---------|----------|------------|
| SP — GAMES | `latency-sp-games-1.streethosting.com.br` | `177.55.0.10` |
| SP — GAMES | `latency-sp-games-2.streethosting.com.br` | `177.55.0.11` |
| SP — EMPRESA | `latency-sp-empresa-1.streethosting.com.br` | `177.56.0.10` |
| SP — EMPRESA | `latency-sp-empresa-2.streethosting.com.br` | `177.56.0.11` |
| SP — NÃO MITIGADA | `latency-sp-raw-1.streethosting.com.br` | `177.57.0.10` |
| SP — NÃO MITIGADA | `latency-sp-raw-2.streethosting.com.br` | `177.57.0.11` |

Replace placeholders with production values before go-live.

## After deploy

Update `config/connectivity.ts` in the Next.js site with final `probeUrl` and `displayAddress` values, then publish the site.

## Acceptance criteria

See §10 in [connectivity-probe-api.md](./connectivity-probe-api.md). Run `make verify` against all six hostnames before go-live.
