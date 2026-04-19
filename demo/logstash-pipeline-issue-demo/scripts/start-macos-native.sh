#!/usr/bin/env bash
# demo/scripts/start-macos-native.sh
#
# Fully native macOS launcher. No Lima. No virtualization layers.
# Both deer-daemon sandboxes AND the Logstash source VM run as native
# QEMU arm64 VMs with HVF acceleration via Hypervisor.framework.
#
# Architecture:
#   Mac: Docker Compose - Redpanda + Elasticsearch + Kibana
#   Mac: deer-daemon (native binary) + QEMU sandboxes (HVF)
#   Mac: deer-source VM (native QEMU arm64, HVF) - Logstash broken pipeline
#
# Requirements:
#   brew install qemu socket_vmnet docker tmux
#   sudo brew services start socket_vmnet
#
# Usage: ./demo/logstash-pipeline-issue-demo/scripts/start-macos-native.sh [--repo-root <path>] [--dry-run]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

DEER_DIR="${HOME}/.deer"
DEER_ASSETS_DIR="${DEER_DIR}/assets"
DEER_OVERLAYS_DIR="${DEER_DIR}/overlays"
DEER_IMAGES_DIR="${DEER_DIR}/images"
DEER_KEYS_DIR="${DEER_DIR}/keys"
DEER_DAEMON_BIN="${DEER_DIR}/bin/deer-daemon"
DEER_DAEMON_CONFIG="${DEER_DIR}/daemon-macos.yaml"
DEER_DAEMON_LOG="${DEER_DIR}/daemon.log"

SOURCE_VM_DIR="${DEER_DIR}/source-vm"
SOURCE_VM_OVERLAY="${SOURCE_VM_DIR}/overlay.qcow2"
SOURCE_VM_CLOUDINIT="${SOURCE_VM_DIR}/cloud-init.iso"
SOURCE_VM_MAC_FILE="${SOURCE_VM_DIR}/mac"
SOURCE_VM_PID_FILE="${SOURCE_VM_DIR}/qemu.pid"
SOURCE_VM_SERIAL_LOG="${SOURCE_VM_DIR}/serial.log"
SOURCE_VM_CPUS=2
SOURCE_VM_MEM_MB=2048

DEER_CLI_CONFIG_DIR="${HOME}/.config/deer"
DEER_CLI_CONFIG="${DEER_CLI_CONFIG_DIR}/config.yaml"
SOURCE_KEY="${DEER_KEYS_DIR}/source_ed25519"
SOURCE_PUB_KEY="${DEER_KEYS_DIR}/source_ed25519.pub"

SSH_CA_KEY="${DEER_DIR}/ssh_ca"
SSH_CA_PUB="${DEER_DIR}/ssh_ca.pub"
SSH_IDENTITY="${DEER_DIR}/identity"

SOCKET_VMNET_CLIENT="/opt/homebrew/opt/socket_vmnet/bin/socket_vmnet_client"
SOCKET_VMNET_PATH="/opt/homebrew/var/run/socket_vmnet"
QEMU_BIN="/opt/homebrew/bin/qemu-system-aarch64"

DAEMON_GRPC_ADDR="localhost:9091"
SOURCE_VM_SSH_USER="deer"

UBUNTU_VERSION="24.04"
UBUNTU_CODENAME="noble"
UBUNTU_ARCH="arm64"

DRY_RUN=0

# ---- Helpers ----

usage() {
    cat <<'EOF'
Usage: ./demo/logstash-pipeline-issue-demo/scripts/start-macos-native.sh [--repo-root <path>] [--dry-run]

Options:
  --repo-root <path>   Repo root on host (default: two levels above this script)
  --dry-run            Print commands without executing
  -h, --help           Show help
EOF
}

log()  { printf '[demo-native] %s\n' "$*"; }
fail() { printf '[demo-native] ERROR: %s\n' "$*" >&2; exit 1; }

run() {
    if [ "$DRY_RUN" -eq 1 ]; then printf '+'; printf ' %q' "$@"; printf '\n'; return; fi
    "$@"
}

generate_mac() {
    printf '52:54:00:%02x:%02x:%02x' $((RANDOM % 256)) $((RANDOM % 256)) $((RANDOM % 256))
}

