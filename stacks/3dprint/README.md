# 3dprint – Bambuddy

Bambuddy: print archive, monitoring, and virtual printer for Bambu Lab printers. This stack runs **without host network**; add printers manually by IP in the Bambuddy UI.

## Komodo configuration

| Variable | Secret? | Description |
|----------|--------|-------------|
| `EXT_PATH` | No | Base path for persistent data (e.g. `/mnt/tank`). Used for `3dprinting/virtual_printer`. |
| `TZ` | No | Timezone (e.g. `Europe/Berlin`). |
| `PORT` | No | HTTP port for Bambuddy (default `8000`). On Lovelace use `8002`; port 8000 is used by Paperless. |
| `DOMAIN` | No | Caddy domain for reverse proxy (e.g. `bambuddy.${DOMAIN}`). |
| `VIRTUAL_PRINTER_PASV_ADDRESS` | No | Optional: Docker host IP for FTP passive mode. Omit if not using virtual printer FTP. |

## Prerequisites

- Bambuddy runs without host network. Add printers manually by IP in the Bambuddy web UI (Settings → Printers).
- The directory `${EXT_PATH}/3dprinting/virtual_printer` is used for the virtual printer; create it if needed (Bambuddy may create it on first use).

## Health / URL

- **Local:** `http://<host>:8000` (or `http://<host>:${PORT}` if overridden; Lovelace uses port 8002).
- **Behind Caddy:** `https://bambuddy.${DOMAIN}` (when `DOMAIN` is set and Caddy is configured).

## Prometheus

Bambuddy can expose metrics at `/api/v1/metrics` when enabled in Settings. The stack adds Prometheus scrape labels; ensure the metrics endpoint is enabled in Bambuddy if you want to scrape.
