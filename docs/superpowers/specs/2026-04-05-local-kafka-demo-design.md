# Local Kafka Demo - Design

## Context

deer.sh needs a self-contained local demo showing the Kafka stub feature end-to-end: a daemon captures from a live Kafka source, creates a sandbox microVM, and an agent investigates a broken Logstash pipeline inside that sandbox. The target audience is enterprise ELK stack users.

Previously this required a paid Hetzner VM. The goal is to run everything locally on a Mac in one command.

## Architecture

```
Mac (Apple Hypervisor.framework)
  |
  +-- Lima VM "fluid-demo" (Linux, 4 CPU / 8 GB / 100 GB)
  |     |
  |     +-- deer-daemon (gRPC :9091, port-forwarded to Mac)
  |     |
  |     +-- Docker Compose (inside Lima)
  |     |     +-- Redpanda          (source Kafka, :9092 internal)
  |     |     +-- weather-producer  (publishes to 5 city topics every 30s)
  |     |     +-- Elasticsearch     (:9200 internal, port-forwarded to Mac)
  |     |     +-- Kibana            (:5601, port-forwarded to Mac -> localhost:5601)
  |     |
  |     +-- QEMU microVM "sandbox"  (created by daemon on demand)
  |           +-- Redpanda stub     (127.0.0.1:9092, injected via cloud-init)
  |           +-- Logstash          (kafka input -> stub, ES output -> Lima Docker ES)
  |
  +-- fluid CLI / TUI  (runs natively on Mac, connects to daemon at localhost:9091)
  +-- Browser          (Kibana at http://localhost:5601)
```

Lima automatically port-forwards anything the guest listens on to the Mac host - no manual config needed.

Networking inside Lima: the sandbox microVM sits on the libvirt bridge (virbr0, gateway 192.168.122.1 by default). It reaches Elasticsearch at the Lima host's bridge IP on port 9200. Docker inside Lima binds ES to 0.0.0.0 so it's reachable from the bridge. The `start.sh` script detects the virbr0 IP and injects it into the Logstash config via a cloud-init environment variable at sandbox creation time.

## Components

### Existing (reuse / adapt)

| File | Role |
|------|------|
| `scripts/run-redpanda-e2e-lima-host.sh` | Lima VM creation and guest setup (QEMU + libvirt) - reuse the Lima provisioning logic |
| `scripts/setup-kafka-demo.sh` | Source of the weather producer script and the intentional grok bug - extract and adapt |
| `deer-daemon` Kafka stub support | Capture from source Kafka, inject Redpanda stub into sandbox via cloud-init |

### New

| File | Description |
|------|-------------|
| `demo/docker-compose.yml` | Redpanda + weather-producer + Elasticsearch + Kibana. Runs inside Lima via `docker compose`. |
| `demo/weather-producer.sh` | Publishes weather data to 5 topics: `new-york`, `chicago`, `sf`, `la`, `indy`. Adapted from `setup-kafka-demo.sh`. |
| `demo/logstash/pipeline.conf` | Kafka input (stub broker) + grok filter (with intentional bug) + ES output. |
| `demo/logstash/logstash.yml` | Logstash config pointing ES output at Lima bridge gateway IP (injected at sandbox creation via cloud-init env var). |
| `demo/kibana/dashboard.ndjson` | Pre-built Kibana dashboard (weather events per city). Imported on startup. |
| `scripts/demo/start.sh` | One-command launcher: ensures Lima VM, installs deps, starts Docker Compose + daemon. |
| `scripts/demo/stop.sh` | Stops daemon + Docker Compose inside Lima. Leaves Lima VM running. |
| `Makefile` targets | `demo-start`, `demo-stop`, `demo-reset` |

## Demo Flow

1. `make demo-start` on Mac
   - Creates/starts Lima VM `fluid-demo` (4 CPU, 8 GB, 100 GB)
   - Installs QEMU + libvirt + Docker inside Lima (idempotent)
   - Starts Docker Compose inside Lima (Redpanda, ES, Kibana, weather producer)
   - Builds and starts deer-daemon inside Lima (in a tmux session named `deer-daemon`)
   - Imports Kibana dashboard
   - Prints: `Daemon ready at localhost:9091 - Kibana at http://localhost:5601`

2. User runs `deer` CLI on Mac, connects to daemon at `localhost:9091`

3. User types: *"Our Logstash pipeline isn't ingesting weather data, can you investigate?"*

4. Agent creates a sandbox with Kafka stub enabled (capturing all 5 city topics from Redpanda)

5. Sandbox boots: cloud-init installs Redpanda stub, starts Logstash

6. Agent SSHes into sandbox, reads `pipeline.conf`, identifies grok bug (missing `\[` and `\]` around timestamp)

7. Agent fixes the config, restarts Logstash

8. Kibana at `localhost:5601` shows all 5 city topics flowing - visual payoff

## The Bug

Weather producer log line format:
```
[2026-03-28T15:30:00Z] WEATHER location="new-york" temp=8.3 wind_speed=14.2 ...
```

Broken grok pattern (omits brackets):
```
%{TIMESTAMP_ISO8601:timestamp} WEATHER location="%{WORD:location}" ...
```

Fixed grok pattern:
```
\[%{TIMESTAMP_ISO8601:timestamp}\] WEATHER location="%{WORD:location}" ...
```

All downstream enrichment filters are guarded on parse success, so the bug silently drops all events. Kibana shows zero data until the fix.

## City Topics

| Topic | Location |
|-------|----------|
| `new-york` | New York, NY (lat 40.71, lon -74.01) |
| `chicago` | Chicago, IL (lat 41.88, lon -87.63) |
| `sf` | San Francisco, CA (lat 37.77, lon -122.42) |
| `la` | Los Angeles, CA (lat 34.05, lon -118.24) |
| `indy` | Indianapolis, IN (lat 39.77, lon -86.16) |

Weather data fetched from Open-Meteo API (no key required), published every 30s.

## Verification

1. `make demo-start` completes without errors
2. `curl http://localhost:9200` returns Elasticsearch cluster info
3. `http://localhost:5601` opens Kibana with the pre-built dashboard (showing zero events)
4. `deer` CLI connects to daemon at `localhost:9091`
5. Agent creates sandbox - sandbox appears in `deer list sandboxes`
6. After agent fixes the grok bug - Kibana dashboard shows events on all 5 city topics
7. `make demo-stop` cleanly stops all services
