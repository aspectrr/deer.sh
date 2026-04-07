#!/usr/bin/env bash
# scripts/demo/stop.sh
#
# Stops deer-daemon and Docker Compose inside the Lima demo VM.
# Leaves the Lima VM running (use demo-reset to fully destroy).
#
# Usage: ./scripts/demo/stop.sh [--repo-root <path>]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
LIMA_NAME="deer-demo"

usage() { printf 'Usage: ./scripts/demo/stop.sh [--repo-root <path>]\n'; }

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

log "Stopping deer-daemon tmux session..."
limactl shell "$LIMA_NAME" -- bash -lc "
    tmux kill-session -t deer-daemon 2>/dev/null || true
    sudo pkill -f deer-daemon 2>/dev/null || true
    echo 'deer-daemon stopped.'
" || true

log "Stopping Docker Compose services..."
limactl shell "$LIMA_NAME" -- bash -lc "
    cd ${REPO_ROOT}/demo
    sudo docker compose down
    echo 'Docker Compose stopped.'
" || true

log "Demo stopped. Lima VM '${LIMA_NAME}' is still running."
log "To fully remove the VM: limactl delete ${LIMA_NAME} --force"
