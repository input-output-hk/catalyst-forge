#!/usr/bin/env bash

set -eo pipefail

LOCAL=0
if [[ "$1" == "--local" ]]; then
  LOCAL=1
fi

# Read JWT token from secrets
echo '>>> Reading JWT token'
JWT_TOKEN=$(cat ./.secrets/jwt.txt)

echo ">>> Creating release"
LATEST_COMMIT=$(git -C git/source rev-parse HEAD)
UPDATED_JSON=$(jq --arg commit "$LATEST_COMMIT" '.source_commit = $commit' data/release.json)

if [[ "$LOCAL" -eq 1 ]]; then
  UPDATED_JSON=$(echo "$UPDATED_JSON" | jq '.source_repo |= sub("gitea"; "localhost")')
fi

JSON=$(echo "$UPDATED_JSON" | curl -s -X POST "http://localhost:3001/release?deploy=true" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -d @-)

DEPLOYMENT_ID=$(echo "$JSON" | jq -r '.deployments[0].id')
RELEASE_ID=$(echo "$JSON" | jq -r '.id')
echo "Release ID: $RELEASE_ID"
echo "Deployment ID: $DEPLOYMENT_ID"
