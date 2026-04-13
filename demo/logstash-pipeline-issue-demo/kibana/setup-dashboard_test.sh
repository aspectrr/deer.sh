#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET="${SCRIPT_DIR}/setup-dashboard.sh"

assert_contains() {
    local haystack="$1" needle="$2"
    if [[ "$haystack" != *"$needle"* ]]; then
        printf 'expected output to contain: %s\n' "$needle" >&2
        exit 1
    fi
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

# Mock curl: /api/status returns "available"; saved_objects returns "{}" success
mkdir -p "$tmpdir/bin"
cat > "$tmpdir/bin/curl" <<'MOCK'
#!/usr/bin/env bash
# Inspect all args to decide the mock response
args="$*"
if echo "$args" | grep -q "api/status"; then
    echo '{"status":{"overall":{"level":"available"}}}'
fi
# All other calls (saved_objects POSTs) return empty success - no output needed
exit 0
MOCK
chmod +x "$tmpdir/bin/curl"

# Happy path: mock curl makes Kibana appear ready and POSTs succeed
output="$(PATH="$tmpdir/bin:$PATH" bash "$TARGET" "http://localhost:5601" 2>&1)"
assert_contains "$output" "Kibana setup complete"
assert_contains "$output" "Index pattern created"
assert_contains "$output" "Dashboard created"
echo "PASS: happy path"

# Unhappy path: when curl always fails, script should exit non-zero with a timeout error
bad_curl="$tmpdir/bin-bad"
mkdir -p "$bad_curl"
cat > "$bad_curl/curl" <<'BADMOCK'
#!/usr/bin/env bash
exit 1
BADMOCK
chmod +x "$bad_curl/curl"

if PATH="$bad_curl:$PATH" bash "$TARGET" "http://localhost:5601" > /dev/null 2>&1; then
    printf 'expected failure when Kibana unreachable\n' >&2
    exit 1
fi
echo "PASS: exits non-zero when Kibana never ready"
