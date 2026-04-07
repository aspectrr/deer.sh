#!/usr/bin/env bash
# prepare-source.sh
#
# Runs INSIDE Lima. Creates a QCOW2 source image with Logstash 8.x pre-installed
# and the demo pipeline configs in /etc/logstash/conf.d/.
#
# Usage: bash prepare-source.sh --repo-root <path> --output <path> [--arch amd64|arm64]
#
# The output image is used by deer-daemon as the source VM for sandbox cloning.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT=""
OUTPUT_IMAGE=""
ARCH="amd64"
DRY_RUN=0

usage() {
    cat <<'EOF'
Usage: bash prepare-source.sh --repo-root <path> --output <path> [--arch amd64|arm64] [--dry-run]

Options:
  --repo-root <path>   Repository root (where demo/ lives)
  --output <path>      Output QCOW2 image path
  --arch <arch>        Guest architecture: amd64 or arm64 (default: amd64)
  --dry-run            Print commands without executing
  -h, --help           Show help
EOF
}

log()  { printf '[prepare-source] %s\n' "$*"; }
fail() { printf '[prepare-source] ERROR: %s\n' "$*" >&2; exit 1; }

run() {
    if [ "$DRY_RUN" -eq 1 ]; then printf '+'; printf ' %q' "$@"; printf '\n'; return; fi
    "$@"
}

while [ "$#" -gt 0 ]; do
    case "$1" in
        --repo-root) REPO_ROOT="$2"; shift 2 ;;
        --output)    OUTPUT_IMAGE="$2"; shift 2 ;;
        --arch)      ARCH="$2"; shift 2 ;;
        --dry-run)   DRY_RUN=1; shift ;;
        -h|--help)   usage; exit 0 ;;
        *) fail "unknown argument: $1" ;;
    esac
done

[ -n "$REPO_ROOT" ]    || fail "--repo-root is required"
[ -n "$OUTPUT_IMAGE" ] || fail "--output is required"
[ -d "$REPO_ROOT" ]    || fail "repo root not found: $REPO_ROOT"
[ "$ARCH" = "amd64" ] || [ "$ARCH" = "arm64" ] || fail "--arch must be amd64 or arm64, got: $ARCH"

PIPELINE_DIR="${REPO_ROOT}/demo/logstash/pipeline"
LOGSTASH_YML="${REPO_ROOT}/demo/logstash/logstash.yml"
WORKDIR="$(mktemp -d /tmp/deer-prepare-source.XXXXXX)"
trap 'rm -rf "$WORKDIR"' EXIT

BASE_IMAGE_URL="https://cloud-images.ubuntu.com/releases/noble/release/ubuntu-24.04-server-cloudimg-${ARCH}.img"
BASE_IMAGE="${WORKDIR}/base.img"
WORK_DISK="${WORKDIR}/work.qcow2"
CLOUD_INIT_ISO="${WORKDIR}/cloud-init.iso"
SEED_DIR="${WORKDIR}/seed"

log "Downloading base Ubuntu 24.04 image..."
run curl -fsSL --progress-bar -o "$BASE_IMAGE" "$BASE_IMAGE_URL"

log "Creating work overlay..."
run qemu-img create -f qcow2 -F qcow2 -b "$BASE_IMAGE" "$WORK_DISK" 20G

log "Writing cloud-init user-data..."
mkdir -p "$SEED_DIR"

cat > "${SEED_DIR}/meta-data" <<EOF
instance-id: logstash-source-$(date +%s)
local-hostname: logstash-source
EOF

cat > "${SEED_DIR}/network-config" <<'EOF'
version: 2
ethernets:
  id0:
    match: {name: "en*"}
    dhcp4: true
EOF

cat > "${SEED_DIR}/user-data" <<CLOUDINIT
#cloud-config
password: ubuntu
chpasswd: {expire: False}
ssh_pwauth: True

packages:
  - wget
  - gnupg
  - curl
  - python3

