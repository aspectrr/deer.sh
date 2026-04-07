#!/bin/bash
# setup-kafka-demo.sh
#
# Sets up a Kafka + Logstash demo environment on Ubuntu.
# Creates two VMs on the libvirt default network:
#
#   kafka-demo-{INDEX}    Apache Kafka 3.7.1 (KRaft mode) + weather data producer
#   logstash-demo-{INDEX} Logstash 8.x with a 6-stage pipeline containing a grok parsing bug
#
# The weather producer fetches live data from Open-Meteo API (no key required)
# and produces structured log lines to the 'weather-logs' Kafka topic every 30s.
#
# The Logstash pipeline has an intentional grok bug: the grok pattern omits the
# brackets (\[ and \]) around the ISO8601 timestamp, causing _grokparsefailure
# on every event. All downstream enrichment filters are guarded on parse success,
# so the bug silently drops all data enrichment.
#
# Log line format produced by weather-producer.sh:
#   [2026-03-28T15:30:00Z] WEATHER location="Berlin" lat=52.52 lon=13.41 temp=8.3 wind_speed=14.2 wind_dir=270 weather_code=3 is_day=1
#
# Usage: sudo ./setup-kafka-demo.sh [VM_INDEX] [--ssh-users-file <path>]
#
# Options:
#   VM_INDEX                  VM index number (default: 1)
#   --ssh-users-file <path>   Path to file with SSH users (one per line: <username> <public-key>)

VM_INDEX=""
SSH_USERS_FILE=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --ssh-users-file)
            SSH_USERS_FILE="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: sudo ./setup-kafka-demo.sh [VM_INDEX] [--ssh-users-file <path>]"
            echo ""
            echo "Options:"
            echo "  VM_INDEX                  VM index number (default: 1)"
            echo "  --ssh-users-file <path>   Path to file with SSH users (one per line: <username> <public-key>)"
            exit 0
            ;;
        *)
            if [[ -z "$VM_INDEX" ]]; then
                VM_INDEX="$1"
            else
                echo "Unknown argument: $1" >&2
                exit 1
            fi
            shift
            ;;
    esac
done

