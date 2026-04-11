#!/usr/bin/env bash
# scripts/test-kafka-es-stub-e2e.sh
#
# End-to-end test: creates a sandbox with kafka_stub + es_stub,
# verifies Redpanda and Elasticsearch start correctly, produces
# a Kafka message, indexes it into ES, and queries ES to confirm.
#
# Prerequisites:
#   - deer-daemon running on localhost:9091
#   - Source VM image "Deer Source VM" available in ~/.deer/images/
#   - CLI built at deer-cli/bin/deer
#
# Usage: ./scripts/test-kafka-es-stub-e2e.sh [--keep]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DEER_CLI="${REPO_ROOT}/deer-cli/bin/deer"

KEEP=0
for arg in "$@"; do
    case "$arg" in
        --keep) KEEP=1 ;;
    esac
done

log()  { printf '[e2e] [%s] %s\n' "$(date +%H:%M:%S)" "$*"; }
fail() { printf '[e2e] [%s] FAIL: %s\n' "$(date +%H:%M:%S)" "$*" >&2; exit 1; }

# Run a command inside the sandbox. Returns only the guest stdout.
sb_run() {
    local raw
    raw=$("$DEER_CLI" sandbox run "$SANDBOX_ID" "$1" 2>&1) || return 1
    local in_stdout=0
    local result=""
    while IFS= read -r line; do
        if [[ "$line" == *"STDOUT:" ]]; then
            in_stdout=1
            continue
        fi
        if [[ "$line" == *"STDERR:" ]]; then
            in_stdout=0
            continue
        fi
        if [ "$in_stdout" -eq 1 ]; then
            # Strip leading whitespace and trailing \r
            line="${line#"${line%%[![:space:]]*}"}"
            line="${line%$'\r'}"
            result="${result}${line}"$'\n'
        fi
    done <<< "$raw"
    printf '%s' "$result"
}

SANDBOX_ID=""
cleanup() {
    if [ -n "$SANDBOX_ID" ] && [ "$KEEP" -eq 0 ]; then
        log "Destroying sandbox $SANDBOX_ID ..."
        "$DEER_CLI" sandbox destroy "$SANDBOX_ID" 2>&1 || true
    elif [ -n "$SANDBOX_ID" ] && [ "$KEEP" -eq 1 ]; then
        log "Keeping sandbox $SANDBOX_ID (--keep)"
    fi
}
trap cleanup EXIT

# ---- Preflight ----

log "Preflight checks ..."
[ -x "$DEER_CLI" ] || fail "CLI not found at $DEER_CLI. Run: cd deer-cli && make build"
"$DEER_CLI" sandbox list >/dev/null 2>&1 || fail "Cannot reach daemon at localhost:9091"
[ -f "$HOME/.deer/images/Deer Source VM.qcow2" ] || fail "Source VM image not found"
log "Preflight: OK"

# ---- Step 1: Create sandbox ----

log "=========================================="
log "Step 1/7: Create sandbox (kafka-stub + es-stub)"
log "=========================================="

CREATE_OUT=$("$DEER_CLI" sandbox create "Deer Source VM" --kafka-stub --es-stub --memory 4096 2>&1) \
    || fail "sandbox create failed: $CREATE_OUT"

SANDBOX_ID=$(echo "$CREATE_OUT" | grep -oE 'sbx-[a-f0-9]+' | head -1)
[ -n "$SANDBOX_ID" ] || fail "no sandbox ID in: $CREATE_OUT"
SANDBOX_IP=$(echo "$CREATE_OUT" | grep -oE 'IP: [0-9.]+')
log "  ID:  $SANDBOX_ID"
log "  $SANDBOX_IP"
log "Step 1: DONE"

# ---- Step 2: Wait for cloud-init ----

log ""
log "=========================================="
log "Step 2/7: Wait for cloud-init (up to 15min)"
log "=========================================="