# Discover IP for a MAC via ARP. Prints IP on success, returns 1 on timeout.
# macOS ARP collapses leading zeros: 52:54:00:4f:fb:3c -> 52:54:0:4f:fb:3c
# We normalize both to colon-free lowercase and compare.
discover_ip_by_mac() {
    local mac="$1"
    local timeout="${2:-120}"
    local ip=""

    # Normalize target MAC to lowercase without colons for comparison
    local mac_normalized=$(echo "$mac" | tr '[:upper:]' '[:lower:]' | tr -d ':')

    log "Discovering IP for MAC ${mac} (up to ${timeout}s)..." >&2
    for _ in $(seq 1 "$timeout"); do
        # Extract ALL IPs from ARP where normalized MAC matches target
        ips=$(arp -an 2>/dev/null | awk -v target="$mac_normalized" '
        {
            # Extract MAC and IP from line like: "? (192.168.105.61) at 52:54:0:4f:fb:3c on bridge100"
            ip = $2
            mac = $4

            # Strip parentheses from IP: (192.168.105.61) -> 192.168.105.61
            gsub(/[()]/, "", ip)

            # Normalize MAC: split by colons, pad each octet to 2 chars, join
            split(mac, octets, ":")
            normalized = ""
            for (i = 1; i <= length(octets); i++) {
                # Pad each octet to 2 hex digits (e.g., "0" -> "00", "4f" -> "4f")
                if (length(octets[i]) == 1) normalized = normalized "0" octets[i]
                else normalized = normalized tolower(octets[i])
            }

            # Compare normalized MAC with target (target already has no colons)
            if (normalized == target || tolower(normalized) == target) {
                print ip
            }
        }')

        # Try each IP to find the one that actually responds
        if [ -n "$ips" ]; then
            for test_ip in $ips; do
                # Verify IP is reachable with a quick ping
                if ping -c 1 -W 1 "$test_ip" >/dev/null 2>&1; then
                    echo "$test_ip"
                    return 0
                fi
            done
        fi
        sleep 1
    done
    return 1
}

# Wait for SSH to accept connections on host:22.
wait_for_ssh() {
    local host="$1"
    local user="$2"
    local key="$3"
    local timeout="${4:-180}"
    log "Waiting for SSH on ${host} (up to ${timeout}s)..."
    for _ in $(seq 1 "$timeout"); do
        ssh -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes -o IdentitiesOnly=yes \
            -i "$key" "${user}@${host}" true 2>/dev/null && return 0
        sleep 3
    done
    return 1
}

# Wait for cloud-init to complete inside the VM.
wait_for_cloud_init() {
    local host="$1"
    local user="$2"
    local key="$3"
    local timeout="${4:-900}"
    log "Waiting for cloud-init on ${host} (up to ${timeout}s)..."
    for _ in $(seq 1 "$timeout"); do
        if ssh -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes -o IdentitiesOnly=yes \
            -i "$key" "${user}@${host}" \
            'test -f /var/lib/cloud/instance/boot-finished' 2>/dev/null; then
            return 0
        fi
        sleep 5
    done
    return 1
}

# Run a command on the source VM via SSH.
ssh_source() {
    local host="$1"
    local cmd="$2"
    ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 -o BatchMode=yes -o IdentitiesOnly=yes \
        -i "$SOURCE_KEY" "${SOURCE_VM_SSH_USER}@${host}" "$cmd"
}

# SCP a local file to the source VM.
scp_to_source() {
    local host="$1"
    local src="$2"
    local dst="$3"
    scp -o StrictHostKeyChecking=no -o ConnectTimeout=10 -o BatchMode=yes -o IdentitiesOnly=yes \
        -i "$SOURCE_KEY" "$src" "${SOURCE_VM_SSH_USER}@${host}:${dst}"
}

# Create a cloud-init NoCloud ISO using mkisofs (or hdiutil as fallback).
create_cloud_init_iso() {
    local iso_path="$1"
    local user_data="$2"
    local meta_data="$3"
    local tmp_dir
    tmp_dir="$(mktemp -d)"
    printf '%s' "$user_data" > "${tmp_dir}/user-data"
    printf '%s' "$meta_data" > "${tmp_dir}/meta-data"

    # Add simple network-config to avoid wildcard hostname issues
    printf 'version: 2\nethernets:\n  eth0:\n    dhcp4: true\n' > "${tmp_dir}/network-config"

    # Try mkisofs first (more reliable on macOS), fall back to hdiutil
    if command -v mkisofs >/dev/null 2>&1; then
        mkisofs -output "${iso_path}" -volid "cidata" -joliet -rock "${tmp_dir}" >/dev/null 2>&1 || {
            rm -rf "${tmp_dir}"
            return 1
        }
    else
        hdiutil makehybrid -o "${iso_path}" -hfs -iso -joliet \
            -default-volume-name cidata "${tmp_dir}" >/dev/null 2>&1 || {
            rm -rf "${tmp_dir}"
            return 1
        }
    fi
    rm -rf "${tmp_dir}"
}

# Boot a QEMU arm64 VM via socket_vmnet (no -daemonize, background process).
# HVF context must stay in the same process - fork via -daemonize is not used.
boot_qemu_vm() {
    local overlay="$1"
    local cloudinit_iso="$2"
    local mac="$3"
    local mem_mb="$4"
    local cpus="$5"
    local serial_log="$6"
    local pid_file="$7"

    "${SOCKET_VMNET_CLIENT}" "${SOCKET_VMNET_PATH}" \
        "${QEMU_BIN}" \
        -M virt \
        -accel hvf -cpu max \
        -m "${mem_mb}" \
        -smp "${cpus}" \
        -kernel "${UBUNTU_KERNEL}" \
        -initrd "${UBUNTU_INITRD}" \
        -append "console=ttyAMA0 root=/dev/vda1 rw rootwait quiet" \
        -drive "id=root,file=${overlay},format=qcow2,if=none" \
        -device virtio-blk-device,drive=root \
        -netdev "socket,id=net0,fd=3" \
        -device "virtio-net-device,netdev=net0,mac=${mac}" \
        -drive "id=cidata,file=${cloudinit_iso},format=raw,readonly=on,if=none" \
        -device "virtio-scsi-device,id=scsi0" \
        -device "scsi-cd,drive=cidata,bus=scsi0.0" \
        -serial "file:${serial_log}" \
        -nographic -nodefaults \
        -rtc clock=vm,base=localtime,driftfix=slew \
        -no-reboot \
        > /dev/null 2>&1 &

    echo $! > "${pid_file}"
}

# ---- Argument parsing ----

while [ "$#" -gt 0 ]; do
    case "$1" in
        --repo-root) REPO_ROOT="$2"; shift 2 ;;
        --dry-run)   DRY_RUN=1; shift ;;
        -h|--help)   usage; exit 0 ;;
        *) fail "unknown argument: $1" ;;
    esac
