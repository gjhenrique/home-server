# Hermes Agent stack

This stack runs the official Hermes Agent Docker image on Lovelace. The supervised gateway handles messaging integrations and the web dashboard is available at `https://hermes.${DOMAIN}` behind the existing Caddy SSO. Hermes binds the dashboard to host loopback only; it is not published directly to the LAN or Internet.

The image is pinned to a stable release tag and registry digest. Hermes state, including provider credentials, gateway configuration, sessions, memories, and skills, persists under `${EXT_PATH}/hermes` and is mounted at `/opt/data`.

## Requirements

- Docker Engine with Compose v2 on the host
- Komodo stack environment values `EXT_PATH`, `DOMAIN`, and `TZ` (already used by the Lovelace stacks)
- A model provider API key, or a Nous Portal account
- Optional: a Telegram bot token or credentials for another supported messaging platform

## Register the stack in Komodo

The `lovelace-hermes` entry in [`komodo-lovelace.toml`](../../komodo-lovelace.toml) registers this directory as a Lovelace stack. Sync it in Komodo after this change is merged. Provider and chat credentials are entered through Hermes setup and stored in the persistent data directory rather than committed to this repository.

## First-time setup

Run setup once before starting the gateway. On Lovelace, create the data directory and make it writable by the image's default Hermes user (UID/GID 10000):

```bash
sudo mkdir -p /mnt/tank/hermes
sudo chown -R 10000:10000 /mnt/tank/hermes
sudo chmod 700 /mnt/tank/hermes
```

Run the interactive setup wizard with the same pinned image used by the stack:

```bash
docker run -it --rm \
  -v /mnt/tank/hermes:/opt/data \
  nousresearch/hermes-agent:v2026.9.14@sha256:99641e57ec762c59e54cb44aa6746b7fc68c18b3c5ddb088af54234c613d9294 \
  setup
```

The wizard writes provider credentials and configuration under `/mnt/tank/hermes`. Keep this directory private and include it in the host's protected backup. Never commit its `.env`, `config.yaml`, sessions, or other runtime state.

If you use Telegram, create the bot with [@BotFather](https://t.me/BotFather) and provide its token in setup or in the dashboard's Telegram configuration. Keep access restricted: use `TELEGRAM_ALLOWED_USERS` for a fixed user allowlist, or use Hermes' DM pairing and approve only your own account. Do not enable an allow-all setting for a bot that can run tools.

Once setup finishes, deploy `lovelace-hermes` from Komodo. Open `https://hermes.${DOMAIN}` and configure the model and messaging channels in the dashboard if you did not configure them in the wizard. Caddy's existing SSO policy protects the dashboard route.

## Security and networking

- The gateway uses host networking, matching Hermes' upstream Docker Compose guidance. The dashboard binds to host `127.0.0.1:9119`, and Caddy proxies to that address with SSO. Do not change the dashboard bind to `0.0.0.0` or publish port 9119.
- The OpenAI-compatible API server is disabled unless explicitly enabled in Hermes configuration. If you enable it later, set an API key and bind it only to a trusted interface or a protected reverse proxy. Do not expose it publicly without authentication.
- The Compose service drops `NET_RAW` and `NET_ADMIN`, enables `no-new-privileges`, and does not mount the Docker socket or host directories beyond Hermes' own data directory. Host networking lets the gateway reach host-local network services, so keep the agent's messaging allowlists and tool approvals enabled.
- Hermes denies gateway users by default. Keep platform allowlists or DM pairing enabled; avoid `GATEWAY_ALLOW_ALL_USERS=true` for an agent with terminal access.
- Back up `${EXT_PATH}/hermes` before upgrades. Do not run two gateway containers against the same Hermes data directory at the same time.

## Day-to-day operations

Run commands from the stack directory on Lovelace:

```bash
# Follow gateway and dashboard logs
docker compose logs -f hermes-gateway

# Run Hermes diagnostics
docker compose exec hermes-gateway hermes doctor

# Recreate after changing configuration or image
docker compose up -d
```

The official image supervises both the gateway and dashboard inside one container. Hermes' supervisor restarts either service independently, and both use the same persistent `/opt/data` directory.

## Upgrades and rollback

1. Back up `${EXT_PATH}/hermes`.
2. Change the versioned image tag and its registry digest in `compose.yaml` to the release you intend to deploy. Hermes publishes stable images for amd64 and arm64; the digest keeps the deployment pinned to the exact image manifest.
3. Sync and deploy the stack in Komodo, or run `docker compose pull && docker compose up -d` on the host.
4. Confirm the gateway and dashboard start and review `docker compose logs hermes-gateway`.

To roll back, stop the stack, restore the matching data backup if the newer release migrated state, change the image tag back, then deploy again. Never run an older image against data already migrated by a newer release without restoring its backup.

## Trying Hermes alongside OpenClaw

This stack uses a separate `${EXT_PATH}/hermes` directory and the `hermes.${DOMAIN}` hostname, so it can be evaluated while OpenClaw remains available. Hermes does not read OpenClaw's configuration or session store. Recreate messaging bots or integrations in Hermes and verify access controls, tool permissions, and backups before deciding whether to stop OpenClaw. Do not point both agents at the same credential or state directory.

## References

- [Hermes Docker setup](https://hermes-agent.nousresearch.com/docs/user-guide/docker)
- [Hermes security guide](https://hermes-agent.nousresearch.com/docs/user-guide/security)
- [Hermes messaging gateway](https://hermes-agent.nousresearch.com/docs/user-guide/messaging)
- [Telegram setup](https://hermes-agent.nousresearch.com/docs/user-guide/messaging/telegram/)
