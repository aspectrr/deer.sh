# Local Kafka Demo Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a one-command local demo (`make demo-start`) that spins up the full fluid Kafka stub flow on a Mac using Lima, showing an agent diagnosing a broken Logstash pipeline with live weather data.

**Architecture:** Lima VM hosts deer-daemon, Docker Compose (Redpanda + Elasticsearch + Kibana + weather producer), and QEMU microVM sandboxes. The Mac runs only the fluid CLI and a browser. Lima port-forwards gRPC (:9091) and Kibana (:5601) to localhost. The sandbox microVM has Logstash pre-installed (from a prepared source image) and a Redpanda stub injected by the daemon at creation time. Logstash reads from the stub and outputs to Elasticsearch at the Lima virbr0 gateway (192.168.122.1:9200).

**Tech Stack:** Lima, QEMU, Docker Compose, Redpanda, Elasticsearch 8.x, Kibana 8.x, Logstash 8.x, Open-Meteo API (no key), Bash

---

## File Map

| File | Purpose |
|------|---------|
| `demo/docker-compose.yml` | Redpanda + ES + Kibana + weather-producer inside Lima |
| `demo/weather-producer.py` | Python producer: publishes weather data to 5 per-city topics every 30s |
| `demo/weather-producer.Dockerfile` | Python image with kafka-python for the weather producer container |
| `demo/logstash/pipeline/01-input-kafka.conf` | Kafka input from stub broker (127.0.0.1:9092) |
| `demo/logstash/pipeline/02-filter-grok.conf` | Grok parser — contains the intentional bug |
| `demo/logstash/pipeline/03-filter-date.conf` | Date parsing (guarded on parse success) |
| `demo/logstash/pipeline/04-filter-mutate.conf` | Field enrichment (guarded) |
| `demo/logstash/pipeline/05-filter-ruby.conf` | Weather code translation (guarded) |
| `demo/logstash/pipeline/06-output-es.conf` | Elasticsearch output to 192.168.122.1:9200 (guarded) |
| `demo/logstash/logstash.yml` | Logstash node settings |
| `demo/prepare-source.sh` | Runs inside Lima: creates the Logstash source VM image |
| `demo/kibana/setup-dashboard.sh` | Calls Kibana API to create index pattern + dashboard |
| `scripts/demo/start.sh` | One-command launcher (runs on Mac, orchestrates Lima) |
| `scripts/demo/stop.sh` | Stops daemon + Docker Compose inside Lima |
| `scripts/demo/start_test.sh` | Dry-run smoke tests for start.sh |
| Root `Makefile` | Add `demo-start`, `demo-stop`, `demo-reset` targets |

---

## Task 1: Docker Compose for demo services

**Files:**
- Create: `demo/docker-compose.yml`

This runs inside Lima (via `docker compose`). Redpanda advertises `localhost:9092` so the daemon (also on Lima) can connect.

- [ ] **Step 1: Create `demo/docker-compose.yml`**

