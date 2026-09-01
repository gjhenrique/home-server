# OpenClaw stack

OpenClaw is an open-source AI agent framework that connects LLMs to your machine for running commands and automating workflows. This stack runs the gateway with **WhatsApp** and **Telegram** as input channels and **OpenRouter** as the LLM layer, behind Caddy SSO and with a Homepage tile.

## Requirements

- Docker (Compose v2)
- Komodo stack env: `EXT_PATH`, `DOMAIN`, `TZ`, `OPENCLAW_GATEWAY_TOKEN`, `OPENROUTER_API_KEY`

## Komodo / Lovelace variables

Configure these in your Komodo server (secrets or env) so the stack can start:

| Variable | Description |
|----------|-------------|
| `OPENCLAW_GATEWAY_TOKEN` | Secret token for Control UI and API; generate once (e.g. `openssl rand -hex 24`) and keep secret. |
| `OPENROUTER_API_KEY` | API key from [OpenRouter](https://openrouter.ai/settings/keys) (format `sk-or-v1-...`). |

`EXT_PATH`, `DOMAIN`, and `TZ` are provided by the existing Lovelace/Komodo config.

## First-time setup

1. **Generate gateway token** (on your machine or server):
   ```bash
   openssl rand -hex 24
   ```
   Set this as `OPENCLAW_GATEWAY_TOKEN` in Komodo (e.g. `[[OPENCLAW_GATEWAY_TOKEN]]` placeholder).

2. **Get OpenRouter API key** from [openrouter.ai/settings/keys](https://openrouter.ai/settings/keys) and set as `OPENROUTER_API_KEY` in Komodo.

3. **Create config directories** on the server (replace `/mnt/tank` with your `EXT_PATH` if different):
   ```bash
   mkdir -p /mnt/tank/openclaw/config /mnt/tank/openclaw/workspace /mnt/tank/openclaw/auth-secrets
   chown -R 1000:1000 /mnt/tank/openclaw/config /mnt/tank/openclaw/workspace /mnt/tank/openclaw/auth-secrets
   ```
   Copy the example config and **set your Control UI origin** (required when behind Caddy, or the gateway will refuse to start):
   ```bash
   cp openclaw.json.example /mnt/tank/openclaw/config/openclaw.json
   # Edit openclaw.json: set gateway.mode=local, gateway.bind=lan, and replace
   # openclaw.YOUR_DOMAIN in gateway.controlUi.allowedOrigins (e.g. ["https://openclaw.lovelace.ts.net"])
   # Optionally set agents.defaults.model.primary. Prefer OPENROUTER_API_KEY via Komodo env.
   ```

4. **Deploy the stack** via Komodo (Sync + Deploy).

5. **Open Control UI** at `https://openclaw.${DOMAIN}` (or your Caddy domain). In **Settings → token**, paste `OPENCLAW_GATEWAY_TOKEN`.

6. **(Optional) Run onboarding** to register OpenRouter in OpenClaw (if not using `openclaw.json`):
   From the server, in the stack directory (e.g. where Komodo deploys the stack):
   ```bash
   docker compose --profile cli run --rm openclaw-cli onboard --auth-choice apiKey --token-provider openrouter --token "$OPENROUTER_API_KEY"
   ```
   Use the same env Komodo injects (or pass vars manually).

## Upgrading (image + WhatsApp plugin)

WhatsApp is an **external plugin** (`@openclaw/whatsapp`). It is not bundled in the core image, so bumping OpenClaw without updating the plugin leaves a host/plugin mismatch that can hang or crash the gateway.

1. **Backup** `${EXT_PATH}/openclaw/config` (at least `openclaw.json` and plugin/state dirs).
2. **Deploy** this stack so the gateway image is `ghcr.io/openclaw/openclaw:2026.8.1` (Komodo Sync + Deploy, or `docker compose pull && docker compose up -d`).
3. **Update the WhatsApp plugin to the same version**, then restart the gateway. From the stack directory on the server (gateway must be running):
   ```bash
   docker compose --profile cli run --rm openclaw-cli plugins update @openclaw/whatsapp@2026.8.1 --accept-capabilities
   docker compose --profile cli run --rm openclaw-cli plugins enable whatsapp --accept-capabilities
   docker restart lovelace-openclaw-openclaw-gateway-1
   ```
   If `plugins update` says the plugin is not installed yet:
   ```bash
   docker compose --profile cli run --rm openclaw-cli plugins install @openclaw/whatsapp@2026.8.1 --accept-capabilities
   ```
4. **Confirm versions** (both should report `2026.8.1`):
   ```bash
   docker compose --profile cli run --rm openclaw-cli --version
   docker compose --profile cli run --rm openclaw-cli plugins list
   docker compose --profile cli run --rm openclaw-cli channels status
   ```
5. If the gateway crash-loops after the image bump, keep the mounted state and run doctor against it, then start the gateway again:
   ```bash
   docker compose run --rm --no-deps --entrypoint node openclaw-gateway dist/index.js doctor --fix
   docker compose up -d openclaw-gateway
   ```

`--accept-capabilities` records consent for the plugin's capability surface. Skipping it is a known 2026.7 → 2026.8 failure mode: the core update succeeds, WhatsApp stays on 2026.7.x, and the gateway later fails to load the channel.

## WhatsApp

WhatsApp is installed as `@openclaw/whatsapp` (pin it to the same version as the gateway image). After the gateway is running:

1. On the server, from the openclaw stack directory:
   ```bash
   docker compose --profile cli run --rm openclaw-cli channels login
   ```
2. Follow the QR flow (e.g. WhatsApp → Linked devices → Link a device) and scan the QR shown in the terminal.
3. Credentials are stored in the mounted config; no need to repeat unless you log out.

## Telegram

Telegram is configured via **config file** (not `channels add`). Add your bot token to `openclaw.json`:

1. Create a bot via [@BotFather](https://t.me/BotFather) and copy the bot token.
2. On the server, edit `${EXT_PATH}/openclaw/config/openclaw.json` and add a `channels.telegram` block:
   ```json
   "channels": {
     "telegram": {
       "enabled": true,
       "botToken": "YOUR_BOT_TOKEN",
       "dmPolicy": "pairing"
     }
   }
   ```
   Restart the gateway: `docker restart lovelace-openclaw-openclaw-gateway-1` (or restart the stack).
3. For first DM access, approve pairing from the server:
   ```bash
   docker compose --profile cli run --rm openclaw-cli pairing list telegram
   docker compose --profile cli run --rm openclaw-cli pairing approve telegram <CODE>
   ```

## Gateway process (Docker)

The gateway container runs `gateway` in the **foreground** as the main process (under `tini`). That is the correct pattern for Docker: use `docker compose up -d` to detach the *container*, not `gateway start` (which targets systemd/launchd on the host).

## CLI usage

One-off commands (channels, onboard, dashboard, etc.) use the `cli` profile and share the gateway network (`ws://127.0.0.1:18789`). From the **stack directory on the server** (gateway must already be running):

```bash
# WhatsApp QR login
docker compose --profile cli run --rm openclaw-cli channels login

# Telegram: add botToken to openclaw.json under channels.telegram, then restart gateway

# Dashboard URL (e.g. to get Control UI link)
docker compose --profile cli run --rm openclaw-cli dashboard --no-open

# List / approve devices
docker compose --profile cli run --rm openclaw-cli devices list
docker compose --profile cli run --rm openclaw-cli devices approve <requestId>
```

For automated messages to a WhatsApp group (e.g. Friday reminders, YouTube live notifications), see [docs/OPENCLAW_WHATSAPP_GROUP_AUTOMATION.md](../../docs/OPENCLAW_WHATSAPP_GROUP_AUTOMATION.md).

## OpenRouter

- **Docs**: [OpenRouter provider](https://docs.openclaw.ai/providers/openrouter)
- **Model format**: `openrouter/provider/model-name`, e.g. `openrouter/anthropic/claude-sonnet-4-5`
- **Default model**: Set in `openclaw.json` under `agents.defaults.model.primary`, or during onboarding. Example:
  ```json5
  {
    "env": { "OPENROUTER_API_KEY": "sk-or-..." },
    "agents": {
      "defaults": {
        "model": { "primary": "openrouter/anthropic/claude-sonnet-4-5" }
      }
    }
  }
  ```
  Prefer setting `OPENROUTER_API_KEY` via environment (Komodo) and only reference the model in `openclaw.json`.

## Caddy and Homepage

- The gateway service has Caddy labels; it will be exposed at `openclaw.${DOMAIN}` with SSO (same as n8n, Grafana).
- Homepage will show an OpenClaw tile under "AI and automation" if it discovers containers by labels.

## Troubleshooting

- **Control UI `proxy_attribution_required`**: OpenClaw 2026.8.1 requires Caddy to overwrite `X-Forwarded-For` with the real client IP. This stack sets that via Compose labels. `gateway.trustedProxies` must include the address OpenClaw actually sees (Docker userland-proxy is the bridge gateway, e.g. `192.168.144.1`, not Caddy's `127.0.0.1`). Recreate the gateway container after label changes so caddy-docker-proxy reloads.
- **WhatsApp missing / "requires capability consent" / plugin version drift**: Core and `@openclaw/whatsapp` must match. Re-run the plugin update in [Upgrading](#upgrading-image--whatsapp-plugin) with `--accept-capabilities`, then restart the gateway. `plugins enable whatsapp --accept-capabilities` alone does **not** bump the plugin version.
- **502 from Caddy / gateway crash-loop**: The gateway refuses to start when bound to LAN without allowed origins. Ensure `${EXT_PATH}/openclaw/config/openclaw.json` exists and has `gateway.controlUi.allowedOrigins` set to your Control UI URL(s), e.g. `["https://openclaw.YOUR_DOMAIN"]`. Copy from `openclaw.json.example` and replace `YOUR_DOMAIN`. Then restart the stack.
- **Unauthorized / disconnected (1008)**: Open Control UI → Settings → token and paste `OPENCLAW_GATEWAY_TOKEN` again. Get a fresh link with `docker compose --profile cli run --rm openclaw-cli dashboard --no-open` if needed.
- **Gateway exits immediately / "gateway.mode=local"**: Ensure `openclaw.json` includes `"gateway": { "mode": "local", "bind": "lan", ... }` (see `openclaw.json.example`). Do not rely on `--allow-unconfigured` in production.
- **Channels not receiving**: Ensure you completed WhatsApp QR login or Telegram `channels add`; credentials live in `${EXT_PATH}/openclaw/config`.
- **Permission errors** on config/workspace: The image runs as `node` (uid 1000). Fix ownership, e.g.:
  ```bash
  sudo chown -R 1000:1000 /mnt/tank/openclaw/config /mnt/tank/openclaw/workspace
  ```
