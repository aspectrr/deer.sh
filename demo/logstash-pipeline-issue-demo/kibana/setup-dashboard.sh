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
curl -sf -X POST "${KIBANA_URL}/api/saved_objects/index-pattern/weather-all?overwrite=true" \
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
curl -sf -X POST "${KIBANA_URL}/api/saved_objects/dashboard/deer-weather-demo?overwrite=true" \
    -H "kbn-xsrf: true" \
    -H "Content-Type: application/json" \
    -d '{
  "attributes": {
    "title": "deer Demo: Weather by City",
    "description": "Live weather data from the Kafka stub pipeline. Events appear here after the Logstash grok bug is fixed.",
    "panelsJSON": "[]",
    "optionsJSON": "{\"useMargins\":true}",
    "version": 1,
    "timeRestore": true,
    "timeTo": "now",
    "timeFrom": "now-1h",
    "refreshInterval": {"pause": false, "value": 10000},
    "kibanaSavedObjectMeta": {
      "searchSourceJSON": "{\"query\":{\"language\":\"kuery\",\"query\":\"\"},\"filter\":[]}"
    }
  }
}' > /dev/null
log "Dashboard created: deer Demo: Weather by City"

log ""
log "Kibana setup complete."
log "  Dashboard: ${KIBANA_URL}/app/dashboards"
log "  Index pattern covers: weather-new-york-*, weather-chicago-*, weather-sf-*, weather-la-*, weather-indy-*"