```yaml
services:
  redpanda:
    image: redpandadata/redpanda:v24.1.13
    command:
      - redpanda
      - start
      - --smp=1
      - --memory=512M
      - --reserve-memory=0M
      - --overprovisioned
      - --node-id=0
      - --check=false
      - --kafka-addr=PLAINTEXT://0.0.0.0:9092
      - --advertise-kafka-addr=PLAINTEXT://localhost:9092
    ports:
      # Bind only to loopback so the daemon can reach Redpanda at localhost:9092
      # but it does NOT conflict with the deer-daemon readiness server which
      # binds to the virbr0 bridge IP (192.168.122.1:9092) for sandbox phone_home.
      - "127.0.0.1:9092:9092"
    healthcheck:
      test: ["CMD", "rpk", "cluster", "info", "--brokers=localhost:9092"]
      interval: 5s
      timeout: 10s
      retries: 12

  redpanda-init:
    image: redpandadata/redpanda:v24.1.13
    depends_on:
      redpanda:
        condition: service_healthy
    entrypoint: >
      bash -c "
        rpk topic create new-york chicago sf la indy
          --brokers redpanda:9092
          --partitions 3
          --replicas 1 2>/dev/null || true
      "

  weather-producer:
    build:
      context: .
      dockerfile: weather-producer.Dockerfile
    depends_on:
      redpanda-init:
        condition: service_completed_successfully
    environment:
      BOOTSTRAP: "redpanda:9092"
    restart: unless-stopped

  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.13.4
    environment:
      - discovery.type=single-node
      - xpack.security.enabled=false
      - ES_JAVA_OPTS=-Xms512m -Xmx512m
    ports:
      - "9200:9200"
    healthcheck:
      test: ["CMD", "curl", "-sf", "http://localhost:9200"]
      interval: 10s
      timeout: 10s
      retries: 18

  kibana:
    image: docker.elastic.co/kibana/kibana:8.13.4
    environment:
      - ELASTICSEARCH_HOSTS=http://elasticsearch:9200
      - SERVER_BASEPATH=
      - XPACK_SECURITY_ENABLED=false
    ports:
      - "5601:5601"
    depends_on:
      elasticsearch:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-sf", "http://localhost:5601/api/status"]
      interval: 10s
      timeout: 10s
      retries: 24
```

- [ ] **Step 2: Validate compose syntax inside Lima**

```bash
limactl shell fluid-demo -- bash -c "cd /path/to/repo/demo && docker compose config --quiet"
```

Expected: no output (valid syntax)

- [ ] **Step 3: Commit**

```bash
git add demo/docker-compose.yml
git commit -m "feat(demo): add docker-compose for redpanda, elasticsearch, kibana, weather-producer"
```

---

## Task 2: Weather producer (5 city topics)

**Files:**
- Create: `demo/weather-producer.py`
- Create: `demo/weather-producer.Dockerfile`

Python producer (no bash/rpk dependency issues). Uses `kafka-python` to produce per-city weather events. Fetches from Open-Meteo API with `urllib` (no curl needed).

- [ ] **Step 1: Create `demo/weather-producer.Dockerfile`**

```dockerfile
FROM python:3.11-slim
RUN pip install --no-cache-dir kafka-python==2.0.2
COPY weather-producer.py /weather-producer.py
CMD ["python3", "/weather-producer.py"]
```

- [ ] **Step 2: Create `demo/weather-producer.py`**

```python
#!/usr/bin/env python3
"""
weather-producer.py

Fetches live weather data from the Open-Meteo API (no API key required)
and produces structured log lines to per-city Kafka topics every 30 seconds.

Topics: new-york, chicago, sf, la, indy

Log line format:
  [2026-03-28T15:30:00Z] WEATHER location="new-york" lat=40.71 lon=-74.01
      temp=8.3 wind_speed=14.2 wind_dir=270 weather_code=3 is_day=1
"""

import os
import sys
import time
import json
import urllib.request
import urllib.error
from datetime import datetime, timezone
from kafka import KafkaProducer

BOOTSTRAP = os.environ.get("BOOTSTRAP", "localhost:9092")

CITIES = [
    ("new-york", 40.71, -74.01),
    ("chicago",  41.88, -87.63),
    ("sf",       37.77, -122.42),
    ("la",       34.05, -118.24),
    ("indy",     39.77, -86.16),
]


def log(msg: str) -> None:
    print(f"[weather-producer] {msg}", flush=True)


def fetch_weather(city: str, lat: float, lon: float) -> str | None:
    url = (
        f"https://api.open-meteo.com/v1/forecast"
        f"?latitude={lat}&longitude={lon}&current_weather=true"
    )
    try:
        with urllib.request.urlopen(url, timeout=10) as resp:
            data = json.loads(resp.read())
    except (urllib.error.URLError, json.JSONDecodeError) as exc:
        log(f"Failed to fetch weather for {city}: {exc}")
        return None

    cw = data.get("current_weather", {})
    ts = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    return (
        f'[{ts}] WEATHER location="{city}" lat={lat} lon={lon}'
        f' temp={cw.get("temperature")} wind_speed={cw.get("windspeed")}'
        f' wind_dir={cw.get("winddirection")} weather_code={cw.get("weathercode")}'
        f' is_day={cw.get("is_day")}'
    )


def main() -> None:
    log(f"Connecting to Kafka at {BOOTSTRAP}...")
    producer = KafkaProducer(
        bootstrap_servers=BOOTSTRAP,
        value_serializer=lambda v: v.encode("utf-8"),
    )
    log(f"Connected. Producing to topics: {[c[0] for c in CITIES]}")

    while True:
        batch_start = time.monotonic()
        for city, lat, lon in CITIES:
            line = fetch_weather(city, lat, lon)
            if line is None:
                continue
            producer.send(city, value=line)
            log(f"Produced to {city}: {line}")
        producer.flush()
        elapsed = time.monotonic() - batch_start
        sleep_for = max(0.0, 30.0 - elapsed)
        time.sleep(sleep_for)


if __name__ == "__main__":
    main()
```

