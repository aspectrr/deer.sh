#!/usr/bin/env bash
# scripts/demo/start.sh
#
# One-command local demo launcher. Runs on Mac.
# Requires: limactl (brew install lima)
#
# Usage: ./scripts/demo/start.sh [--repo-root <path>] [--dry-run]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
LIMA_NAME="deer-demo"
LIMA_CPUS=4
LIMA_MEMORY="8GiB"
LIMA_DISK="100GiB"
DRY_RUN=0

SOURCE_IMAGE_PATH="/var/lib/deer-demo/logstash-source.qcow2"
DAEMON_WORKDIR="/var/lib/deer-demo/daemon"
DAEMON_CONFIG="/var/lib/deer-demo/daemon.yaml"
ASSETS_DIR="/var/lib/deer-demo/assets"

usage() {
    cat <<'EOF'
Usage: ./scripts/demo/start.sh [--repo-root <path>] [--dry-run]

Options:
  --repo-root <path>   Repo root on host (default: two levels above this script)
  --dry-run            Print commands without executing
  -h, --help           Show help
EOF
}

log()  { printf '[demo-start] %s\n' "$*"; }
fail() { printf '[demo-start] ERROR: %s\n' "$*" >&2; exit 1; }

run_host() {
    if [ "$DRY_RUN" -eq 1 ]; then printf '+'; printf ' %q' "$@"; printf '\n'; return; fi
    "$@"
}

run_guest() {
    local cmd="$1"
    if [ "$DRY_RUN" -eq 1 ]; then
        printf '+ limactl shell %s -- bash -lc %q\n' "$LIMA_NAME" "$cmd"
        return
    fi
    limactl shell "$LIMA_NAME" -- bash -lc "$cmd"
}

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
    command -v limactl >/dev/null 2>&1 || fail "limactl not found. Install with: brew install lima"
    [ -d "$REPO_ROOT" ] || fail "repo root not found: $REPO_ROOT"
fi

# ---- Lima VM ----
log "Ensuring Lima VM '${LIMA_NAME}'..."
LIMA_HOME="${LIMA_HOME:-$HOME/.lima}"
LIMA_CONFIG_PATH="${LIMA_HOME}/${LIMA_NAME}/lima.yaml"

if [ "$DRY_RUN" -eq 0 ] && [ ! -f "$LIMA_CONFIG_PATH" ]; then
    log "Creating Lima VM '${LIMA_NAME}' (${LIMA_CPUS} CPU, ${LIMA_MEMORY}, ${LIMA_DISK})..."
    LIMA_TEMPLATE="$(mktemp /tmp/${LIMA_NAME}.XXXXXX.yaml)"
    cat > "$LIMA_TEMPLATE" <<EOF
base: template://ubuntu-lts
cpus: ${LIMA_CPUS}
memory: ${LIMA_MEMORY}
disk: ${LIMA_DISK}
containerd:
  user: false
EOF
    run_host limactl start --name "$LIMA_NAME" "$LIMA_TEMPLATE"
    rm -f "$LIMA_TEMPLATE"
else
    run_host limactl start "$LIMA_NAME" 2>/dev/null || true
fi

# ---- Guest deps ----
log "Installing guest dependencies (idempotent)..."
run_guest "
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive
sudo apt-get update -qq
sudo apt-get install -y -qq \
    qemu-system qemu-utils libvirt-daemon-system libvirt-clients \
    bridge-utils iproute2 cloud-image-utils tmux curl netcat-openbsd \
    golang-go
# Install Docker CE from official script (idempotent, works on any Ubuntu version)
if ! command -v docker >/dev/null 2>&1; then
    curl -fsSL https://get.docker.com | sudo sh
fi
sudo systemctl enable --now libvirtd docker
sudo virsh net-autostart default 2>/dev/null || true
sudo virsh net-start default 2>/dev/null || true
sudo usermod -aG docker \$(whoami) || true
"

GUEST_REPO="${REPO_ROOT}"

# ---- Docker Compose ----
log "Starting Docker Compose services (Redpanda, Elasticsearch, Kibana, weather-producer)..."
run_guest "
set -euo pipefail
cd ${GUEST_REPO}/demo
sudo docker compose up -d
echo 'Waiting for Elasticsearch...'
for i in \$(seq 1 36); do
    curl -sf http://localhost:9200 >/dev/null 2>&1 && echo 'Elasticsearch ready.' && break
    echo \"Waiting for ES... (\${i}/36)\"
    sleep 5
done
echo 'Waiting for Kibana...'
for i in \$(seq 1 36); do
    curl -sf http://localhost:5601/api/status | grep -q '\"level\":\"available\"' 2>/dev/null && echo 'Kibana ready.' && break
    echo \"Waiting for Kibana... (\${i}/36)\"
    sleep 5
done
bash ${GUEST_REPO}/demo/kibana/setup-dashboard.sh http://localhost:5601
"

# ---- Detect guest architecture ----
if [ "$DRY_RUN" -eq 1 ]; then
    GUEST_ARCH="amd64"
else
    ARCH_OUT=$(limactl shell "$LIMA_NAME" -- uname -m 2>/dev/null || echo "x86_64")
    GUEST_ARCH="amd64"
    [[ "$ARCH_OUT" == "aarch64" ]] && GUEST_ARCH="arm64"
