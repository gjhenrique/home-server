# Paperless-ngx

This stack upgrades Paperless-ngx from 2.20.15 to 3.3.4, using the existing
SQLite database and document directories. Follow the
[upstream v3 migration guide](https://github.com/paperless-ngx/paperless-ngx/blob/v3.3.4/docs/migration-v3.md)
before deployment.

## Before deployment

1. Confirm the running instance is on **2.20.15**. Older installations must
   upgrade to that version first.
2. In Komodo, create a variable named `PAPERLESS_SECRET_KEY` and enable its
   **Secret** option before syncing the stack resources. Generate its value on
   your computer with:
   `python3 -c 'import secrets; print(secrets.token_urlsafe(64))'`.
   Paste the output as the variable value without additional quotes. Both Lisa
   and Lovelace stack definitions reference `[[PAPERLESS_SECRET_KEY]]` and pass
   it to the container. Keep the value stable across deployments and store a
   recovery copy securely; never commit it. If an existing custom key is in use,
   retain it. A new key invalidates sessions and signed tokens.
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

After creating the Komodo secret, sync the updated resources and deploy the
Paperless stack. Startup runs database migrations and automatically rebuilds
the search index. Allow this to finish before resuming imports.

- `PAPERLESS_SECRET_KEY` is required; Compose fails early when it is missing.
- `PAPERLESS_OCR_LANGUAGES` installs extra Tesseract packs in the container;
  Lovelace includes Portuguese (`por`) for documents configured to use it.
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

## AI on Lovelace with OpenRouter

Only the `lovelace-paperless` Komodo stack selects both `compose.yaml` and
`compose.ai.yaml`. Lisa continues to use the base Compose file, which does not
enable AI or require an OpenRouter key.

The override enables AI suggestions, document chat, and similar-document retrieval.
It uses OpenRouter's OpenAI-compatible endpoint at
`https://openrouter.ai/api/v1`, with `google/gemini-3-flash-preview` for suggestions/chat
and `qwen/qwen3-embedding-8b` for the document index. Models can be changed
in the override; generation models must support tool calling. Changing the
embedding model requires rebuilding the LLM index.

### Secret setup

1. In [OpenRouter API Keys](https://openrouter.ai/settings/keys), create a key
   named `paperless-lovelace`. Set a spending limit appropriate to your usage
   and ensure the OpenRouter account has credits. This is a provider-issued
   API key, not a locally generated random string.
2. In Komodo Variables, create **`PAPERLESS_OPENROUTER_API_KEY`**, enable
   **Secret**, and paste the OpenRouter key as its value without extra quotes.
   Keep the existing `PAPERLESS_SECRET_KEY` signing secret as-is: it serves a
   separate purpose. The dedicated API key lets you track and limit Paperless
   usage independently of other applications.
3. Sync the Komodo resources after this PR is merged, then redeploy only
   `lovelace-paperless`. Its resource definition passes the secret into Compose;
   the override injects it as `PAPERLESS_AI_LLM_API_KEY`. Embeddings use the same
   API key and endpoint. No embedding-specific secret is needed.

Enabling the hosted backend sends document text and metadata to OpenRouter and
its selected providers. Both generation and embedding requests incur usage
charges. With the embedding backend enabled, Paperless indexes documents on its
default daily schedule. Review OpenRouter's data-handling settings and your key's
spending limit before deployment. No remote OCR is configured; OCR remains local.

### Verify AI features

Check **Settings → Application Configuration** for existing AI overrides: values
stored in the database take precedence over environment variables. Clear or
align any conflicting values with the override, including the AI enable switch,
backend, model, endpoint, API key, and embedding settings.

Try **Suggest** on one document and check OpenRouter activity for a successful
request. Under **Settings → Documents**, select AI suggestions (or both AI and
ML); disable automatic inbox suggestions if you prefer to request them manually.

Wait for the daily index task, or build the initial index on Lovelace with:

```bash
docker exec lovelace-paperless-webserver-1 document_llmindex update
```

This sends existing document text for embedding and may generate charges. Once
indexing finishes, test the document-chat control and similar-document retrieval.
Check worker logs for provider errors: 401 usually indicates an invalid key,
402 insufficient credits or a spending limit, and 429 a rate limit. If you change
the embedding model, use `document_llmindex rebuild` instead of `update`.

To disable the repository-configured AI features, remove `compose.ai.yaml` from
Lovelace's `file_paths` and redeploy. Also clear any AI settings enabled directly
in Application Configuration, since those override the environment.

References: [Paperless AI features](https://docs.paperless-ngx.com/advanced_usage/#ai-features),
[Paperless AI settings](https://docs.paperless-ngx.com/configuration/#ai),
[OpenRouter embeddings](https://openrouter.ai/docs/api/api-reference/embeddings/create-embeddings).
