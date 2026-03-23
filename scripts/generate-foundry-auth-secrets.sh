#!/usr/bin/env bash

# generate-foundry-auth-secrets.sh
#
# Purpose:
#   Generate Foundry API authentication secrets and upload them to AWS Secrets Manager
#   as a single JSON secret at the path `shared-services/foundry/auth`.
#
# What it creates:
#   - ES256 private/public key pair (PEM) for signing/serving access tokens
#   - Invite HMAC secret (random 32 bytes, base64)
#   - Refresh HMAC secret (random 32 bytes, base64)
#
# The JSON structure stored in Secrets Manager:
#   {
#     "version": 1,
#     "algorithm": "ES256",
#     "created_at": "<ISO-8601 UTC>",
#     "jwt_private_key_pem": "...",
#     "jwt_public_key_pem": "...",
#     "invite_hmac_secret": "...",
#     "refresh_hmac_secret": "..."
#   }
#
# Usage:
#   bash scripts/generate-foundry-auth-secrets.sh \
#     --region <aws-region> \
#     [--profile <aws-profile>] \
#     [--secret-name shared-services/foundry/auth] \
#     [--force]
#
# Requirements:
#   - bash, openssl, aws-cli v2, jq
#   - AWS credentials with permissions:
#       secretsmanager:CreateSecret
#       secretsmanager:PutSecretValue
#       secretsmanager:DescribeSecret
#
# Notes:
#   - This script fails fast and will not overwrite an existing secret unless --force is provided.
#   - Temporary key files are cleaned up on exit.

set -euo pipefail
IFS=$'\n\t'

log() { printf "[%s] %s\n" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*"; }
err() { printf "[ERROR] %s\n" "$*" >&2; }

require_bin() {
  if ! command -v "$1" >/dev/null 2>&1; then
    err "Required binary '$1' not found in PATH"
    exit 1
  fi
}

require_bin openssl
require_bin aws
require_bin jq

REGION=""
PROFILE=""
SECRET_NAME="shared-services/foundry/auth"
FORCE="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --region)
      REGION="${2:-}"; shift 2;;
    --profile)
      PROFILE="${2:-}"; shift 2;;
    --secret-name)
      SECRET_NAME="${2:-}"; shift 2;;
    --force)
      FORCE="true"; shift;;
    -h|--help)
      sed -n '1,80p' "$0"; exit 0;;
    *)
      err "Unknown argument: $1"; exit 1;;
  esac
done

if [[ -z "$REGION" ]]; then
  err "--region is required"
  exit 1
fi

AWS_ARGS=("--region" "$REGION")
if [[ -n "$PROFILE" ]]; then
  AWS_ARGS+=("--profile" "$PROFILE")
fi

# Check if secret exists
SECRET_EXISTS="false"
if aws secretsmanager describe-secret "${AWS_ARGS[@]}" --secret-id "$SECRET_NAME" >/dev/null 2>&1; then
  SECRET_EXISTS="true"
fi

if [[ "$SECRET_EXISTS" == "true" && "$FORCE" != "true" ]]; then
  err "Secret '$SECRET_NAME' already exists. Use --force to create a new version."
  exit 1
fi

WORKDIR="$(mktemp -d -t foundry-auth-XXXXXX)"
cleanup() {
  rm -rf "$WORKDIR" || true
}
trap cleanup EXIT

PRIV="$WORKDIR/private.pem"
PUB="$WORKDIR/public.pem"

log "Generating ES256 (P-256) keypair..."
openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out "$PRIV" >/dev/null 2>&1
openssl ec -in "$PRIV" -pubout -out "$PUB" >/dev/null 2>&1

log "Generating HMAC secrets..."
INVITE_HS="$(openssl rand -base64 32)"
REFRESH_HS="$(openssl rand -base64 32)"

CREATED_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# Build JSON with jq to handle PEM newlines safely
SECRET_JSON="$(
  jq -n \
    --arg version "1" \
    --arg alg "ES256" \
    --arg created "$CREATED_AT" \
    --rawfile priv "$PRIV" \
    --rawfile pub "$PUB" \
    --arg invite "$INVITE_HS" \
    --arg refresh "$REFRESH_HS" \
    '{
      version: ($version|tonumber),
      algorithm: $alg,
      created_at: $created,
      jwt_private_key_pem: $priv,
      jwt_public_key_pem: $pub,
      invite_hmac_secret: $invite,
      refresh_hmac_secret: $refresh
    }'
)"

if [[ "$SECRET_EXISTS" == "true" ]]; then
  log "Uploading new secret version to '$SECRET_NAME'..."
  ARN="$(aws secretsmanager put-secret-value "${AWS_ARGS[@]}" \
    --secret-id "$SECRET_NAME" \
    --secret-string "$SECRET_JSON" \
    --query 'ARN' --output text)"
else
  log "Creating secret '$SECRET_NAME'..."
  ARN="$(aws secretsmanager create-secret "${AWS_ARGS[@]}" \
    --name "$SECRET_NAME" \
    --description "Foundry API authentication secrets (ES256 + HMACs)" \
    --secret-string "$SECRET_JSON" \
    --query 'ARN' --output text)"
fi

log "Success. Secret ARN: $ARN"
log "Keys generated at runtime were not persisted and have been removed."

