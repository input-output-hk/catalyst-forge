#!/bin/bash
set -e

# Trust the CA certificate if available
if [ -f /certs/ca.crt ]; then
    echo "Installing CA certificate..."
    cp /certs/ca.crt /usr/local/share/ca-certificates/registry-ca.crt
    update-ca-certificates
fi

exec /app/renderer