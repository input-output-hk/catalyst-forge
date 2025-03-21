#!/usr/bin/env bash

set -euo pipefail

if [[ -n "${DEBUG_SLEEP}" ]]; then
    echo "Sleeping for ${DEBUG_SLEEP} seconds..."
    sleep "${DEBUG_SLEEP}"
fi

# Only run database initialization if DB_INIT is set
if [[ -n "${DB_INIT}" ]]; then
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
    psql -h "${DB_HOST}" -p "${DB_PORT}" \
        -d "${DB_ROOT_NAME}" \
        -v dbName="${DB_NAME}" \
        -v dbDescription="Foundry API Database" \
        -v dbUser="${DB_USER}" \
        -v dbUserPw="${DB_PASSWORD}" \
        -v dbSuperUser="${DB_SUPER_USER}" \
        -f sql/setup.sql

    echo "Database initialization complete."
fi

echo "Starting Foundry API server..."
exec "/app/foundry-api"
