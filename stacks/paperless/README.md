# Paperless-ngx

This stack upgrades Paperless-ngx from 2.20.15 to 3.3.4, using the existing
SQLite database and document directories. Follow the
[upstream v3 migration guide](https://github.com/paperless-ngx/paperless-ngx/blob/v3.3.4/docs/migration-v3.md)
before deployment.

## Before deployment

1. Confirm the running instance is on **2.20.15**. Older installations must
   upgrade to that version first.
2. Prepare the persistent signing key on the Docker host before deployment:
   `sudo python3 stacks/paperless/prepare-secret.py /mnt/tank` (Lovelace), or
   use `/mnt/external` for Lisa. This creates
   `${EXT_PATH}/documents/secrets/paperless-secret-key` with restricted
   permissions and preserves it on subsequent runs. Its value is never printed.
   Docker Compose mounts it as a secret and Paperless reads it through
   `PAPERLESS_SECRET_KEY_FILE`; no Komodo secret variable is needed.
   If a custom key is already configured on the running instance, securely
   place that value in the file instead of generating a replacement.
   A new key invalidates existing sessions and signed tokens. Back up this file
   with the documents and keep it stable across deployments.
3. On x86 hosts, confirm the CPU supports the x86-64-v2 baseline, including
   SSE4.2, required by the bundled NumPy. Check for `sse4_2` in `/proc/cpuinfo`;
   consult the migration guide for older CPUs. This check does not apply to ARM.
4. Confirm any mobile apps or API integrations support API version 9 or later.
   If legacy Paperless document encryption is in use, run `decrypt_documents`
   on v2 before taking the final backup and upgrading.
5. Let active imports finish and pause scanner uploads, manual changes, and
   other producers. Stop the `webserver` service before backing up so its
   SQLite database and document files are consistent.
6. Back up the entire `${EXT_PATH}/documents` directory, deployment
   configuration, and secrets. The document directory is
   `/mnt/external/documents` on Lisa and `/mnt/tank/documents` on Lovelace.
   Check the actual Backrest plan and a restore: a path ending in `/paperless`
   does not cover these files. Preserve this backup separately from later v3
   backups. Ensure enough free disk space for database migrations and rebuilding
   the search index.

## Deployment and behavior

After preparing the secret file, sync the updated Komodo resources and deploy the
Paperless stack. Startup runs database migrations and automatically rebuilds
the search index. Allow this to finish before resuming imports.

- The secret file must exist on the Docker host before deployment. Paperless
  loads its required `PAPERLESS_SECRET_KEY` from the mounted file at startup.
- `PAPERLESS_ARCHIVE_FILE_GENERATION=always` preserves the v2 default of
  generating archive PDFs. Review OCR settings in the application too, since
  database settings can override environment defaults. The custom scanner PDF
  OCR arguments remain in place and need an ingestion smoke test.
- `PAPERLESS_CONSUMER_DELETE_DUPLICATES=true` prevents duplicate imports by
  deleting matching input files from the consume directory. v3 otherwise
  allows duplicate documents by default.
- Proxy addresses use a comma-separated list. The allauth proxy count is one
  for the host-network Caddy proxy in this repository; adjust it if the actual
  request path has additional trusted proxy hops. Keep the backend bound to
  loopback, and ensure Caddy replaces or strips client-supplied identity
  headers before forwarding requests when remote-user authentication is used.
- Existing task history is cleared by the upgrade. Plain full-text queries no
  longer implicitly search notes and custom fields; use `notes.note:` and
  `custom_fields.value:` when needed. Explicit saved-view field prefixes are
  migrated automatically.

## Verification

Check startup logs for completed migrations, search indexing, and healthy workers.
Compare document counts and open/download existing original and archive files.
Test full-text search and important saved views, Caddy SSO, password login, and
API-token access. Import a scanner PDF (including one that previously needed the
OCR workaround), an IMAP email with attachments, and a duplicate. Confirm the
duplicate input is deleted without creating another document. Resume producers
only after these checks pass.

## Rollback

Stop v3 and pause producers. Restore the complete pre-upgrade document directory,
configuration, and secrets, then redeploy the 2.20.15 image. **Changing only the
image tag is not a rollback:** v2 must not use the database migrated by v3.
Preserve any documents added after the backup separately for later re-import.
