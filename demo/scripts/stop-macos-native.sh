#!/usr/bin/env bash
# demo/scripts/stop-macos-native.sh
#
# Stop all deer.sh macOS native demo components:
# - deer-daemon (tmux session)
# - deer-source VM (QEMU)
# - Docker Compose services (Redpanda, Elasticsearch, Kibana)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

DEER_DIR="${HOME}/.deer"
SOURCE_VM_DIR="${DEER_DIR}/source-vm"
SOURCE_VM_PID_FILE="${SOURCE_VM_DIR}/qemu.pid"

log() {
    printf '[demo-native] %s\n' "$*"
}

stop_daemon() {
    log "Stopping deer-daemon..."
    if tmux has-session -t deer-daemon 2>/dev/null; then
        tmux kill-session -t deer-daemon 2>/dev/null || true
        log "  Killed tmux session: deer-daemon"
    fi
    pkill -f "deer-daemon.*daemon-macos.yaml" 2>/dev/null || true
    pkill -f "deer-daemon.*serve" 2>/dev/null || true
    # Also kill any tmux sessions that might be running deer-daemon
    for session in $(tmux list-sessions 2>/dev/null | grep -oE '^[0-9]+' | head -10); do
        tmux kill-session -t "$session" 2>/dev/null || true
    done
    log "  daemon stopped"
}

stop_source_vm() {
    log "Stopping deer-source VM..."
    if [ -f "$SOURCE_VM_PID_FILE" ]; then
        pid="$(cat "$SOURCE_VM_PID_FILE" 2>/dev/null || true)"
        if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null || true
            sleep 1
            if kill -0 "$pid" 2>/dev/null; then
                kill -9 "$pid" 2>/dev/null || true
            fi
            log "  Killed source VM (pid ${pid})"
        fi
        rm -f "$SOURCE_VM_PID_FILE"
    fi
    # Fallback: kill by QEMU command line
    pkill -f "qemu-system-aarch64.*deer-source" 2>/dev/null || true
    # Also kill any socket_vmnet child processes
    pkill -f "qemu-system-aarch64" 2>/dev/null || true
    log "  source VM stopped"
}

stop_docker_compose() {
    log "Stopping Docker Compose services..."
    if [ -f "${REPO_ROOT}/demo/docker-compose.yml" ]; then
        docker compose -f "${REPO_ROOT}/demo/docker-compose.yml" down 2>/dev/null || true
        log "  Docker Compose services stopped"
    else
        log "  docker-compose.yml not found, skipping"
    fi
}

cleanup_pids() {
    log "Cleaning up stale PIDs..."
    shopt -s nullglob
    for pidfile in "$DEER_DIR"/overlays/*/qemu.pid; do
        [ -f "$pidfile" ] || continue
        pid="$(cat "$pidfile" 2>/dev/null || true)"
        if [ -n "$pid" ]; then
            if kill -0 "$pid" 2>/dev/null; then
                kill "$pid" 2>/dev/null || true
                log "  Killed orphaned QEMU (pid ${pid})"
            fi
            rm -f "$pidfile"
        fi
    done
    shopt -u nullglob
}

cleanup_sandboxes() {
    log "Destroying all running sandboxes..."
    shopt -s nullglob
    for pidfile in "$DEER_DIR"/overlays/*/qemu.pid; do
        [ -f "$pidfile" ] || continue
        sandbox_dir="$(dirname "$pidfile")"
        sandbox_id="$(basename "$sandbox_dir")"
        pid="$(cat "$pidfile" 2>/dev/null || true)"
        if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null || true
            sleep 0.5
            if kill -0 "$pid" 2>/dev/null; then
                kill -9 "$pid" 2>/dev/null || true
            fi
            log "  Killed sandbox ${sandbox_id} (pid ${pid})"
        fi
    done
    shopt -u nullglob
    log "  Removing sandbox overlay directories..."
    rm -rf "${DEER_DIR}/overlays/"* 2>/dev/null || true
    log "  sandboxes cleaned up"
}

cleanup_keys() {
    log "Removing SSH keys and certificates..."
    # Preserve source_ed25519 key so source VM SSH works across restarts
    mv "${DEER_DIR}/keys/source_ed25519" "${DEER_DIR}/source_ed25519.bak" 2>/dev/null || true
    mv "${DEER_DIR}/keys/source_ed25519.pub" "${DEER_DIR}/source_ed25519.pub.bak" 2>/dev/null || true
    rm -rf "${DEER_DIR}/keys/"* 2>/dev/null || true
    mv "${DEER_DIR}/source_ed25519.bak" "${DEER_DIR}/keys/source_ed25519" 2>/dev/null || true
    mv "${DEER_DIR}/source_ed25519.pub.bak" "${DEER_DIR}/keys/source_ed25519.pub" 2>/dev/null || true
    rm -f "${DEER_DIR}/ssh_ca" "${DEER_DIR}/ssh_ca.pub" 2>/dev/null || true
    rm -f "${DEER_DIR}/identity" "${DEER_DIR}/identity.pub" 2>/dev/null || true
    rm -f "${DEER_DIR}/sandbox-host.db" 2>/dev/null || true
    rm -f "${DEER_DIR}/daemon-audit.jsonl" 2>/dev/null || true
    log "  keys and state cleaned up (source SSH key preserved)"
}

cleanup_backups() {
    log "Cleaning up old config backups..."
    shopt -s nullglob
    for backup in "${DEER_DIR}/daemon.yaml.backup."*; do
        [ -f "$backup" ] || continue
        rm -f "$backup" 2>/dev/null || true
        log "  Removed $(basename "$backup")"
    done
    shopt -u nullglob
}

main() {
    log "Stopping deer.sh macOS native demo..."
    echo ""
    stop_daemon
    stop_source_vm
    stop_docker_compose
    cleanup_sandboxes
    cleanup_keys
    cleanup_pids
    cleanup_backups
    echo ""
    log "---- All services stopped ----"
    echo ""
    log "To restart: ./demo/scripts/start-macos-native.sh"
}

main "$@"
