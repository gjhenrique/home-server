# PMQ on Lovelace

[PMQ (Poor Man's Quick)](https://github.com/pedro-stanaka/pmq) hosts deployed
static applications, per-app JSON documents and object files in one Go
process. **PMQ is a full-trust system**: only allowlisted Google accounts
reach app hosts, and every one of them can reach every app. Do not widen
access without revisiting that assumption.

## Routing

- `APP.quick.${DOMAIN}` — app hosts. Caddy terminates TLS (wildcard
  certificate via the existing Cloudflare DNS-01 setup), authorizes against
  the existing Google SSO portal with the dedicated `pmq/app-user` role,
  injects `X-PMQ-User` from the email claim, and proxies to PMQ on
  `127.0.0.1:8080`. `/_pmq/admin/*` is blocked on app hosts.
- `pmq-admin.quick.${DOMAIN}` — operator host. Only reachable from
  `PMQ_OPERATOR_CIDRS` (LAN/tailnet), only routes `/_pmq/admin/*`, and PMQ
  still requires the `Authorization: Bearer` deploy token. Point CLI admin
  commands at `https://pmq-admin.quick.${DOMAIN}`.
- PMQ runs in the host network namespace but listens only on
  `127.0.0.1:8080` (`network_mode: host` + `PMQ_LISTEN=127.0.0.1:8080`),
  so Caddy's host-network connections arrive from exactly `127.0.0.1` —
  the only peer PMQ trusts (`PMQ_TRUSTED_PROXIES=127.0.0.1/32`). A
  bridge network with a published loopback port would make the peer the
  bridge gateway instead, which is not a stable or trustworthy address.
  Other containers and the LAN cannot reach a loopback listener at all.

## Prerequisites (one-time, operator)

1. **Komodo secrets** (lovelace instance):
   - `PMQ_DEPLOY_TOKEN` — random 32-byte hex (`openssl rand -hex 32`).
   - `PMQ_ALLOWED_EMAILS` — regex of Google emails that get `pmq/app-user`;
     reuse the value of `CADDY_ALLOWED_EMAILS` or a stricter subset.
   - `PMQ_OPERATOR_CIDRS` — source CIDRs allowed on the operator host, e.g.
     `192.168.0.0/16 100.64.0.0/10`.
2. **Host data directory**: the image runs as uid/gid `100:101`.

   ```sh
   install -d -o 100 -g 101 /mnt/tank/pmq
   ```

3. **DNS**: verify `hello.quick.${DOMAIN}` resolves to the server from LAN
   and tailnet. If the existing AdGuard wildcard rewrite for the apex domain
   does not cover deeper labels, add a rewrite for
   `*.quick.${DOMAIN}` → server IP. No public DNS record is required; the
   wildcard certificate is issued via the existing Cloudflare DNS-01 token.

No PostgreSQL, object storage or AI provider is required: PMQ uses its
filesystem storage in `/data`.

## AI gateway

The OpenAI-compatible gateway at `POST /_pmq/ai/chat/completions` (relative
URL from any app host) is enabled via Komodo secrets: `PMQ_AI_MODELS`,
`PMQ_AI_DEFAULT_MODEL`, `PMQ_AI_PRICING` and `PMQ_OPENROUTER_API_KEY`.
The current binding is `gemini-flash=google/gemini-3.5-flash-lite|stream`
through OpenRouter. Pricing from the OpenRouter model card enables the
global budget cutoff (`PMQ_AI_GLOBAL_BUDGET_USD: 5`, resets at UTC
midnight). Rotate the key by updating the `PMQ_OPENROUTER_API_KEY`
Komodo secret and redeploying this stack.

## Deploy

The `lovelace-pmq` Komodo stack deploys `stacks/pmq/compose.yaml` on
Lovelace only (it is defined solely in `komodo-lovelace.toml`, with
`server = "lovelace"`, and is listed in the Lovelace deploy procedure).
Caddy picks up the new sites from `stacks/caddy/Caddyfile` on the next
`lovelace-caddy` deploy.

## Post-deploy acceptance

From a client inside `PMQ_OPERATOR_CIDRS`:

```sh
DOMAIN=tanaka.dev.br PMQ_PROBE_IP=<lovelace-lan-ip> ./acceptance.sh
```

Then, signed in with an allowlisted account, deploy the hello-world example
and open it:

```sh
pmq deploy ./hello-world --name hello-world \
  --server https://pmq-admin.quick.${DOMAIN}
```

## Backups

Backrest can snapshot `/mnt/tank/pmq`, but SQLite and release pointers are
only consistent when PMQ is stopped (see upstream
[production and recovery](https://github.com/pedro-stanaka/pmq/blob/main/docs/proxy/production.md)).
Prefer a stopped-PMQ archive over live file copies of the bind mount.

## Upgrades

Renovate bumps the image tag in `compose.yaml`. PMQ data layouts move
forward with releases; before upgrading, pin the current image tag for
rollback and take a consistent backup as above.