- [ ] **Step 3: Smoke test (can be run on Mac if kafka-python is installed)**

```bash
pip install kafka-python==2.0.2 --quiet
python3 -c "
import sys
sys.path.insert(0, 'demo')
# Test only the fetch logic, not Kafka connection
import urllib.request, json
from datetime import datetime, timezone
url = 'https://api.open-meteo.com/v1/forecast?latitude=40.71&longitude=-74.01&current_weather=true'
with urllib.request.urlopen(url, timeout=10) as r:
    data = json.loads(r.read())
cw = data['current_weather']
ts = datetime.now(timezone.utc).strftime('%Y-%m-%dT%H:%M:%SZ')
line = f'[{ts}] WEATHER location=\"new-york\" lat=40.71 lon=-74.01 temp={cw[\"temperature\"]} wind_speed={cw[\"windspeed\"]} wind_dir={cw[\"winddirection\"]} weather_code={cw[\"weathercode\"]} is_day={cw[\"is_day\"]}'
assert line.startswith('[') and 'WEATHER location=\"new-york\"' in line, f'unexpected: {line}'
print('PASS:', line)
"
```

Expected: `PASS: [2026-...] WEATHER location="new-york" ...`

- [ ] **Step 4: Commit**

```bash
git add demo/weather-producer.py demo/weather-producer.Dockerfile
git commit -m "feat(demo): add python weather producer for 5 city topics"
```

---

## Task 3: Logstash pipeline config files

**Files:**
- Create: `demo/logstash/pipeline/01-input-kafka.conf`
- Create: `demo/logstash/pipeline/02-filter-grok.conf`
- Create: `demo/logstash/pipeline/03-filter-date.conf`
- Create: `demo/logstash/pipeline/04-filter-mutate.conf`
- Create: `demo/logstash/pipeline/05-filter-ruby.conf`
- Create: `demo/logstash/pipeline/06-output-es.conf`
- Create: `demo/logstash/logstash.yml`

Input reads from the Redpanda stub at `127.0.0.1:9092` (always loopback in sandbox). ES output points to `192.168.122.1:9200` (Lima virbr0 gateway - where Docker ES is reachable from inside the sandbox).

- [ ] **Step 1: Create `demo/logstash/pipeline/01-input-kafka.conf`**

```ruby
input {
  kafka {
    bootstrap_servers => "127.0.0.1:9092"
    topics            => ["new-york", "chicago", "sf", "la", "indy"]
    group_id          => "logstash-weather-consumer"
    auto_offset_reset => "earliest"
    decorate_events   => true
    codec => plain {
      charset => "UTF-8"
    }
  }
}
```

- [ ] **Step 2: Create `demo/logstash/pipeline/02-filter-grok.conf`**

```ruby
filter {
  # BUG: Missing \[ and \] around timestamp. The log format is:
  #   [2026-03-28T15:30:00Z] WEATHER location="new-york" ...
  # but this pattern omits the brackets, causing _grokparsefailure
  # on every event. All downstream filters are guarded on parse success,
  # so no enrichment occurs and no data reaches Elasticsearch.
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
```

- [ ] **Step 3: Create `demo/logstash/pipeline/03-filter-date.conf`**

```ruby
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
```