done

# ---- Prerequisites ----

log "Checking prerequisites..."
if [ "$DRY_RUN" -eq 0 ]; then
    [ "$(uname -s)" = "Darwin" ] || fail "macOS only"
    [ "$(uname -m)" = "arm64" ]  || fail "Apple Silicon (arm64) required for HVF"
    command -v "${QEMU_BIN}" >/dev/null 2>&1 || fail "qemu not found: brew install qemu"
    command -v docker          >/dev/null 2>&1 || fail "docker not found: install Docker Desktop"
    command -v tmux            >/dev/null 2>&1 || fail "tmux not found: brew install tmux"
    [ -x "$SOCKET_VMNET_CLIENT" ] || fail "socket_vmnet_client not found. Run: brew install socket_vmnet && sudo brew services start socket_vmnet"
    [ -S "$SOCKET_VMNET_PATH" ]   || fail "socket_vmnet socket not found at ${SOCKET_VMNET_PATH}. Run: sudo brew services start socket_vmnet"
    [ -d "$REPO_ROOT" ]           || fail "repo root not found: $REPO_ROOT"
fi

# ---- Create directories ----

log "Creating directories..."
run mkdir -p \
    "$DEER_ASSETS_DIR" \
    "$DEER_OVERLAYS_DIR" \
    "$DEER_IMAGES_DIR" \
    "$DEER_KEYS_DIR" \
    "${DEER_DIR}/bin" \
    "$SOURCE_VM_DIR" \
    "$DEER_CLI_CONFIG_DIR"

