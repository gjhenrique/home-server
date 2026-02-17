# Donetick

Open-source task and chore management (natural language tasks, recurring chores, assignee rotation, points).

- **Caddy** – reverse proxy at `chores.${DOMAIN}` with SSO (Google OAuth gate).
- **Auth**: Caddy SSO is a **gate only**. Donetick does not support forward-auth/remote-user headers (unlike Paperless). After passing Caddy you sign in to Donetick with username/password, or configure Donetick’s built-in OIDC (e.g. Google) for single sign-on inside the app.

## Paperless-style SSO (for reference)

Paperless can reuse Caddy’s SSO as real login:

1. Caddy’s `mypolicy` uses `inject headers with claims`, so after Google OAuth it adds headers (e.g. `X-Token-User-Email`) to the request.
2. Paperless is configured with `PAPERLESS_ENABLE_HTTP_REMOTE_USER: true` and `PAPERLESS_HTTP_REMOTE_USER_HEADER_NAME: HTTP_X_TOKEN_USER_EMAIL`, so it trusts that header and logs the user in (or creates an account).

Donetick has no equivalent “trust proxy header as user” option; it only supports OIDC or local auth. So we use Caddy SSO only as a gate here.

## Optional: Donetick OIDC (Google)

To get single sign-on *inside* Donetick (no second login), configure Donetick’s OIDC to use Google (or another provider). Add env vars (or config) for the OAuth2 provider; redirect URL should be `https://chores.${DOMAIN}/auth/oauth2` (confirm path in [Donetick OIDC docs](https://docs.donetick.com/advance-settings/openid-connect-setup/)).

## Environment

| Variable | Required | Description |
|----------|----------|-------------|
| `EXT_PATH` | Yes | Base path for persistent data (e.g. `/mnt/tank`). |
| `DOMAIN` | Yes | Caddy domain (e.g. from `[[CADDY_DOMAIN]]`). |
| `DONETICK_JWT_SECRET` | Yes | JWT secret for sessions (e.g. `openssl rand -base64 32`). |

`EXT_PATH` and `DOMAIN` come from Lovelace/Komodo. Set `DONETICK_JWT_SECRET` in your Komodo secrets.

## Data

- `${EXT_PATH}/donetick/data` – SQLite DB and uploads.
