#!/usr/bin/env bash

# Function to run CLI commands
run_cli() {
    (cd ../../cli && go run cmd/main.go --api-url http://localhost:5050 -vvv "$@")
}

echo '>>> Registering user'
run_cli api register -f -e test@test.com

echo '>>> Logging in as admin'
run_cli api login --token "$(cat .auth/jwt.txt)"

echo '>>> Activating user'
KID="$(run_cli api auth users pending -j --email test@test.com | jq -r '.[0].kid')"
run_cli api auth users activate --email test@test.com
run_cli api auth keys activate --email test@test.com "${KID}"

echo '>>> Creating admin role'
run_cli api auth roles create --admin admin

echo '>>> Assigning user to admin role'
run_cli api auth users roles assign test@test.com admin
