# API de Probe de Latência — Teste de Conectividade

Documentação para implementar e fazer deploy dos **endpoints HTTP(S)** usados pela página `/ferramentas/teste-conectividade` no site StreetHosting.

**Público:** agente/desenvolvedor que vai subir a infra (nginx, Worker, microserviço, etc.)  
**Frontend (já existente):** `page-modules/ConnectivityTest.tsx`, `lib/connectivity/probe.ts`, `config/connectivity.ts`

---

## 1. Visão geral

### O que NÃO é

- **Não é ICMP ping.** Navegadores não expõem ping ICMP para IPs arbitrários.
- **Não há API REST no Next.js** hoje que calcule latência no servidor. O RTT é medido **no navegador do usuário** com `fetch()` até URLs configuradas.

### O que é

Uma **API HTTP mínima de probe** — um endpoint leve em cada nó de rede que:

1. Responde rápido (ideal: corpo vazio, status `204`).
2. Permite **CORS** a partir de `https://streethosting.com.br` (e preview Vercel, se aplicável).
3. Está em **HTTPS** (obrigatório em produção; mixed content bloqueia `http://`).

O frontend dispara **6 amostras GET** por alvo, com **350 ms** entre amostras e **timeout de 8 s** por requisição. Métricas calculadas no cliente: mínimo, média, máximo e % de perda.

```
┌─────────────┐     GET /ping (×6)      ┌──────────────────────────┐
│  Navegador  │ ──────────────────────► │ Probe API (seu deploy)   │
│  (usuário)  │ ◄────────────────────── │ IP da rede SP-GAMES etc. │
└─────────────┘     RTT medido local    └──────────────────────────┘
```

---

## 2. Inventário de endpoints (deploy atual)

Cada **rede** tem **2 nós** (2 IPs distintos no mesmo prefixo `/24`). Total: **6 URLs de probe**.

| Rede (UI) | ID rede | Prefixo (informativo) | Hostname sugerido | IP exibido (placeholder) |
|-----------|---------|------------------------|-------------------|---------------------------|
| SP — GAMES | `sp-games` | `177.55.0.0/24` | `latency-sp-games-1.streethosting.com.br` | `177.55.0.10` |
| SP — GAMES | `sp-games` | | `latency-sp-games-2.streethosting.com.br` | `177.55.0.11` |
| SP — EMPRESA | `sp-empresa` | `177.56.0.0/24` | `latency-sp-empresa-1.streethosting.com.br` | `177.56.0.10` |
| SP — EMPRESA | `sp-empresa` | | `latency-sp-empresa-2.streethosting.com.br` | `177.56.0.11` |
| SP — NÃO MITIGADA | `sp-nao-mitigada` | `177.57.0.0/24` | `latency-sp-raw-1.streethosting.com.br` | `177.57.0.10` |
| SP — NÃO MITIGADA | `sp-nao-mitigada` | | `latency-sp-raw-2.streethosting.com.br` | `177.57.0.11` |

**Substitua** prefixos, IPs e hostnames pelos valores reais da operação antes do go-live.

**Path padrão:** `/ping`  
**URL completa de exemplo:** `https://latency-sp-games-1.streethosting.com.br/ping`

Após o deploy, atualize `config/connectivity.ts` no repositório do site com `probeUrl` e `displayAddress` finais.

---

## 3. Contrato HTTP da Probe API

### 3.1 Rotas obrigatórias

| Método | Path | Descrição |
|--------|------|-----------|
| `GET` | `/ping` | Probe principal — deve responder em &lt; 50 ms de processamento no servidor |
| `HEAD` | `/ping` | Opcional; mesmo comportamento que GET (sem body) |
| `OPTIONS` | `/ping` | **Obrigatório** para preflight CORS |

Não é necessário body JSON na resposta. O frontend **não lê o corpo** — só mede tempo até o fim da requisição.

### 3.2 Resposta de sucesso

**Recomendado:**

```http
HTTP/1.1 204 No Content
Cache-Control: no-store, no-cache, must-revalidate
Access-Control-Allow-Origin: https://streethosting.com.br
Access-Control-Allow-Methods: GET, HEAD, OPTIONS
Access-Control-Max-Age: 86400
```

