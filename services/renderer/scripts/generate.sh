#!/bin/bash
set -euo pipefail

# Generate gRPC Go code from proto files
protoc \
  --go_out=. \
  --go_opt=paths=source_relative \
  --go-grpc_out=. \
  --go-grpc_opt=paths=source_relative \
  -I proto \
  -I ../../lib/schema/proto \
  proto/renderer.proto