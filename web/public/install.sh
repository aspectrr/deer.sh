#!/bin/sh
set -e

echo "Installing Fluid..."

if ! command -v go >/dev/null 2>&1; then
    echo "Error: 'go' is not installed. Please install Go first: https://go.dev/doc/install"
    exit 1
fi

echo "Running: go install github.com/aspectrr/deer.sh/deer-cli/cmd/deer@latest"
go install github.com/aspectrr/deer.sh/deer-cli/cmd/deer@latest

echo ""
echo "Fluid installed successfully!"
echo "Ensure that $(go env GOPATH)/bin is in your PATH."
echo "Run 'deer --help' to get started."
