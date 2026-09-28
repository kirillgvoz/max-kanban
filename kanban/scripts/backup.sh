#!/bin/sh
set -eu

: "${POSTGRES_USER:=kanban}"
: "${POSTGRES_DB:=kanban}"
: "${BACKUP_DIR:=./backups}"

mkdir -p "$BACKUP_DIR"
timestamp=$(date -u +%Y%m%dT%H%M%SZ)
backup="$BACKUP_DIR/kanban-$timestamp.sql.gz"

docker compose exec -T postgres pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" | gzip > "$backup"
printf 'Backup completed: %s\n' "$backup"