**Alternativa aceita:** `200 OK` com body vazio ou `{"ok":true}` — o frontend considera sucesso se `fetch` não lançar exceção (modo CORS) ou completar (modo `no-cors` opaco).

### 3.3 Preflight CORS (`OPTIONS`)

```http
OPTIONS /ping HTTP/1.1
Origin: https://streethosting.com.br
Access-Control-Request-Method: GET
```

Resposta:

```http
HTTP/1.1 204 No Content
Access-Control-Allow-Origin: https://streethosting.com.br
Access-Control-Allow-Methods: GET, HEAD, OPTIONS
Access-Control-Max-Age: 86400
```

### 3.4 Headers CORS — origens permitidas

Em **produção**, libere pelo menos:

| Origin | Motivo |
|--------|--------|
| `https://streethosting.com.br` | Site principal |
| `https://www.streethosting.com.br` | Se usar redirect www |

Em **staging** (opcional):

| Origin | Motivo |
|--------|--------|
| `https://*.vercel.app` | Não é válido literalmente em `Allow-Origin` — use lista explícita ou reflexão controlada por allowlist |
| `http://localhost:3000` | Dev local |

**Importante:** `Access-Control-Allow-Origin` só aceita **um** origin por resposta (ou `*` sem credentials — aqui não usamos credentials). Para múltiplos ambientes, implemente allowlist no servidor:

```
se Origin ∈ ALLOWED_ORIGINS → echo Origin
senão → omitir header (preflight falha)
```

### 3.5 Headers que o servidor NÃO deve exigir

O frontend envia:

```http
GET /ping HTTP/1.1
Host: latency-sp-games-1.streethosting.com.br
Origin: https://streethosting.com.br
Cache-Control: no-store
```

- **Sem** `Authorization`, cookies ou API keys.
- **Sem** corpo na requisição.

### 3.6 Cache

Sempre enviar:

```http
Cache-Control: no-store, no-cache, must-revalidate
Pragma: no-cache
```

Evita CDN/browser mascarar RTT real.

### 3.7 TLS e DNS

| Requisito | Detalhe |
|-----------|---------|
| Certificado | Válido para o hostname do probe (wildcard `*.streethosting.com.br` é o caminho mais simples) |
| DNS | Registro `A`/`AAAA` do hostname → IP do nó na rede correta |
| Binding | O tráfego deve sair/entrar pelo **mesmo POP/rede** que o teste representa (Games vs Empresa vs Não mitigada) |

Probing direto por IP (`https://177.55.0.10/ping`) exige certificado com **IP no SAN** — evite; use hostname.

---

## 4. Comportamento do cliente (referência para testes)

Implementação: `lib/connectivity/probe.ts`.

| Parâmetro | Valor |
|-----------|-------|
| Amostras por alvo | 6 |
| Intervalo entre amostras | 350 ms |
| Timeout por requisição | 8000 ms |
| Método HTTP | `GET` |
| `credentials` | `omit` |
| `cache` | `no-store` |

**Fluxo por amostra:**

1. Tenta `fetch(url, { mode: 'cors' })`.
2. Se falhar, tenta `fetch(url, { mode: 'no-cors' })` (menos confiável; use só como fallback).
3. `ok: true` se a promise resolver sem abort/timeout.
4. RTT = `round(performance.now() - start)` em milissegundos.

**Agregação (no frontend):**

| Métrica | Cálculo |
|---------|---------|
| Melhor | `min(ms)` das amostras com `ok: true` |
| Média | `round(mean(ms))` das amostras ok |
| Máximo | `max(ms)` das amostras ok |
| Perda | `round((falhas / total) * 100)%` |

Se **todas** as 6 amostras falharem → UI mostra erro no card da rede.

**Carga estimada por usuário que testa 3 redes de uma vez:**

- 3 redes × 2 nós × 6 GET ≈ **36 requisições** em ~15–20 s (sequencial por nó no código atual).

Planeje rate limit generoso para usuários legítimos, mas proteja contra abuso (ver §7).

---

## 5. Implementações de referência

### 5.1 nginx (recomendado no bare metal / VM)