# ---- Docker Compose ----

log "Starting Docker Compose services (Redpanda, Elasticsearch, Kibana)..."
export HOST_IP="${HOST_IP:-$(ipconfig getifaddr en0 2>/dev/null || route -n get default 2>/dev/null | awk '/gateway/{print $2}' || echo '127.0.0.1')}"
run docker compose -f "${REPO_ROOT}/demo/logstash-pipeline-issue-demo/docker-compose.yml" up -d

if [ "$DRY_RUN" -eq 0 ]; then
    log "Waiting for Elasticsearch..."
    for i in $(seq 1 36); do
        curl -sf http://localhost:9200 >/dev/null 2>&1 && log "Elasticsearch ready." && break
        log "  waiting... (${i}/36)"; sleep 5
    done

    log "Waiting for Kibana..."
    for i in $(seq 1 36); do
        curl -sf http://localhost:5601/api/status 2>/dev/null | grep -q '"level":"available"' && log "Kibana ready." && break
        log "  waiting... (${i}/36)"; sleep 5
    done

    if [ -f "${REPO_ROOT}/demo/logstash-pipeline-issue-demo/kibana/setup-dashboard.sh" ]; then
        log "Setting up Kibana dashboard..."
        bash "${REPO_ROOT}/demo/logstash-pipeline-issue-demo/kibana/setup-dashboard.sh" http://localhost:5601
    fi
fi

# ---- Download arm64 microVM assets ----

UBUNTU_KERNEL="${DEER_ASSETS_DIR}/ubuntu-${UBUNTU_VERSION}-server-cloudimg-${UBUNTU_ARCH}-vmlinuz-generic"
UBUNTU_INITRD="${DEER_ASSETS_DIR}/ubuntu-${UBUNTU_VERSION}-server-cloudimg-${UBUNTU_ARCH}-initrd-generic"
UBUNTU_IMAGE="${DEER_IMAGES_DIR}/deer-source-vm.qcow2"

if [ ! -f "$UBUNTU_KERNEL" ] || [ ! -f "$UBUNTU_INITRD" ] || [ ! -f "$UBUNTU_IMAGE" ]; then
    log "Downloading Ubuntu ${UBUNTU_VERSION} arm64 cloud images..."
    ASSETS_SCRIPT="${REPO_ROOT}/demo/logstash-pipeline-issue-demo/scripts/download-microvm-assets.sh"
    if [ -f "$ASSETS_SCRIPT" ]; then
        run bash "$ASSETS_SCRIPT" --arch "$UBUNTU_ARCH" --output-dir "$DEER_ASSETS_DIR" --images-dir "$DEER_IMAGES_DIR"
    else
        BASE_URL="https://cloud-images.ubuntu.com/releases/${UBUNTU_CODENAME}/release"
        [ -f "$UBUNTU_KERNEL" ] || run curl -fL -o "$UBUNTU_KERNEL" \
            "${BASE_URL}/unpacked/ubuntu-${UBUNTU_VERSION}-server-cloudimg-${UBUNTU_ARCH}-vmlinuz-generic"
        [ -f "$UBUNTU_INITRD" ] || run curl -fL -o "$UBUNTU_INITRD" \
            "${BASE_URL}/unpacked/ubuntu-${UBUNTU_VERSION}-server-cloudimg-${UBUNTU_ARCH}-initrd-generic"
        if [ ! -f "$UBUNTU_IMAGE" ]; then
            run curl -fL -o "${UBUNTU_IMAGE}.tmp" \
                "${BASE_URL}/ubuntu-${UBUNTU_VERSION}-server-cloudimg-${UBUNTU_ARCH}.img"
            run qemu-img convert -f qcow2 -O qcow2 "${UBUNTU_IMAGE}.tmp" "$UBUNTU_IMAGE"
            rm -f "${UBUNTU_IMAGE}.tmp"
        fi
    fi
    # Rename downloaded image if assets script used the old name
    OLD_IMAGE="${DEER_IMAGES_DIR}/ubuntu-${UBUNTU_VERSION}-${UBUNTU_ARCH}.qcow2"
    if [ -f "$OLD_IMAGE" ] && [ ! -f "$UBUNTU_IMAGE" ]; then
        run mv "$OLD_IMAGE" "$UBUNTU_IMAGE"
    fi
