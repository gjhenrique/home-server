# Mere Medical stack

[Mere Medical](https://github.com/cfu288/mere-medical) is an offline-first, self-hosted web app to aggregate and sync medical records from multiple patient portals (Epic MyChart, Cerner, OnPatient/DrChrono, VA, Healow, etc.) in one place.

- **Caddy** – reverse proxy at `health.${DOMAIN}` with SSO (Google OAuth)
- **Homepage** – tile under "Health" linking to `https://health.${DOMAIN}`

## Auth and SSO

- **Caddy SSO** protects access to the app: users must sign in with Google (your existing auth portal) before reaching Mere Medical. Unauthenticated requests are redirected to the login page.
- **Mere Medical itself** does not support forward-auth or remote-user headers. The app is “local first” with no built-in account system; it does not read `X-Token-User-Email` or similar headers to log users in. So we use SSO only as a **gate** in front of the app: only people who passed Caddy OAuth can open `https://health.${DOMAIN}`. Inside the app, there is no separate login step unless you link a patient portal (Epic, OnPatient, etc.).

## Komodo variables

| Variable | Required | Description |
|----------|----------|-------------|
| `PUBLIC_URL` | Yes | Full URL where Mere Medical is accessible (must include `https://` or `http://`). Set to `https://health.${DOMAIN}` when using Caddy. |
| `ONPATIENT_CLIENT_ID` / `ONPATIENT_CLIENT_SECRET` | No | OnPatient/DrChrono – see [OnPatient setup](https://meremedical.co/docs/getting-started/onpatient-setup) |
| `EPIC_CLIENT_ID_R4` / `EPIC_SANDBOX_CLIENT_ID_R4` | No | Epic MyChart – see [Epic setup](https://meremedical.co/docs/getting-started/epic-setup) |
| `CERNER_CLIENT_ID` | No | Cerner Health |
| `VERADIGM_CLIENT_ID` | No | Veradigm |
| `VA_CLIENT_ID` | No | VA (Veterans Affairs; sandbox only for now) |
| `HEALOW_CLIENT_ID` / `HEALOW_CLIENT_SECRET` | No | Healow (eClinicalWorks) – see [Healow setup](https://meremedical.co/docs/getting-started/healow-setup) |

`EXT_PATH` and `DOMAIN` are provided by the existing Lovelace/Komodo config. `PUBLIC_URL` is set in the stack environment to `https://health.[[CADDY_DOMAIN]]` so it matches your Caddy domain.