```nginx
server {
    listen 443 ssl http2;
    server_name latency-sp-games-1.streethosting.com.br;

    # ssl_certificate / ssl_certificate_key ...

    location /ping {
        # Allowlist de Origin (ajuste conforme ambiente)
        set $cors_origin "";
        if ($http_origin ~* ^https://(www\.)?streethosting\.com\.br$) {
            set $cors_origin $http_origin;
        }
        if ($http_origin = "http://localhost:3000") {
            set $cors_origin $http_origin;
        }

        if ($request_method = OPTIONS) {
            add_header Access-Control-Allow-Origin $cors_origin always;
            add_header Access-Control-Allow-Methods "GET, HEAD, OPTIONS" always;
            add_header Access-Control-Max-Age 86400 always;
            add_header Content-Length 0;
            return 204;
        }

        add_header Access-Control-Allow-Origin $cors_origin always;
        add_header Access-Control-Allow-Methods "GET, HEAD, OPTIONS" always;
        add_header Cache-Control "no-store, no-cache, must-revalidate" always;
        return 204;
    }
}
```

Repita o `server {}` por hostname/IP, ou use `map` + SNI se centralizar.

### 5.2 Cloudflare Worker (edge leve)

```js
const ALLOWED = new Set([
  "https://streethosting.com.br",
  "https://www.streethosting.com.br",
  "http://localhost:3000",
]);

export default {
  async fetch(request) {
    const url = new URL(request.url);
    if (url.pathname !== "/ping") {
      return new Response("Not Found", { status: 404 });
    }

    const origin = request.headers.get("Origin") ?? "";
    const headers = {
      "Cache-Control": "no-store, no-cache, must-revalidate",
      ...(ALLOWED.has(origin)
        ? {
            "Access-Control-Allow-Origin": origin,
            "Access-Control-Allow-Methods": "GET, HEAD, OPTIONS",
            "Access-Control-Max-Age": "86400",
          }
        : {}),
    };

    if (request.method === "OPTIONS") {
      return new Response(null, { status: 204, headers });
    }
    if (request.method === "GET" || request.method === "HEAD") {
      return new Response(null, { status: 204, headers });
    }
    return new Response("Method Not Allowed", { status: 405, headers });
  },
};
```

Roteie cada hostname para o Worker ou para origin no IP correto conforme a rede.

### 5.3 Node.js (Express mínimo)

```js
import express from "express";

const ALLOWED = new Set([
  "https://streethosting.com.br",
  "https://www.streethosting.com.br",
]);

const app = express();

app.use("/ping", (req, res) => {
  const origin = req.headers.origin;
  if (origin && ALLOWED.has(origin)) {
    res.setHeader("Access-Control-Allow-Origin", origin);
    res.setHeader("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS");
    res.setHeader("Access-Control-Max-Age", "86400");
  }
  res.setHeader("Cache-Control", "no-store, no-cache, must-revalidate");

  if (req.method === "OPTIONS") return res.status(204).end();
  if (req.method === "GET" || req.method === "HEAD") return res.status(204).end();
  res.status(405).end();
});

app.listen(8443);
```

---

## 6. Checklist de deploy

### Por nó (×6)

- [ ] DNS `A`/`AAAA` apontando para o IP correto da rede
- [ ] TLS válido no hostname
- [ ] `GET /ping` → `204` em &lt; 100 ms (latência de processamento)
- [ ] `OPTIONS /ping` com CORS correto
- [ ] `Cache-Control: no-store` presente
- [ ] Tráfego passa pela rede anunciada (Games / Empresa / Raw)

### Integração com o site

- [ ] Atualizar `config/connectivity.ts` com `probeUrl` e `displayAddress` reais
- [ ] Deploy do site Next.js
- [ ] Testar em `https://streethosting.com.br/ferramentas/teste-conectividade`
- [ ] Testar a partir de rede residencial + 4G (validar RTT plausível)

### Comandos de verificação manual

```bash
# Preflight
curl -i -X OPTIONS "https://latency-sp-games-1.streethosting.com.br/ping" \
  -H "Origin: https://streethosting.com.br" \
  -H "Access-Control-Request-Method: GET"

# Probe
curl -i "https://latency-sp-games-1.streethosting.com.br/ping" \
  -H "Origin: https://streethosting.com.br"
```