fi

# ---- SSH keys ----

if [ ! -f "$SOURCE_KEY" ]; then
    log "Generating source SSH key..."
    run ssh-keygen -t ed25519 -f "$SOURCE_KEY" -N "" -C "deer-source-access"
fi
if [ ! -f "$SSH_CA_KEY" ]; then
    log "Generating SSH CA key..."
    run ssh-keygen -t ed25519 -f "$SSH_CA_KEY" -N "" -C "deer-daemon-ca"
fi
if [ ! -f "$SSH_IDENTITY" ]; then
    log "Generating SSH identity key..."
    run ssh-keygen -t ed25519 -f "$SSH_IDENTITY" -N "" -C "deer-daemon-identity"
fi

# ---- Source VM: boot native QEMU arm64 + HVF ----

log "Preparing deer-source VM (native QEMU arm64, HVF)..."

# Stop existing source VM if running
if [ "$DRY_RUN" -eq 0 ]; then
    # Kill any existing QEMU processes for deer-source
    pkill -f "qemu-system-aarch64.*deer-source" 2>/dev/null || true
    sleep 1
    # Also kill by PID file if exists
    if [ -f "$SOURCE_VM_PID_FILE" ]; then
        old_pid="$(cat "$SOURCE_VM_PID_FILE" 2>/dev/null || true)"
        if [ -n "$old_pid" ] && kill -0 "$old_pid" 2>/dev/null; then
            log "Stopping existing source VM (pid ${old_pid})..."
            kill "$old_pid" 2>/dev/null || true
            sleep 1
            if kill -0 "$old_pid" 2>/dev/null; then
                kill -9 "$old_pid" 2>/dev/null || true
            fi
        fi
        rm -f "$SOURCE_VM_PID_FILE"
    fi
fi

# Generate a FRESH MAC address each run to avoid ARP stale entries
# Reusing MAC causes ARP confusion with old IPs
if [ "$DRY_RUN" -eq 0 ]; then
    generate_mac > "$SOURCE_VM_MAC_FILE"
    SOURCE_VM_MAC="$(cat "$SOURCE_VM_MAC_FILE")"
else
    SOURCE_VM_MAC="52:54:00:ab:cd:ef"
fi

# Create QCOW2 overlay (recreate for clean state)
log "Creating source VM overlay..."
run rm -f "$SOURCE_VM_OVERLAY"
run qemu-img create -f qcow2 -b "$UBUNTU_IMAGE" -F qcow2 "$SOURCE_VM_OVERLAY"
run qemu-img resize "$SOURCE_VM_OVERLAY" 20G

# Detect macOS host IP (reachable from VMs via socket_vmnet gateway)
HOST_IP=""
if [ "$DRY_RUN" -eq 0 ]; then
    HOST_IP="$(ipconfig getifaddr en0 2>/dev/null || route -n get default 2>/dev/null | awk '/gateway/{print $2}' || echo "127.0.0.1")"
    log "Host IP for VM networking: ${HOST_IP}"
fi

# Build cloud-init user-data: set up SSH user + install Logstash
SOURCE_PUB_KEY_CONTENT=""
if [ "$DRY_RUN" -eq 0 ]; then
    SOURCE_PUB_KEY_CONTENT="$(cat "$SOURCE_PUB_KEY")"
fi

SOURCE_USER_DATA="#cloud-config
users:
  - name: ${SOURCE_VM_SSH_USER}
    sudo: ALL=(ALL) NOPASSWD:ALL
    shell: /bin/bash
    ssh_authorized_keys:
      - ${SOURCE_PUB_KEY_CONTENT}
