#!/bin/bash

set -e

CERT_DIR="$(dirname "$0")/certs"
mkdir -p "$CERT_DIR"

openssl genrsa -out "$CERT_DIR/server.key" 4096

openssl req -new \
  -key "$CERT_DIR/server.key" \
  -out "$CERT_DIR/server.csr" \
  -subj "/C=US/ST=Test/L=Test/O=Test/CN=localhost"

openssl x509 -req -days 365 \
  -in "$CERT_DIR/server.csr" \
  -signkey "$CERT_DIR/server.key" \
  -out "$CERT_DIR/server.crt" \
  -extensions v3_req \
  -extfile <(cat <<EOF
[v3_req]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = *.localhost
IP.1 = 127.0.0.1
IP.2 = ::1
EOF
)

rm "$CERT_DIR/server.csr"
echo "Test certificates generated:"
echo "  Certificate: $CERT_DIR/server.crt"
echo "  Private Key: $CERT_DIR/server.key"
