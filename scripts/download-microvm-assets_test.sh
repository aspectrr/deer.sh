#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
TARGET="${REPO_ROOT}/scripts/download-microvm-assets.sh"

assert_contains() {
    local haystack="$1"
    local needle="$2"
    if [[ "$haystack" != *"$needle"* ]]; then
        printf 'expected output to contain: %s\n' "$needle" >&2
        exit 1
    fi
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

output="$("$TARGET" --dry-run --output-dir "$tmpdir/amd64-assets")"
assert_contains "$output" "https://cloud-images.ubuntu.com/releases/noble/release/ubuntu-24.04-server-cloudimg-amd64.img"
assert_contains "$output" "https://cloud-images.ubuntu.com/releases/noble/release/unpacked/ubuntu-24.04-server-cloudimg-amd64-vmlinuz-generic"
assert_contains "$output" "https://cloud-images.ubuntu.com/releases/noble/release/unpacked/ubuntu-24.04-server-cloudimg-amd64-initrd-generic"
assert_contains "$output" "DEER_E2E_BASE_IMAGE=${tmpdir}/amd64-assets/ubuntu-24.04-server-cloudimg-amd64.qcow2"
assert_contains "$output" "DEER_E2E_KERNEL=${tmpdir}/amd64-assets/ubuntu-24.04-server-cloudimg-amd64-vmlinuz-generic"
assert_contains "$output" "DEER_E2E_INITRD=${tmpdir}/amd64-assets/ubuntu-24.04-server-cloudimg-amd64-initrd-generic"

arm_output="$("$TARGET" --dry-run --arch arm64 --output-dir "$tmpdir/arm64-assets")"
assert_contains "$arm_output" "https://cloud-images.ubuntu.com/releases/noble/release/ubuntu-24.04-server-cloudimg-arm64.img"
assert_contains "$arm_output" "ln -sf ubuntu-24.04-server-cloudimg-arm64.img ${tmpdir}/arm64-assets/ubuntu-24.04-server-cloudimg-arm64.qcow2"

rerun_dir="$tmpdir/rerun-assets"
mkdir -p "$rerun_dir"
printf 'dummy-image\n' > "$rerun_dir/ubuntu-24.04-server-cloudimg-amd64.img"
printf 'dummy-kernel\n' > "$rerun_dir/ubuntu-24.04-server-cloudimg-amd64-vmlinuz-generic"
printf 'dummy-initrd\n' > "$rerun_dir/ubuntu-24.04-server-cloudimg-amd64-initrd-generic"
(
    cd "$rerun_dir"
    sha256sum \
      ubuntu-24.04-server-cloudimg-amd64.img \
      ubuntu-24.04-server-cloudimg-amd64-vmlinuz-generic \
      ubuntu-24.04-server-cloudimg-amd64-initrd-generic > SHA256SUMS
    ln -s ubuntu-24.04-server-cloudimg-amd64.img ubuntu-24.04-server-cloudimg-amd64.qcow2
)
rerun_output="$("$TARGET" --output-dir "$rerun_dir")"
assert_contains "$rerun_output" "DEER_E2E_BASE_IMAGE=${rerun_dir}/ubuntu-24.04-server-cloudimg-amd64.qcow2"