- [ ] **Step 4: Create `demo/logstash/pipeline/04-filter-mutate.conf`**

```ruby
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
```

- [ ] **Step 5: Create `demo/logstash/pipeline/05-filter-ruby.conf`**

```ruby
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
```

- [ ] **Step 6: Create `demo/logstash/pipeline/06-output-es.conf`**

```ruby
output {
  if "weather_parsed" in [tags] {
    elasticsearch {
      hosts    => ["http://192.168.122.1:9200"]
      index    => "weather-%{[location]}-%{+YYYY.MM.dd}"
      action   => "index"
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
}
```

- [ ] **Step 7: Create `demo/logstash/logstash.yml`**

```yaml
node.name: fluid-demo-sandbox
pipeline.workers: 2
pipeline.batch.size: 125
pipeline.batch.delay: 50
log.level: warn
path.logs: /var/log/logstash
xpack.monitoring.enabled: false
```

- [ ] **Step 8: Commit**

```bash
git add demo/logstash/
git commit -m "feat(demo): add logstash pipeline config with intentional grok bug"
```

---

## Task 4: Source VM image preparation script

**Files:**
- Create: `demo/prepare-source.sh`

Runs inside Lima. Creates a QCOW2 disk image with Logstash pre-installed and the pipeline configs baked in. This is a one-time step (~10 min) that `start.sh` skips if the image already exists.

The script boots a QEMU microVM from the Ubuntu cloud image, runs cloud-init to install Logstash + copy pipeline files, waits for completion, then powers off and keeps the QCOW2.

- [ ] **Step 1: Create `demo/prepare-source.sh`**

```bash
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
    if [ "$DRY_RUN" -eq 1 ]; then printf '+ %s\n' "$*"; return; fi
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

PIPELINE_DIR="${REPO_ROOT}/demo/logstash/pipeline"
LOGSTASH_YML="${REPO_ROOT}/demo/logstash/logstash.yml"
WORKDIR="$(mktemp -d /tmp/fluid-prepare-source.XXXXXX)"
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

# Encode pipeline files as base64 for embedding in user-data
encode_file() { base64 < "$1" | tr -d '\n'; }

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
      wget -qO - https://artifacts.elastic.co/GPG-KEY-elasticsearch | apt-key add -
      echo "deb https://artifacts.elastic.co/packages/8.x/apt stable main" \
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
  - cloud-init status --wait
  - poweroff
CLOUDINIT

log "Building cloud-init ISO..."
run cloud-localds "$CLOUD_INIT_ISO" \
    "${SEED_DIR}/user-data" \
    "${SEED_DIR}/meta-data" \
    --network-config "${SEED_DIR}/network-config"

QEMU_BIN="qemu-system-${ARCH}"
[ "$ARCH" = "amd64" ] && QEMU_BIN="qemu-system-x86_64"
[ "$ARCH" = "arm64" ] && QEMU_BIN="qemu-system-aarch64"

log "Booting setup VM (this takes ~10 minutes)..."
log "Output: /var/log/logstash-setup.log inside the VM"

QEMU_ARGS=(
    -nographic
    -machine type=q35,accel=tcg
    -cpu max
    -smp 2
    -m 2048
    -drive "file=${WORK_DISK},if=virtio,cache=unsafe"
    -drive "file=${CLOUD_INIT_ISO},if=virtio,format=raw"
    -netdev user,id=net0
    -device virtio-net-pci,netdev=net0
    -serial stdio
    -no-reboot
)

if [ "$ARCH" = "arm64" ]; then
    QEMU_ARGS+=(-machine virt -bios /usr/share/qemu-efi-aarch64/QEMU_EFI.fd)
fi

run "$QEMU_BIN" "${QEMU_ARGS[@]}"

log "VM powered off. Converting to final image..."
run qemu-img convert -f qcow2 -O qcow2 "$WORK_DISK" "$OUTPUT_IMAGE"

log "Source image ready: ${OUTPUT_IMAGE}"
if [ "$DRY_RUN" -eq 0 ]; then
    qemu-img info "$OUTPUT_IMAGE"
fi
```

