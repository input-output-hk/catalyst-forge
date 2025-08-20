#!/usr/bin/env sh
set -euo pipefail

: "${PGHOST:=postgres}"
: "${PGPORT:=5432}"
: "${PGUSER:=postgres}"
: "${PGPASSWORD:=postgres}"

echo "Waiting for Postgres at ${PGHOST}:${PGPORT}..."
until pg_isready -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" >/dev/null 2>&1; do
  sleep 1
done

psql_url() { printf "postgresql://%s:%s@%s:%s/%s" "$PGUSER" "$PGPASSWORD" "$PGHOST" "$PGPORT" "$1"; }

create_db() {
  db="$1"
  echo "Ensuring database '$db' exists..."
  if ! psql "$(psql_url postgres)" -v ON_ERROR_STOP=1 -tAc "SELECT 1 FROM pg_database WHERE datname='${db}'" | grep -q 1; then
    psql "$(psql_url postgres)" -v ON_ERROR_STOP=1 -c "CREATE DATABASE ${db}"
    echo "Created database '${db}'."
  else
    echo "Database '${db}' already exists."
  fi
}

create_db kratos
create_db hydra
create_db keto

echo "All done."
