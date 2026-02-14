# n8n stack (full setup)

Production-style n8n with:

- **PostgreSQL 16** – persistent database (no SQLite)
- **Redis 7** – queue backend for execution queue
- **Queue mode** – main instance + worker for better throughput
- **Caddy** – reverse proxy at `n8n.${DOMAIN}` with SSO
- **Homepage** – tile under category "AI and automation"

## Komodo / Lovelace variables

Configure these in your Komodo server (secrets or env) so the stack can start:

| Variable | Example | Description |
|----------|---------|-------------|
| `N8N_ENCRYPTION_KEY` | (random 32+ chars) | Used to encrypt credentials; generate once and keep secret. |
| `N8N_POSTGRES_USER` | `postgres` | Postgres superuser (used by init script). |
| `N8N_POSTGRES_PASSWORD` | (secret) | Postgres superuser password. |
| `N8N_POSTGRES_DB` | `n8n` | Database name. |
| `N8N_POSTGRES_NON_ROOT_USER` | `n8n` | Dedicated DB user for n8n. |
| `N8N_POSTGRES_NON_ROOT_PASSWORD` | (secret) | Password for the n8n DB user. |

`CADDY_DOMAIN` and `EXT_PATH` are provided by the existing Lovelace/Komodo config.

## Generate encryption key

```bash
openssl rand -hex 32
```

Use the output as `N8N_ENCRYPTION_KEY` and never change it after workflows/credentials exist.
