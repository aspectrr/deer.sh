#!/usr/bin/env bash
# prepare-source.sh
#
# Creates a QCOW2 source image with Logstash 8.x pre-installed
# and the demo pipeline configs in /etc/logstash/conf.d/.
#
# Optionally starts Docker Compose (Redpanda + ES) so Logstash can process
# real data during the build, seeding realistic error logs into the image.
#
# Usage: bash prepare-source.sh --repo-root <path> --output <path> [--arch amd64|arm64] [--with-docker]
#
# The output image is used by deer-daemon as the source VM for sandbox cloning.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT=""
OUTPUT_IMAGE=""
ARCH="amd64"
DRY_RUN=0
WITH_DOCKER=0

usage() {
    cat <<'EOF'
Usage: bash prepare-source.sh --repo-root <path> --output <path> [--arch amd64|arm64] [--with-docker] [--dry-run]

Options:
  --repo-root <path>   Repository root (where demo/ lives)
  --output <path>      Output QCOW2 image path
  --arch <arch>        Guest architecture: amd64 or arm64 (default: amd64)
  --with-docker        Start Docker Compose so Logstash generates real logs
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
        --repo-root)  REPO_ROOT="$2"; shift 2 ;;
        --output)     OUTPUT_IMAGE="$2"; shift 2 ;;
        --arch)       ARCH="$2"; shift 2 ;;
        --with-docker) WITH_DOCKER=1; shift ;;
        --dry-run)    DRY_RUN=1; shift ;;
        -h|--help)    usage; exit 0 ;;
        *) fail "unknown argument: $1" ;;
    esac
done

[ -n "$REPO_ROOT" ]    || fail "--repo-root is required"
[ -n "$OUTPUT_IMAGE" ] || fail "--output is required"
[ -d "$REPO_ROOT" ]    || fail "repo root not found: $REPO_ROOT"
[ "$ARCH" = "amd64" ] || [ "$ARCH" = "arm64" ] || fail "--arch must be amd64 or arm64, got: $ARCH"

PIPELINE_DIR="${REPO_ROOT}/demo/logstash-pipeline-issue-demo/logstash/pipeline"
LOGSTASH_YML="${REPO_ROOT}/demo/logstash-pipeline-issue-demo/logstash/logstash.yml"
COMPOSE_FILE="${REPO_ROOT}/demo/logstash-pipeline-issue-demo/docker-compose.yml"
WORKDIR="$(mktemp -d /tmp/deer-prepare-source.XXXXXX)"
trap 'rm -rf "$WORKDIR"' EXIT

BASE_IMAGE_URL="https://cloud-images.ubuntu.com/releases/noble/release/ubuntu-24.04-server-cloudimg-${ARCH}.img"
BASE_IMAGE="${WORKDIR}/base.img"
WORK_DISK="${WORKDIR}/work.qcow2"
CLOUD_INIT_ISO="${WORKDIR}/cloud-init.iso"
SEED_DIR="${WORKDIR}/seed"

# QEMU user-mode networking gateway (host as seen from guest)
QEMU_HOST_IP="10.0.2.2"

# ---- Optionally start Docker Compose ----

