#!/usr/bin/env bash

set -euo pipefail

# Disable AWS CLI pager to prevent interactive prompts
export AWS_PAGER=""

# Source environment variables if .env file exists
if [[ -f .env ]]; then
    echo ">>> Loading environment variables from .env"
    source .env
fi

# Validate required environment variables
if [[ -z "${FOUNDRY_API_CERTS_SECRET:-}" ]]; then
    echo "ERROR: FOUNDRY_API_CERTS_SECRET environment variable is required"
    echo "Please set it in a .env file or export it directly"
    exit 1
fi

if [[ -z "${FOUNDRY_OPERATOR_TOKEN_SECRET:-}" ]]; then
    echo "ERROR: FOUNDRY_OPERATOR_TOKEN_SECRET environment variable is required"
    echo "Please set it in a .env file or export it directly"
    exit 1
fi

cleanup() {
    echo ">>> Cleaning up certificates"
    rm -rf certs/
    rm -rf jwt/
}
trap cleanup EXIT

echo ">>> Generating certificates"
earthly --config "" --artifact +certs/certs .

echo ">>> Uploading certificates to AWS Secrets Manager"
PUBLIC_CERT=$(cat certs/public.pem)
PRIVATE_CERT=$(cat certs/private.pem)

SECRET_JSON=$(jq -n \
  --arg public "$PUBLIC_CERT" \
  --arg private "$PRIVATE_CERT" \
  '{
    "public.pem": $public,
    "private.pem": $private
  }')

if aws secretsmanager describe-secret \
  --secret-id "$FOUNDRY_API_CERTS_SECRET" \
  --region "${AWS_REGION:-eu-central-1}" >/dev/null 2>&1; then
  echo ">>> Secret exists, updating..."
  aws secretsmanager put-secret-value \
    --secret-id "$FOUNDRY_API_CERTS_SECRET" \
    --secret-string "$SECRET_JSON" \
    --region "${AWS_REGION:-eu-central-1}"
else
  echo ">>> Secret does not exist, creating..."
  aws secretsmanager create-secret \
    --name "$FOUNDRY_API_CERTS_SECRET" \
    --secret-string "$SECRET_JSON" \
    --region "${AWS_REGION:-eu-central-1}"
fi

echo ">>> Generating operator token"
earthly --config "" --artifact +jwt/jwt .

echo ">>> Uploading operator token to AWS Secrets Manager"
JWT_TOKEN=$(cat jwt/token.txt)
TOKEN_JSON=$(jq -n \
  --arg token "$JWT_TOKEN" \
  '{
    "token": $token
  }')

if aws secretsmanager describe-secret \
  --secret-id "$FOUNDRY_OPERATOR_TOKEN_SECRET" \
  --region "${AWS_REGION:-eu-central-1}" >/dev/null 2>&1; then
  echo ">>> Operator token secret exists, updating..."
  aws secretsmanager put-secret-value \
    --secret-id "$FOUNDRY_OPERATOR_TOKEN_SECRET" \
    --secret-string "$TOKEN_JSON" \
    --region "${AWS_REGION:-eu-central-1}"
else
  echo ">>> Operator token secret does not exist, creating..."
  aws secretsmanager create-secret \
    --name "$FOUNDRY_OPERATOR_TOKEN_SECRET" \
    --secret-string "$TOKEN_JSON" \
    --region "${AWS_REGION:-eu-central-1}"
fi

echo ">>> Certificates and operator token uploaded successfully to AWS Secrets Manager"