#!/bin/bash
# scripts/generate.sh

set -e

echo "Generating SDK..."

# Generate with custom templates
docker run --rm \
  --user "$(id -u):$(id -g)" \
  -v ${PWD}/..:/local \
  openapitools/openapi-generator-cli generate --skip-validate-spec \
  -i /local/api/docs/openapi.yaml \
  -g python \
  -o /local/sdk/fluid-py/ \
  -c /local/sdk/.openapi-generator/config.yaml \
  -t /local/sdk/.openapi-generator/templates/python/

echo "Running polish script..."
python3 scripts/polish_sdk.py

echo "Formatting code..."
cd fluid-py
pip install -r requirements.txt
black .
isort .

echo "Finished!"
