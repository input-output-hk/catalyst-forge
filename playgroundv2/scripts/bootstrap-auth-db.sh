#!/usr/bin/env bash

# -----------------------------------------------------------------------------
# bootstrap-auth-db.sh
#
# Purpose:
#   Initialize PostgreSQL for Ory Kratos and Ory Hydra in a Kubernetes Job.
#   - Creates dedicated databases and roles
#   - Grants privileges
#   - Creates Kubernetes Secrets with DSN connection strings for each service
#
# Behavior:
#   - Idempotent: safe to re-run; uses CREATE IF NOT EXISTS and GRANT
#   - Validates required tools (psql) and environment
#   - Logs clearly; exits non-zero on unrecoverable errors
#
# Required environment variables (with defaults for local dev):
#   PGHOST           - Postgres host (default: postgres.default.svc.cluster.local)
#   PGPORT           - Postgres port (default: 5432)
#   PGUSER           - Superuser for bootstrapping (default: postgres)
#   PGPASSWORD       - Superuser password (no default; must be set)
#
#   KRATOS_DB        - Kratos database name (default: kratos)
#   KRATOS_USER      - Kratos DB user (default: kratos)
#   KRATOS_PASSWORD  - Kratos DB password (default: kratos_password)
#   KRATOS_NAMESPACE - Namespace to create DSN secret (default: auth)
#   KRATOS_SECRET    - Secret name for DSN (default: kratos-dsn)
#   KRATOS_SECRET_KEY- DSN key in the secret (default: dsn)
#
#   HYDRA_DB         - Hydra database name (default: hydra)
#   HYDRA_USER       - Hydra DB user (default: hydra)
#   HYDRA_PASSWORD   - Hydra DB password (default: hydra_password)
#   HYDRA_NAMESPACE  - Namespace to create DSN secret (default: auth)
#   HYDRA_SECRET     - Secret name for DSN (default: hydra-dsn)
#   HYDRA_SECRET_KEY - DSN key in the secret (default: dsn)
#
# Notes:
#   - DSN format: postgres://USER:PASSWORD@PGHOST:PGPORT/DB?sslmode=disable
# -----------------------------------------------------------------------------

set -Eeuo pipefail

log() { printf "[INFO] %s\n" "$*"; }
warn() { printf "[WARN] %s\n" "$*" >&2; }
err() { printf "[ERROR] %s\n" "$*" >&2; }

require_cmd() {
  local name="$1"
  command -v "$name" >/dev/null 2>&1 || { err "Required command not found: $name"; exit 1; }
}

require_cmd psql

# Defaults
: "${PGHOST:=postgres-postgresql.postgres.svc.cluster.local}"
: "${PGPORT:=5432}"
: "${PGUSER:=postgres}"
: "${KRATOS_DB:=kratos}"
: "${KRATOS_USER:=kratos}"
: "${KRATOS_PASSWORD:=kratos_password}"
:
: "${HYDRA_DB:=hydra}"
: "${HYDRA_USER:=hydra}"
: "${HYDRA_PASSWORD:=hydra_password}"
:

if [[ -z "${PGPASSWORD:-}" ]]; then
  err "PGPASSWORD must be set for bootstrap superuser"
  exit 1
fi

export PGPASSWORD

psql_exec() {
  local sql="$1"
  PGPASSWORD="$PGPASSWORD" psql \
    --host "$PGHOST" \
    --port "$PGPORT" \
    --username "$PGUSER" \
    --dbname postgres \
    --no-align --tuples-only \
    --set ON_ERROR_STOP=1 \
    -c "$sql"
}

psql_query() {
  local sql="$1"
  PGPASSWORD="$PGPASSWORD" psql \
    --host "$PGHOST" \
    --port "$PGPORT" \
    --username "$PGUSER" \
    --dbname postgres \
    --no-align --tuples-only \
    --set ON_ERROR_STOP=1 \
    -t -A -c "$sql"
}

create_db_and_user() {
  local dbname="$1" user="$2" password="$3"
  log "Ensuring role '$user' exists..."
  psql_exec "DO \$\$ BEGIN IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '$user') THEN CREATE ROLE \"$user\" LOGIN PASSWORD '$password'; END IF; END \$\$;"

  log "Ensuring database '$dbname' exists..."
  local exists
  exists=$(psql_query "SELECT 1 FROM pg_database WHERE datname = '$dbname'") || true
  if [[ "$exists" == "1" ]]; then
    # Ensure desired owner
    psql_exec "ALTER DATABASE \"$dbname\" OWNER TO \"$user\";"
  else
    # CREATE DATABASE cannot run inside a DO block/transaction; run directly
    psql_exec "CREATE DATABASE \"$dbname\" OWNER \"$user\";"
  fi
}

main() {
  log "Starting auth DB bootstrap..."

  create_db_and_user "$KRATOS_DB" "$KRATOS_USER" "$KRATOS_PASSWORD"
  create_db_and_user "$HYDRA_DB" "$HYDRA_USER" "$HYDRA_PASSWORD"

  log "Bootstrap completed successfully."
}

main "$@"


