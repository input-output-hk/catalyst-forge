#!/usr/bin/env bash

set -euo pipefail

if [ -z "$CONFIG_PATH" ]; then
    echo ">>> Error: CONFIG_PATH environment variable is not set"
    exit 1
fi

if [ ! -f "$CONFIG_PATH" ]; then
    echo ">>> Error: Config file does not exist"
    exit 1
fi

args=()
args+=("--config" "$CONFIG_PATH")

echo ">>> Starting operator with flags: ${args[*]}"

exec /app/foundry-operator "${args[@]}"
