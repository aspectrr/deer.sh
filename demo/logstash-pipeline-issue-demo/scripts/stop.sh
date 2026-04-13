#!/usr/bin/env bash
# demo/scripts/stop.sh
#
# Stops deer-daemon and Docker Compose for the demo.
# Leaves Lima VMs running (use demo-reset to fully destroy).
#
# Usage: ./demo/scripts/stop.sh [--repo-root <path>]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

LIMA_SOURCE="deer-source"
LIMA_SANDBOX="deer-sandbox"
SSH_CONFIG="${HOME}/.ssh/config"

usage() { printf 'Usage: ./demo/scripts/stop.sh [--repo-root <path>]\n'; }

log()  { printf '[demo-stop] %s\n' "$*"; }
fail() { printf '[demo-stop] ERROR: %s\n' "$*" >&2; exit 1; }

while [ "$#" -gt 0 ]; do
    case "$1" in
        --repo-root) REPO_ROOT="$2"; shift 2 ;;
        -h|--help)   usage; exit 0 ;;
        *) fail "unknown argument: $1" ;;
    esac
done

command -v limactl >/dev/null 2>&1 || fail "limactl not found. Install with: brew install lima"

log "Stopping deer-daemon on '${LIMA_SANDBOX}'..."
limactl shell "$LIMA_SANDBOX" -- bash -lc "
    tmux kill-session -t deer-daemon 2>/dev/null || true
    sudo pkill -f deer-daemon 2>/dev/null || true
    echo 'deer-daemon stopped.'
" 2>/dev/null || true

log "Stopping Docker Compose services on Mac..."
docker compose -f "${REPO_ROOT}/demo/docker-compose.yml" down 2>/dev/null || true

log "Removing logstash-source SSH config entry..."
if grep -q '# deer-demo-source-start' "$SSH_CONFIG" 2>/dev/null; then
    TMPFILE="$(mktemp)"
    awk '/^# deer-demo-source-start/{skip=1} skip{if(/^# deer-demo-source-end/){skip=0; next}; next} {print}' "$SSH_CONFIG" > "$TMPFILE"
    mv "$TMPFILE" "$SSH_CONFIG"
    log "Removed logstash-source from ${SSH_CONFIG}"
fi

log ""
log "Demo stopped. Lima VMs '${LIMA_SOURCE}' and '${LIMA_SANDBOX}' are still running."
log "To fully remove: make demo-reset"
