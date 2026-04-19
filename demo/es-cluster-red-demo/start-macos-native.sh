#!/usr/bin/env bash
# demo/es-cluster-red-demo/start-macos-native.sh
#
# Spins up a 5-node Elasticsearch cluster as native QEMU arm64 VMs on macOS.
# After the cluster goes GREEN, one data node is killed and shard allocation
# is disabled to force a YELLOW cluster state. The deer agent must then SSH
# into each node in read-only mode to diagnose what happened.
#
# Cluster topology:
#   es-node-1..3: master-eligible + data + ingest
#   es-node-4..5: data-only
#   es-node-5 is killed after GREEN to trigger YELLOW state
#
# Each VM has two SSH users:
#   root  - authorized with your ~/.ssh/id_ed25519.pub (or id_rsa.pub)
#   deer  - authorized with the deer source key (read-only agent access)
#
# Requirements:
#   brew install qemu socket_vmnet cdrtools
#   sudo brew services start socket_vmnet
#
# Usage: ./start-macos-native.sh [--repo-root <path>] [--dry-run]
#
# Environment:
#   ES_DEMO_NODE_MEM   VM memory per node in MB (default: 1536)
#   ES_DEMO_HEAP       ES JVM heap size           (default: 512m)
#   ES_DEMO_CPUS       VM CPUs per node            (default: 2)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# ---- Configuration ----

DEER_DIR="${HOME}/.deer"
DEER_ASSETS_DIR="${DEER_DIR}/assets"
DEER_OVERLAYS_DIR="${DEER_DIR}/overlays"
DEER_IMAGES_DIR="${DEER_DIR}/images"
DEER_KEYS_DIR="${DEER_DIR}/keys"
DEER_DAEMON_BIN="${DEER_DIR}/bin/deer-daemon"
DEER_DAEMON_CONFIG="${DEER_DIR}/daemon-es-cluster.yaml"
DEER_DAEMON_LOG="${DEER_DIR}/daemon-es-cluster.log"

CLUSTER_DIR="${DEER_DIR}/es-cluster"

SOURCE_KEY="${DEER_KEYS_DIR}/source_ed25519"
SOURCE_PUB_KEY="${DEER_KEYS_DIR}/source_ed25519.pub"

SSH_CA_KEY="${DEER_DIR}/ssh_ca"
SSH_CA_PUB="${DEER_DIR}/ssh_ca.pub"
SSH_IDENTITY="${DEER_DIR}/identity"

DEER_CLI_CONFIG_DIR="${HOME}/.config/deer"
DEER_CLI_CONFIG="${DEER_CLI_CONFIG_DIR}/config.yaml"

SOCKET_VMNET_CLIENT="/opt/homebrew/opt/socket_vmnet/bin/socket_vmnet_client"
SOCKET_VMNET_PATH="/opt/homebrew/var/run/socket_vmnet"
QEMU_BIN="/opt/homebrew/bin/qemu-system-aarch64"

DAEMON_GRPC_ADDR="localhost:9091"
SSH_USER="deer"

UBUNTU_VERSION="24.04"
UBUNTU_CODENAME="noble"
UBUNTU_ARCH="arm64"

ES_VERSION="8.13.4"

NUM_NODES=5
NODE_MEM_MB="${ES_DEMO_NODE_MEM:-1536}"
NODE_CPUS="${ES_DEMO_CPUS:-2}"
ES_HEAP="${ES_DEMO_HEAP:-512m}"

NODE_NAMES=("es-node-1" "es-node-2" "es-node-3" "es-node-4" "es-node-5")
NODE_ROLES=("master,data,ingest" "master,data,ingest" "master,data,ingest" "data" "data")
KILL_NODE_IDX=4  # 0-indexed: es-node-5

DRY_RUN=0

# ---- Helpers ----

usage() {
    cat <<'EOF'
Usage: ./start-macos-native.sh [--repo-root <path>] [--dry-run]

Spin up a 5-node ES cluster, wait for GREEN, then kill one node to go YELLOW.

Options:
  --repo-root <path>   Repo root (default: two levels above this script)
  --dry-run            Print commands without executing
  -h, --help           Show help

Environment:
  ES_DEMO_NODE_MEM   VM memory per node MB (default: 1536)
  ES_DEMO_HEAP       ES JVM heap               (default: 512m)
  ES_DEMO_CPUS       VM CPUs per node           (default: 2)
EOF
}

log()  { printf '[es-cluster] %s\n' "$*"; }
fail() { printf '[es-cluster] ERROR: %s\n' "$*" >&2; exit 1; }

