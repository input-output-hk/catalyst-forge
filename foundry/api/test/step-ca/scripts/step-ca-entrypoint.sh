#!/bin/bash
set -e

# Define paths and variables
STEP_PATH="/home/step"
CA_CONFIG="$STEP_PATH/config/ca.json"
PROVISIONER_NAME="foundry.projectcatalyst.io"
PUBLIC_KEY_PATH="/foundry-keys/public.pem"

# 1. Wait for the public key to be created by the 'auth' container.
# This completely solves the startup race condition.
echo "Waiting for public key to be created at $PUBLIC_KEY_PATH..."
while [ ! -f "$PUBLIC_KEY_PATH" ]; do
  sleep 2
done
echo "Public key found. Proceeding with setup."
echo "Public key: $(cat $PUBLIC_KEY_PATH)"

# 2. Install dependencies. We revert to the amd64 binary, which works under emulation.
echo "Downloading jq binary (amd64) to /tmp..."
curl -s -L -o /tmp/jq https://github.com/jqlang/jq/releases/download/jq-1.7.1/jq-linux-amd64
chmod +x /tmp/jq
echo "jq downloaded successfully."

# 3. Initialize Step CA if not already configured
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

# 4. Add the JWT provisioner if it doesn't exist, using jq to check
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

# 5. Copy the root CA to the /foundry-keys directory
echo "Copying root CA to /foundry-keys directory..."
cp "$STEP_PATH/certs/root_ca.crt" /data/root_ca.crt

# 6. Start the Step CA server
echo "Starting Step CA server..."
exec step-ca "$CA_CONFIG" --password-file <(echo "$DOCKER_STEPCA_INIT_PASSWORD")