for i in $(seq 1 60); do
    ELAPSED=$((i * 15))

    # Check cloud-init status (SSH may not be ready yet)
    STATUS_RAW=$(sb_run "cloud-init status 2>&1" 2>/dev/null) || {
        log "  [${ELAPSED}s] SSH not ready, retrying ..."
        sleep 15
        continue
    }
    STATUS=$(echo "$STATUS_RAW" | grep -oE 'status: [a-z]+' | awk '{print $2}' || true)

    if [ "$STATUS" = "done" ]; then
        log "  [${ELAPSED}s] cloud-init DONE"
        break
    fi

    # Show progress every 30s
    if [ $((i % 2)) -eq 0 ]; then
        # Try to get current stage from logs
        STAGE=$(sb_run "sudo tail -5 /var/log/cloud-init-output.log 2>/dev/null | grep -oE 'deer (redpanda|elasticsearch|notify) [a-z _]+(start|complete|ready|fail)' | tail -1" 2>/dev/null) || STAGE=""
        if [ -n "$STAGE" ]; then
            log "  [${ELAPSED}s] running — $(echo "$STAGE" | head -1)"
        else
            SVC=$(sb_run "systemctl is-active deer-redpanda.service deer-elasticsearch.service 2>&1" 2>/dev/null) || SVC="unknown"
            RP_STATE=$(echo "$SVC" | head -1 | tr -d '\n')
            ES_STATE=$(echo "$SVC" | tail -1 | tr -d '\n')
            log "  [${ELAPSED}s] running — redpanda:${RP_STATE:-?} es:${ES_STATE:-?}"
        fi
    fi

    if [ "$i" -eq 60 ]; then
        log "TIMEOUT after 15min. Diagnostics:"
        sb_run "cloud-init status --long 2>&1" 2>/dev/null || true
        sb_run "sudo systemctl status deer-redpanda.service --no-pager 2>&1" 2>/dev/null || true
        sb_run "sudo systemctl status deer-elasticsearch.service --no-pager 2>&1" 2>/dev/null || true
        sb_run "df -h / 2>&1" 2>/dev/null || true
        fail "cloud-init did not finish in 15min"
    fi

    sleep 15
done
log "Step 2: DONE"

# ---- Step 3: Verify Redpanda ----

log ""
log "=========================================="
log "Step 3/7: Verify Redpanda"
log "=========================================="

PORTS=$(sb_run "ss -ltn | grep -E ':9092|:9200'" 2>/dev/null) || PORTS=""
log "  Listening ports: $(echo "$PORTS" | tr '\n' ' ')"

if ! echo "$PORTS" | grep -q ':9092'; then
    log "  Port 9092 not listening. Diagnostics:"
    sb_run "sudo systemctl status deer-redpanda.service --no-pager 2>&1" 2>/dev/null | while read -r l; do log "    $l"; done
    fail "Redpanda not listening on 9092"
fi

BROKERS=$(sb_run "curl -s http://localhost:9092/v1/brokers" 2>/dev/null) || fail "Redpanda broker query failed"
log "  Brokers: $BROKERS"
log "Step 3: PASS"

# ---- Step 4: Verify Elasticsearch ----

log ""
log "=========================================="
log "Step 4/7: Verify Elasticsearch"
log "=========================================="

if ! echo "$PORTS" | grep -q ':9200'; then
    log "  Port 9200 not listening. Giving ES more time (60s) ..."
    sleep 60
    PORTS=$(sb_run "ss -ltn | grep -E ':9200'" 2>/dev/null) || PORTS=""
    if ! echo "$PORTS" | grep -q ':9200'; then
        log "  Still not listening. Diagnostics:"
        sb_run "sudo systemctl status deer-elasticsearch.service --no-pager 2>&1" 2>/dev/null | while read -r l; do log "    $l"; done
        sb_run "sudo journalctl -u deer-elasticsearch.service --no-pager -n 20 2>&1" 2>/dev/null | while read -r l; do log "    $l"; done
        fail "Elasticsearch not listening on 9200"
    fi
