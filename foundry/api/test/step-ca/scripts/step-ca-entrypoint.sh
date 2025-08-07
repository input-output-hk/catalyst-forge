#!/bin/bash
set -e

# Define paths and variables
STEP_PATH="/home/step"
CA_CONFIG="$STEP_PATH/config/ca.json"
PROVISIONER_NAME="foundry.projectcatalyst.io"
PUBLIC_KEY_PATH="/foundry-keys/public.pem"

# 1. Fix permissions for the /foundry-keys directory
# This ensures the step user can write to the mounted volume
echo "Fixing permissions for /foundry-keys directory..."
if [ -d "/foundry-keys" ]; then
    # Get the step user's UID and GID
    STEP_UID=$(id -u step 2>/dev/null || echo "1000")
    STEP_GID=$(id -g step 2>/dev/null || echo "1000")

    # Change ownership of the directory to step user
    chown -R $STEP_UID:$STEP_GID /foundry-keys 2>/dev/null || true

    # Ensure the directory is writable
    chmod 755 /foundry-keys 2>/dev/null || true
    echo "Permissions fixed for /foundry-keys directory"
else
    echo "Warning: /foundry-keys directory not found"
fi

# 2. Wait for the public key to be created by the 'auth' container.
# This completely solves the startup race condition.
echo "Waiting for public key to be created at $PUBLIC_KEY_PATH..."
while [ ! -f "$PUBLIC_KEY_PATH" ]; do
  sleep 2
done
echo "Public key found. Proceeding with setup."
echo "Public key: $(cat $PUBLIC_KEY_PATH)"

# 3. Install dependencies. We revert to the amd64 binary, which works under emulation.
echo "Downloading jq binary (amd64) to /tmp..."
curl -s -L -o /tmp/jq https://github.com/jqlang/jq/releases/download/jq-1.7.1/jq-linux-amd64
chmod +x /tmp/jq
echo "jq downloaded successfully."

# 4. Initialize Step CA if not already configured
if [ ! -f "$CA_CONFIG" ]; then
    echo "Initializing Step CA..."
    step ca init --name "Foundry Test CA" \
        --provisioner admin \
        --dns step-ca,localhost \
        --address ":9000" \
        --password-file <(echo "$DOCKER_STEPCA_INIT_PASSWORD") \
        --provisioner-password-file <(echo "$DOCKER_STEPCA_INIT_PASSWORD")
else
    echo "Step CA already initialized."
fi

# 5. Add the JWT provisioner if it doesn't exist, using jq to check
echo "Checking for JWT provisioner '$PROVISIONER_NAME'..."
if ! /tmp/jq -e --arg name "$PROVISIONER_NAME" 'any(.authority.provisioners[]; .name == $name)' "$CA_CONFIG" > /dev/null; then
    echo "Provisioner not found. Adding JWT provisioner '$PROVISIONER_NAME'..."

    step ca provisioner add "$PROVISIONER_NAME" \
      --type JWK \
      --public-key "$PUBLIC_KEY_PATH" \
      --ca-config "$CA_CONFIG"

    # echo "Adding audiences field to new provisioner..."
    # /tmp/jq --arg name "$PROVISIONER_NAME" '(.authority.provisioners[] | select(.name == $name)) |= . + {"audience": ["step-ca"]}' \
    # "$CA_CONFIG" > "/tmp/ca.json" && mv "/tmp/ca.json" "$CA_CONFIG"

    echo "Provisioner setup complete."
else
    echo "JWT provisioner '$PROVISIONER_NAME' already exists."
fi

echo "Provisioner config: $(cat $CA_CONFIG)"

# 6. Copy the root CA to the /foundry-keys directory
echo "Copying root CA to /foundry-keys directory..."
# Try cp first, fallback to cat if cp fails due to permissions
if ! cp "$STEP_PATH/certs/root_ca.crt" /foundry-keys/root_ca.crt 2>/dev/null; then
    echo "cp failed, trying cat method..."
    cat "$STEP_PATH/certs/root_ca.crt" > /foundry-keys/root_ca.crt
fi

# 7. Start the Step CA server
echo "Starting Step CA server..."
exec step-ca "$CA_CONFIG" --password-file <(echo "$DOCKER_STEPCA_INIT_PASSWORD")