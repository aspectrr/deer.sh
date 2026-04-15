#!/usr/bin/env bash
# demo/es-cluster-red-demo/stop-macos-native.sh
#
# Stop all ES cluster yellow demo components:
# - deer-daemon (tmux session)
# - All ES node VMs (QEMU)
# - Clean up overlays, keys, state

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

DEER_DIR="${HOME}/.deer"
CLUSTER_DIR="${DEER_DIR}/es-cluster"
NUM_NODES=5

log() {
    printf '[es-cluster] %s\n' "$*"
}

stop_daemon() {
    log "Stopping deer-daemon..."
    if tmux has-session -t deer-daemon 2>/dev/null; then
        tmux kill-session -t deer-daemon 2>/dev/null || true
        log "  Killed tmux session: deer-daemon"
    fi
    pkill -f "deer-daemon.*daemon-es-cluster.yaml" 2>/dev/null || true
    pkill -f "deer-daemon.*serve" 2>/dev/null || true
    log "  daemon stopped"
}

stop_cluster_vms() {
    log "Stopping ES cluster VMs..."
    for i in $(seq 1 "$NUM_NODES"); do
        node_dir="${CLUSTER_DIR}/node-${i}"
        pid_file="${node_dir}/qemu.pid"

        if [ -f "$pid_file" ]; then
            pid="$(cat "$pid_file" 2>/dev/null || true)"
            if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
                log "  Killing node-${i} (pid ${pid})..."
                kill "$pid" 2>/dev/null || true
                sleep 1
                if kill -0 "$pid" 2>/dev/null; then
                    kill -9 "$pid" 2>/dev/null || true
                fi
            fi
            rm -f "$pid_file"
        fi
    done

    # Fallback: kill any remaining QEMU processes
    pkill -f "qemu-system-aarch64" 2>/dev/null || true
    log "  all VMs stopped"
}

cleanup_overlays() {
    log "Removing cluster overlays..."
    if [ -d "$CLUSTER_DIR" ]; then
        for i in $(seq 1 "$NUM_NODES"); do
            node_dir="${CLUSTER_DIR}/node-${i}"
            rm -f "${node_dir}/overlay.qcow2" 2>/dev/null || true
            rm -f "${node_dir}/cloud-init.iso" 2>/dev/null || true
            rm -f "${node_dir}/ip" 2>/dev/null || true
        done
    fi
    rm -rf "${DEER_DIR}/overlays/"* 2>/dev/null || true
    log "  overlays cleaned up"
}

cleanup_keys() {
    log "Removing SSH keys and certificates..."
    mv "${DEER_DIR}/keys/source_ed25519" "${DEER_DIR}/source_ed25519.bak" 2>/dev/null || true
    mv "${DEER_DIR}/keys/source_ed25519.pub" "${DEER_DIR}/source_ed25519.pub.bak" 2>/dev/null || true
    rm -rf "${DEER_DIR}/keys/"* 2>/dev/null || true
    mv "${DEER_DIR}/source_ed25519.bak" "${DEER_DIR}/keys/source_ed25519" 2>/dev/null || true
    mv "${DEER_DIR}/source_ed25519.pub.bak" "${DEER_DIR}/keys/source_ed25519.pub" 2>/dev/null || true
    rm -f "${DEER_DIR}/ssh_ca" "${DEER_DIR}/ssh_ca.pub" 2>/dev/null || true
    rm -f "${DEER_DIR}/identity" "${DEER_DIR}/identity.pub" 2>/dev/null || true
    rm -f "${DEER_DIR}/sandbox-host.db" 2>/dev/null || true
    rm -f "${DEER_DIR}/daemon-es-cluster.yaml" 2>/dev/null || true
    rm -f "${DEER_DIR}/daemon-audit.jsonl" 2>/dev/null || true
    log "  keys and state cleaned up (source SSH key preserved)"
}

cleanup_pids() {
    log "Cleaning up stale PIDs..."
    shopt -s nullglob
    for pidfile in "$DEER_DIR"/overlays/*/qemu.pid; do
        [ -f "$pidfile" ] || continue
        pid="$(cat "$pidfile" 2>/dev/null || true)"
        if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null || true
            log "  Killed orphaned QEMU (pid ${pid})"
        fi
        rm -f "$pidfile"
    done
    shopt -u nullglob
}

cleanup_config() {
    log "Removing deer CLI config..."
    rm -f "${HOME}/.config/deer/config.yaml" 2>/dev/null || true
    log "  config removed"
}

cleanup_ssh_config() {
    log "Removing ES cluster SSH config entries..."
    SSH_CONFIG="${HOME}/.ssh/config"
    if grep -q '# deer-es-cluster-start' "$SSH_CONFIG" 2>/dev/null; then
        TMPFILE="$(mktemp)"
        awk '/^# deer-es-cluster-start/{skip=1} skip{if(/^# deer-es-cluster-end/){skip=0; next}; next} {print}' "$SSH_CONFIG" > "$TMPFILE"
        mv "$TMPFILE" "$SSH_CONFIG"
        log "  Removed ES cluster hosts from ${SSH_CONFIG}"
    fi
}

main() {
    echo ""
    log "Stopping ES cluster yellow demo..."
    echo ""
    stop_daemon
    stop_cluster_vms
    cleanup_overlays
    cleanup_keys
    cleanup_pids
    cleanup_config
    cleanup_ssh_config
    echo ""
    log "---- All services stopped ----"
    echo ""
    log "To restart: ./demo/es-cluster-red-demo/start-macos-native.sh"
}

main "$@"