Esperado: `204`, headers `Access-Control-Allow-Origin` e `Cache-Control: no-store`.

No navegador (DevTools → Console):

```js
const t0 = performance.now();
await fetch("https://latency-sp-games-1.streethosting.com.br/ping", {
  mode: "cors",
  cache: "no-store",
});
console.log(Math.round(performance.now() - t0), "ms");
```

---

## 7. Segurança e limites

| Risco | Mitigação sugerida |
|-------|-------------------|
| Abuso / DDoS reflexivo | Rate limit por IP no nginx/CF (ex.: 60 req/min por IP) |
| Scan de infra | Não exponha versões, banners ou paths além de `/ping` |
| CORS aberto (`*`) | Evitar; usar allowlist de origins |
| Log excessivo | Não logar cada GET em disco; métricas agregadas no edge bastam |
| Amplificação | Resposta sem body (204); sem compressão desnecessária |

A Probe API **não deve**:

- Executar código arbitrário
- Aceitar POST com payload grande
- Proxificar tráfego para outros hosts
- Retornar dados sensíveis

---

## 8. API opcional de metadados (futuro — não implementada)

Se no futuro quiserem **configuração dinâmica** (sem redeploy do Next.js), pode-se adicionar um serviço separado:

### `GET https://latency-api.streethosting.com.br/v1/networks`

```json
{
  "version": 1,
  "networks": [
    {
      "id": "sp-games",
      "name": "SP — GAMES",
      "description": "Rede com mitigação orientada a jogos.",
      "prefix": "177.55.0.0/24",
      "targets": [
        {
          "id": "games-a",
          "label": "Nó A",
          "displayAddress": "177.55.0.10",
          "probeUrl": "https://latency-sp-games-1.streethosting.com.br/ping"
        }
      ]
    }
  ]
}
```

| Requisito | Detalhe |
|-----------|---------|
| CORS | Mesma allowlist do §3.4 |
| Cache | `Cache-Control: public, max-age=300` (config muda pouco) |
| Auth | Pública (read-only) |

O frontend atual **não consome** esse endpoint; hoje tudo vem de `config/connectivity.ts`. Documentado apenas para evolução.

---

## 9. O que NÃO implementar nesta API

| Item | Motivo |
|------|--------|
| ICMP echo no servidor retornado ao browser | Impossível expor ao JS de forma portável |
| Medição server-side apresentada como “seu ping” | Seria latência datacenter→nó, não usuário→nó |
| WebSocket obrigatório | GET + CORS é suficiente e mais simples |
| Autenticação por usuário | Ferramenta pública |

Para **Minecraft** (jogadores online, MOTD, etc.), use o produto **MC Status** (`mcstatus.streethosting.com.br`) — escopo diferente.

---

## 10. Critérios de aceite (go-live)

1. Os **6** `probeUrl` respondem `204` com CORS para `https://streethosting.com.br`.
2. Na página de teste, cada rede mostra RTT numérico (não “—” nem 100% perda) a partir de rede brasileira típica.
3. `OPTIONS` e `GET` funcionam sem redirect HTTP→HTTPS que quebre preflight (redirects devem ser só na camada DNS/ingress, ou HTTPS desde o primeiro byte).
4. IPs exibidos na UI correspondem aos nós reais documentados internamente.
5. Prefixos na UI batem com o BGP/anúncio de cada rede.

---

## 11. Referências no repositório do site

| Arquivo | Conteúdo |
|---------|----------|
| `config/connectivity.ts` | Redes, IPs exibidos, URLs de probe |
| `lib/connectivity/probe.ts` | Lógica de amostragem e RTT |
| `page-modules/ConnectivityTest.tsx` | UI |
| `app/ferramentas/teste-conectividade/page.tsx` | Rota pública |

---

## 12. Resumo para o agente de deploy

**Entregável:** 6 endpoints HTTPS `GET /ping` (e `OPTIONS` com CORS), um por hostname, cada um terminando no IP da rede correta (2 por prefixo × 3 redes).

**Contrato mínimo:** `204 No Content` + `Access-Control-Allow-Origin` (allowlist) + `Cache-Control: no-store`.

**Depois do deploy:** atualizar `config/connectivity.ts` e publicar o site.
