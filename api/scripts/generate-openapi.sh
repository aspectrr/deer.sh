#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Generate swagger docs
swag init --dir cmd/server,./internal/rest,./internal/orchestrator,./internal/store,./internal/error --generalInfo main.go --parseDependency --parseInternal

# Convert swagger to OpenAPI 3.0
docker run --rm \
  -v "$(pwd)":/workspace \
  openapitools/openapi-generator-cli generate \
  -i /workspace/docs/swagger.yaml \
  -g openapi-yaml \
  -o /workspace/docs

mv docs/openapi/openapi.yaml docs/
rm -rf docs/docs.go docs/swagger.json docs/swagger.yaml docs/README.md docs/.openapi-generator-ignore docs/.openapi-generator/ docs/openapi/