fi

ES_INFO=$(sb_run "curl -s http://localhost:9200/") || fail "ES query failed"
ES_VERSION=$(echo "$ES_INFO" | grep -oE '"number" : "[^"]+"' | head -1)
log "  Version: $ES_VERSION"

HEALTH=$(sb_run "curl -s http://localhost:9200/_cluster/health") || fail "ES health check failed"
log "  Health: $(echo "$HEALTH" | grep -oE '"status":"[^"]+"')"
log "Step 4: PASS"

# ---- Step 5: Kafka produce/consume ----

log ""
log "=========================================="
log "Step 5/7: Kafka produce/consume"
log "=========================================="

TOPIC_OUT=$(sb_run "/usr/bin/rpk topic create stub-test-topic --brokers localhost:9092 2>&1") || fail "topic create failed: $TOPIC_OUT"
log "  Topic: $(echo "$TOPIC_OUT" | tr -d '\n')"

PRODUCE_OUT=$(sb_run "echo '{\"test\":\"e2e\",\"value\":42}' | /usr/bin/rpk topic produce stub-test-topic --brokers localhost:9092 2>&1") || fail "produce failed"
log "  Produced: $(echo "$PRODUCE_OUT" | tr -d '\n')"

CONSUME_OUT=$(sb_run "/usr/bin/rpk topic consume stub-test-topic --brokers localhost:9092 -n 1 2>&1") || fail "consume failed"
echo "$CONSUME_OUT" | tr -d '\r' | grep -q 'stub-test-topic' || fail "consume missing expected topic: $CONSUME_OUT"
log "  Consumed OK"
log "Step 5: PASS"

# ---- Step 6: ES index/search ----

log ""
log "=========================================="
log "Step 6/7: ES index/search"
log "=========================================="

INDEX_OUT=$(sb_run "curl -s -X POST 'http://localhost:9200/stub-test-index/_doc' -H 'Content-Type: application/json' -d '{\"test\":\"e2e\",\"status\":\"pass\"}'") || fail "index failed"
echo "$INDEX_OUT" | grep -q '"result":"created"' || fail "index unexpected: $INDEX_OUT"
log "  Indexed: $(echo "$INDEX_OUT" | grep -oE '"_id":"[^"]+"')"

sleep 1

SEARCH_OUT=$(sb_run "curl -s 'http://localhost:9200/stub-test-index/_search?q=e2e'") || fail "search failed"
echo "$SEARCH_OUT" | tr -d '\r' | grep -q '"status":"pass"' || fail "search missing doc: $SEARCH_OUT"
HITS=$(echo "$SEARCH_OUT" | grep -oE '"value":[0-9]+' | head -1 | cut -d: -f2)
log "  Search hits: $HITS"
log "Step 6: PASS"

# ---- Step 7: Disk ----

log ""
log "=========================================="
log "Step 7/7: Disk space"
log "=========================================="

DISK=$(sb_run "df -h / 2>&1")
PCT=$(echo "$DISK" | awk 'NR==2{print $5}')
AVAIL=$(echo "$DISK" | awk 'NR==2{print $4}')
log "  Usage: $PCT  Available: $AVAIL"
PCT_NUM=${PCT%\%}
[ "$PCT_NUM" -lt 95 ] || fail "disk ${PCT_NUM}% full"
log "Step 7: PASS"

# ---- Summary ----

log ""
log "=========================================="
log "  ALL 7 TESTS PASSED"
log "=========================================="
log "  Sandbox:  $SANDBOX_ID"
log "  $SANDBOX_IP"
log "  Redpanda: localhost:9092"
log "  ES:       localhost:9200 ($ES_VERSION)"
log "  Disk:     $PCT used"
log "=========================================="
