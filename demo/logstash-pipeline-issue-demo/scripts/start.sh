#!/usr/bin/env bash
# demo/scripts/start.sh
#
# One-command local demo launcher. Runs on Mac.
# Requires: limactl (brew install lima), Docker Desktop
#
# Architecture:
#   Mac: Docker Compose - Redpanda + Elasticsearch + Kibana
#   Lima VM "deer-source":  Logstash (broken pipeline, for agent inspection)
#   Lima VM "deer-sandbox": deer-daemon + QEMU sandboxes
#
# Usage: ./demo/scripts/start.sh [--repo-root <path>] [--dry-run]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

LIMA_SOURCE="deer-source"
LIMA_SANDBOX="deer-sandbox"
LIMA_SOURCE_CPUS=2
LIMA_SOURCE_MEMORY="4GiB"
LIMA_SOURCE_DISK="20GiB"
LIMA_SANDBOX_CPUS=4
LIMA_SANDBOX_MEMORY="8GiB"
LIMA_SANDBOX_DISK="100GiB"

SOURCE_IMAGE_PATH="/var/lib/deer-demo/logstash-source.qcow2"
DAEMON_WORKDIR="/var/lib/deer-demo/daemon"
DAEMON_CONFIG="/var/lib/deer-demo/daemon.yaml"
ASSETS_DIR="/var/lib/deer-demo/assets"

DEER_CONFIG_DIR="${HOME}/.config/deer"
DEER_CONFIG="${DEER_CONFIG_DIR}/config.yaml"
DEER_KEY_DIR="${DEER_CONFIG_DIR}/keys"
SOURCE_KEY="${DEER_KEY_DIR}/source_ed25519"
SOURCE_PUB_KEY_FILE="${DEER_KEY_DIR}/source_ed25519.pub"

SSH_CONFIG="${HOME}/.ssh/config"

DRY_RUN=0

usage() {
    cat <<'EOF'
Usage: ./demo/scripts/start.sh [--repo-root <path>] [--dry-run]

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

run_source() {
    local cmd="$1"
    if [ "$DRY_RUN" -eq 1 ]; then
        printf '+ limactl shell %s -- bash -lc %q\n' "$LIMA_SOURCE" "$cmd"
        return
    fi
    limactl shell "$LIMA_SOURCE" -- bash -lc "$cmd"
}

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
        *) fail "unknown argument: $1" ;;
    esac
done

# ---- Prerequisites ----
log "Checking prerequisites..."
if [ "$DRY_RUN" -eq 0 ]; then
    command -v limactl >/dev/null 2>&1 || fail "limactl not found. Install with: brew install lima"
    command -v docker  >/dev/null 2>&1 || fail "docker not found. Install Docker Desktop."
    [ -d "$REPO_ROOT" ] || fail "repo root not found: $REPO_ROOT"
fi

# ---- Docker Compose on Mac ----
log "Starting Docker Compose services on Mac (Redpanda, Elasticsearch, Kibana, weather-producer)..."
run_host docker compose -f "${REPO_ROOT}/demo/docker-compose.yml" up -d

if [ "$DRY_RUN" -eq 0 ]; then
    log "Waiting for Elasticsearch..."
    for i in $(seq 1 36); do
        curl -sf http://localhost:9200 >/dev/null 2>&1 && log "Elasticsearch ready." && break
        log "Waiting for ES... (${i}/36)"
        sleep 5
    done

    log "Waiting for Kibana..."
    for i in $(seq 1 36); do
        curl -sf http://localhost:5601/api/status 2>/dev/null | grep -q '"level":"available"' && log "Kibana ready." && break
        log "Waiting for Kibana... (${i}/36)"
        sleep 5
    done

    log "Setting up Kibana dashboard..."
    bash "${REPO_ROOT}/demo/kibana/setup-dashboard.sh" http://localhost:5601
fi

# ---- Lima VM "deer-source" ----
log "Ensuring Lima VM '${LIMA_SOURCE}' (${LIMA_SOURCE_CPUS} CPU, ${LIMA_SOURCE_MEMORY})..."
LIMA_HOME="${LIMA_HOME:-$HOME/.lima}"
SOURCE_CONFIG_PATH="${LIMA_HOME}/${LIMA_SOURCE}/lima.yaml"

