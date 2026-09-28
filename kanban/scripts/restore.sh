#!/bin/sh
set -eu

: "${POSTGRES_USER:=kanban}"
: "${POSTGRES_DB:=kanban}"

if [ "$#" -ne 1 ]; then
  printf 'Usage: %s [--yes] <backup-file.sql.gz>\n' "$0" >&2
  exit 2
fi

confirmed="no"
backup=""
for argument in "$@"; do
  case "$argument" in
    --yes) confirmed="yes" ;;
    *) backup="$argument" ;;
  esac
done

if [ -z "$backup" ] || [ ! -f "$backup" ]; then
  printf 'Backup file not found: %s\n' "$backup" >&2
  exit 2
fi
if [ "$confirmed" != "yes" ]; then
  printf 'This will overwrite database %s. Re-run with --yes to continue.\n' "$POSTGRES_DB" >&2
  exit 2
fi

gunzip -c "$backup" | docker compose exec -T postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1
printf 'Restore completed from: %s\n' "$backup"
