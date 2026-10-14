#!/bin/sh
set -eu
umask 077
runner=${GOOSE_ENTRYPOINT:-/goose-docker/entrypoint.sh}
dir=${GOOSE_MIGRATION_DIR:-/migrations}
lockfile=${SQLITE_LOCK_FILE:-/journal/journal.db.lock}
case "${GOOSE_COMMAND:-up}" in
  create)
    log=$(mktemp)
    tmp=
    trap 'rm -f "$log"; [ -z "$tmp" ] || rm -f "$tmp"' EXIT
    if "$runner" >"$log" 2>&1; then
      cat "$log"
    else
      code=$?
      cat "$log" >&2
      exit "$code"
    fi
    file=$(sed -n 's/.*Created new file: //p' "$log")
    case "$file" in
      "$dir"/[0-9]*_*.sql) ;;
      *) echo 'Cannot identify the SQL migration created by goose' >&2; exit 1 ;;
    esac
    tmp=$(mktemp "$dir/.up-only-XXXXXX")
    awk '/^-- \+goose Down/{exit} {print}' "$file" >"$tmp"
    chmod 644 "$tmp"
    mv "$tmp" "$file"
    ;;
  up|status|validate)
    for file in "$dir"/*.sql; do
      if grep -Eiq '^--[[:space:]]+\+goose[[:space:]]+Down' "$file"; then
        echo "Down is not permitted in solver migrations: $file" >&2
        exit 1
      fi
    done
    exec flock --exclusive --nonblock "$lockfile" "$runner"
    ;;
  *) echo 'Only create, up, status and validate are supported' >&2; exit 1 ;;
esac