runcmd:
  - systemctl mask serial-getty@ttyAMA0.service || systemctl mask serial-getty@ttyS0.service || true
  - systemctl stop serial-getty@ttyAMA0.service || systemctl stop serial-getty@ttyS0.service || true
  - systemctl start ssh || systemctl start sshd || true
  - systemctl enable ssh || systemctl enable sshd || true
  # Explicitly set up authorized_keys for both deer and root
  # (cloud-init users directive may be skipped if base image has stale state)
  - mkdir -p /home/${SOURCE_VM_SSH_USER}/.ssh /root/.ssh
  - echo '${SOURCE_PUB_KEY_CONTENT}' > /home/${SOURCE_VM_SSH_USER}/.ssh/authorized_keys
  - echo '${SOURCE_PUB_KEY_CONTENT}' > /root/.ssh/authorized_keys
  - chown -R ${SOURCE_VM_SSH_USER}:${SOURCE_VM_SSH_USER} /home/${SOURCE_VM_SSH_USER}/.ssh
  - chmod 700 /home/${SOURCE_VM_SSH_USER}/.ssh /root/.ssh
  - chmod 600 /home/${SOURCE_VM_SSH_USER}/.ssh/authorized_keys /root/.ssh/authorized_keys
  - touch /var/lib/cloud/instance/boot-finished || true
"

SOURCE_META_DATA="instance-id: deer-source-$(date +%s)
local-hostname: deer-source"

log "Creating source VM cloud-init ISO..."
if [ "$DRY_RUN" -eq 0 ]; then
    create_cloud_init_iso "$SOURCE_VM_CLOUDINIT" "$SOURCE_USER_DATA" "$SOURCE_META_DATA"
fi

# Boot source VM in background (no -daemonize: HVF context must not fork)
log "Booting source VM (arm64 + HVF)..."
if [ "$DRY_RUN" -eq 0 ]; then
    boot_qemu_vm \
        "$SOURCE_VM_OVERLAY" \
        "$SOURCE_VM_CLOUDINIT" \
        "$SOURCE_VM_MAC" \
        "$SOURCE_VM_MEM_MB" \
        "$SOURCE_VM_CPUS" \
        "$SOURCE_VM_SERIAL_LOG" \
        "$SOURCE_VM_PID_FILE"
    log "Source VM booted (pid $(cat "$SOURCE_VM_PID_FILE"))."
fi

# ---- Build deer-daemon ----

log "Building deer-daemon..."
run go build -o "$DEER_DAEMON_BIN" "${REPO_ROOT}/deer-daemon/cmd/deer-daemon"

# ---- Write daemon config ----

log "Writing daemon config..."
if [ "$DRY_RUN" -eq 0 ]; then
    # Create symlink from daemon.yaml to daemon-macos.yaml
    # This ensures daemon uses our macos-native config even with tmux
    ln -sf "$DEER_DAEMON_CONFIG" "${DEER_DIR}/daemon.yaml"

    # Also write daemon-macos.yaml directly
    cat > "$DEER_DAEMON_CONFIG" <<EOF
provider: microvm

daemon:
  listen_addr: ":9091"
  enabled: true

microvm:
  qemu_binary: ${QEMU_BIN}
  accel: hvf
  socket_vmnet_client: ${SOCKET_VMNET_CLIENT}
  socket_vmnet_path: ${SOCKET_VMNET_PATH}
  kernel_path: ${UBUNTU_KERNEL}
  initrd_path: ${UBUNTU_INITRD}
  root_device: /dev/vda1
  work_dir: ${DEER_OVERLAYS_DIR}
  default_vcpus: 2
  default_memory_mb: 2048
  ip_discovery_timeout: 2m
  readiness_timeout: 10m

network:
  default_bridge: vmnet
  dhcp_mode: arp

image:
  base_dir: ${DEER_IMAGES_DIR}

ssh:
  ca_key_path: ${SSH_CA_KEY}
  ca_pub_key_path: ${SSH_CA_PUB}
  key_dir: ${DEER_KEYS_DIR}
  cert_ttl: 30m
  default_user: sandbox
  identity_file: ${SSH_IDENTITY}

state:
  db_path: ${DEER_DIR}/sandbox-host.db
EOF
fi

# ---- Start deer-daemon ----

log "Starting deer-daemon (native macOS, HVF)..."
if [ "$DRY_RUN" -eq 0 ]; then
    pkill -f "deer-daemon.*daemon-macos.yaml" 2>/dev/null || true
    sleep 1
fi