if [ "$DRY_RUN" -eq 0 ] && [ ! -f "$SOURCE_CONFIG_PATH" ]; then
    log "Creating Lima VM '${LIMA_SOURCE}'..."
    SOURCE_TEMPLATE="$(mktemp /tmp/${LIMA_SOURCE}.XXXXXX.yaml)"
    cat > "$SOURCE_TEMPLATE" <<EOF
base: template://ubuntu-lts
cpus: ${LIMA_SOURCE_CPUS}
memory: ${LIMA_SOURCE_MEMORY}
disk: ${LIMA_SOURCE_DISK}
containerd:
  user: false
EOF
    run_host limactl start --name "$LIMA_SOURCE" "$SOURCE_TEMPLATE"
    rm -f "$SOURCE_TEMPLATE"
else
    run_host limactl start "$LIMA_SOURCE" 2>/dev/null || true
fi

# ---- Provision Logstash on Lima VM "deer-source" ----
log "Installing and configuring Logstash on '${LIMA_SOURCE}'..."
run_source "
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive

# Install Logstash 8.x
if ! command -v logstash >/dev/null 2>&1; then
    echo '[source] Installing Logstash 8.x...'
    sudo wget -qO /usr/share/keyrings/elasticsearch.asc https://artifacts.elastic.co/GPG-KEY-elasticsearch
    echo 'deb [signed-by=/usr/share/keyrings/elasticsearch.asc] https://artifacts.elastic.co/packages/8.x/apt stable main' \
        | sudo tee /etc/apt/sources.list.d/elastic-8.x.list > /dev/null
    sudo apt-get update -qq
    sudo apt-get install -y -qq logstash

    echo '[source] Installing Kafka input plugin...'
    sudo /usr/share/logstash/bin/logstash-plugin install logstash-input-kafka 2>/dev/null || true
fi

# Write pipeline configs from repo (Lima shares host filesystem)
sudo mkdir -p /etc/logstash/conf.d /var/log/logstash /usr/share/logstash/data
sudo chown logstash:logstash /var/log/logstash /usr/share/logstash/data 2>/dev/null || true

