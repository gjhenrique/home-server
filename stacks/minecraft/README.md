# Minecraft and Playit

This Komodo stack runs a Minecraft Java server with a Playit tunnel.

## Security

- Keep Minecraft online authentication and the player whitelist enabled.
- Expose only the Minecraft Java game port. Keep RCON internal and Playit on its own Docker bridge.
- Use rootless Docker for both Periphery and Compose. Restrict access to Docker and Komodo to trusted operators.
- Keep image versions pinned and review security updates before deploying them.

## Required secret

Create `MINECRAFT_PLAYIT_SECRET_KEY` in Komodo and mark it as secret. Use the
key for the Playit agent associated with the Minecraft Java tunnel. Compose
passes the value to Playit as `SECRET_KEY`. Store the key in Komodo and rotate
it if exposed.

## Monitoring

The `mc-monitor` sidecar exports Vanilla Minecraft status metrics on host
loopback at `127.0.0.1:9225`. Prometheus scrapes this endpoint through its
`minecraft` job; the metrics are not exposed through Caddy. Set the Komodo
variable `MINECRAFT_MONITOR_SERVERS` to a comma-separated list of Java endpoints
with explicit ports, for example `192.168.1.219:25565`.

### Exported metrics

| Metric | Meaning |
| --- | --- |
| `minecraft_status_healthy` | `1` when the server responds to a Minecraft status query; otherwise `0` |
| `minecraft_status_response_time_seconds` | Time taken to query the server |
| `minecraft_status_players_online_count` | Current number of online players |
| `minecraft_status_players_max_count` | Maximum player capacity |

The metrics include `server_host`, `server_port`, `server_edition`, and
`server_version` labels. Prometheus also exposes `up{job="minecraft"}` for
exporter scrape health, which is distinct from Minecraft server health.

The exporter does not provide TPS/MSPT, CPU, memory, JVM, player-name, entity,
or chunk metrics. Those require additional instrumentation, such as a server
mod/plugin, JMX exporter, or cAdvisor.

### Grafana dashboard

[`grafana-dashboard.json`](grafana-dashboard.json) provides server health,
response time, online players, and player capacity panels. Import it in Grafana
under **Dashboards → New → Import**, upload the JSON file, and select the
Prometheus data source. The imported dashboard is available at
`/d/minecraft-server-overview/minecraft-server-overview`.