run tmux new-session -d -s deer-daemon \
    "\"${DEER_DAEMON_BIN}\" serve -config \"${DEER_DAEMON_CONFIG}\" 2>&1 | tee \"${DEER_DAEMON_LOG}\""

if [ "$DRY_RUN" -eq 0 ]; then
    log "Waiting for daemon gRPC on :9091..."
    for i in $(seq 1 20); do
        nc -z localhost 9091 >/dev/null 2>&1 && log "Daemon ready." && break
        sleep 1
    done
fi

# ---- Discover source VM IP + deploy pipeline ----

SOURCE_IP=""
if [ "$DRY_RUN" -eq 0 ]; then
    SOURCE_IP="$(discover_ip_by_mac "$SOURCE_VM_MAC" 120)" || fail "Could not discover source VM IP. Check ${SOURCE_VM_SERIAL_LOG}"
    log "Source VM IP: ${SOURCE_IP}"

    wait_for_ssh "$SOURCE_IP" "$SOURCE_VM_SSH_USER" "$SOURCE_KEY" 180 || {
        # If deer user SSH fails, try root (cloud-init may have skipped user creation
        # on a base image with stale cloud-init state)
        log "deer user SSH failed, bootstrapping via root..."
        ROOT_SSH_KEY="${HOME}/.ssh/id_rsa"
        for _ in $(seq 1 60); do
            ssh -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes \
                "root@${SOURCE_IP}" true 2>/dev/null && break
            sleep 3
        done || fail "Root SSH also failed. Check ${SOURCE_VM_SERIAL_LOG}"
        log "Fixing deer user authorized_keys via root..."
        ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 -o BatchMode=yes \
            "root@${SOURCE_IP}" "mkdir -p /home/${SOURCE_VM_SSH_USER}/.ssh && echo '${SOURCE_PUB_KEY_CONTENT}' > /home/${SOURCE_VM_SSH_USER}/.ssh/authorized_keys && chown -R ${SOURCE_VM_SSH_USER}:${SOURCE_VM_SSH_USER} /home/${SOURCE_VM_SSH_USER}/.ssh && chmod 700 /home/${SOURCE_VM_SSH_USER}/.ssh && chmod 600 /home/${SOURCE_VM_SSH_USER}/.ssh/authorized_keys"
        wait_for_ssh "$SOURCE_IP" "$SOURCE_VM_SSH_USER" "$SOURCE_KEY" 60 || \
            fail "SSH to source VM timed out after bootstrap. Check ${SOURCE_VM_SERIAL_LOG}"
    }

    log "Growing VM filesystem..."
    ssh_source "$SOURCE_IP" "sudo growpart /dev/vda 1 && sudo resize2fs /dev/vda1" || true

    log "Fixing time sync on source VM..."
    HOST_TIME="$(date -u '+%Y-%m-%d %H:%M:%S')"
    ssh_source "$SOURCE_IP" "sudo date -u -s '${HOST_TIME}' 2>/dev/null || sudo timedatectl set-ntp true 2>/dev/null || true" || true

    log "Installing Logstash on source VM..."
    log "  Updating apt..."
    ssh_source "$SOURCE_IP" "sudo apt-get update -qq" || \
        fail "apt-get update failed"

    log "  Installing prerequisites..."
    ssh_source "$SOURCE_IP" "sudo apt-get install -y -qq apt-transport-https wget gnupg" || \
        fail "prerequisites install failed"

    log "  Adding Elastic GPG key..."
    ssh_source "$SOURCE_IP" "wget -qO - https://artifacts.elastic.co/GPG-KEY-elasticsearch | sudo gpg --batch --yes --dearmor -o /usr/share/keyrings/elasticsearch-keyring.gpg" || \
        fail "GPG key import failed"

    log "  Adding Elastic apt repo..."
    ssh_source "$SOURCE_IP" "echo 'deb [signed-by=/usr/share/keyrings/elasticsearch-keyring.gpg] https://artifacts.elastic.co/packages/8.x/apt stable main' | sudo tee /etc/apt/sources.list.d/elastic-8.x.list > /dev/null" || \
        fail "apt repo add failed"

    log "  Updating apt with Elastic repo..."
    ssh_source "$SOURCE_IP" "sudo apt-get update -qq" || \
        fail "apt-get update (with Elastic) failed"

    log "  Installing logstash package..."
    ssh_source "$SOURCE_IP" "sudo apt-get install -y -qq logstash" || \
        fail "logstash install failed"

    log "Deploying Logstash pipeline configs..."
    ssh_source "$SOURCE_IP" "sudo mkdir -p /etc/logstash/conf.d"
    for conf in "${REPO_ROOT}/demo/logstash-pipeline-issue-demo/logstash/pipeline/"*.conf; do
        scp_to_source "$SOURCE_IP" "$conf" "/tmp/$(basename "$conf")"
        ssh_source "$SOURCE_IP" "sudo mv /tmp/$(basename "$conf") /etc/logstash/conf.d/"
    done

    if [ -f "${REPO_ROOT}/demo/logstash-pipeline-issue-demo/logstash/station_timezones.csv" ]; then
        scp_to_source "$SOURCE_IP" "${REPO_ROOT}/demo/logstash-pipeline-issue-demo/logstash/station_timezones.csv" "/tmp/station_timezones.csv"
        ssh_source "$SOURCE_IP" "sudo mv /tmp/station_timezones.csv /etc/logstash/"
    fi

    if [ -f "${REPO_ROOT}/demo/logstash-pipeline-issue-demo/logstash/logstash.yml" ]; then
        scp_to_source "$SOURCE_IP" "${REPO_ROOT}/demo/logstash-pipeline-issue-demo/logstash/logstash.yml" "/tmp/logstash.yml"
        ssh_source "$SOURCE_IP" "sudo mv /tmp/logstash.yml /etc/logstash/logstash.yml"
    fi

    # Patch Kafka and ES addresses to point at macOS host
    log "Patching pipeline addresses to host IP ${HOST_IP}..."
    ssh_source "$SOURCE_IP" "sudo sed -i 's|[0-9]\+\.[0-9]\+\.[0-9]\+\.[0-9]\+:909[0-9]|${HOST_IP}:9093|g' /etc/logstash/conf.d/01-input-kafka.conf 2>/dev/null || true"
    ssh_source "$SOURCE_IP" "sudo sed -i 's|[0-9]\+\.[0-9]\+\.[0-9]\+\.[0-9]\+:9200|${HOST_IP}:9200|g' /etc/logstash/conf.d/15-output-es.conf 2>/dev/null || true"

    ssh_source "$SOURCE_IP" "sudo systemctl restart logstash || sudo systemctl start logstash"
    log "Logstash started on source VM."