for conf in ${REPO_ROOT}/demo/logstash/pipeline/*.conf; do
    sudo cp \"\$conf\" /etc/logstash/conf.d/
done

sudo cp ${REPO_ROOT}/demo/logstash/station_timezones.csv /etc/logstash/station_timezones.csv

# Override Kafka bootstrap: use Redpanda EXTERNAL listener on Mac host
# host.lima.internal resolves to Mac host IP from inside Lima
sudo sed -i 's|127.0.0.1:9092|host.lima.internal:9093|g' /etc/logstash/conf.d/01-input-kafka.conf

# Override Elasticsearch output: use Mac host ES
sudo sed -i 's|192.168.122.1:9200|host.lima.internal:9200|g' /etc/logstash/conf.d/15-output-es.conf

# Write logstash.yml
sudo cp ${REPO_ROOT}/demo/logstash/logstash.yml /etc/logstash/logstash.yml

# Tune JVM memory
sudo mkdir -p /etc/systemd/system/logstash.service.d
printf '[Service]\nEnvironment=\"LS_JAVA_OPTS=-Xms512m -Xmx512m\"\n' | sudo tee /etc/systemd/system/logstash.service.d/override.conf > /dev/null
sudo sed -i 's/-Xms1g/-Xms512m/g' /etc/logstash/jvm.options 2>/dev/null || true
sudo sed -i 's/-Xmx1g/-Xmx512m/g' /etc/logstash/jvm.options 2>/dev/null || true

sudo systemctl daemon-reload
sudo systemctl enable logstash
sudo systemctl restart logstash
echo '[source] Logstash started.'
"

# ---- Set up deer-readonly on Lima VM "deer-source" ----
log "Setting up deer-readonly user on '${LIMA_SOURCE}'..."

# Generate source key pair on Mac if not exists
if [ "$DRY_RUN" -eq 0 ]; then
    mkdir -p "$DEER_KEY_DIR"
    chmod 700 "$DEER_KEY_DIR"
    if [ ! -f "$SOURCE_KEY" ]; then
        ssh-keygen -t ed25519 -f "$SOURCE_KEY" -N "" -C "deer-demo-source" -q
        log "Generated source SSH key at ${SOURCE_KEY}"
    fi
fi

# Write restricted shell and pubkey to temp files in repo dir so Lima can read
# them via the shared host filesystem (avoids bash quoting/expansion issues).
DEER_SHELL_TMP="${REPO_ROOT}/demo/.deer-readonly-shell.tmp"
DEER_PUBKEY_TMP="${REPO_ROOT}/demo/.deer-source-pubkey.tmp"
trap 'rm -f "$DEER_SHELL_TMP" "$DEER_PUBKEY_TMP"' EXIT

if [ "$DRY_RUN" -eq 0 ]; then
    # Extract restricted shell from Go source.
    # The first line is: const RestrictedShellScript = `#!/bin/bash
    # so #!/bin/bash is on the same line as the backtick - print it, then following lines.
    awk '/^const RestrictedShellScript = `/{sub(/^const RestrictedShellScript = `/, ""); print; found=1; next} found && /^`$/{exit} found{print}' \
        "${REPO_ROOT}/deer-cli/internal/readonly/shell.go" > "$DEER_SHELL_TMP"
    cp "$SOURCE_PUB_KEY_FILE" "$DEER_PUBKEY_TMP"
else
    printf '+ [extract restricted shell to %s]\n' "$DEER_SHELL_TMP"
    printf '+ [copy pubkey to %s]\n' "$DEER_PUBKEY_TMP"
fi

if [ "$DRY_RUN" -eq 0 ]; then
    # Lima shares the host repo filesystem - read temp files directly from Lima VM
    limactl shell "$LIMA_SOURCE" -- bash -lc "
set -euo pipefail
sudo cp '${DEER_SHELL_TMP}' /usr/local/bin/deer-readonly-shell
sudo chmod 755 /usr/local/bin/deer-readonly-shell

id deer-readonly >/dev/null 2>&1 || sudo useradd -r -s /usr/local/bin/deer-readonly-shell -m deer-readonly
sudo usermod -s /usr/local/bin/deer-readonly-shell deer-readonly
sudo usermod -a -G systemd-journal deer-readonly 2>/dev/null || true

sudo mkdir -p /home/deer-readonly/.ssh
sudo chmod 700 /home/deer-readonly/.ssh
sudo cp '${DEER_PUBKEY_TMP}' /home/deer-readonly/.ssh/authorized_keys
sudo chmod 600 /home/deer-readonly/.ssh/authorized_keys
sudo chown -R deer-readonly:deer-readonly /home/deer-readonly/.ssh

sudo systemctl restart sshd 2>/dev/null || sudo systemctl restart ssh
echo '[source] deer-readonly user configured.'
"
else
    printf '+ [setup deer-readonly on %s via shared temp files]\n' "$LIMA_SOURCE"
fi

rm -f "$DEER_SHELL_TMP" "$DEER_PUBKEY_TMP"
trap - EXIT

# ---- Configure ~/.ssh/config for logstash-source ----
log "Configuring ~/.ssh/config for logstash-source..."

if [ "$DRY_RUN" -eq 0 ]; then
    # Get Lima VM 1 SSH port (Lima v2 uses "-o Port=N", older used "-p N")
    LIMA_SOURCE_PORT=$(limactl show-ssh "$LIMA_SOURCE" 2>/dev/null | grep -oE 'Port=[0-9]+' | head -1 | cut -d= -f2)
    [ -z "$LIMA_SOURCE_PORT" ] && LIMA_SOURCE_PORT=$(limactl show-ssh "$LIMA_SOURCE" 2>/dev/null | grep -oE ' -p [0-9]+' | awk '{print $2}' | head -1)
    [ -z "$LIMA_SOURCE_PORT" ] && LIMA_SOURCE_PORT="60022"
    log "Lima '${LIMA_SOURCE}' SSH port: ${LIMA_SOURCE_PORT}"

    # Remove old deer-demo-source entry
    if grep -q '# deer-demo-source-start' "$SSH_CONFIG" 2>/dev/null; then
        TMPFILE="$(mktemp)"
        awk '/^# deer-demo-source-start/{skip=1} skip{if(/^# deer-demo-source-end/){skip=0; next}; next} {print}' "$SSH_CONFIG" > "$TMPFILE"
        mv "$TMPFILE" "$SSH_CONFIG"
    fi

    mkdir -p "$(dirname "$SSH_CONFIG")"
    cat >> "$SSH_CONFIG" <<EOF

# deer-demo-source-start
Host logstash-source
  HostName 127.0.0.1
  Port ${LIMA_SOURCE_PORT}
  User deer-readonly
  IdentityFile ${SOURCE_KEY}
  StrictHostKeyChecking no
  UserKnownHostsFile /dev/null
# deer-demo-source-end
EOF
    log "Added 'logstash-source' to ${SSH_CONFIG}"
else
    printf '+ [update %s with logstash-source entry]\n' "$SSH_CONFIG"
    LIMA_SOURCE_PORT="60022"
fi

# ---- Lima VM "deer-sandbox" ----
log "Ensuring Lima VM '${LIMA_SANDBOX}' (${LIMA_SANDBOX_CPUS} CPU, ${LIMA_SANDBOX_MEMORY})..."
SANDBOX_CONFIG_PATH="${LIMA_HOME}/${LIMA_SANDBOX}/lima.yaml"

if [ "$DRY_RUN" -eq 0 ] && [ ! -f "$SANDBOX_CONFIG_PATH" ]; then
    log "Creating Lima VM '${LIMA_SANDBOX}'..."
    SANDBOX_TEMPLATE="$(mktemp /tmp/${LIMA_SANDBOX}.XXXXXX.yaml)"
    cat > "$SANDBOX_TEMPLATE" <<EOF
base: template://ubuntu-lts
cpus: ${LIMA_SANDBOX_CPUS}
memory: ${LIMA_SANDBOX_MEMORY}
disk: ${LIMA_SANDBOX_DISK}
containerd:
  user: false
EOF
    run_host limactl start --name "$LIMA_SANDBOX" "$SANDBOX_TEMPLATE"
    rm -f "$SANDBOX_TEMPLATE"
else
    run_host limactl start "$LIMA_SANDBOX" 2>/dev/null || true
fi

# ---- Guest deps for deer-sandbox (no Docker needed) ----
log "Installing guest dependencies on '${LIMA_SANDBOX}' (idempotent)..."
run_sandbox "
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive
sudo apt-get update -qq
sudo apt-get install -y -qq \
    qemu-system qemu-utils libvirt-daemon-system libvirt-clients \
    bridge-utils iproute2 cloud-image-utils tmux curl netcat-openbsd \
    golang-go socat
sudo systemctl enable --now libvirtd
sudo virsh net-autostart default 2>/dev/null || true
sudo virsh net-start default 2>/dev/null || true
"

# ---- Elasticsearch proxy in deer-sandbox ----
# QEMU sandbox VMs send to 192.168.122.1:9200 (virbr0 gateway).
# Proxy that address to Elasticsearch on Mac at host.lima.internal:9200.
log "Setting up Elasticsearch proxy on '${LIMA_SANDBOX}' (192.168.122.1:9200 -> Mac ES)..."
run_sandbox "
set -euo pipefail
ES_HOST=\$(getent hosts host.lima.internal | awk '{print \$1}' | head -1)
[ -z \"\$ES_HOST\" ] && { echo 'ERROR: host.lima.internal not resolving'; exit 1; }

sudo tee /etc/systemd/system/deer-es-proxy.service > /dev/null <<UNIT
[Unit]
Description=Elasticsearch proxy for sandbox VMs (192.168.122.1:9200 -> Mac ES)
After=network.target libvirtd.service

[Service]
ExecStart=/usr/bin/socat TCP-LISTEN:9200,bind=192.168.122.1,reuseaddr,fork TCP:\${ES_HOST}:9200
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
UNIT

sudo systemctl daemon-reload
sudo systemctl enable --now deer-es-proxy
echo 'ES proxy started: 192.168.122.1:9200 -> '\${ES_HOST}':9200'
"

# ---- Detect guest architecture ----
if [ "$DRY_RUN" -eq 1 ]; then
    GUEST_ARCH="amd64"
else
    ARCH_OUT=$(limactl shell "$LIMA_SANDBOX" -- uname -m 2>/dev/null || echo "x86_64")
    GUEST_ARCH="amd64"
    [[ "$ARCH_OUT" == "aarch64" ]] && GUEST_ARCH="arm64"
fi
log "Sandbox guest architecture: ${GUEST_ARCH}"

# ---- Source VM image ----
log "Checking Logstash source VM image on '${LIMA_SANDBOX}'..."
run_sandbox "
if [ -f '${SOURCE_IMAGE_PATH}' ]; then
    echo 'Source image already exists, skipping prepare-source.sh.'
else
    sudo mkdir -p \$(dirname '${SOURCE_IMAGE_PATH}')
    sudo chown \$(id -u):\$(id -g) \$(dirname '${SOURCE_IMAGE_PATH}')
    echo 'Preparing Logstash source VM image (~10 min)...'
    bash ${REPO_ROOT}/demo/prepare-source.sh \
        --repo-root ${REPO_ROOT} \
        --output ${SOURCE_IMAGE_PATH} \
        --arch ${GUEST_ARCH}
fi
"

# ---- microVM kernel + initrd ----
log "Downloading microVM kernel and initrd assets on '${LIMA_SANDBOX}'..."
run_sandbox "
sudo mkdir -p ${ASSETS_DIR}
sudo chown \$(id -u):\$(id -g) ${ASSETS_DIR}
bash ${REPO_ROOT}/scripts/download-microvm-assets.sh \
    --arch ${GUEST_ARCH} \
    --output-dir ${ASSETS_DIR}
"

KERNEL_PATH="${ASSETS_DIR}/ubuntu-24.04-server-cloudimg-${GUEST_ARCH}-vmlinuz-generic"
INITRD_PATH="${ASSETS_DIR}/ubuntu-24.04-server-cloudimg-${GUEST_ARCH}-initrd-generic"
IMAGES_DIR="$(dirname "${SOURCE_IMAGE_PATH}")"
QEMU_BINARY="qemu-system-x86_64"
[ "${GUEST_ARCH}" = "arm64" ] && QEMU_BINARY="qemu-system-aarch64"

# ---- deer-daemon config ----
log "Writing deer-daemon config on '${LIMA_SANDBOX}'..."
run_sandbox "
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

# ---- Generate SSH CA and identity keys ----
log "Generating SSH CA and identity keys on '${LIMA_SANDBOX}'..."
run_sandbox "
set -euo pipefail
mkdir -p /var/lib/deer-demo
[ -f /var/lib/deer-demo/ssh_ca ] || ssh-keygen -t ed25519 -f /var/lib/deer-demo/ssh_ca -N '' -C 'deer-sandbox-ca' -q
[ -f /var/lib/deer-demo/identity ] || ssh-keygen -t ed25519 -f /var/lib/deer-demo/identity -N '' -C 'deer-sandbox-identity' -q
echo 'SSH CA and identity keys ready.'
"

# ---- Build and start deer-daemon ----
log "Building and starting deer-daemon on '${LIMA_SANDBOX}'..."
run_sandbox "
set -euo pipefail
mkdir -p /var/lib/deer-demo/bin ${DAEMON_WORKDIR}/overlays ${DAEMON_WORKDIR}/keys
cd ${REPO_ROOT}/deer-daemon
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

# ---- Update ~/.config/deer/config.yaml ----
log "Updating deer CLI config with logstash-source host..."
if [ "$DRY_RUN" -eq 0 ]; then
    mkdir -p "$DEER_CONFIG_DIR"
    python3 - <<PYEOF
import yaml, os

cfg_path = '${DEER_CONFIG}'
cfg = {}
if os.path.exists(cfg_path):
    with open(cfg_path) as f:
        cfg = yaml.safe_load(f) or {}

hosts = [h for h in cfg.get('hosts', []) if h.get('name') != 'logstash-source']
hosts.append({
    'name': 'logstash-source',
    'address': '127.0.0.1',
    'ssh_user': 'deer-readonly',
    'ssh_port': int('${LIMA_SOURCE_PORT}'),
    'ssh_key_path': '${SOURCE_KEY}',
    'prepared': True,
})
cfg['hosts'] = hosts

with open(cfg_path, 'w') as f:
    yaml.dump(cfg, f, default_flow_style=False, allow_unicode=True)
print('deer config updated: logstash-source host added.')
PYEOF
else
    printf '+ [update %s with logstash-source host]\n' "$DEER_CONFIG"
fi

# ---- Summary ----
log ""
log "============================================================"
log "deer demo is ready!"
log ""
log "  Daemon gRPC:  localhost:9091 (port-forwarded from ${LIMA_SANDBOX})"
log "  Kibana:       http://localhost:5601 (Mac Docker)"
log "  Source host:  logstash-source (Lima VM '${LIMA_SOURCE}', deer-readonly SSH)"
log ""
log "  Build and connect the deer CLI:"
log "    cd deer-cli && make build"
log "    ./bin/deer connect localhost:9091 --insecure"
log "    ./bin/deer"
log ""
log "  Source host registered: logstash-source"
log "  Try this prompt:"
log "    'Our Logstash pipeline is running and data is flowing to Elasticsearch,"
log "     but all weather events have UTC timestamps instead of local station time."
log "     Can you investigate why the timezone enrichment is not working?'"
log "============================================================"
