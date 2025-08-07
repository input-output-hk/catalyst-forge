#!/bin/bash
set -e

REGISTRY_URL=${REGISTRY_URL:-localhost:5000}
MODULE_NAME=${MODULE_NAME:-example-app}
MODULE_VERSION=${MODULE_VERSION:-v1.0.0}

echo "Publishing KCL module to $REGISTRY_URL"

# Trust the CA certificate if available
if [ -f /certs/ca.crt ]; then
    echo "Installing CA certificate..."
    cp /certs/ca.crt /usr/local/share/ca-certificates/registry-ca.crt
    update-ca-certificates
fi

# Wait for registry to be available (using HTTPS)
for i in {1..30}; do
    if curl -f --cacert /certs/ca.crt https://$REGISTRY_URL/v2/ 2>/dev/null; then
        echo "Registry is ready"
        break
    fi
    echo "Waiting for registry... ($i/30)"
    sleep 2
done

cd kcl-module

echo "Testing KCL module compilation..."
kcl run . -D deployment='{"env":"test","instance":"test","name":"test","namespace":"default","values":{"image":"nginx","replicas":1,"port":80},"version":"1.0.0"}' || {
    echo "Module compilation test failed"
    exit 1
}

echo "Pushing KCL module to oci://$REGISTRY_URL/$MODULE_NAME:$MODULE_VERSION..."
kcl mod push oci://$REGISTRY_URL/$MODULE_NAME?tag=$MODULE_VERSION
echo "Successfully published $MODULE_NAME:$MODULE_VERSION to $REGISTRY_URL"