write_files:
  - path: /opt/setup-logstash.sh
    permissions: '0755'
    content: |
      #!/bin/bash
      set -euo pipefail
      echo "[setup] Installing Logstash 8.x..."
      wget -qO /usr/share/keyrings/elasticsearch.asc https://artifacts.elastic.co/GPG-KEY-elasticsearch
      echo "deb [signed-by=/usr/share/keyrings/elasticsearch.asc] https://artifacts.elastic.co/packages/8.x/apt stable main" \
          > /etc/apt/sources.list.d/elastic-8.x.list
      apt-get update -qq
      apt-get install -y -qq logstash
      echo "[setup] Installing Kafka input plugin..."
      /usr/share/logstash/bin/logstash-plugin install logstash-input-kafka 2>/dev/null || true
      mkdir -p /etc/logstash/conf.d /var/log/logstash
      chown logstash:logstash /var/log/logstash
      sed -i 's/-Xms1g/-Xms512m/g' /etc/logstash/jvm.options 2>/dev/null || true
      sed -i 's/-Xmx1g/-Xmx512m/g' /etc/logstash/jvm.options 2>/dev/null || true
      echo "[setup] Logstash installed."
      touch /var/lib/cloud/instance/logstash-setup-done

  - path: /etc/logstash/conf.d/01-input-kafka.conf
    content: |
$(sed 's/^/      /' "${PIPELINE_DIR}/01-input-kafka.conf")

  - path: /etc/logstash/conf.d/02-filter-grok.conf
    content: |
$(sed 's/^/      /' "${PIPELINE_DIR}/02-filter-grok.conf")

  - path: /etc/logstash/conf.d/03-filter-date.conf
    content: |
$(sed 's/^/      /' "${PIPELINE_DIR}/03-filter-date.conf")

  - path: /etc/logstash/conf.d/04-filter-mutate.conf
    content: |
$(sed 's/^/      /' "${PIPELINE_DIR}/04-filter-mutate.conf")

  - path: /etc/logstash/conf.d/05-filter-ruby.conf
    content: |
$(sed 's/^/      /' "${PIPELINE_DIR}/05-filter-ruby.conf")

  - path: /etc/logstash/conf.d/06-output-es.conf
    content: |
$(sed 's/^/      /' "${PIPELINE_DIR}/06-output-es.conf")

  - path: /etc/logstash/logstash.yml
    content: |
$(sed 's/^/      /' "${LOGSTASH_YML}")

  - path: /etc/systemd/system/logstash.service.d/override.conf
    content: |
      [Service]
      Environment="LS_JAVA_OPTS=-Xms512m -Xmx512m"

runcmd:
  - /opt/setup-logstash.sh >> /var/log/logstash-setup.log 2>&1
  - systemctl daemon-reload
  - systemctl enable logstash
  - poweroff
CLOUDINIT

log "Building cloud-init ISO..."
run cloud-localds "$CLOUD_INIT_ISO" \
    "${SEED_DIR}/user-data" \
    "${SEED_DIR}/meta-data" \
    --network-config "${SEED_DIR}/network-config"

QEMU_BIN="qemu-system-x86_64"
[ "$ARCH" = "arm64" ] && QEMU_BIN="qemu-system-aarch64"

# Set machine type per arch - do NOT append -machine twice
MACHINE_ARG="type=q35,accel=tcg"
[ "$ARCH" = "arm64" ] && MACHINE_ARG="virt"

log "Booting setup VM (this takes ~10 minutes)..."
log "Output: /var/log/logstash-setup.log inside the VM"

# -serial stdio is omitted: -nographic already redirects the first serial port to stdio.
# Adding both causes "cannot use stdio by multiple character devices" on QEMU 6+.
QEMU_ARGS=(
    -nographic
    -machine "$MACHINE_ARG"
    -cpu max
    -smp 2
    -m 2048
    -drive "file=${WORK_DISK},if=virtio,cache=unsafe"
    -drive "file=${CLOUD_INIT_ISO},if=virtio,format=raw"
    -netdev user,id=net0
    -device virtio-net-pci,netdev=net0
    -no-reboot
)

if [ "$ARCH" = "arm64" ]; then
    QEMU_ARGS+=(-bios /usr/share/qemu-efi-aarch64/QEMU_EFI.fd)
fi

run "$QEMU_BIN" "${QEMU_ARGS[@]}"

log "VM powered off. Converting to final image..."
run qemu-img convert -f qcow2 -O qcow2 "$WORK_DISK" "$OUTPUT_IMAGE"

log "Source image ready: ${OUTPUT_IMAGE}"
if [ "$DRY_RUN" -eq 0 ]; then
    qemu-img info "$OUTPUT_IMAGE"
fi