fi
log "Guest architecture: ${GUEST_ARCH}"

# ---- Source VM image ----
log "Checking Logstash source VM image..."
run_guest "
if [ -f '${SOURCE_IMAGE_PATH}' ]; then
    echo 'Source image already exists, skipping prepare-source.sh.'
else
    sudo mkdir -p \$(dirname '${SOURCE_IMAGE_PATH}')
    sudo chown \$(id -u):\$(id -g) \$(dirname '${SOURCE_IMAGE_PATH}')
    echo 'Preparing Logstash source VM image (~10 min)...'
    bash ${GUEST_REPO}/demo/prepare-source.sh \
        --repo-root ${GUEST_REPO} \
        --output ${SOURCE_IMAGE_PATH} \
        --arch ${GUEST_ARCH}
fi
"

# ---- microVM kernel + initrd ----
log "Downloading microVM kernel and initrd assets..."
run_guest "
sudo mkdir -p ${ASSETS_DIR}
sudo chown \$(id -u):\$(id -g) ${ASSETS_DIR}
bash ${GUEST_REPO}/scripts/download-microvm-assets.sh \
    --arch ${GUEST_ARCH} \
    --output-dir ${ASSETS_DIR}
"

# Compute asset paths directly (download-microvm-assets.sh uses deterministic names)
KERNEL_PATH="${ASSETS_DIR}/ubuntu-24.04-server-cloudimg-${GUEST_ARCH}-vmlinuz-generic"
INITRD_PATH="${ASSETS_DIR}/ubuntu-24.04-server-cloudimg-${GUEST_ARCH}-initrd-generic"
IMAGES_DIR="$(dirname "${SOURCE_IMAGE_PATH}")"
QEMU_BINARY="qemu-system-x86_64"
[ "${GUEST_ARCH}" = "arm64" ] && QEMU_BINARY="qemu-system-aarch64"

# ---- deer-daemon config ----
log "Writing deer-daemon config..."
run_guest "
sudo mkdir -p ${DAEMON_WORKDIR} ${IMAGES_DIR} ${ASSETS_DIR}
sudo chown -R \$(id -u):\$(id -g) \$(dirname ${DAEMON_WORKDIR}) ${IMAGES_DIR} ${ASSETS_DIR}
sudo tee ${DAEMON_CONFIG} > /dev/null <<DAEMONYAML
daemon:
  listen_addr: ':9091'
  enabled: true

provider: microvm

microvm:
  qemu_binary: ${QEMU_BINARY}
  accel: tcg
  kernel_path: ${KERNEL_PATH}
  initrd_path: ${INITRD_PATH}
  root_device: /dev/vda1
  work_dir: ${DAEMON_WORKDIR}/overlays
  default_vcpus: 2
  default_memory_mb: 2048
  ip_discovery_timeout: 2m
  readiness_timeout: 15m

network:
  default_bridge: virbr0
  dhcp_mode: arp

image:
  base_dir: ${IMAGES_DIR}

state:
  db_path: ${DAEMON_WORKDIR}/state.db

ssh:
  ca_key_path: /var/lib/deer-demo/ssh_ca
  ca_pub_key_path: /var/lib/deer-demo/ssh_ca.pub
  key_dir: ${DAEMON_WORKDIR}/keys
  default_user: sandbox
  identity_file: /var/lib/deer-demo/identity

libvirt:
  uri: qemu:///system
  network: default

janitor:
  interval: 1m
  default_ttl: 24h
DAEMONYAML
echo 'Daemon config written to ${DAEMON_CONFIG}'
"

# ---- Build and start deer-daemon ----
log "Building and starting deer-daemon inside Lima..."
run_guest "
set -euo pipefail
mkdir -p /var/lib/deer-demo/bin ${DAEMON_WORKDIR}/overlays ${DAEMON_WORKDIR}/keys
cd ${GUEST_REPO}/deer-daemon
GOTOOLCHAIN=auto GOCACHE=/var/tmp/deer-daemon-go-build go build -o /var/lib/deer-demo/bin/deer-daemon ./cmd/deer-daemon
tmux new-session -d -s deer-daemon 2>/dev/null || true
tmux send-keys -t deer-daemon \
    'sudo /var/lib/deer-demo/bin/deer-daemon -config ${DAEMON_CONFIG} 2>&1 | tee ${DAEMON_WORKDIR}/daemon.log' \
    Enter
echo 'Waiting for daemon gRPC port...'
for i in \$(seq 1 20); do
    nc -z localhost 9091 2>/dev/null && echo 'Daemon ready.' && break
    echo \"Waiting for daemon... (\${i}/20)\"
    sleep 3
done
"

log ""
log "============================================================"
log "deer demo is ready!"
log ""
log "  Daemon gRPC:  localhost:9091 (port-forwarded from Lima)"
log "  Kibana:       http://localhost:5601 (port-forwarded from Lima)"
log ""
log "  Connect the deer CLI:"
log "    deer connect localhost:9091"
log "    deer"
log ""
log "  Source VM available: logstash-source"
log "  Try this prompt:"
log "    'Our Logstash pipeline is not ingesting weather data - investigate'"
log "============================================================"