run() {
    if [ "$DRY_RUN" -eq 1 ]; then printf '+'; printf ' %q' "$@"; printf '\n'; return; fi
    "$@"
}

generate_mac() {
    printf '52:54:00:%02x:%02x:%02x' $((RANDOM % 256)) $((RANDOM % 256)) $((RANDOM % 256))
}

discover_ip_by_mac() {
    local mac="$1"
    local timeout="${2:-120}"
    local mac_normalized
    mac_normalized=$(echo "$mac" | tr '[:upper:]' '[:lower:]' | tr -d ':')

    log "Discovering IP for MAC ${mac} (up to ${timeout}s)..." >&2
    for _ in $(seq 1 "$timeout"); do
        local ips
        ips=$(arp -an 2>/dev/null | awk -v target="$mac_normalized" '
        {
            ip = $2
            mac = $4
            gsub(/[()]/, "", ip)
            split(mac, octets, ":")
            normalized = ""
            for (i = 1; i <= length(octets); i++) {
                if (length(octets[i]) == 1) normalized = normalized "0" octets[i]
                else normalized = normalized tolower(octets[i])
            }
            if (normalized == target || tolower(normalized) == target) {
                print ip
            }
        }')

        if [ -n "$ips" ]; then
            for test_ip in $ips; do
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

wait_for_ssh() {
    local host="$1" user="$2" key="$3"
    local timeout="${4:-180}"
    log "Waiting for SSH on ${host} (up to ${timeout}s)..."
    for _ in $(seq 1 "$timeout"); do
        ssh -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes -o IdentitiesOnly=yes \
            -i "$key" "${user}@${host}" true 2>/dev/null && return 0
        sleep 3
    done
    return 1
}

ssh_node() {
    local host="$1" cmd="$2"
    ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 -o BatchMode=yes -o IdentitiesOnly=yes \
        -i "$SOURCE_KEY" "${SSH_USER}@${host}" "$cmd"
}

scp_to_node() {
    local host="$1" src="$2" dst="$3"
    scp -o StrictHostKeyChecking=no -o ConnectTimeout=10 -o BatchMode=yes -o IdentitiesOnly=yes \
        -i "$SOURCE_KEY" "$src" "${SSH_USER}@${host}:${dst}"
}

create_cloud_init_iso() {
    local iso_path="$1" user_data="$2" meta_data="$3"
    local tmp_dir
    tmp_dir="$(mktemp -d)"
    printf '%s' "$user_data" > "${tmp_dir}/user-data"
    printf '%s' "$meta_data" > "${tmp_dir}/meta-data"
    printf 'version: 2\nethernets:\n  eth0:\n    dhcp4: true\n' > "${tmp_dir}/network-config"

    if command -v mkisofs >/dev/null 2>&1; then
        mkisofs -output "${iso_path}" -volid "cidata" -joliet -rock "${tmp_dir}" >/dev/null 2>&1 || {
            rm -rf "${tmp_dir}"; return 1
        }
    else
        hdiutil makehybrid -o "${iso_path}" -hfs -iso -joliet \
            -default-volume-name cidata "${tmp_dir}" >/dev/null 2>&1 || {
            rm -rf "${tmp_dir}"; return 1
        }
    fi
    rm -rf "${tmp_dir}"
}

boot_qemu_vm() {
    local overlay="$1" cloudinit="$2" mac="$3" mem="$4" cpus="$5" serial_log="$6" pid_file="$7"

    "${SOCKET_VMNET_CLIENT}" "${SOCKET_VMNET_PATH}" \
        "${QEMU_BIN}" \
        -M virt \
        -accel hvf -cpu max \
        -m "${mem}" \
        -smp "${cpus}" \
        -kernel "${UBUNTU_KERNEL}" \
        -initrd "${UBUNTU_INITRD}" \
        -append "console=ttyAMA0 root=/dev/vda1 rw rootwait quiet" \
        -drive "id=root,file=${overlay},format=qcow2,if=none" \
        -device virtio-blk-device,drive=root \
        -netdev "socket,id=net0,fd=3" \
        -device "virtio-net-device,netdev=net0,mac=${mac}" \
        -drive "id=cidata,file=${cloudinit},format=raw,readonly=on,if=none" \
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
    command -v tmux            >/dev/null 2>&1 || fail "tmux not found: brew install tmux"
    [ -x "$SOCKET_VMNET_CLIENT" ] || fail "socket_vmnet_client not found. Run: brew install socket_vmnet"
    [ -S "$SOCKET_VMNET_PATH" ]   || fail "socket_vmnet socket not found at ${SOCKET_VMNET_PATH}. Run: sudo brew services start socket_vmnet"
    [ -d "$REPO_ROOT" ]           || fail "repo root not found: $REPO_ROOT"
fi

# ---- Detect user's root SSH key ----

ROOT_PUB_KEY=""
if [ "$DRY_RUN" -eq 0 ]; then
    for key in "${HOME}/.ssh/id_ed25519.pub" "${HOME}/.ssh/id_rsa.pub" "${HOME}/.ssh/id_ecdsa.pub"; do
        if [ -f "$key" ]; then
            ROOT_PUB_KEY="$(cat "$key")"
            log "Using root SSH key: $(basename "$key")"
            break
        fi
    done
    [ -n "$ROOT_PUB_KEY" ] || fail "No SSH public key found in ~/.ssh/. Need id_ed25519.pub or id_rsa.pub for root access."
fi

# ---- Create directories ----

log "Creating directories..."
run mkdir -p \
    "$DEER_ASSETS_DIR" \
    "$DEER_OVERLAYS_DIR" \
    "$DEER_IMAGES_DIR" \
    "$DEER_KEYS_DIR" \
    "${DEER_DIR}/bin" \
    "$CLUSTER_DIR" \
    "$DEER_CLI_CONFIG_DIR"

for i in $(seq 1 "$NUM_NODES"); do
    run mkdir -p "${CLUSTER_DIR}/node-${i}"
done

# ---- Download arm64 microVM assets ----

UBUNTU_KERNEL="${DEER_ASSETS_DIR}/ubuntu-${UBUNTU_VERSION}-server-cloudimg-${UBUNTU_ARCH}-vmlinuz-generic"
UBUNTU_INITRD="${DEER_ASSETS_DIR}/ubuntu-${UBUNTU_VERSION}-server-cloudimg-${UBUNTU_ARCH}-initrd-generic"
UBUNTU_IMAGE="${DEER_IMAGES_DIR}/deer-source-vm.qcow2"

if [ ! -f "$UBUNTU_KERNEL" ] || [ ! -f "$UBUNTU_INITRD" ] || [ ! -f "$UBUNTU_IMAGE" ]; then
    log "Downloading Ubuntu ${UBUNTU_VERSION} arm64 cloud images..."
    ASSETS_SCRIPT="${REPO_ROOT}/scripts/download-microvm-assets.sh"
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
    OLD_IMAGE="${DEER_IMAGES_DIR}/ubuntu-${UBUNTU_VERSION}-${UBUNTU_ARCH}.qcow2"
    if [ -f "$OLD_IMAGE" ] && [ ! -f "$UBUNTU_IMAGE" ]; then
        run mv "$OLD_IMAGE" "$UBUNTU_IMAGE"
    fi
fi

# ---- SSH keys ----

if [ ! -f "$SOURCE_KEY" ]; then
    log "Generating deer source SSH key..."
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

SOURCE_PUB_KEY_CONTENT=""
if [ "$DRY_RUN" -eq 0 ]; then
    SOURCE_PUB_KEY_CONTENT="$(cat "$SOURCE_PUB_KEY")"
fi

# ---- Phase 1: Boot all VMs ----

log "Phase 1: Booting ${NUM_NODES} ES node VMs..."

NODE_IPS=()
NODE_MACS=()

for i in $(seq 1 "$NUM_NODES"); do
    idx=$((i - 1))
    name="${NODE_NAMES[$idx]}"
    node_dir="${CLUSTER_DIR}/node-${i}"
    overlay="${node_dir}/overlay.qcow2"
    cloudinit="${node_dir}/cloud-init.iso"
    mac_file="${node_dir}/mac"
    pid_file="${node_dir}/qemu.pid"
    serial_log="${node_dir}/serial.log"
    ip_file="${node_dir}/ip"

    # Stop existing VM if running
    if [ "$DRY_RUN" -eq 0 ]; then
        if [ -f "$pid_file" ]; then
            old_pid="$(cat "$pid_file" 2>/dev/null || true)"
            if [ -n "$old_pid" ] && kill -0 "$old_pid" 2>/dev/null; then
                log "Stopping existing ${name} (pid ${old_pid})..."
                kill "$old_pid" 2>/dev/null || true
                sleep 1
                kill -9 "$old_pid" 2>/dev/null || true
            fi
            rm -f "$pid_file"
        fi
    fi

    # Generate MAC
    if [ "$DRY_RUN" -eq 0 ]; then
        generate_mac > "$mac_file"
        mac="$(cat "$mac_file")"
    else
        mac="52:54:00:ab:cd:e${i}"
    fi
    NODE_MACS+=("$mac")

    # Create overlay
    log "Creating overlay for ${name}..."
    run rm -f "$overlay"
    run qemu-img create -f qcow2 -b "$UBUNTU_IMAGE" -F qcow2 "$overlay"
    run qemu-img resize "$overlay" 10G

    # Cloud-init
    META_DATA="instance-id: ${name}-$(date +%s)
local-hostname: ${name}"

    USER_DATA="#cloud-config
users:
  - name: ${SSH_USER}
    sudo: ALL=(ALL) NOPASSWD:ALL
    shell: /bin/bash
    ssh_authorized_keys:
      - ${SOURCE_PUB_KEY_CONTENT}
  - name: root
    ssh_authorized_keys:
      - ${ROOT_PUB_KEY}
runcmd:
  - systemctl mask serial-getty@ttyAMA0.service || systemctl mask serial-getty@ttyS0.service || true
  - systemctl stop serial-getty@ttyAMA0.service || systemctl stop serial-getty@ttyS0.service || true
  - systemctl start ssh || systemctl start sshd || true
  - systemctl enable ssh || systemctl enable sshd || true
  - mkdir -p /root/.ssh
  - echo '${ROOT_PUB_KEY}' > /root/.ssh/authorized_keys
  - chmod 700 /root/.ssh
  - chmod 600 /root/.ssh/authorized_keys
  - echo '${SOURCE_PUB_KEY_CONTENT}' >> /root/.ssh/authorized_keys
  - sysctl -w vm.max_map_count=262144
  - echo 'vm.max_map_count=262144' >> /etc/sysctl.d/99-elasticsearch.conf
  - touch /var/lib/cloud/instance/boot-finished || true
"

    log "Creating cloud-init ISO for ${name}..."
    if [ "$DRY_RUN" -eq 0 ]; then
        create_cloud_init_iso "$cloudinit" "$USER_DATA" "$META_DATA"
    fi

    # Boot
    log "Booting ${name} (arm64 + HVF)..."
    if [ "$DRY_RUN" -eq 0 ]; then
        boot_qemu_vm "$overlay" "$cloudinit" "$mac" "$NODE_MEM_MB" "$NODE_CPUS" "$serial_log" "$pid_file"
        log "  ${name} booted (pid $(cat "$pid_file"))"
    fi
done

# ---- Phase 2: Discover IPs ----

log "Phase 2: Discovering VM IPs..."
if [ "$DRY_RUN" -eq 0 ]; then
    for i in $(seq 1 "$NUM_NODES"); do
        name="${NODE_NAMES[$((i-1))]}"
        mac="${NODE_MACS[$((i-1))]}"
        ip_file="${CLUSTER_DIR}/node-${i}/ip"

        ip="$(discover_ip_by_mac "$mac" 120)" || fail "Could not discover IP for ${name}. Check ${CLUSTER_DIR}/node-${i}/serial.log"
        echo "$ip" > "$ip_file"
        NODE_IPS+=("$ip")
        log "  ${name}: ${ip}"
    done
fi

# ---- Phase 3: Wait for SSH + install ES ----

log "Phase 3: Waiting for SSH and installing Elasticsearch on all nodes..."
if [ "$DRY_RUN" -eq 0 ]; then
    # Wait for SSH on all nodes
    for i in $(seq 1 "$NUM_NODES"); do
        name="${NODE_NAMES[$((i-1))]}"
        ip="${NODE_IPS[$((i-1))]}"
        wait_for_ssh "$ip" "$SSH_USER" "$SOURCE_KEY" 180 || {
            log "deer user SSH failed for ${name}, trying root bootstrap..."
            for _ in $(seq 1 60); do
                ssh -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes \
                    "root@${ip}" true 2>/dev/null && break
                sleep 3
            done || fail "SSH failed for ${name}. Check ${CLUSTER_DIR}/node-${i}/serial.log"
            ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 -o BatchMode=yes \
                "root@${ip}" "mkdir -p /home/${SSH_USER}/.ssh && echo '${SOURCE_PUB_KEY_CONTENT}' > /home/${SSH_USER}/.ssh/authorized_keys && chown -R ${SSH_USER}:${SSH_USER} /home/${SSH_USER}/.ssh && chmod 700 /home/${SSH_USER}/.ssh && chmod 600 /home/${SSH_USER}/.ssh/authorized_keys" || true
            wait_for_ssh "$ip" "$SSH_USER" "$SOURCE_KEY" 60 || \
                fail "SSH to ${name} timed out. Check ${CLUSTER_DIR}/node-${i}/serial.log"
        }
    done

    # Grow filesystem + fix time on all nodes
    for i in $(seq 1 "$NUM_NODES"); do
        ip="${NODE_IPS[$((i-1))]}"
        name="${NODE_NAMES[$((i-1))]}"
        log "  Preparing ${name}..."
        ssh_node "$ip" "sudo growpart /dev/vda 1 && sudo resize2fs /dev/vda1" || true
        HOST_TIME="$(date -u '+%Y-%m-%d %H:%M:%S')"
        ssh_node "$ip" "sudo date -u -s '${HOST_TIME}' 2>/dev/null || sudo timedatectl set-ntp true 2>/dev/null || true" || true
    done

    # Install ES on all nodes in parallel
    log "Installing Elasticsearch on all ${NUM_NODES} nodes (parallel)..."
    INSTALL_PIDS=()
    for i in $(seq 1 "$NUM_NODES"); do
        ip="${NODE_IPS[$((i-1))]}"
        name="${NODE_NAMES[$((i-1))]}"
        (
            log "  [${name}] Installing prerequisites..."
            ssh_node "$ip" "sudo apt-get update -qq && sudo apt-get install -y -qq apt-transport-https wget gnupg" || \
                { log "  [${name}] ERROR: prerequisites failed"; exit 1; }

            log "  [${name}] Adding Elastic GPG key..."
            ssh_node "$ip" "wget -qO - https://artifacts.elastic.co/GPG-KEY-elasticsearch | sudo gpg --batch --yes --dearmor -o /usr/share/keyrings/elasticsearch-keyring.gpg" || \
                { log "  [${name}] ERROR: GPG key failed"; exit 1; }

            log "  [${name}] Adding Elastic apt repo..."
            ssh_node "$ip" "echo 'deb [signed-by=/usr/share/keyrings/elasticsearch-keyring.gpg] https://artifacts.elastic.co/packages/8.x/apt stable main' | sudo tee /etc/apt/sources.list.d/elastic-8.x.list > /dev/null" || \
                { log "  [${name}] ERROR: apt repo failed"; exit 1; }

            log "  [${name}] Installing Elasticsearch..."
            ssh_node "$ip" "sudo apt-get update -qq && sudo apt-get install -y -qq elasticsearch" || \
                { log "  [${name}] ERROR: ES install failed"; exit 1; }

            log "  [${name}] Elasticsearch installed."
        ) &
        INSTALL_PIDS+=($!)
    done

    # Wait for all installs
    FAILED=0
    for pid in "${INSTALL_PIDS[@]}"; do
        if ! wait "$pid"; then
            FAILED=$((FAILED + 1))
        fi
    done
    [ "$FAILED" -eq 0 ] || fail "${FAILED} node(s) failed ES installation"
fi

# ---- Phase 4: Configure & start ES cluster ----

log "Phase 4: Configuring and starting Elasticsearch cluster..."
if [ "$DRY_RUN" -eq 0 ]; then
    SEED_HOSTS="[\"${NODE_IPS[0]}\",\"${NODE_IPS[1]}\",\"${NODE_IPS[2]}\"]"
    INITIAL_MASTERS="[\"es-node-1\",\"es-node-2\",\"es-node-3\"]"

    for i in $(seq 1 "$NUM_NODES"); do
        idx=$((i - 1))
        ip="${NODE_IPS[$idx]}"
        name="${NODE_NAMES[$idx]}"
        roles="${NODE_ROLES[$idx]}"

        log "  Configuring ${name} (roles: ${roles})..."

        # Write elasticsearch.yml
        ssh_node "$ip" "sudo tee /etc/elasticsearch/elasticsearch.yml > /dev/null" <<EOF
cluster.name: deer-demo
node.name: ${name}
node.roles: [${roles}]
path.data: /var/lib/elasticsearch
path.logs: /var/log/elasticsearch
network.host: 0.0.0.0
http.port: 9200
discovery.seed_hosts: ${SEED_HOSTS}
cluster.initial_master_nodes: ${INITIAL_MASTERS}
xpack.security.enabled: false
xpack.security.http.ssl.enabled: false
xpack.security.transport.ssl.enabled: false
xpack.ml.enabled: false
EOF

        # Write JVM heap options
        ssh_node "$ip" "sudo tee /etc/elasticsearch/jvm.options.d/heap.options > /dev/null" <<EOF
-Xms${ES_HEAP}
-Xmx${ES_HEAP}
EOF

        # Ensure data dir ownership
        ssh_node "$ip" "sudo chown -R elasticsearch:elasticsearch /var/lib/elasticsearch /var/log/elasticsearch" || true

        log "  Starting ES on ${name}..."
        ssh_node "$ip" "sudo systemctl daemon-reload && sudo systemctl enable elasticsearch && sudo systemctl start elasticsearch"
    done

    # Wait for cluster to be GREEN
    log "Waiting for cluster to form and go GREEN (this may take a few minutes)..."
    MASTER_IP="${NODE_IPS[0]}"
    for attempt in $(seq 1 60); do
        HEALTH=$(ssh_node "$MASTER_IP" "curl -sf 'http://localhost:9200/_cluster/health?pretty'" 2>/dev/null || echo "")
        STATUS=$(echo "$HEALTH" | grep '"status"' | head -1 | sed 's/.*: "//;s/".*//' || true)
        if [ "$STATUS" = "green" ]; then
            log "Cluster is GREEN! (${attempt}/60)"
            break
        fi
        log "  Cluster status: ${STATUS:-pending} (${attempt}/60)..."
        sleep 10
    done

    # Final health check
    HEALTH=$(ssh_node "$MASTER_IP" "curl -sf 'http://localhost:9200/_cluster/health?pretty'" 2>/dev/null || echo "")
    STATUS=$(echo "$HEALTH" | grep '"status"' | head -1 | sed 's/.*: "//;s/".*//' || true)
    log "Cluster health: ${STATUS}"
    if [ "$STATUS" != "green" ]; then
        log "WARNING: Cluster did not reach GREEN. Continuing anyway..."
    fi
fi

# ---- Phase 5: Ingest test data ----

log "Phase 5: Creating test index and ingesting data..."
if [ "$DRY_RUN" -eq 0 ]; then
    MASTER_IP="${NODE_IPS[0]}"

    # Create index with shards and replicas
    ssh_node "$MASTER_IP" "curl -sf -X PUT 'http://localhost:9200/app-logs' -H 'Content-Type: application/json' -d '
{
  \"settings\": {
    \"number_of_shards\": 5,
    \"number_of_replicas\": 1
  },
  \"mappings\": {
    \"properties\": {
      \"timestamp\": { \"type\": \"date\" },
      \"level\": { \"type\": \"keyword\" },
      \"message\": { \"type\": \"text\" },
      \"service\": { \"type\": \"keyword\" },
      \"host\": { \"type\": \"keyword\" },
      \"request_time_ms\": { \"type\": \"float\" },
      \"status_code\": { \"type\": \"integer\" }
    }
  }
}'" || log "WARNING: Index creation may have failed"

    # Ingest sample data
    ssh_node "$MASTER_IP" "curl -sf -X POST 'http://localhost:9200/app-logs/_bulk' -H 'Content-Type: application/json' -d '
{\"index\":{}}
{\"timestamp\":\"2025-01-15T10:30:00Z\",\"level\":\"INFO\",\"message\":\"Request processed successfully\",\"service\":\"api-gateway\",\"host\":\"web-01\",\"request_time_ms\":45.2,\"status_code\":200}
{\"index\":{}}
{\"timestamp\":\"2025-01-15T10:30:01Z\",\"level\":\"WARN\",\"message\":\"High memory usage detected on node\",\"service\":\"monitor\",\"host\":\"web-02\",\"request_time_ms\":12.1,\"status_code\":200}
{\"index\":{}}
{\"timestamp\":\"2025-01-15T10:30:02Z\",\"level\":\"ERROR\",\"message\":\"Database connection timeout after 30s\",\"service\":\"user-service\",\"host\":\"app-01\",\"request_time_ms\":30012.5,\"status_code\":503}
{\"index\":{}}
{\"timestamp\":\"2025-01-15T10:30:03Z\",\"level\":\"INFO\",\"message\":\"Cache hit for session lookup\",\"service\":\"session-service\",\"host\":\"app-02\",\"request_time_ms\":2.3,\"status_code\":200}
{\"index\":{}}
{\"timestamp\":\"2025-01-15T10:30:04Z\",\"level\":\"ERROR\",\"message\":\"Failed to write to Kafka topic events\",\"service\":\"event-processor\",\"host\":\"app-01\",\"request_time_ms\":8500.1,\"status_code\":500}
{\"index\":{}}
{\"timestamp\":\"2025-01-15T10:30:05Z\",\"level\":\"INFO\",\"message\":\"Health check passed\",\"service\":\"load-balancer\",\"host\":\"lb-01\",\"request_time_ms\":1.1,\"status_code\":200}
{\"index\":{}}
{\"timestamp\":\"2025-01-15T10:30:06Z\",\"level\":\"WARN\",\"message\":\"Disk usage at 85 percent on data volume\",\"service\":\"monitor\",\"host\":\"data-01\",\"request_time_ms\":5.4,\"status_code\":200}
{\"index\":{}}
{\"timestamp\":\"2025-01-15T10:30:07Z\",\"level\":\"INFO\",\"message\":\"Scheduled job completed successfully\",\"service\":\"cron-runner\",\"host\":\"app-03\",\"request_time_ms\":125.7,\"status_code\":200}
{\"index\":{}}
{\"timestamp\":\"2025-01-15T10:30:08Z\",\"level\":\"ERROR\",\"message\":\"SSL certificate expiring in 3 days for api.example.com\",\"service\":\"cert-checker\",\"host\":\"monitor-01\",\"request_time_ms\":8.9,\"status_code\":200}
{\"index\":{}}
{\"timestamp\":\"2025-01-15T10:30:09Z\",\"level\":\"INFO\",\"message\":\"User login successful from 10.0.1.50\",\"service\":\"auth-service\",\"host\":\"app-01\",\"request_time_ms\":23.4,\"status_code\":200}
'" || log "WARNING: Data ingestion may have partially failed"

    log "Test data ingested."

    # Wait for shards to settle
    sleep 5
fi

# ---- Phase 6: Trigger YELLOW state ----

log "Phase 6: Triggering YELLOW cluster state..."
if [ "$DRY_RUN" -eq 0 ]; then
    MASTER_IP="${NODE_IPS[0]}"
    KILL_IP="${NODE_IPS[$KILL_NODE_IDX]}"
    KILL_NAME="${NODE_NAMES[$KILL_NODE_IDX]}"

    # Disable replica shard allocation so the cluster stays yellow
    log "Disabling replica shard allocation..."
    ssh_node "$MASTER_IP" "curl -sf -X PUT 'http://localhost:9200/_cluster/settings' -H 'Content-Type: application/json' -d '
{
  \"persistent\": {
    \"cluster.routing.allocation.enable\": \"primaries\"
  }
}'" || log "WARNING: Could not disable allocation"

    sleep 2

    # Kill the target node
    log "Killing ${KILL_NAME} at ${KILL_IP} to trigger YELLOW state..."
    KILL_PID_FILE="${CLUSTER_DIR}/node-$((KILL_NODE_IDX + 1))/qemu.pid"
    if [ -f "$KILL_PID_FILE" ]; then
        kill_pid="$(cat "$KILL_PID_FILE" 2>/dev/null || true)"
        if [ -n "$kill_pid" ] && kill -0 "$kill_pid" 2>/dev/null; then
            kill "$kill_pid" 2>/dev/null || true
            sleep 2
            kill -9 "$kill_pid" 2>/dev/null || true
        fi
    fi
    log "${KILL_NAME} killed."

    # Wait for cluster to detect the missing node
    log "Waiting for cluster to reflect YELLOW state..."
    for attempt in $(seq 1 30); do
        HEALTH=$(ssh_node "$MASTER_IP" "curl -sf 'http://localhost:9200/_cluster/health?pretty'" 2>/dev/null || echo "")
        STATUS=$(echo "$HEALTH" | grep '"status"' | head -1 | sed 's/.*: "//;s/".*//' || true)
        if [ "$STATUS" = "yellow" ] || [ "$STATUS" = "red" ]; then
            log "Cluster is now ${STATUS}! (${attempt}/30)"
            break
        fi
        log "  Cluster status: ${STATUS:-pending} (${attempt}/30)..."
        sleep 5
    done

    # Final verification
    HEALTH=$(ssh_node "$MASTER_IP" "curl -sf 'http://localhost:9200/_cluster/health?pretty'" 2>/dev/null || echo "")
    STATUS=$(echo "$HEALTH" | grep '"status"' | head -1 | sed 's/.*: "//;s/".*//' || true)
    log "Final cluster status: ${STATUS}"
fi

# ---- Phase 7: Build & start deer-daemon ----

log "Phase 7: Building and starting deer-daemon..."
run go build -o "$DEER_DAEMON_BIN" "${REPO_ROOT}/deer-daemon/cmd/deer-daemon"

if [ "$DRY_RUN" -eq 0 ]; then
    ln -sf "$DEER_DAEMON_CONFIG" "${DEER_DIR}/daemon.yaml"

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
  default_memory_mb: 1536
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

if [ "$DRY_RUN" -eq 0 ]; then
    pkill -f "deer-daemon.*daemon-es-cluster.yaml" 2>/dev/null || true
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

# ---- Write deer CLI config ----

# log "Writing deer CLI config..."
# if [ "$DRY_RUN" -eq 0 ]; then
#     run mkdir -p "$DEER_CLI_CONFIG_DIR"

#     # Build source_hosts YAML
#     SOURCE_HOSTS=""
#     for i in $(seq 1 "$NUM_NODES"); do
#         idx=$((i - 1))
#         ip="${NODE_IPS[$idx]}"
#         name="${NODE_NAMES[$idx]}"
#         STATUS_MARKER="online"
#         if [ "$idx" -eq "$KILL_NODE_IDX" ]; then
#             STATUS_MARKER="OFFLINE (killed)"
#         fi
#         SOURCE_HOSTS="${SOURCE_HOSTS}  - address: ${ip}
#     name: ${name}
#     ssh_user: ${SSH_USER}
#     ssh_port: 22
#     type: ssh
# "
#     done

#     cat > "$DEER_CLI_CONFIG" <<EOF
# daemon:
#   address: ${DAEMON_GRPC_ADDR}
#   insecure: true

# ssh:
#   identity_file: ${SOURCE_KEY}
#   default_user: ${SSH_USER}

# source_hosts:
# ${SOURCE_HOSTS}
# EOF
# fi

# ---- Configure ~/.ssh/config for ES cluster nodes ----

SSH_CONFIG="${HOME}/.ssh/config"

log "Configuring ~/.ssh/config for ES cluster nodes..."
if [ "$DRY_RUN" -eq 0 ]; then
    # Remove old deer-es-cluster block
    if grep -q '# deer-es-cluster-start' "$SSH_CONFIG" 2>/dev/null; then
        TMPFILE="$(mktemp)"
        awk '/^# deer-es-cluster-start/{skip=1} skip{if(/^# deer-es-cluster-end/){skip=0; next}; next} {print}' "$SSH_CONFIG" > "$TMPFILE"
        mv "$TMPFILE" "$SSH_CONFIG"
    fi

    mkdir -p "$(dirname "$SSH_CONFIG")"
    SSH_BLOCK=""
    for i in $(seq 1 "$NUM_NODES"); do
        idx=$((i - 1))
        ip="${NODE_IPS[$idx]}"
        name="${NODE_NAMES[$idx]}"
        SSH_BLOCK="${SSH_BLOCK}
Host ${name}
  HostName ${ip}
  User root
  IdentityFile ${HOME}/.ssh/id_ed25519
  StrictHostKeyChecking no
  UserKnownHostsFile /dev/null
"
    done

    cat >> "$SSH_CONFIG" <<EOF

# deer-es-cluster-start${SSH_BLOCK}# deer-es-cluster-end
EOF
    log "Added ES cluster hosts to ${SSH_CONFIG}"
fi

# ---- Summary ----

echo ""
log "============================================================"
log "  ES Cluster Yellow Demo Ready"
log "============================================================"
echo ""
if [ "$DRY_RUN" -eq 0 ]; then
    log "  Cluster name:    deer-demo"
    log "  ES version:      ${ES_VERSION}"
    log "  Cluster status:  ${STATUS}"
    echo ""
    log "  Node IPs:"
    for i in $(seq 1 "$NUM_NODES"); do
        idx=$((i - 1))
        ip="${NODE_IPS[$idx]}"
        name="${NODE_NAMES[$idx]}"
        roles="${NODE_ROLES[$idx]}"
        marker=""
        if [ "$idx" -eq "$KILL_NODE_IDX" ]; then
            marker=" <-- KILLED (this is the problem)"
        fi
        log "    ${name}: ${ip}  (${roles})${marker}"
    done
    echo ""
    log "  Daemon:          ${DAEMON_GRPC_ADDR}  (tmux: deer-daemon)"
    log "  ES API:          http://${NODE_IPS[0]}:9200"
    echo ""
    log "  Root SSH:        ssh root@<ip>  (uses your personal key)"
    log "  Deer SSH:        ssh -i ${SOURCE_KEY} deer@<ip>"
    echo ""
    log "  Diagnose with:"
    log "    deer connect ${DAEMON_GRPC_ADDR} --insecure"
    echo ""
    log "  Quick health check:"
    log "    ssh -i ${SOURCE_KEY} deer@${NODE_IPS[0]} 'curl -sf http://localhost:9200/_cluster/health?pretty'"
    log "    ssh -i ${SOURCE_KEY} deer@${NODE_IPS[0]} 'curl -sf http://localhost:9200/_cat/nodes?v'"
    log "    ssh -i ${SOURCE_KEY} deer@${NODE_IPS[0]} 'curl -sf http://localhost:9200/_cat/shards?v'"
    echo ""
fi
log "Teardown:"
log "  ./stop-macos-native.sh"
log "============================================================"
