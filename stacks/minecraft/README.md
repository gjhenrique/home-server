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
