#!/usr/bin/env bash
# demo/scripts/status-macos-native.sh
#
# Show status of all deer.sh macOS native demo components.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

DEER_DIR="${HOME}/.deer"
SOURCE_VM_DIR="${DEER_DIR}/source-vm"
SOURCE_VM_PID_FILE="${SOURCE_VM_DIR}/qemu.pid"
DEER_DAEMON_PID="${DEER_DIR}/daemon-macos.pid"

log() {
    printf '[status] %s\n' "$*"
}

check_daemon() {
    log "Checking deer-daemon..."
    if tmux has-session -t deer-daemon 2>/dev/null; then
        echo "  ✓ Running in tmux session: deer-daemon"
    elif pgrep -f "deer-daemon.*daemon-macos.yaml" >/dev/null 2>&1; then
        local pid=$(pgrep -f "deer-daemon.*daemon-macos.yaml")
        echo "  ✓ Running (pid ${pid})"
    else
        echo "  ✗ Not running"
        return 1
    fi

    if lsof -i :9091 >/dev/null 2>&1; then
        echo "  ✓ Listening on :9091 (gRPC)"
    else
        echo "  ✗ Not listening on :9091"
    fi
}

check_source_vm() {
    log "Checking deer-source VM..."
    if [ -f "$SOURCE_VM_PID_FILE" ]; then
        local pid=$(cat "$SOURCE_VM_PID_FILE" 2>/dev/null || true)
        if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
            echo "  ✓ Running (pid ${pid})"
        else
            echo "  ✗ PID file exists but process not running"
            return 1
        fi
    elif pgrep -f "qemu-system-aarch64.*deer-source" >/dev/null 2>&1; then
        local pid=$(pgrep -f "qemu-system-aarch64.*deer-source")
        echo "  ✓ Running (pid ${pid}) - orphaned (no PID file)"
    else
        echo "  ✗ Not running"
        return 1
    fi

    # Show IP if known
    if [ -f "$SOURCE_VM_DIR/mac" ]; then
        local mac=$(cat "$SOURCE_VM_DIR/mac")
        local ip=$(arp -an 2>/dev/null | grep -iE "[0-9a-f]+:[0-9a-f]+:[0-9a-f]+:[0-9a-f]+:[0-9a-f]+:[0-9a-f]+" | while read line; do
            arp_mac=$(echo "$line" | grep -oiE '[0-9a-f]+:[0-9a-f]+:[0-9a-f]+:[0-9a-f]+:[0-9a-f]+:[0-9a-f]+' | head -1)
            arp_mac_padded=$(echo "$arp_mac" | awk -F: '{for(i=1;i<=NF;i++) printf "%02s:", $i}' | sed 's/:$//' | tr '[:upper:]' '[:lower:]')
            mac_normalized=$(echo "$mac" | tr '[:upper:]' '[:lower:]')
            if [ "$arp_mac_padded" = "$mac_normalized" ]; then
                echo "$line" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+'
            fi
        done | head -1)

        if [ -n "$ip" ]; then
            echo "  ✓ IP: ${ip} (MAC: ${mac})"
        else
            echo "  ⚠ MAC known (${mac}) but IP not found in ARP table"
        fi
    fi
}

check_docker_compose() {
    log "Checking Docker Compose services..."
    if [ ! -f "${REPO_ROOT}/demo/docker-compose.yml" ]; then
        echo "  ✗ docker-compose.yml not found"
        return 1
    fi

    local output
    output=$(docker compose -f "${REPO_ROOT}/demo/docker-compose.yml" ps 2>/dev/null) || {
        echo "  ✗ Docker Compose not running"
        return 1
    }

    echo "$output" | grep -q "healthy" && echo "  ✓ All services healthy" || echo "  ⚠ Some services may not be ready"

    # Show port bindings
    if lsof -i :9200 >/dev/null 2>&1; then echo "  ✓ Elasticsearch: http://localhost:9200"; fi
    if lsof -i :5601 >/dev/null 2>&1; then echo "  ✓ Kibana: http://localhost:5601"; fi
    if lsof -i :9092 >/dev/null 2>&1; then echo "  ✓ Redpanda: localhost:9092"; fi
}

check_socket_vmnet() {
    log "Checking socket_vmnet..."
    if pgrep -f "socket_vmnet" >/dev/null 2>&1; then
        echo "  ✓ Running"
    else
        echo "  ✗ Not running (required for VM networking)"
        echo "    Fix: sudo brew services start socket_vmnet"
        return 1
    fi

    if [ -S "/opt/homebrew/var/run/socket_vmnet" ]; then
        echo "  ✓ Socket exists at /opt/homebrew/var/run/socket_vmnet"
    else
        echo "  ✗ Socket not found"
        return 1
    fi
}

check_logs() {
    log "Recent log activity..."
    if [ -f "${DEER_DIR}/daemon.log" ]; then
        echo "  Daemon log (last 3 lines):"
        tail -3 "${DEER_DIR}/daemon.log" 2>/dev/null | sed 's/^/    /' || echo "    (empty or unreadable)"
    fi
    if [ -f "${SOURCE_VM_DIR}/serial.log" ]; then
        echo "  Source VM serial log (last 3 lines):"
        tail -3 "${SOURCE_VM_DIR}/serial.log" 2>/dev/null | sed 's/^/    /' || echo "    (empty or unreadable)"
    fi
}

main() {
    echo ""
    log "---- deer.sh macOS native demo status ----"
    echo ""
    check_daemon || true
    echo ""
    check_source_vm || true
    echo ""
    check_docker_compose || true
    echo ""
    check_socket_vmnet || true
    echo ""
    check_logs
    echo ""
    log "---- End of status ----"
    echo ""
    log "Actions:"
    log "  Start:   ./demo/scripts/start-macos-native.sh"
    log "  Stop:    ./demo/scripts/stop-macos-native.sh"
    log "  Status:  ./demo/scripts/status-macos-native.sh"
}

main "$@"
