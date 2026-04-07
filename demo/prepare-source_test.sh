#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET="${SCRIPT_DIR}/prepare-source.sh"

assert_contains() {
    local haystack="$1" needle="$2"
    if [[ "$haystack" != *"$needle"* ]]; then
        printf 'expected output to contain: %s\n' "$needle" >&2
        exit 1
    fi
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

# Create minimal pipeline files for the test
mkdir -p "$tmpdir/demo/logstash/pipeline"
touch "$tmpdir/demo/logstash/pipeline/01-input-kafka.conf"
touch "$tmpdir/demo/logstash/pipeline/02-filter-grok.conf"
touch "$tmpdir/demo/logstash/pipeline/03-filter-date.conf"
touch "$tmpdir/demo/logstash/pipeline/04-filter-mutate.conf"
touch "$tmpdir/demo/logstash/pipeline/05-filter-ruby.conf"
touch "$tmpdir/demo/logstash/pipeline/06-output-es.conf"
touch "$tmpdir/demo/logstash/logstash.yml"

assert_fails() {
    local desc="$1"; shift
    if bash "$TARGET" "$@" > /dev/null 2>&1; then
        printf 'expected failure for: %s\n' "$desc" >&2
        exit 1
    fi
}

# Happy path: amd64 dry-run
output="$(bash "$TARGET" --dry-run --repo-root "$tmpdir" --output "$tmpdir/output.qcow2" 2>&1)"
assert_contains "$output" "ubuntu-24.04-server-cloudimg-amd64.img"
assert_contains "$output" "qemu-img create"
assert_contains "$output" "qemu-system-x86_64"
echo "PASS: amd64 dry-run"

# arm64 dry-run: should produce arm64 binary and NOT double -machine
output64="$(bash "$TARGET" --dry-run --arch arm64 --repo-root "$tmpdir" --output "$tmpdir/output.qcow2" 2>&1)"
assert_contains "$output64" "ubuntu-24.04-server-cloudimg-arm64.img"
assert_contains "$output64" "qemu-system-aarch64"
assert_contains "$output64" "QEMU_EFI.fd"
# Ensure machine type appears exactly once (no double -machine)
count="$(printf '%s\n' "$output64" | grep -c -- '-machine' || true)"
if [ "$count" -ne 1 ]; then
    printf 'expected exactly 1 -machine flag, got %s\n' "$count" >&2
    exit 1
fi
echo "PASS: arm64 dry-run, no double -machine"

# Missing --repo-root
assert_fails "missing --repo-root" --dry-run --output "$tmpdir/output.qcow2"
echo "PASS: missing --repo-root fails"

# Missing --output
assert_fails "missing --output" --dry-run --repo-root "$tmpdir"
echo "PASS: missing --output fails"

# Invalid --arch
assert_fails "invalid --arch" --dry-run --arch sparc --repo-root "$tmpdir" --output "$tmpdir/output.qcow2"
echo "PASS: invalid --arch fails"

# Non-existent --repo-root
assert_fails "non-existent --repo-root" --dry-run --repo-root /does/not/exist --output "$tmpdir/output.qcow2"
echo "PASS: non-existent --repo-root fails"
