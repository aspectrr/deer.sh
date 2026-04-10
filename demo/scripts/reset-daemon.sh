#!/usr/bin/env bash
# demo/scripts/reset-daemon.sh
#
# Wipes all sandbox artifacts, state, and SSH keys from the deer-sandbox Lima
# VM, then rebuilds and restarts deer-daemon from source.
#
# Safe to run while the daemon is running - kills it first.
#
# Usage: ./demo/scripts/reset-daemon.sh [--repo-root <path>] [--dry-run]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

LIMA_SANDBOX="deer-sandbox"
DAEMON_WORKDIR="/var/lib/deer-demo/daemon"
DAEMON_CONFIG="/var/lib/deer-demo/daemon.yaml"

DRY_RUN=0

usage() {
    cat <<'EOF'
Usage: ./demo/scripts/reset-daemon.sh [--repo-root <path>] [--dry-run]

Options:
  --repo-root <path>   Repo root on host (default: two levels above this script)
  --dry-run            Print commands without executing
  -h, --help           Show help
EOF
}

log()  { printf '[reset-daemon] %s\n' "$*"; }
fail() { printf '[reset-daemon] ERROR: %s\n' "$*" >&2; exit 1; }

run_sandbox() {
    local cmd="$1"
    if [ "$DRY_RUN" -eq 1 ]; then
        printf '+ limactl shell %s -- bash -lc %q\n' "$LIMA_SANDBOX" "$cmd"
        return
    fi
    limactl shell "$LIMA_SANDBOX" -- bash -lc "$cmd"
}

while [ "$#" -gt 0 ]; do
    case "$1" in
        --repo-root) REPO_ROOT="$2"; shift 2 ;;
        --dry-run)   DRY_RUN=1; shift ;;
        -h|--help)   usage; exit 0 ;;
        *) fail "Unknown argument: $1" ;;
    esac
done

# ---- Stop daemon ----
log "Stopping deer-daemon..."
run_sandbox "
    sudo pkill -f deer-daemon 2>/dev/null || true
    tmux kill-session -t deer-daemon 2>/dev/null || true
    sleep 1
    echo 'daemon stopped'
"

# ---- Wipe sandbox artifacts ----
log "Wiping sandbox overlays, state DB, and SSH keys..."
run_sandbox "
    sudo rm -rf ${DAEMON_WORKDIR}/overlays/*
    sudo rm -f  ${DAEMON_WORKDIR}/state.db
    sudo rm -f  /var/lib/deer-demo/ssh_ca /var/lib/deer-demo/ssh_ca.pub
    sudo rm -f  /var/lib/deer-demo/identity /var/lib/deer-demo/identity.pub
    sudo rm -rf ${DAEMON_WORKDIR}/keys/*
    echo 'artifacts wiped'
"

# ---- Regenerate SSH CA and identity keys ----
log "Generating fresh SSH CA and identity keys..."
run_sandbox "
    set -euo pipefail
    sudo mkdir -p /var/lib/deer-demo
    sudo chown \$(id -u):\$(id -g) /var/lib/deer-demo
    [ -f /var/lib/deer-demo/ssh_ca ] || ssh-keygen -t ed25519 -f /var/lib/deer-demo/ssh_ca -N '' -C 'deer-sandbox-ca' -q
    [ -f /var/lib/deer-demo/identity ] || ssh-keygen -t ed25519 -f /var/lib/deer-demo/identity -N '' -C 'deer-sandbox-identity' -q
    sudo mkdir -p ${DAEMON_WORKDIR}/overlays ${DAEMON_WORKDIR}/keys
    echo 'keys ready'
"

# ---- Rebuild deer-daemon ----
log "Building deer-daemon from source..."
run_sandbox "
    set -euo pipefail
    mkdir -p /var/lib/deer-demo/bin
    cd ${REPO_ROOT}/deer-daemon
    GOTOOLCHAIN=auto GOCACHE=/var/tmp/deer-daemon-go-build go build -o /var/lib/deer-demo/bin/deer-daemon ./cmd/deer-daemon
    echo 'build ok'
"

# ---- Start daemon ----
log "Starting deer-daemon..."
run_sandbox "
    set -euo pipefail
    tmux new-session -d -s deer-daemon 2>/dev/null || true
    tmux send-keys -t deer-daemon \
        'sudo /var/lib/deer-demo/bin/deer-daemon -config ${DAEMON_CONFIG} 2>&1 | tee ${DAEMON_WORKDIR}/daemon.log' \
        Enter
    echo 'Waiting for daemon gRPC port...'
    for i in \$(seq 1 20); do
        nc -z localhost 9091 2>/dev/null && echo 'Daemon ready on :9091' && break
        echo \"Waiting... (\${i}/20)\"
        sleep 3
    done
"

log "Reset complete. Connect with: deer connect localhost:9091 --insecure"