- [ ] **Step 2: Make executable**

```bash
chmod +x demo/prepare-source.sh
```

- [ ] **Step 3: Write dry-run test**

Create `demo/prepare-source_test.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET="${SCRIPT_DIR}/prepare-source.sh"

assert_contains() {
    local haystack="$1" needle="$2"
    if [[ "$haystack" != *"$needle"* ]]; then
        printf 'expected output to contain: %s\n' "$needle" >&2
        exit 1
    fi
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

# Create minimal pipeline files for the test
mkdir -p "$tmpdir/demo/logstash/pipeline"
for n in 01 02 03 04 05 06; do
    printf '# placeholder\n' > "$tmpdir/demo/logstash/pipeline/${n}-test.conf"
done
printf '# placeholder\n' > "$tmpdir/demo/logstash/logstash.yml"
touch "$tmpdir/demo/logstash/pipeline/01-input-kafka.conf"
touch "$tmpdir/demo/logstash/pipeline/02-filter-grok.conf"
touch "$tmpdir/demo/logstash/pipeline/03-filter-date.conf"
touch "$tmpdir/demo/logstash/pipeline/04-filter-mutate.conf"
touch "$tmpdir/demo/logstash/pipeline/05-filter-ruby.conf"
touch "$tmpdir/demo/logstash/pipeline/06-output-es.conf"

output="$(bash "$TARGET" --dry-run --repo-root "$tmpdir" --output "$tmpdir/output.qcow2" 2>&1)"
assert_contains "$output" "ubuntu-24.04-server-cloudimg-amd64.img"
assert_contains "$output" "qemu-img create"
assert_contains "$output" "qemu-system-x86_64"
echo "PASS"
```

- [ ] **Step 4: Run the test**

```bash
bash demo/prepare-source_test.sh
```

Expected: `PASS`

- [ ] **Step 5: Commit**

```bash
git add demo/prepare-source.sh demo/prepare-source_test.sh
git commit -m "feat(demo): add logstash source VM preparation script"
```

---

## Task 5: Kibana dashboard setup script

**Files:**
- Create: `demo/kibana/setup-dashboard.sh`

Uses the Kibana Saved Objects API to create an index pattern and a simple dashboard showing weather events per city. Runs after Kibana is healthy.

- [ ] **Step 1: Create `demo/kibana/setup-dashboard.sh`**

```bash
#!/usr/bin/env bash
# setup-dashboard.sh
#
# Creates Kibana index pattern and weather dashboard via the Saved Objects API.
# Run after Kibana is healthy at $KIBANA_URL.
#
# Usage: bash setup-dashboard.sh [KIBANA_URL]

set -euo pipefail

KIBANA_URL="${1:-http://localhost:5601}"

log() { printf '[kibana-setup] %s\n' "$*"; }

log "Waiting for Kibana at ${KIBANA_URL}..."
for i in $(seq 1 30); do
    if curl -sf "${KIBANA_URL}/api/status" | grep -q '"level":"available"' 2>/dev/null; then
        log "Kibana is ready."
        break
    fi
    [ "$i" -lt 30 ] || { log "ERROR: Kibana not ready after 5 minutes"; exit 1; }
    log "Waiting... (${i}/30)"
    sleep 10
done

log "Creating index pattern for weather-* indices..."
curl -sf -X POST "${KIBANA_URL}/api/saved_objects/index-pattern/weather-all" \
    -H "kbn-xsrf: true" \
    -H "Content-Type: application/json" \
    -d '{
  "attributes": {
    "title": "weather-*",
    "timeFieldName": "@timestamp"
  }
}' > /dev/null
log "Index pattern created: weather-*"

log "Creating weather dashboard..."
curl -sf -X POST "${KIBANA_URL}/api/saved_objects/dashboard/fluid-weather-demo" \
    -H "kbn-xsrf: true" \
    -H "Content-Type: application/json" \
    -d '{
  "attributes": {
    "title": "fluid Demo: Weather by City",
    "description": "Live weather data from the Kafka stub pipeline. Events appear here after the Logstash grok bug is fixed.",
    "panelsJSON": "[]",
    "optionsJSON": "{\"useMargins\":true}",
    "version": 1,
    "timeRestore": true,
    "timeTo": "now",
    "timeFrom": "now-1h",
    "refreshInterval": "{\"pause\":false,\"value\":10000}",
    "kibanaSavedObjectMeta": {
      "searchSourceJSON": "{\"query\":{\"language\":\"kuery\",\"query\":\"\"},\"filter\":[]}"
    }
  }
}' > /dev/null
log "Dashboard created: fluid Demo: Weather by City"

log ""
log "Kibana setup complete."
log "  Dashboard: ${KIBANA_URL}/app/dashboards"
log "  Index pattern covers: weather-new-york-*, weather-chicago-*, weather-sf-*, weather-la-*, weather-indy-*"
```

