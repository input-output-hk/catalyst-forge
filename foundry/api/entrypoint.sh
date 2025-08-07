#!/usr/bin/env bash

set -euo pipefail

if [[ -n "${DEBUG_SLEEP:-}" ]]; then
    echo "Sleeping for ${DEBUG_SLEEP} seconds..."
    sleep "${DEBUG_SLEEP}"
fi

# Only run database initialization if DB_INIT is set
if [[ -n "${DB_INIT:-}" ]]; then
    echo "Initializing database..."

    if [[ -z "${DB_SUPER_USER}" ]]; then
        echo "Error: DB_SUPER_USER must be set when DB_INIT is enabled"
        exit 1
    fi

    if [[ -z "${DB_SUPER_PASSWORD}" ]]; then
        echo "Error: DB_SUPER_PASSWORD must be set when DB_INIT is enabled"
        exit 1
    fi

    if [[ -z "${DB_ROOT_NAME}" ]]; then
        echo "Error: DB_ROOT_NAME must be set when DB_INIT is enabled"
        exit 1
    fi

    export PGUSER="${DB_SUPER_USER}"
    export PGPASSWORD="${DB_SUPER_PASSWORD}"
    psql -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -d "${DB_ROOT_NAME}" \
        -v dbName="${DB_NAME}" \
        -v dbDescription="Foundry API Database" \
        -v dbUser="${DB_USER}" \
        -v dbUserPw="${DB_PASSWORD}" \
        -v dbSuperUser="${DB_SUPER_USER}" \
        -f sql/setup.sql

    echo "Database initialization complete."
fi

# Fetch root CA from Step CA server if STEPCA_ROOT_CA is set
if [[ -n "${STEPCA_ROOT_CA:-}" ]]; then
    echo "Fetching root CA from Step CA server..."

    # Wait for Step CA server to be ready
    echo "Waiting for Step CA server to be available..."
    until curl -f -k "${STEPCA_BASE_URL}/health" >/dev/null 2>&1; do
        echo "Step CA server not ready, waiting..."
        sleep 2
    done
    echo "Step CA server is ready."

    # Create the directory if it doesn't exist
    mkdir -p "$(dirname "${STEPCA_ROOT_CA}")"

    # Fetch the root CA
    echo "Downloading root CA from ${STEPCA_BASE_URL}/roots.pem..."
    if curl -f -k "${STEPCA_BASE_URL}/roots.pem" > "${STEPCA_ROOT_CA}"; then
        echo "Root CA downloaded successfully to ${STEPCA_ROOT_CA}"
        echo "Root CA contents (first 3 lines):"
        head -3 "${STEPCA_ROOT_CA}"
    else
        echo "ERROR: Failed to download root CA from Step CA server"
        exit 1
    fi
fi

echo "Starting Foundry API server..."
exec /app/foundry-api run