VM_INDEX="${VM_INDEX:-1}"

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info()    { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn()    { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $1" >&2; }

if [[ $EUID -ne 0 ]]; then
    log_error "This script must be run as root"
    exit 1
fi

if ! command -v virsh &>/dev/null || ! command -v virt-install &>/dev/null; then
    log_error "Required commands (virsh, virt-install) not found."
    log_error "Please run setup-ubuntu.sh first to install dependencies."
    exit 1
fi

# ============================================================================
# Ensure default network is active
# ============================================================================
log_info "Ensuring default network is active..."
if ! virsh net-info default &>/dev/null; then
    virsh net-define /usr/share/libvirt/networks/default.xml || true
fi
if ! virsh net-list | grep -q "default.*active"; then
    virsh net-start default || true
    virsh net-autostart default || true
fi
log_success "Default network is active."

IMAGE_DIR="/var/lib/libvirt/images"
CLOUD_INIT_DIR="${IMAGE_DIR}/cloud-init"
BASE_IMAGE="ubuntu-22.04-minimal-cloudimg-amd64.img"
BASE_IMAGE_URL="https://cloud-images.ubuntu.com/minimal/releases/jammy/release/${BASE_IMAGE}"
BASE_IMAGE_PATH="${IMAGE_DIR}/${BASE_IMAGE}"

mkdir -p "$IMAGE_DIR" "$CLOUD_INIT_DIR"

if [[ ! -f "$BASE_IMAGE_PATH" ]]; then
    log_info "Downloading Ubuntu Minimal Cloud Image (approx 300MB)..."
    wget -q --show-progress -O "$BASE_IMAGE_PATH" "$BASE_IMAGE_URL"
    log_success "Image downloaded."
else
    log_info "Base image already exists at $BASE_IMAGE_PATH"
fi

# Add SSH public keys to KVM host for proxy jump access
if [[ -n "$SSH_USERS_FILE" ]] && [[ -f "$SSH_USERS_FILE" ]]; then
    log_info "Adding SSH public keys to KVM host authorized_keys..."
    HOST_SSH_DIR="/root/.ssh"
    mkdir -p "$HOST_SSH_DIR"
    chmod 700 "$HOST_SSH_DIR"
    touch "$HOST_SSH_DIR/authorized_keys"
    chmod 600 "$HOST_SSH_DIR/authorized_keys"
    while IFS= read -r line || [[ -n "$line" ]]; do
        [[ -z "$line" ]] && continue
        [[ "$line" =~ ^#.*$ ]] && continue
        pubkey="${line#* }"
        if ! grep -qF "$pubkey" "$HOST_SSH_DIR/authorized_keys"; then
            echo "$pubkey" >> "$HOST_SSH_DIR/authorized_keys"
            username="${line%% *}"
            log_success "Added key for ${username} to host authorized_keys"
        fi
    done < "$SSH_USERS_FILE"
fi

CREATED_VM_NAMES=()
CREATED_VM_IPS=()
CREATED_VM_MACS=()
CREATED_VM_DISKS=()

# ============================================================================
# write_cloud_init_base: Write meta-data and network-config for a VM
# ============================================================================
write_cloud_init_base() {
    local vm_name="$1"
    local seed_dir="${CLOUD_INIT_DIR}/${vm_name}"
    mkdir -p "$seed_dir"

    cat > "${seed_dir}/meta-data" <<EOF
instance-id: ${vm_name}-$(date +%s)
local-hostname: ${vm_name}
EOF

    cat > "${seed_dir}/network-config" <<'EOF'
version: 2
ethernets:
  id0:
    match:
      name: en*
    dhcp4: true
EOF
}

# ============================================================================
# boot_vm: Boot a VM from existing cloud-init files and wait for IP
#
# Arguments:
#   $1 - vm_name
#   $2 - vm_index (for deterministic MAC)
#   $3 - mac_prefix (e.g. "52:54:02")
#   $4 - memory_mb
#   $5 - vcpus
#   $6 - disk_gb
#
# Returns (via echo): the VM IP address
# ============================================================================
boot_vm() {
    local vm_name="$1"
    local vm_index="$2"
    local mac_prefix="$3"
    local memory_mb="$4"
    local vcpus="$5"
    local disk_gb="$6"

    local vm_disk="${IMAGE_DIR}/${vm_name}.qcow2"
    local seed_dir="${CLOUD_INIT_DIR}/${vm_name}"

    log_info "Creating disk for '${vm_name}' (${disk_gb}GB)..."
    [[ -f "$vm_disk" ]] && rm -f "$vm_disk"
    qemu-img create -f qcow2 -F qcow2 -b "$BASE_IMAGE_PATH" "$vm_disk" "${disk_gb}G"

    local mac_suffix
    mac_suffix=$(printf '%02x:%02x:%02x' $((vm_index / 256 / 256 % 256)) $((vm_index / 256 % 256)) $((vm_index % 256)))
    local mac_address="${mac_prefix}:${mac_suffix}"

    log_info "Booting '${vm_name}' (${memory_mb}MB, ${vcpus} vCPUs, MAC ${mac_address})..."

    virt-install \
        --name "${vm_name}" \
        --memory "${memory_mb}" \
        --vcpus "${vcpus}" \
        --disk "${vm_disk},device=disk,bus=virtio" \
        --cloud-init user-data="${seed_dir}/user-data",meta-data="${seed_dir}/meta-data",network-config="${seed_dir}/network-config" \
        --os-variant ubuntu22.04 \
        --import \
        --noautoconsole \
        --graphics none \
        --console pty,target_type=serial \
        --network network=default,model=virtio,mac="${mac_address}"

    log_success "VM '${vm_name}' started!"

    local max_wait=180
    local wait_interval=5
    local elapsed=0
    local vm_ip=""

    log_info "Waiting for '${vm_name}' to obtain IP address..."
    while [[ $elapsed -lt $max_wait ]]; do
        vm_ip=$(virsh domifaddr "${vm_name}" --source lease 2>/dev/null | grep -oE '([0-9]{1,3}\.){3}[0-9]{1,3}' | head -1 || true)
        if [[ -n "$vm_ip" ]]; then
            log_success "'${vm_name}' IP: ${vm_ip}"
            ssh-keygen -f /root/.ssh/known_hosts -R "$vm_ip" 2>/dev/null || true
            break
        fi
        log_info "Waiting for '${vm_name}' IP... (${elapsed}s / ${max_wait}s)"
        sleep $wait_interval
        elapsed=$((elapsed + wait_interval))
    done

    if [[ -z "$vm_ip" ]]; then
        log_warn "'${vm_name}' did not get IP within ${max_wait}s"
        log_warn "Check: virsh domifaddr ${vm_name} --source lease"
        log_warn "Check: virsh console ${vm_name} (login: ubuntu/ubuntu)"
    fi

    local vm_mac
    vm_mac=$(virsh domiflist "${vm_name}" 2>/dev/null | grep -oE '([0-9a-f]{2}:){5}[0-9a-f]{2}' | head -1 || true)

    CREATED_VM_NAMES+=("$vm_name")
    CREATED_VM_IPS+=("$vm_ip")
    CREATED_VM_MACS+=("$vm_mac")
    CREATED_VM_DISKS+=("$vm_disk")

    echo "$vm_ip"
}

# ============================================================================
# STEP 1: Create kafka-demo VM cloud-init
# ============================================================================
KAFKA_VM="kafka-demo-${VM_INDEX}"
LOGSTASH_VM="logstash-demo-${VM_INDEX}"

log_info "Creating cloud-init for '${KAFKA_VM}'..."
write_cloud_init_base "$KAFKA_VM"
KAFKA_SEED_DIR="${CLOUD_INIT_DIR}/${KAFKA_VM}"

# Write base cloud-config header
cat > "${KAFKA_SEED_DIR}/user-data" <<'ENDOFCLOUD'
#cloud-config
password: ubuntu
chpasswd: { expire: False }
ssh_pwauth: True
ENDOFCLOUD

# Append SSH users if provided
if [[ -n "$SSH_USERS_FILE" ]] && [[ -f "$SSH_USERS_FILE" ]]; then
    echo "" >> "${KAFKA_SEED_DIR}/user-data"
    echo "users:" >> "${KAFKA_SEED_DIR}/user-data"
    echo "  - default" >> "${KAFKA_SEED_DIR}/user-data"
    while IFS= read -r line || [[ -n "$line" ]]; do
        [[ -z "$line" ]] && continue
        [[ "$line" =~ ^#.*$ ]] && continue
        username="${line%% *}"
        pubkey="${line#* }"
        cat >> "${KAFKA_SEED_DIR}/user-data" <<EOF
  - name: ${username}
    sudo: ALL=(ALL) NOPASSWD:ALL
    shell: /bin/bash
    ssh_authorized_keys:
      - ${pubkey}
EOF
    done < "$SSH_USERS_FILE"
fi

# Append packages, write_files, and runcmd
# NOTE: single-quoted heredoc - no shell expansion inside
cat >> "${KAFKA_SEED_DIR}/user-data" <<'ENDOFCLOUD'

packages:
  - qemu-guest-agent
  - openjdk-17-jre-headless
  - curl
  - jq
  - python3

write_files:
  - path: /opt/setup-kafka.sh
    permissions: '0755'
    content: |
      #!/bin/bash
      set -euo pipefail
      KAFKA_VERSION="3.7.1"
      SCALA_VERSION="2.13"
      KAFKA_DIR="/opt/kafka"
      KAFKA_TGZ="/tmp/kafka.tgz"
      KAFKA_LOG_DIR="/var/log/kafka"
      KAFKA_DATA_DIR="/var/lib/kafka/data"

      echo "[kafka-setup] Downloading Kafka ${KAFKA_VERSION}..."
      curl -fsSL \
          "https://downloads.apache.org/kafka/${KAFKA_VERSION}/kafka_${SCALA_VERSION}-${KAFKA_VERSION}.tgz" \
          -o "$KAFKA_TGZ" || \
      curl -fsSL \
          "https://archive.apache.org/dist/kafka/${KAFKA_VERSION}/kafka_${SCALA_VERSION}-${KAFKA_VERSION}.tgz" \
          -o "$KAFKA_TGZ"

      echo "[kafka-setup] Extracting Kafka..."
      tar -xzf "$KAFKA_TGZ" -C /opt/
      ln -sfn "/opt/kafka_${SCALA_VERSION}-${KAFKA_VERSION}" "$KAFKA_DIR"
      rm -f "$KAFKA_TGZ"

      mkdir -p "$KAFKA_LOG_DIR" "$KAFKA_DATA_DIR"
      chown ubuntu:ubuntu "$KAFKA_LOG_DIR"

      cat > "${KAFKA_DIR}/config/kraft/server.properties" <<KAFKACFG
      process.roles=broker,controller
      node.id=1
      controller.quorum.voters=1@localhost:9093
      listeners=PLAINTEXT://:9092,CONTROLLER://:9093
      inter.broker.listener.name=PLAINTEXT
      advertised.listeners=PLAINTEXT://localhost:9092
      controller.listener.names=CONTROLLER
      listener.security.protocol.map=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
      log.dirs=${KAFKA_DATA_DIR}
      num.partitions=3
      default.replication.factor=1
      log.retention.hours=24
      log.retention.bytes=536870912
      KAFKACFG

      # Strip leading whitespace from config (heredoc indentation artifact)
      sed -i 's/^      //' "${KAFKA_DIR}/config/kraft/server.properties"

      CLUSTER_ID=$(${KAFKA_DIR}/bin/kafka-storage.sh random-uuid)
      echo "[kafka-setup] Cluster ID: ${CLUSTER_ID}"
      ${KAFKA_DIR}/bin/kafka-storage.sh format \
          -t "$CLUSTER_ID" \
          -c "${KAFKA_DIR}/config/kraft/server.properties"

      cat > /etc/systemd/system/kafka.service <<SVCEOF
      [Unit]
      Description=Apache Kafka (KRaft)
      After=network.target

      [Service]
      Type=simple
      User=ubuntu
      Environment="JAVA_HOME=/usr/lib/jvm/java-17-openjdk-amd64"
      Environment="LOG_DIR=${KAFKA_LOG_DIR}"
      ExecStart=${KAFKA_DIR}/bin/kafka-server-start.sh ${KAFKA_DIR}/config/kraft/server.properties
      ExecStop=${KAFKA_DIR}/bin/kafka-server-stop.sh
      Restart=on-failure
      RestartSec=10
      StandardOutput=journal
      StandardError=journal

      [Install]
      WantedBy=multi-user.target
      SVCEOF

      sed -i 's/^      //' /etc/systemd/system/kafka.service

      systemctl daemon-reload
      systemctl enable kafka
      systemctl start kafka

      echo "[kafka-setup] Waiting for Kafka broker to be ready..."
      for i in $(seq 1 36); do
          if ${KAFKA_DIR}/bin/kafka-topics.sh \
                  --list \
                  --bootstrap-server localhost:9092 &>/dev/null; then
              echo "[kafka-setup] Kafka is ready."
              break
          fi
          echo "[kafka-setup] Waiting... (${i}/36)"
          sleep 5
      done

      ${KAFKA_DIR}/bin/kafka-topics.sh \
          --create \
          --topic weather-logs \
          --bootstrap-server localhost:9092 \
          --partitions 3 \
          --replication-factor 1 \
          --if-not-exists

      echo "[kafka-setup] Topic 'weather-logs' created."
      chown -R ubuntu:ubuntu "${KAFKA_DIR}" "${KAFKA_DATA_DIR}"

      systemctl enable weather-producer
      systemctl start weather-producer
      echo "[kafka-setup] Done."

  - path: /home/ubuntu/weather-producer.sh
    owner: ubuntu:ubuntu
    permissions: '0755'
    content: |
      #!/bin/bash
      # weather-producer.sh
      #
      # Fetches live weather data from the Open-Meteo API (no API key required)
      # for 8 cities and produces structured log lines to Kafka every 30 seconds.
      #
      # Output log line format:
      #   [ISO8601] WEATHER location="CITY" lat=LAT lon=LON temp=TEMP
      #             wind_speed=WIND wind_dir=DIR weather_code=CODE is_day=0|1
      KAFKA_DIR="/opt/kafka"
      TOPIC="weather-logs"
      BOOTSTRAP="localhost:9092"

      CITIES=(
          "Berlin,52.52,13.41"
          "London,51.51,-0.13"
          "Tokyo,35.69,139.69"
          "NewYork,40.71,-74.01"
          "Sydney,-33.87,151.21"
          "SaoPaulo,-23.55,-46.63"
          "Mumbai,19.08,72.88"
          "Cairo,30.04,31.24"
      )

      log() { echo "[weather-producer] $*" >&2; }

      fetch_weather() {
          local city="$1" lat="$2" lon="$3"
          local response temp wind_speed wind_dir weather_code is_day ts

          response=$(curl -sf --max-time 10 \
              "https://api.open-meteo.com/v1/forecast?latitude=${lat}&longitude=${lon}&current_weather=true" \
              2>/dev/null) || return 1

          temp=$(echo "$response" | python3 -c \
              "import sys,json; d=json.load(sys.stdin)['current_weather']; print(d['temperature'])" \
              2>/dev/null) || return 1
          wind_speed=$(echo "$response" | python3 -c \
              "import sys,json; d=json.load(sys.stdin)['current_weather']; print(d['windspeed'])" \
              2>/dev/null) || return 1
          wind_dir=$(echo "$response" | python3 -c \
              "import sys,json; d=json.load(sys.stdin)['current_weather']; print(d['winddirection'])" \
              2>/dev/null) || return 1
          weather_code=$(echo "$response" | python3 -c \
              "import sys,json; d=json.load(sys.stdin)['current_weather']; print(d['weathercode'])" \
              2>/dev/null) || return 1
          is_day=$(echo "$response" | python3 -c \
              "import sys,json; d=json.load(sys.stdin)['current_weather']; print(d['is_day'])" \
              2>/dev/null) || return 1

          ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
          printf '[%s] WEATHER location="%s" lat=%s lon=%s temp=%s wind_speed=%s wind_dir=%s weather_code=%s is_day=%s\n' \
              "$ts" "$city" "$lat" "$lon" "$temp" "$wind_speed" "$wind_dir" "$weather_code" "$is_day"
      }

      log "Starting. Topic: ${TOPIC} on ${BOOTSTRAP}"

      while true; do
          batch_start=$(date +%s)
          for entry in "${CITIES[@]}"; do
              IFS=',' read -r city lat lon <<< "$entry"
              line=$(fetch_weather "$city" "$lat" "$lon") || {
                  log "Failed to fetch weather for ${city}, skipping"
                  continue
              }
              echo "$line" | \
                  "${KAFKA_DIR}/bin/kafka-console-producer.sh" \
                  --bootstrap-server "${BOOTSTRAP}" \
                  --topic "${TOPIC}" \
                  2>/dev/null && \
              log "Produced: ${line}" || \
              log "Failed to produce for ${city}"
          done
          elapsed=$(( $(date +%s) - batch_start ))
          remaining=$(( 30 - elapsed ))
          [[ $remaining -gt 0 ]] && sleep "$remaining"
      done

  - path: /etc/systemd/system/weather-producer.service
    content: |
      [Unit]
      Description=Weather Data Kafka Producer
      After=network.target kafka.service
      Requires=kafka.service

      [Service]
      Type=simple
      User=ubuntu
      ExecStart=/home/ubuntu/weather-producer.sh
      Restart=on-failure
      RestartSec=15
      StandardOutput=journal
      StandardError=journal

      [Install]
      WantedBy=multi-user.target

runcmd:
  - systemctl enable qemu-guest-agent
  - systemctl start qemu-guest-agent
  - /opt/setup-kafka.sh >> /var/log/kafka-setup.log 2>&1
ENDOFCLOUD

# ============================================================================
# STEP 2: Boot kafka-demo VM and capture IP
# ============================================================================
log_info "Booting '${KAFKA_VM}'..."
KAFKA_IP=$(boot_vm "$KAFKA_VM" "$VM_INDEX" "52:54:02" 4096 2 20)

if [[ -z "$KAFKA_IP" ]]; then
    log_error "Could not determine kafka-demo IP. Logstash will need manual configuration."
    KAFKA_IP="UNKNOWN"
fi

# ============================================================================
# STEP 3: Create logstash-demo VM cloud-init
# ============================================================================
log_info "Creating cloud-init for '${LOGSTASH_VM}' (kafka bootstrap: ${KAFKA_IP})..."
write_cloud_init_base "$LOGSTASH_VM"
LOGSTASH_SEED_DIR="${CLOUD_INIT_DIR}/${LOGSTASH_VM}"

cat > "${LOGSTASH_SEED_DIR}/user-data" <<'ENDOFCLOUD'
#cloud-config
password: ubuntu
chpasswd: { expire: False }
ssh_pwauth: True
ENDOFCLOUD

if [[ -n "$SSH_USERS_FILE" ]] && [[ -f "$SSH_USERS_FILE" ]]; then
    echo "" >> "${LOGSTASH_SEED_DIR}/user-data"
    echo "users:" >> "${LOGSTASH_SEED_DIR}/user-data"
    echo "  - default" >> "${LOGSTASH_SEED_DIR}/user-data"
    while IFS= read -r line || [[ -n "$line" ]]; do
        [[ -z "$line" ]] && continue
        [[ "$line" =~ ^#.*$ ]] && continue
        username="${line%% *}"
        pubkey="${line#* }"
        cat >> "${LOGSTASH_SEED_DIR}/user-data" <<EOF
  - name: ${username}
    sudo: ALL=(ALL) NOPASSWD:ALL
    shell: /bin/bash
    ssh_authorized_keys:
      - ${pubkey}
EOF
    done < "$SSH_USERS_FILE"
fi

# Append the logstash setup. Uses KAFKA_IP_PLACEHOLDER which is sed-replaced below.
cat >> "${LOGSTASH_SEED_DIR}/user-data" <<'ENDOFCLOUD'

packages:
  - qemu-guest-agent
  - wget
  - gnupg

write_files:
  - path: /opt/setup-logstash.sh
    permissions: '0755'
    content: |
      #!/bin/bash
      set -euo pipefail

      echo "[logstash-setup] Installing Logstash 8.x..."
      wget -qO - https://artifacts.elastic.co/GPG-KEY-elasticsearch | apt-key add -
      echo "deb https://artifacts.elastic.co/packages/8.x/apt stable main" \
          > /etc/apt/sources.list.d/elastic-8.x.list
      apt-get update -qq
      apt-get install -y -qq logstash

      echo "[logstash-setup] Installing Kafka input plugin..."
      /usr/share/logstash/bin/logstash-plugin install logstash-input-kafka 2>/dev/null || true

      mkdir -p /etc/logstash/conf.d

      # ------------------------------------------------------------------
      # 01-input-kafka.conf
      # Reads from the 'weather-logs' topic on the kafka-demo VM.
      # ------------------------------------------------------------------
      cat > /etc/logstash/conf.d/01-input-kafka.conf <<'PIPEEOF'
      input {
        kafka {
          bootstrap_servers => "KAFKA_IP_PLACEHOLDER:9092"
          topics            => ["weather-logs"]
          group_id          => "logstash-weather-consumer"
          auto_offset_reset => "earliest"
          decorate_events   => true
          codec => plain {
            charset => "UTF-8"
          }
        }
      }
      PIPEEOF
      sed -i 's/^      //' /etc/logstash/conf.d/01-input-kafka.conf

      # ------------------------------------------------------------------
      # 02-filter-grok.conf
      #
      # BUG: The grok pattern starts with %{TIMESTAMP_ISO8601:event_timestamp}
      # but the actual log line wraps the timestamp in square brackets:
      #   [2026-03-28T15:30:00Z] WEATHER ...
      #
      # Fix: replace the pattern start with:
      #   \[%{TIMESTAMP_ISO8601:event_timestamp}\]
      # ------------------------------------------------------------------
      cat > /etc/logstash/conf.d/02-filter-grok.conf <<'PIPEEOF'
      filter {
        # BUG: Missing \[ and \] around timestamp. The log format is:
        #   [2026-03-28T15:30:00Z] WEATHER location="Berlin" ...
        # but this pattern omits the brackets, causing _grokparsefailure
        # on every event. All downstream filters are guarded on parse success
        # so no enrichment occurs and failures pile up silently.
        #
        # Fix: change the pattern start to:
        #   \[%{TIMESTAMP_ISO8601:event_timestamp}\] WEATHER ...
        grok {
          match => {
            "message" => "%{TIMESTAMP_ISO8601:event_timestamp} WEATHER location=\"%{DATA:location}\" lat=%{NUMBER:latitude:float} lon=%{NUMBER:longitude:float} temp=%{NUMBER:temperature:float} wind_speed=%{NUMBER:wind_speed:float} wind_dir=%{NUMBER:wind_dir:int} weather_code=%{NUMBER:weather_code:int} is_day=%{NUMBER:is_day:int}"
          }
          add_tag      => ["weather_parsed"]
          tag_on_failure => ["_grokparsefailure"]
        }
      }
      PIPEEOF
      sed -i 's/^      //' /etc/logstash/conf.d/02-filter-grok.conf

      # ------------------------------------------------------------------
      # 03-filter-date.conf
      # Parse event_timestamp into @timestamp. Skipped on grok failure.
      # ------------------------------------------------------------------
      cat > /etc/logstash/conf.d/03-filter-date.conf <<'PIPEEOF'
      filter {
        if "weather_parsed" in [tags] {
          date {
            match    => ["event_timestamp", "yyyy-MM-dd'T'HH:mm:ss'Z'", "ISO8601"]
            target   => "@timestamp"
            timezone => "UTC"
          }
          mutate {
            remove_field => ["event_timestamp"]
          }
        }
      }
      PIPEEOF
      sed -i 's/^      //' /etc/logstash/conf.d/03-filter-date.conf

      # ------------------------------------------------------------------
      # 04-filter-mutate.conf
      # Unit conversions and field enrichment. Skipped on grok failure.
      # ------------------------------------------------------------------
      cat > /etc/logstash/conf.d/04-filter-mutate.conf <<'PIPEEOF'
      filter {
        if "weather_parsed" in [tags] {
          ruby {
            code => "
              if event.get('temperature')
                c = event.get('temperature').to_f
                event.set('temperature_f', (c * 9.0 / 5.0 + 32.0).round(1))
              end
            "
          }

          ruby {
            code => "
              dir = event.get('wind_dir').to_i
              compass = case dir
                when   0..22   then 'N'
                when  23..67   then 'NE'
                when  68..112  then 'E'
                when 113..157  then 'SE'
                when 158..202  then 'S'
                when 203..247  then 'SW'
                when 248..292  then 'W'
                when 293..337  then 'NW'
                else                'N'
              end
              event.set('wind_direction_compass', compass)
            "
          }

          mutate {
            add_field => {
              "pipeline_version" => "1.0.0"
              "data_source"      => "open-meteo"
            }
          }
        }
      }
      PIPEEOF
      sed -i 's/^      //' /etc/logstash/conf.d/04-filter-mutate.conf

      # ------------------------------------------------------------------
      # 05-filter-ruby.conf
      # Translates WMO weather codes to human-readable descriptions and
      # tags extreme temperature events. Skipped on grok failure.
      # ------------------------------------------------------------------
      cat > /etc/logstash/conf.d/05-filter-ruby.conf <<'PIPEEOF'
      filter {
        if "weather_parsed" in [tags] {
          ruby {
            code => "
              code = event.get('weather_code').to_i
              description = case code
                when 0        then 'Clear sky'
                when 1        then 'Mainly clear'
                when 2        then 'Partly cloudy'
                when 3        then 'Overcast'
                when 45, 48   then 'Foggy'
                when 51, 53, 55 then 'Light drizzle'
                when 61, 63, 65 then 'Rain'
                when 71, 73, 75 then 'Snow'
                when 80, 81, 82 then 'Rain showers'
                when 85, 86   then 'Snow showers'
                when 95       then 'Thunderstorm'
                when 96, 99   then 'Thunderstorm with hail'
                else          'Unknown'
              end
              event.set('weather_description', description)
              precip_codes = [51,53,55,61,63,65,71,73,75,80,81,82,85,86,95,96,99]
              event.set('is_precipitation', precip_codes.include?(code))
            "
          }

          if [temperature] and [temperature] > 35 {
            mutate { add_tag => ["extreme_heat"] }
          }
          if [temperature] and [temperature] < -10 {
            mutate { add_tag => ["extreme_cold"] }
          }
        }
      }
      PIPEEOF
      sed -i 's/^      //' /etc/logstash/conf.d/05-filter-ruby.conf

      # ------------------------------------------------------------------
      # 06-output-file.conf
      # Parsed events -> structured JSON log. Failures -> failure log.
      # All events -> stdout for journalctl debugging.
      # ------------------------------------------------------------------
      cat > /etc/logstash/conf.d/06-output-file.conf <<'PIPEEOF'
      output {
        if "weather_parsed" in [tags] {
          file {
            path  => "/var/log/logstash/weather-output.log"
            codec => json_lines
          }
        }

        if "_grokparsefailure" in [tags] {
          file {
            path  => "/var/log/logstash/weather-parse-failures.log"
            codec => line {
              format => "%{@timestamp} PARSE_FAILURE: %{message}"
            }
          }
        }

        stdout {
          codec => rubydebug {
            metadata => false
          }
        }
      }
      PIPEEOF
      sed -i 's/^      //' /etc/logstash/conf.d/06-output-file.conf

      # Configure Logstash pipeline
      cat > /etc/logstash/pipelines.yml <<'PIPEEOF'
      - pipeline.id: weather
        path.config: "/etc/logstash/conf.d/*.conf"
        pipeline.workers: 2
        pipeline.batch.size: 125
      PIPEEOF
      sed -i 's/^      //' /etc/logstash/pipelines.yml

      mkdir -p /var/log/logstash
      chown logstash:logstash /var/log/logstash

      # Reduce JVM heap to avoid OOM on a 4GB VM
      sed -i 's/-Xms1g/-Xms512m/g' /etc/logstash/jvm.options 2>/dev/null || true

      systemctl daemon-reload
      systemctl enable logstash
      systemctl start logstash

      echo "[logstash-setup] Done."
      echo "[logstash-setup] Pipeline failures (expected with the bug):"
      echo "[logstash-setup]   tail -f /var/log/logstash/weather-parse-failures.log"
      echo "[logstash-setup] To see the broken grok:"
      echo "[logstash-setup]   cat /etc/logstash/conf.d/02-filter-grok.conf"

runcmd:
  - systemctl enable qemu-guest-agent
  - systemctl start qemu-guest-agent
  - /opt/setup-logstash.sh >> /var/log/logstash-setup.log 2>&1
ENDOFCLOUD

# Inject kafka IP into logstash user-data (replaces placeholder in two places:
# the 01-input-kafka.conf content and the display comment at the end)
log_info "Injecting kafka IP (${KAFKA_IP}) into logstash cloud-init..."
sed -i "s/KAFKA_IP_PLACEHOLDER/${KAFKA_IP}/g" "${LOGSTASH_SEED_DIR}/user-data"
log_success "Kafka IP injected."

# ============================================================================
# STEP 4: Boot logstash-demo VM
# ============================================================================
log_info "Booting '${LOGSTASH_VM}'..."
LOGSTASH_IP=$(boot_vm "$LOGSTASH_VM" "$VM_INDEX" "52:54:03" 4096 2 10)

# ============================================================================
# STEP 5: Final summary
# ============================================================================
echo ""
echo "============================================================================"
log_success "Kafka + Logstash demo environment ready!"
echo "============================================================================"
echo ""
echo "VMs created:"
for i in "${!CREATED_VM_NAMES[@]}"; do
    echo "  ${CREATED_VM_NAMES[$i]}"
    echo "    Disk:  ${CREATED_VM_DISKS[$i]}"
    [[ -n "${CREATED_VM_MACS[$i]}" ]] && echo "    MAC:   ${CREATED_VM_MACS[$i]}"
    if [[ -n "${CREATED_VM_IPS[$i]}" ]]; then
        echo "    IP:    ${CREATED_VM_IPS[$i]}"
    else
        echo "    IP:    (pending - check with 'virsh domifaddr ${CREATED_VM_NAMES[$i]} --source lease')"
    fi
done
echo ""
echo "Services (start ~5 min after VM boot while cloud-init runs):"
echo "  ${KAFKA_VM}:"
echo "    - Kafka broker on :9092 (KRaft mode, no ZooKeeper)"
echo "    - Topic: weather-logs (3 partitions)"
echo "    - weather-producer.service: fetches Open-Meteo data every 30s"
echo "    - Setup log: /var/log/kafka-setup.log"
echo ""
echo "  ${LOGSTASH_VM}:"
echo "    - Logstash 8.x consuming weather-logs from kafka-demo"
echo "    - 6-stage pipeline in /etc/logstash/conf.d/"
echo "    - BUG in 02-filter-grok.conf: missing brackets around timestamp"
echo "    - Failures accumulate in /var/log/logstash/weather-parse-failures.log"
echo "    - Setup log: /var/log/logstash-setup.log"
echo ""
echo "The grok bug (02-filter-grok.conf):"
echo "  Pattern expects: <ISO8601_TIMESTAMP> WEATHER ..."
echo "  Log format is:   [<ISO8601_TIMESTAMP>] WEATHER ..."
echo "  Fix: change pattern start to: \[%{TIMESTAMP_ISO8601:event_timestamp}\]"
echo ""
echo "Login: ubuntu / ubuntu"
if [[ -n "$SSH_USERS_FILE" ]] && [[ -f "$SSH_USERS_FILE" ]]; then
    echo "SSH Users:"
    while IFS= read -r line || [[ -n "$line" ]]; do
        [[ -z "$line" ]] && continue
        [[ "$line" =~ ^#.*$ ]] && continue
        username="${line%% *}"
        echo "  ${username} (key-based auth)"
    done < "$SSH_USERS_FILE"
fi
echo ""
echo "Useful commands:"
echo "  virsh list --all"
for i in "${!CREATED_VM_NAMES[@]}"; do
    echo "  virsh domifaddr ${CREATED_VM_NAMES[$i]} --source lease"
done
echo ""