- [ ] **Step 2: Make executable**

```bash
chmod +x demo/kibana/setup-dashboard.sh
```

- [ ] **Step 3: Commit**

```bash
git add demo/kibana/setup-dashboard.sh
git commit -m "feat(demo): add kibana dashboard setup script"
```

---

## Task 6: Demo start script

**Files:**
- Create: `scripts/demo/start.sh`

Runs on the Mac. Orchestrates everything: Lima VM, Docker Compose inside Lima, source VM image preparation, deer-daemon inside Lima.

- [ ] **Step 1: Create `scripts/demo/start.sh`**

```bash
#!/usr/bin/env bash
# scripts/demo/start.sh
#
# One-command local demo launcher.
# Run on Mac. Requires: limactl, docker (for compose syntax check only)
#
# Usage: ./scripts/demo/start.sh [--repo-root <path>] [--dry-run]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
LIMA_NAME="fluid-demo"
LIMA_CPUS=4
LIMA_MEMORY="8GiB"
LIMA_DISK="100GiB"
DRY_RUN=0

SOURCE_IMAGE_PATH="/var/lib/fluid-demo/logstash-source.qcow2"
DAEMON_WORKDIR="/var/lib/fluid-demo/daemon"

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
    if [ "$DRY_RUN" -eq 1 ]; then printf '+ %s\n' "$*"; return; fi
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
LIMA_CONFIG_PATH="${LIMA_HOME:-$HOME/.lima}/${LIMA_NAME}/lima.yaml"

if [ ! -f "$LIMA_CONFIG_PATH" ]; then
    log "Creating Lima VM '${LIMA_NAME}' (${LIMA_CPUS} CPU, ${LIMA_MEMORY}, ${LIMA_DISK})..."
    LIMA_TEMPLATE="$(mktemp /tmp/${LIMA_NAME}.XXXXXX.yaml)"
    cat > "$LIMA_TEMPLATE" <<EOF
base: template://ubuntu
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
    bridge-utils iproute2 cloud-image-utils tmux curl \
    golang-go docker.io docker-compose-plugin
sudo systemctl enable --now libvirtd docker
sudo virsh net-autostart default
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

# ---- Source VM image ----
log "Checking Logstash source VM image..."
ARCH_OUT=$(limactl shell "$LIMA_NAME" -- uname -m 2>/dev/null || echo "x86_64")
GUEST_ARCH="amd64"
[[ "$ARCH_OUT" == "aarch64" ]] && GUEST_ARCH="arm64"

run_guest "
if [ -f '${SOURCE_IMAGE_PATH}' ]; then
    echo 'Source image already exists, skipping preparation.'
else
    sudo mkdir -p \$(dirname ${SOURCE_IMAGE_PATH})
    echo 'Preparing Logstash source VM image (~10 min)...'
    bash ${GUEST_REPO}/demo/prepare-source.sh \
        --repo-root ${GUEST_REPO} \
        --output ${SOURCE_IMAGE_PATH} \
        --arch ${GUEST_ARCH}
fi
"

# ---- microVM assets (kernel + initrd) ----
log "Downloading microVM kernel and initrd assets..."
ASSETS_DIR="/var/lib/fluid-demo/assets"
run_guest "
sudo mkdir -p ${ASSETS_DIR}
bash ${GUEST_REPO}/scripts/download-microvm-assets.sh --output-dir ${ASSETS_DIR}
"

# ---- deer-daemon config ----
log "Writing deer-daemon config..."
DAEMON_CONFIG="/var/lib/fluid-demo/daemon.yaml"
IMAGES_DIR="${SOURCE_IMAGE_PATH%/*}"

run_guest "
sudo mkdir -p ${DAEMON_WORKDIR} ${IMAGES_DIR}

# Link or copy source image into images dir so daemon discovers it
[ -f '${SOURCE_IMAGE_PATH}' ] && sudo ln -sf ${SOURCE_IMAGE_PATH} ${IMAGES_DIR}/logstash-source.qcow2 || true

# Read asset paths from download script output
KERNEL_PATH=\$(bash ${GUEST_REPO}/scripts/download-microvm-assets.sh --output-dir ${ASSETS_DIR} 2>/dev/null | grep FLUID_E2E_KERNEL | cut -d= -f2)
INITRD_PATH=\$(bash ${GUEST_REPO}/scripts/download-microvm-assets.sh --output-dir ${ASSETS_DIR} 2>/dev/null | grep FLUID_E2E_INITRD | cut -d= -f2)

sudo tee ${DAEMON_CONFIG} > /dev/null <<DAEMONYAML
daemon:
  listen_addr: ':9091'
  enabled: true

provider: microvm

microvm:
  qemu_binary: qemu-system-x86_64
  accel: tcg
  kernel_path: \${KERNEL_PATH}
  initrd_path: \${INITRD_PATH}
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
  ca_key_path: /var/lib/fluid-demo/ssh_ca
  ca_pub_key_path: /var/lib/fluid-demo/ssh_ca.pub
  key_dir: ${DAEMON_WORKDIR}/keys
  default_user: sandbox
  identity_file: /var/lib/fluid-demo/identity

libvirt:
  uri: qemu:///system
  network: default

janitor:
  interval: 1m
  default_ttl: 24h
DAEMONYAML
echo 'Daemon config written to ${DAEMON_CONFIG}'
"

# ---- deer-daemon ----
log "Building and starting deer-daemon..."
run_guest "
set -euo pipefail
cd ${GUEST_REPO}/deer-daemon
go build -o bin/deer-daemon ./cmd/deer-daemon
sudo mkdir -p ${DAEMON_WORKDIR}
tmux new-session -d -s deer-daemon 2>/dev/null || true
tmux send-keys -t deer-daemon \
    'sudo ${GUEST_REPO}/deer-daemon/bin/deer-daemon -config ${DAEMON_CONFIG} 2>&1 | tee ${DAEMON_WORKDIR}/daemon.log' \
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
log "fluid demo is ready!"
log ""
log "  Daemon gRPC:  localhost:9091"
log "  Kibana:       http://localhost:5601"
log ""
log "  Run the fluid CLI on your Mac:"
log "    deer connect localhost:9091"
log "    fluid"
log ""
log "  Source VM for sandbox creation: logstash-source"
log "  Prompt the agent:"
log "    'Our Logstash pipeline isn't ingesting weather data, investigate'"
log "============================================================"
```