fi

# ---- Write deer CLI config ----

# log "Writing deer CLI config..."
# if [ "$DRY_RUN" -eq 0 ]; then
#     run mkdir -p "$DEER_CLI_CONFIG_DIR"
#     cat > "$DEER_CLI_CONFIG" <<EOF
# daemon:
#   address: ${DAEMON_GRPC_ADDR}
#   insecure: true

# ssh:
#   identity_file: ${SOURCE_KEY}
#   default_user: ${SOURCE_VM_SSH_USER}

# source_hosts:
#   - address: ${SOURCE_IP}
#     ssh_user: ${SOURCE_VM_SSH_USER}
#     ssh_port: 22
#     type: ssh
# EOF
# fi

# ---- Summary ----

log ""
log "---- deer.sh native macOS demo ready (no Lima) ----"
log ""
log "  Daemon:         ${DAEMON_GRPC_ADDR}  (tmux: deer-daemon)"
log "  Source VM:      ${SOURCE_IP:-<pending>}  (arm64 QEMU/HVF, Logstash)"
log "  Elasticsearch:  http://localhost:9200"
log "  Kibana:         http://localhost:5601"
log ""
log "Connect:"
log "  deer connect ${DAEMON_GRPC_ADDR} --insecure"
log ""
log "Logs:"
log "  tmux attach -t deer-daemon"
log "  tail -f ${SOURCE_VM_SERIAL_LOG}"
log "  tail -f ${DEER_DAEMON_LOG}"
log ""
log "Teardown:"
log "  kill \$(cat ${SOURCE_VM_PID_FILE})"
log "  docker compose -f ${REPO_ROOT}/demo/logstash-pipeline-issue-demo/docker-compose.yml down"
