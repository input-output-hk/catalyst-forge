#!/usr/bin/env bash

set -euo pipefail

if [[ -n "${DEBUG_SLEEP:-}" ]]; then
    echo "Sleeping for ${DEBUG_SLEEP} seconds..."
    sleep "${DEBUG_SLEEP}"
fi

# Only run database initialization if DB_INIT is set
if [[ -n "${DATABASE_INIT:-}" ]]; then
    echo "Initializing database..."

    if [[ -z "${DATABASE_ROOT_USER}" ]]; then
        echo "Error: DATABASE_ROOT_USER must be set when DB_INIT is enabled"
        exit 1
    fi

    if [[ -z "${DATABASE_ROOT_PASSWORD}" ]]; then
        echo "Error: DATABASE_ROOT_PASSWORD must be set when DB_INIT is enabled"
        exit 1
    fi

    if [[ -z "${DATABASE_ROOT_NAME}" ]]; then
        echo "Error: DATABASE_ROOT_NAME must be set when DB_INIT is enabled"
        exit 1
    fi

    export PGUSER="${DATABASE_ROOT_USER}"
    export PGPASSWORD="${DATABASE_ROOT_PASSWORD}"
    psql -h "${DATABASE_HOST}" \
        -p "${DATABASE_PORT}" \
        -d "${DATABASE_ROOT_NAME}" \
        -v dbName="${DATABASE_NAME}" \
        -v dbDescription="Foundry API Database" \
        -v dbUser="${DATABASE_USER}" \
        -v dbUserPw="${DATABASE_PASSWORD}" \
        -v dbSuperUser="${DATABASE_ROOT_USER}" \
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

if [[ -n "${SEED_ADMIN:-}" ]]; then
    echo "Seeding admin user..."
    /app/foundry-api seed --email "${SEED_ADMIN}"
fi

echo "Starting Foundry API server..."
if [[ "$#" -gt 0 ]]; then
    echo "Forwarding args to foundry-api:" "$@"
    exec /app/foundry-api "$@"
else
    exec /app/foundry-api run
fi