- [ ] **Step 2: Make executable**

```bash
chmod +x scripts/demo/start.sh
```

- [ ] **Step 3: Write dry-run test**

Create `scripts/demo/start_test.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET="${SCRIPT_DIR}/start.sh"

assert_contains() {
    local haystack="$1" needle="$2"
    if [[ "$haystack" != *"$needle"* ]]; then
        printf 'expected output to contain: %s\n' "$needle" >&2
        exit 1
    fi
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

output="$(bash "$TARGET" --dry-run --repo-root "$tmpdir" 2>&1)"
assert_contains "$output" "limactl start"
assert_contains "$output" "docker compose up"
assert_contains "$output" "deer-daemon -config"
assert_contains "$output" "prepare-source.sh"
assert_contains "$output" "download-microvm-assets.sh"
echo "PASS"
```

- [ ] **Step 4: Run dry-run test**

```bash
bash scripts/demo/start_test.sh
```

Expected: `PASS`

- [ ] **Step 5: Commit**

```bash
git add scripts/demo/start.sh scripts/demo/start_test.sh
git commit -m "feat(demo): add one-command demo launcher script"
```

---

## Task 7: Stop script and Makefile targets

**Files:**
- Create: `scripts/demo/stop.sh`
- Modify: `Makefile` (root - create if missing, or add to deer-daemon/Makefile)

