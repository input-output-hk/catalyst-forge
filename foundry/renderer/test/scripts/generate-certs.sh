#!/bin/bash
set -e

CERT_DIR="/certs"
CA_KEY="$CERT_DIR/ca.key"
CA_CERT="$CERT_DIR/ca.crt"
REGISTRY_KEY="$CERT_DIR/registry.key"
REGISTRY_CERT="$CERT_DIR/registry.crt"
REGISTRY_CSR="$CERT_DIR/registry.csr"

echo "Generating certificates for Docker registry..."

# Create certificate directory
mkdir -p $CERT_DIR

# Generate CA private key
openssl genrsa -out $CA_KEY 4096

# Generate CA certificate
openssl req -new -x509 -days 365 -key $CA_KEY -out $CA_CERT \
    -subj "/C=US/ST=State/L=City/O=CatalystForge/CN=CatalystForge CA"

# Generate registry private key
openssl genrsa -out $REGISTRY_KEY 4096

# Create certificate request for registry
openssl req -new -key $REGISTRY_KEY -out $REGISTRY_CSR \
    -subj "/C=US/ST=State/L=City/O=CatalystForge/CN=registry"

# Create extensions file for SAN
cat > $CERT_DIR/v3.ext <<EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = registry
DNS.2 = localhost
DNS.3 = registry.local
IP.1 = 127.0.0.1
EOF

# Sign the registry certificate with CA
openssl x509 -req -in $REGISTRY_CSR -CA $CA_CERT -CAkey $CA_KEY -CAcreateserial \
    -out $REGISTRY_CERT -days 365 -sha256 -extfile $CERT_DIR/v3.ext

# Clean up
rm $REGISTRY_CSR $CERT_DIR/v3.ext

echo "Certificates generated successfully!"
echo "CA Certificate: $CA_CERT"
echo "Registry Certificate: $REGISTRY_CERT"
echo "Registry Key: $REGISTRY_KEY"

# Make certificates readable
chmod 644 $CA_CERT $REGISTRY_CERT
chmod 600 $REGISTRY_KEY $CA_KEY