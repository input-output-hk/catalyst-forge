#!/usr/bin/env bash

set -e

if [ ! -f /data/private.pem ]; then
  /app/foundry-api auth init --output-dir /data
fi