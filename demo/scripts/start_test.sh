#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET="${SCRIPT_DIR}/start.sh"

assert_contains() {
    local haystack="$1" needle="$2"
    if [[ "$haystack" != *"$needle"* ]]; then
        printf 'expected output to contain: %s\n' "$needle" >&2
        exit 1
    fi
}

assert_fails() {
    local desc="$1"; shift
    if bash "$TARGET" "$@" > /dev/null 2>&1; then
        printf 'expected failure for: %s\n' "$desc" >&2
        exit 1
    fi
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

# Happy path: dry-run should print all major orchestration steps
output="$(bash "$TARGET" --dry-run --repo-root "$tmpdir" 2>&1)"
assert_contains "$output" "limactl start"
assert_contains "$output" "docker compose up"
assert_contains "$output" "prepare-source.sh"
assert_contains "$output" "download-microvm-assets.sh"
assert_contains "$output" "deer-daemon -config"
echo "PASS: dry-run prints all orchestration steps"

# Unknown flag should fail
assert_fails "unknown flag" --dry-run --repo-root "$tmpdir" --unknown-flag
echo "PASS: unknown flag fails"
