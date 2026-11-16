#!/bin/sh
set -eu
umask 077
: "${SLOPPER_GATEWAY_CONTAINER:?set the gateway container name}"
: "${SLOPPER_POSTGRES_CONTAINER:?set the InsForge PostgreSQL container name}"
: "${SLOPPER_BACKUP_DIR:=/mnt/tank/slopper/backups}"
mkdir -p "$SLOPPER_BACKUP_DIR"
find "$SLOPPER_BACKUP_DIR" -type f -name 'postgres-*.dump' -mtime +14 -delete
stamp=$(date -u +%Y%m%dT%H%M%SZ)
dump="$SLOPPER_BACKUP_DIR/postgres-$stamp.dump"
docker stop "$SLOPPER_GATEWAY_CONTAINER" >/dev/null
restart_gateway() { docker start "$SLOPPER_GATEWAY_CONTAINER" >/dev/null || true; }
trap restart_gateway EXIT HUP INT TERM
docker exec "$SLOPPER_POSTGRES_CONTAINER" sh -ec 'pg_dump -U "$POSTGRES_USER" --format=custom "$POSTGRES_DB"' >"$dump"
test -s "$dump"
sync "$dump"
printf 'Created consistent PostgreSQL dump: %s\n' "$dump"