- [ ] **Step 1: Create `scripts/demo/stop.sh`**

```bash
#!/usr/bin/env bash
# scripts/demo/stop.sh
#
# Stops deer-daemon and Docker Compose inside the Lima demo VM.
# Leaves the Lima VM running.
#
# Usage: ./scripts/demo/stop.sh [--repo-root <path>]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
LIMA_NAME="fluid-demo"

usage() { printf 'Usage: ./scripts/demo/stop.sh [--repo-root <path>]\n'; }

log()  { printf '[demo-stop] %s\n' "$*"; }
fail() { printf '[demo-stop] ERROR: %s\n' "$*" >&2; exit 1; }

while [ "$#" -gt 0 ]; do
    case "$1" in
        --repo-root) REPO_ROOT="$2"; shift 2 ;;
        -h|--help)   usage; exit 0 ;;
        *) fail "unknown argument: $1" ;;
    esac
done

command -v limactl >/dev/null 2>&1 || fail "limactl not found"

log "Stopping deer-daemon tmux session..."
limactl shell "$LIMA_NAME" -- bash -lc "
    tmux kill-session -t deer-daemon 2>/dev/null || true
    sudo pkill -f deer-daemon 2>/dev/null || true
    echo 'deer-daemon stopped.'
" || true

log "Stopping Docker Compose services..."
limactl shell "$LIMA_NAME" -- bash -lc "
    cd ${REPO_ROOT}/demo
    sudo docker compose down
    echo 'Docker Compose stopped.'
" || true

log "Demo stopped. Lima VM '${LIMA_NAME}' is still running."
log "To fully remove the VM: limactl delete ${LIMA_NAME}"
```

- [ ] **Step 2: Make executable**

```bash
chmod +x scripts/demo/stop.sh
```

- [ ] **Step 3: Check if a root Makefile exists**

```bash
ls Makefile 2>/dev/null && echo "exists" || echo "missing"
```

- [ ] **Step 4: Create root `Makefile` with demo targets (or add to existing)**

If no root Makefile exists, create it:

```makefile
.PHONY: demo-start demo-stop demo-reset

demo-start:
	bash scripts/demo/start.sh

demo-stop:
	bash scripts/demo/stop.sh

demo-reset:
	bash scripts/demo/stop.sh || true
	limactl delete fluid-demo --force 2>/dev/null || true
	@echo "Demo fully reset. Run 'make demo-start' to start fresh."
```

If a root Makefile already exists, append these targets.

- [ ] **Step 5: Commit**

```bash
git add scripts/demo/stop.sh Makefile
git commit -m "feat(demo): add stop script and Makefile demo-start/stop/reset targets"
```

---

## Verification

Run these to confirm the demo works end-to-end:

```bash
# 1. Dry-run the start script (no Lima needed)
bash scripts/demo/start.sh --dry-run
# Expected: prints all commands without executing

# 2. Run dry-run tests
bash scripts/demo/start_test.sh
bash demo/prepare-source_test.sh
# Expected: PASS PASS

# 3. Full run (requires limactl + ~20 min for first boot + source image)
make demo-start
# Expected: "fluid demo is ready!" with localhost:9091 and localhost:5601

# 4. Verify services
curl http://localhost:9200          # ES cluster info
curl http://localhost:5601/api/status | grep available  # Kibana healthy

# 5. Connect CLI and run demo
deer connect localhost:9091
fluid
# In TUI: "Our Logstash pipeline isn't ingesting weather data, investigate"
# Watch agent create sandbox, find grok bug, fix it

# 6. Verify Kibana shows data
open http://localhost:5601/app/dashboards
# Dashboard "fluid Demo: Weather by City" should show events

# 7. Stop
make demo-stop
```
