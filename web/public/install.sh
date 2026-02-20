#!/bin/sh
set -e

echo "Installing Fluid..."

if ! command -v go >/dev/null 2>&1; then
    echo "Error: 'go' is not installed. Please install Go first: https://go.dev/doc/install"
    exit 1
fi

echo "Running: go install github.com/aspectrr/fluid.sh/fluid/cmd/fluid@latest"
go install github.com/aspectrr/fluid.sh/fluid/cmd/fluid@latest

echo ""
echo "Fluid installed successfully!"
echo "Ensure that $(go env GOPATH)/bin is in your PATH."
echo "Run 'fluid --help' to get started."
