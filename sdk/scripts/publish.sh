#!/bin/bash
set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}Publishing SDK...${NC}"

# Check if version is provided
if [ -z "$1" ]; then
    echo -e "${YELLOW}Usage: ./scripts/publish.sh <version> [testpypi|pypi]${NC}"
    exit 1
fi

VERSION=$1
TARGET=${2:-testpypi}

echo "Version: $VERSION"
echo "Target: $TARGET"

# Update version in config
# sed -i "s/packageVersion: .*/packageVersion: $VERSION/" .openapi-generator/config.yaml

# Generate SDK
echo -e "${BLUE}Generating SDK...${NC}"
bash scripts/generate.sh

# Build
echo -e "${BLUE}Building package...${NC}"
cd deer-py
python3 -m build

# Check
echo -e "${BLUE}Checking package...${NC}"
twine check dist/*

# Publish
if [ "$TARGET" = "pypi" ]; then
    echo -e "${YELLOW}Publishing to PyPI...${NC}"
    read -p "Are you sure? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        twine upload dist/*
    fi
else
    echo -e "${BLUE}Publishing to TestPyPI...${NC}"
    twine upload --repository testpypi dist/* --verbose
fi

echo -e "${GREEN}Published successfully!${NC}"
