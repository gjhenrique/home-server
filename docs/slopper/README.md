# Slopper operations

Slopper is a private static app host backed by one shared InsForge/PostgreSQL deployment on Lovelace, with file objects stored in Backblaze B2 through its S3-compatible API. Browser apps never receive InsForge keys. App pages and APIs require the existing Caddy Google SSO session; Caddy checks the socket peer against `SLOPPER_PRIVATE_CIDRS` with `remote_ip`, so forwarded headers cannot grant access. Empty CIDR configuration falls back to `0.0.0.0/32` and denies ordinary clients.

## Install and required secrets

The InsForge Compose configuration and initialization SQL are adapted from the official InsForge source at [commit `c1c886e`](https://github.com/InsForge/InsForge/tree/c1c886e775d286433e4b50c78c24335783e8fc00/deploy/docker-compose). PostgreSQL, PostgREST, and the backend run locally; file objects go directly to a pre-created B2 bucket.

First create a private Backblaze B2 bucket and an application key scoped to that bucket with read/write access. Use the bucket region exactly as shown in Backblaze; set `SLOPPER_B2_ENDPOINT_URL` to `https://s3.<region>.backblazeb2.com` and `SLOPPER_B2_REGION` to `<region>`. The bucket is not created automatically. Create Komodo secrets for `SLOPPER_POSTGRES_USER`, `SLOPPER_POSTGRES_PASSWORD`, `SLOPPER_POSTGRES_DB`, `SLOPPER_JWT_SECRET`, `SLOPPER_ENCRYPTION_KEY`, `SLOPPER_ADMIN_USERNAME`, `SLOPPER_ADMIN_PASSWORD`, `SLOPPER_ACCESS_API_KEY`, `SLOPPER_ACCESS_ANON_KEY`, `SLOPPER_B2_BUCKET`, `SLOPPER_B2_REGION`, `SLOPPER_B2_ENDPOINT_URL`, `SLOPPER_B2_APPLICATION_KEY_ID`, `SLOPPER_B2_APPLICATION_KEY`, `SLOPPER_OPENROUTER_API_KEY`, `SLOPPER_AI_MODEL`, `SLOPPER_GATEWAY_PROXY_SECRET`, `SLOPPER_GATEWAY_CSRF_SECRET`, and `SLOPPER_PRIVATE_CIDRS`. Supply private CIDRs as a space-separated list, for example `192.168.0.0/16 100.64.0.0/10`. The gateway receives only `SLOPPER_ACCESS_ANON_KEY` for InsForge REST authorization; the administrator API key and root administrator credentials stay on the InsForge stack/operator side. Its JWT secret is the existing `CADDY_JWT_SHARED_KEY`, because it verifies the AuthCrunch browser session. Use distinct random values for database JWT, encryption, proxy, and CSRF secrets. Never put actual values in `.env` files or git.

`SLOPPER_AI_MODEL` is mandatory. Choose the model explicitly; no paid model is selected implicitly. InsForge's implemented `/api/ai/chat/completion` route is currently deprecated upstream, so treat it as a compatibility surface and review upstream changes before upgrades.

The Komodo procedure deploys `lovelace-insforge` and `lovelace-slopper` after source sync and pulls. Persistent state is under `/mnt/tank/slopper` (PostgreSQL, InsForge logs/storage, hosted sites, and database dumps). Object traffic goes from InsForge to Backblaze B2 over HTTPS; credentials remain server-side. InsForge administrative HTTP is loopback on 127.0.0.1:7133 and its auth listener on 7131; the admin proxy, gateway, and static NGINX are loopback on 7130, 7132, and 8088. No Docker socket is mounted.

## DNS and private network

Add an AdGuard wildcard rewrite for `*.slopper.tanaka.dev.br` to Lovelace's LAN address. Tailnet clients need the existing subnet route and split-DNS resolver for `slopper.tanaka.dev.br`. Do not enable Funnel for Slopper; the current Funnel webhook-only configuration stays unchanged. The Caddy wildcard uses Cloudflare DNS-01 and routes `api` and `admin` before app hosts.

Check the router has no unintended TCP 80/443 forwarding, and check IPv6 separately: remove any public AAAA record or IPv6 router exposure unless the firewall has an equivalent restriction. If LAN or tailnet clients reach Caddy over IPv6, add the actual trusted LAN and Tailscale IPv6 CIDRs to `SLOPPER_PRIVATE_CIDRS` too. From a public network, test with valid SNI and Host:

```sh
curl --resolve demo.slopper.tanaka.dev.br:443:PUBLIC_IP https://demo.slopper.tanaka.dev.br/
curl --resolve api.slopper.tanaka.dev.br:443:PUBLIC_IP https://api.slopper.tanaka.dev.br/demo/db/demo
```

Both must be rejected by the private ingress matcher. Repeat over IPv6 where the host has a public address. Do not infer rejection from DNS alone.

## Application names and resources

App names are lowercase DNS labels of at most 32 characters; `api` and `admin` are reserved. Local table and bucket names use lowercase letters, digits, and underscore, start with a letter, and are at most 24 characters. SQL tables use `public.slp_<app>__<table>`. Private buckets use `slp_<app>`; local bucket labels map to the one private bucket for their app. The gateway reads `stacks/slopper/resources/manifest.json` read-only and denies resources absent from it. Update it only as part of administrator initialization. The gateway verifies AuthCrunch's `AUTHP_ACCESS_TOKEN` cookie by HMAC and checks its signed `roles` claim; the separate session tracking cookie is not an authorization credential.

For each registered resource, an administrator should add a numbered SQL migration under `resources/migrations/` and apply it with the InsForge administrative database connection. The migration must create the table, policy and indexes; then create the private bucket through the administrator MCP tool and update the manifest. Example:

```sql
create table if not exists public.slp_slopper_demo__demo (
  id bigint generated always as identity primary key,
  message text not null,
  created_at timestamptz not null default now()
);
alter table public.slp_slopper_demo__demo enable row level security;
grant select, insert, update, delete on public.slp_slopper_demo__demo to anon;
create policy "Slopper demo rows" on public.slp_slopper_demo__demo
  for all to anon using (true) with check (true);
```

InsForge's admin key and database credentials are for controlled operator use only. Static app deployments cannot run migrations. Keep per-app policies narrow; the sample policy is only for a throwaway demo resource.

## Admin console and MCP

`admin.slopper.tanaka.dev.br` is private and additionally gated by existing SSO and InsForge's own administrator login. For local administration, establish an SSH tunnel through the administrator's SSH channel, then visit `http://127.0.0.1:17133` through the tunnel to the InsForge listener. Do not expose this port on LAN or WAN. Configure the pinned `@insforge/mcp` package in stdio mode on the admin workstation, pointing it at the forwarded local URL and supplying the admin API key from a local secret manager. Example MCP entry (replace the value locally; do not commit it):

```json
{
  "mcpServers": {
    "insforge": {
      "command": "npx",
      "args": ["-y", "@insforge/mcp@1.2.11"],
      "env": { "API_KEY": "<local-admin-key>", "API_BASE_URL": "http://127.0.0.1:17133" }
    }
  }
}
```

Create the tunnel with `ssh -N -L 17133:127.0.0.1:7133 <administrator-host>` using normal host-key verification. This is stdio transport; credentials stay on the operator workstation.

## Deploy CLI

Build with `go build -o slopper ./cmd/slopper`. The remote SSH account must use normal host-key verification and an sshd forced command that invokes the fixed receiver with no PTY, forwarding, or shell:

```text
Match User slopper-deploy
  ForceCommand /usr/local/libexec/slopper receive
  DisableForwarding yes
  PermitTTY no
  AllowTcpForwarding no
  X11Forwarding no
```

Install the receiver binary at `/usr/local/libexec/slopper`, set `SLOPPER_ROOT=/mnt/tank/slopper/sites` in its restricted service environment, and allow that account to write only to that directory. Provision keys using an SSH authorized key with `restrict` options as well. The client sends a JSON request then a bounded tar stream over stdin; all upload content is treated as archive data and remote commands are never composed from user input.

```sh
slopper deploy --host slopper-deploy@lovelace --app my-app --dir dist
slopper list --host slopper-deploy@lovelace --app my-app
slopper status --host slopper-deploy@lovelace --app my-app
slopper rollback --host slopper-deploy@lovelace --app my-app --release RELEASE_ID
slopper delete --host slopper-deploy@lovelace --app my-app --yes
```

Archives must contain `index.html`; links, special files, dotfiles, traversal, duplicate names, and uploads over 100 MiB are rejected. Releases are immutable content hashes; deployment atomically switches `current`, and keeps the active release plus recent rollback targets. `delete` removes hosting files only. Database rows and objects need separate administrator removal. NGINX serves the tree read-only, with immutable caching for hashed assets, HTML revalidation, SPA navigation, and 404s for missing assets.

## Client API

`demo/slopper-demo/src/helper.ts` is the reference helper. Same-origin routes are `/_slopper/identity`, `/_slopper/csrf`, `/_slopper/db/<table>`, `/_slopper/objects/<bucket>/<key>`, and `/_slopper/ai/chat/completion`. The API host accepts equivalent app-qualified paths such as `/my-app/db/items`. Mutations require same-origin HTTPS `Origin` and `X-Slopper-CSRF`; the token is bound to the signed session subject and expiry. API 401s are JSON so the helper can start a top-level SSO flow. Use the existing portal logout URL to clear the shared session cookie. Logout cannot revoke a copied stateless JWT; copied tokens remain valid through expiry, consistent with [AuthCrunch's documented stateless-token behavior](https://docs.authcrunch.com/docs/authenticate/logout).

The REST gateway only forwards registered CRUD, private object operations capped at 100 MiB, and the configured AI completion route. For InsForge storage and AI routes it injects the server-side anonymous key as a Bearer token after removing client-supplied Authorization and API-key headers. SQL, RPC, schema/auth management, secrets, provider setup, compute, relationship expansion, and arbitrary paths are not exposed. Browser cross-origin access is disabled.

## Backups and recovery

Add `/mnt/tank/slopper` to the existing Backrest plan on Lovelace. Before each scheduled backup, pause admin changes and run `stacks/slopper/backup.sh` with the gateway and PostgreSQL container names. It stops gateway writes, creates a custom-format `pg_dump` under `/mnt/tank/slopper/backups`, then restarts the gateway. Backrest captures that dump together with releases, manifest/migrations, and InsForge data. B2 stores the objects independently; use B2 lifecycle/versioning and a separate backup policy if you need object recovery. Include a protected secret backup in an encrypted Backrest repository; never place plaintext secrets in the app data tree. Keep admin changes paused until the Backrest snapshot completes so the database dump and B2 object state stay coordinated.

Restore into a separate destination first: restore secret material into a temporary secret store, restore PostgreSQL dump, restore or verify the B2 bucket separately, restore manifest/migrations and site releases, then start the stack with a separate Compose project and ports. Verify identity, CRUD, object integrity and static release routing there. Never test recovery against production data.

## Acceptance and upgrade checklist

Run the Go and demo builds and static configuration checks before deployment. A live acceptance operator must verify Google login, cross-app SSO, portal logout, wildcard TLS, LAN access, tailnet access, and public ingress rejection with valid SNI and Host. Also verify persistence after recreation, uploads/downloads, deployment, rollback, and backup restore. No live login or production deployment is implied by this repository change.

For upgrades, inspect the matching InsForge release compose, initialization SQL, image digest and deprecated AI route. Back up first, update the pinned digest and vendored init SQL as one change, restore in an isolated project, and then deploy through the explicit Komodo procedure.
