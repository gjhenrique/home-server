# Slopper implementation notes

- Keep InsForge, PostgreSQL, gateway, NGINX, and admin listeners private; preserve the loopback host bindings and internal Docker networks. InsForge object storage uses the configured external S3 endpoint; do not add a public file-serving path.
- Never add credentials to browser code, static files, committed `.env` files, or Docker build contexts. The root `.dockerignore` intentionally limits the gateway context.
- Keep the shared Go validation in `internal/slopper`; use only registered tables and buckets from `resources/manifest.json`.
- Add resource schema changes as ordered administrator migrations. Static deployment commands must never apply migrations or alter object/database data.
- Keep the API same-origin, session verified, CSRF protected for writes, and private ingress based on `remote_ip`.
- The gateway may hold only the InsForge anonymous API key, never the administrative API key or root credentials.
- Do not deploy to Lovelace, push changes, or change unrelated stacks as part of Slopper implementation work.
- Run `go test -race ./...`, `go build ./cmd/slopper`, the demo `npm run check` and `npm run build`, both Compose config checks, NGINX syntax validation, and Caddy adaptation when changing the relevant paths.
