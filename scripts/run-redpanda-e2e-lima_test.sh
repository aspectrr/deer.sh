#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
TARGET="${REPO_ROOT}/scripts/run-redpanda-e2e-lima.sh"

assert_contains() {
    local haystack="$1"
    local needle="$2"
    if [[ "$haystack" != *"$needle"* ]]; then
        printf 'expected output to contain: %s\n' "$needle" >&2
        exit 1
    fi
}

repo_fixture="$(mktemp -d)"
trap 'rm -rf "$repo_fixture"' EXIT
mkdir -p "$repo_fixture/deer-daemon"

output="$("$TARGET" --dry-run --repo-root "$repo_fixture" --arch arm64 --no-download)"
assert_contains "$output" "cd ${repo_fixture}/deer-daemon"
assert_contains "$output" "DEER_E2E_BASE_IMAGE=${repo_fixture}/.cache/deer/e2e/noble-arm64/ubuntu-24.04-server-cloudimg-arm64.qcow2"
assert_contains "$output" "DEER_E2E_KERNEL=${repo_fixture}/.cache/deer/e2e/noble-arm64/ubuntu-24.04-server-cloudimg-arm64-vmlinuz-generic"
assert_contains "$output" "DEER_E2E_INITRD=${repo_fixture}/.cache/deer/e2e/noble-arm64/ubuntu-24.04-server-cloudimg-arm64-initrd-generic"
assert_contains "$output" "DEER_E2E_QEMU_BINARY=qemu-system-aarch64"
assert_contains "$output" "DEER_E2E_BRIDGE=virbr0"
assert_contains "$output" "DEER_E2E_DHCP_MODE=libvirt"
assert_contains "$output" "DEER_E2E_ROOT_DEVICE=/dev/vda1"
assert_contains "$output" "DEER_E2E_STARTUP_TIMEOUT=25m"
assert_contains "$output" "TMPDIR=/var/tmp"
assert_contains "$output" "GOTOOLCHAIN=auto"
assert_contains "$output" "GOCACHE=/var/tmp/deer-daemon-go-build"
assert_contains "$output" "go test -timeout 30m -v -run TestProviderIntegration_RedpandaStartsInGuest ./internal/provider/microvm"

keep_output="$("$TARGET" --dry-run --repo-root "$repo_fixture" --arch arm64 --no-download --workdir /var/tmp/deer-e2e-debug --keep-workdir)"
assert_contains "$keep_output" "DEER_E2E_WORKDIR=/var/tmp/deer-e2e-debug"
assert_contains "$keep_output" "DEER_E2E_KEEP_WORKDIR=1"

download_output="$("$TARGET" --dry-run --repo-root "$repo_fixture" --arch amd64)"
assert_contains "$download_output" "${repo_fixture}/scripts/download-microvm-assets.sh --arch amd64 --output-dir ${repo_fixture}/.cache/deer/e2e/noble-amd64"