if [ "$WITH_DOCKER" -eq 1 ]; then
    log "Starting Docker Compose (Redpanda, Elasticsearch, weather-producer)..."
    export HOST_IP="127.0.0.1"
    run docker compose -f "$COMPOSE_FILE" up -d

    log "Waiting for Redpanda to be healthy..."
    for i in $(seq 1 30); do
        docker compose -f "$COMPOSE_FILE" exec redpanda rpk cluster info --brokers=localhost:9092 >/dev/null 2>&1 && break
        log "  waiting... (${i}/30)"
        sleep 5
    done

    log "Waiting for Elasticsearch..."
    for i in $(seq 1 30); do
        curl -sf http://localhost:9200 >/dev/null 2>&1 && break
        log "  waiting... (${i}/30)"
        sleep 5
    done

    # Patch pipeline configs to point at QEMU gateway IP
    log "Patching pipeline configs to use QEMU host IP ${QEMU_HOST_IP}..."
    PATCHED_DIR="${WORKDIR}/patched-pipeline"
    mkdir -p "$PATCHED_DIR"
    for conf in "${PIPELINE_DIR}"/*.conf; do
        fname="$(basename "$conf")"
        sed \
            -e "s|127\.0\.0\.1:9092|${QEMU_HOST_IP}:9092|g" \
            -e "s|127\.0\.0\.1:9093|${QEMU_HOST_IP}:9093|g" \
            -e "s|192\.168\.[0-9]*\.[0-9]*:9200|${QEMU_HOST_IP}:9200|g" \
            -e "s|127\.0\.0\.1:9200|${QEMU_HOST_IP}:9200|g" \
            "$conf" > "${PATCHED_DIR}/${fname}"
    done
    PIPELINE_DIR="$PATCHED_DIR"
fi

# ---- Download base image ----

log "Downloading base Ubuntu 24.04 image..."
run curl -fsSL --progress-bar -o "$BASE_IMAGE" "$BASE_IMAGE_URL"

log "Creating work overlay..."
run qemu-img create -f qcow2 -F qcow2 -b "$BASE_IMAGE" "$WORK_DISK" 20G

# ---- Cloud-init ----

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

# Pre-compute pipeline write_files for cloud-init
PIPELINE_WRITE_FILES=""
for conf in "${PIPELINE_DIR}"/*.conf; do
    fname="$(basename "$conf")"
    PIPELINE_WRITE_FILES+="  - path: /etc/logstash/conf.d/${fname}
    content: |
$(sed 's/^/      /' "$conf")

"
done

CSV_B64="$(base64 -i "${REPO_ROOT}/demo/logstash-pipeline-issue-demo/logstash/station_timezones.csv" | tr -d '\n')"

# Determine how long to run Logstash
if [ "$WITH_DOCKER" -eq 1 ]; then
    LOGSTASH_RUN_TIME=60
else
    LOGSTASH_RUN_TIME=0
fi

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
      apt-get install -y -qq apt-transport-https wget gnupg
      wget -qO - https://artifacts.elastic.co/GPG-KEY-elasticsearch | gpg --batch --yes --dearmor -o /usr/share/keyrings/elasticsearch-keyring.gpg
      echo "deb [signed-by=/usr/share/keyrings/elasticsearch-keyring.gpg] https://artifacts.elastic.co/packages/8.x/apt stable main" \
          > /etc/apt/sources.list.d/elastic-8.x.list
      apt-get update -qq
      apt-get install -y logstash
      echo "[setup] Installing Kafka input plugin..."
      /usr/share/logstash/bin/logstash-plugin install logstash-input-kafka 2>/dev/null || true
      mkdir -p /etc/logstash/conf.d /var/log/logstash
      chown -R logstash:logstash /etc/logstash /var/log/logstash /usr/share/logstash/data
      sed -i 's/-Xms1g/-Xms512m/g' /etc/logstash/jvm.options 2>/dev/null || true
      sed -i 's/-Xmx1g/-Xmx512m/g' /etc/logstash/jvm.options 2>/dev/null || true
      echo "[setup] Logstash installed."
      touch /var/lib/cloud/instance/logstash-setup-done

${PIPELINE_WRITE_FILES}
  - path: /etc/logstash/station_timezones.csv
    encoding: b64
    content: ${CSV_B64}

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
  - systemctl start logstash
  - sleep ${LOGSTASH_RUN_TIME}
  - systemctl stop logstash
  - chown -R logstash:logstash /var/log/logstash /var/lib/logstash
  - poweroff
CLOUDINIT

log "Building cloud-init ISO..."
SEED_TMP="${WORKDIR}/iso_seed"
mkdir -p "$SEED_TMP"
printf '%s' "$(cat "${SEED_DIR}/user-data")" > "${SEED_TMP}/user-data"
printf '%s' "$(cat "${SEED_DIR}/meta-data")" > "${SEED_TMP}/meta-data"
printf 'version: 2\nethernets:\n  id0:\n    match: {name: "en*"}\n    dhcp4: true\n' > "${SEED_TMP}/network-config"

if command -v mkisofs >/dev/null 2>&1; then
    run mkisofs -output "$CLOUD_INIT_ISO" -volid "cidata" -joliet -rock "$SEED_TMP"
elif command -v hdiutil >/dev/null 2>&1; then
    run hdiutil makehybrid -o "$CLOUD_INIT_ISO" -hfs -iso -joliet -default-volume-name cidata "$SEED_TMP"
else
    fail "Neither mkisofs nor hdiutil found. Install with: brew install cdrtools"
fi

# ---- Boot QEMU ----

QEMU_BIN="qemu-system-x86_64"
[ "$ARCH" = "arm64" ] && QEMU_BIN="qemu-system-aarch64"

MACHINE_ARG="type=q35,accel=tcg"
[ "$ARCH" = "arm64" ] && MACHINE_ARG="virt"

if [ "$WITH_DOCKER" -eq 1 ]; then
    log "Booting setup VM with Docker services (this takes ~12 minutes)..."
    log "Logstash will process real data for ${LOGSTASH_RUN_TIME}s to seed logs."
else
    log "Booting setup VM (this takes ~10 minutes)..."
fi
log "Setup log: /var/log/logstash-setup.log inside the VM"

if [ "$WITH_DOCKER" -eq 1 ]; then
    NETDEV="user,id=net0"
else
    NETDEV="user,id=net0"
fi

QEMU_ARGS=(
    -nographic
    -machine "$MACHINE_ARG"
    -cpu max
    -smp 2
    -m 2048
    -drive "file=${WORK_DISK},if=virtio,cache=unsafe"
    -drive "file=${CLOUD_INIT_ISO},if=virtio,format=raw"
    -netdev "$NETDEV"
    -device virtio-net-pci,netdev=net0
    -no-reboot
)

if [ "$ARCH" = "arm64" ]; then
    if [ -f /opt/homebrew/share/qemu/edk2-aarch64-code.fd ]; then
        QEMU_ARGS+=(-bios /opt/homebrew/share/qemu/edk2-aarch64-code.fd)
    elif [ -f /usr/share/qemu-efi-aarch64/QEMU_EFI.fd ]; then
        QEMU_ARGS+=(-bios /usr/share/qemu-efi-aarch64/QEMU_EFI.fd)
    else
        fail "No arm64 EFI firmware found. Install qemu (brew install qemu) or qemu-efi-aarch64"
    fi
fi

run "$QEMU_BIN" "${QEMU_ARGS[@]}"

# ---- Teardown Docker ----

if [ "$WITH_DOCKER" -eq 1 ]; then
    log "Stopping Docker Compose..."
    run docker compose -f "$COMPOSE_FILE" down
fi

# ---- Finalize image ----

log "VM powered off. Converting to final image..."
run qemu-img convert -f qcow2 -O qcow2 "$WORK_DISK" "$OUTPUT_IMAGE"

log "Source image ready: ${OUTPUT_IMAGE}"
if [ "$DRY_RUN" -eq 0 ]; then
    qemu-img info "$OUTPUT_IMAGE"
fi
