#!/usr/bin/env bash

# -----------------------------------------------------------------------------
# seed-secrets.sh
#
# Seeds LocalStack Secrets Manager (running in-cluster) with:
# - db/foundry: {host, port, username, password}
# - db/root_account: {username, password}
#
# The script execs into the LocalStack pod to run `awslocal` so we don't need
# port-forwarding. It is idempotent: updates value if the secret already exists.
#
# Requirements:
# - kubectl, jq, awk, sed
# - LocalStack installed via Helm and running in namespace `localstack`
# - ESO deployed and ClusterSecretStore configured to use LocalStack
#
# Usage:
#   seed-secrets.sh [--pg-namespace NS] [--pg-release NAME] \
#                   [--pg-host HOST] [--pg-port PORT] \
#                   [--db-user USER] [--db-pass PASS] \
#                   [--root-user USER] [--root-pass PASS]
#
# Defaults:
#   --pg-namespace default
#   --pg-release   postgres            (Bitnami chart: service = "${release}-postgresql")
#   --pg-port      5432
#   --db-user      foundry
#   --db-pass      (random 24B base64 if not provided)
#   --root-user    postgres            (matches Terraform var pg_username)
#   --root-pass    postgres            (matches Terraform var pg_password)
# -----------------------------------------------------------------------------

set -Eeuo pipefail

log() { printf "[INFO] %s\n" "$*"; }
warn() { printf "[WARN] %s\n" "$*" >&2; }
err() { printf "[ERROR] %s\n" "$*" >&2; }

need() { command -v "$1" >/dev/null 2>&1 || { err "Missing dependency: $1"; exit 1; }; }
need kubectl
need jq

PG_NAMESPACE="default"
PG_RELEASE="postgres"
PG_PORT="5432"
DB_USER="foundry"
DB_PASS=""
ROOT_USER="postgres"
ROOT_PASS="postgres"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --pg-namespace) PG_NAMESPACE="$2"; shift 2 ;;
    --pg-release) PG_RELEASE="$2"; shift 2 ;;
    --pg-host) PG_HOST_OVERRIDE="$2"; shift 2 ;;
    --pg-port) PG_PORT="$2"; shift 2 ;;
    --db-user) DB_USER="$2"; shift 2 ;;
    --db-pass) DB_PASS="$2"; shift 2 ;;
    --root-user) ROOT_USER="$2"; shift 2 ;;
    --root-pass) ROOT_PASS="$2"; shift 2 ;;
    -h|--help)
      sed -n '1,80p' "$0" | sed 's/^# \{0,1\}//' | sed '/^-\{5,\}$/q'
      exit 0 ;;
    *) err "Unknown argument: $1"; exit 1 ;;
  esac
done

# Derive Postgres service DNS
PG_SVC="${PG_RELEASE}-postgresql"
if [[ -n "${PG_HOST_OVERRIDE:-}" ]]; then
  PG_HOST="$PG_HOST_OVERRIDE"
else
  PG_HOST="${PG_SVC}.${PG_NAMESPACE}.svc.cluster.local"
fi

# Generate random password for app DB user if not provided
if [[ -z "$DB_PASS" ]]; then
  if command -v openssl >/dev/null 2>&1; then
    DB_PASS=$(openssl rand -base64 24 | tr -d '\n' | sed 's#/##g')
  else
    DB_PASS=$(python3 - <<'PY'
import secrets, string
alphabet = string.ascii_letters + string.digits
print(''.join(secrets.choice(alphabet) for _ in range(32)))
PY
)
  fi
fi

# Find LocalStack pod
LS_POD=$(kubectl -n localstack get pods -o jsonpath='{.items[?(@.status.phase=="Running")].metadata.name}' | tr ' ' '\n' | grep '^localstack-' | head -n1 || true)
if [[ -z "$LS_POD" ]]; then
  err "Could not find running LocalStack pod in namespace 'localstack'"
  exit 1
fi
log "Using LocalStack pod: $LS_POD"

aws_in_pod() {
  local args=("$@")
  kubectl -n localstack exec "$LS_POD" -- awslocal "${args[@]}"
}

ensure_secret() {
  local name="$1" json="$2"
  if aws_in_pod secretsmanager describe-secret --secret-id "$name" >/dev/null 2>&1; then
    log "Updating secret value: $name"
    aws_in_pod secretsmanager put-secret-value --secret-id "$name" --secret-string "$json" >/dev/null
  else
    log "Creating secret: $name"
    aws_in_pod secretsmanager create-secret --name "$name" --secret-string "$json" >/dev/null
  fi
}

# Prepare payloads
DB_FOUNDATION_JSON=$(jq -cn --arg host "$PG_HOST" --arg port "$PG_PORT" --arg user "$DB_USER" --arg pass "$DB_PASS" '{host:$host, port:$port, username:$user, password:$pass}')
DB_ROOT_JSON=$(jq -cn --arg user "$ROOT_USER" --arg pass "$ROOT_PASS" '{username:$user, password:$pass}')

log "Seeding shared-services/db/foundry (app user)"
ensure_secret "shared-services/db/foundry" "$DB_FOUNDATION_JSON"

log "Seeding shared-services/db/root_account (superuser)"
ensure_secret "shared-services/db/root_account" "$DB_ROOT_JSON"

cat <<EON

Seeded secrets in LocalStack Secrets Manager:
  - shared-services/db/foundry         (user=$DB_USER, host=$PG_HOST, port=$PG_PORT)
  - shared-services/db/root_account    (user=$ROOT_USER)

Note: Store the generated app DB password if you need it:
  DB_USER: $DB_USER
  DB_PASS: $DB_PASS
EON


