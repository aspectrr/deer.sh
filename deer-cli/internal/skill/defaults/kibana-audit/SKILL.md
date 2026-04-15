---
name: kibana-audit
description: >
  Enable and configure Kibana audit logging for saved object access, logins, and space
  operations. Use when setting up Kibana audit, filtering events, or correlating Kibana
  and ES audit logs.
metadata:
  author: elastic
  version: 0.1.0
  source: elastic/agent-skills//skills/kibana/kibana-audit
---

# Kibana Audit Logging

Enable and configure audit logging for Kibana via `kibana.yml`. Covers application-layer security events: saved object
CRUD, login/logout, session expiry, and space operations.

## Enable Kibana Audit Logging

```yaml
xpack.security.audit.enabled: true
xpack.security.audit.appender:
  type: rolling-file
  fileName: /path/to/kibana/data/audit.log
  policy:
    type: time-interval
    interval: 24h
  strategy:
    type: numeric
    max: 10
```

A Kibana restart is required after changes.

## Event Types

| Event action                       | Description                                  |
| ---------------------------------- | -------------------------------------------- |
| `saved_object_create`              | A saved object was created                   |
| `saved_object_get`                 | A saved object was read                      |
| `saved_object_update`              | A saved object was updated                   |
| `saved_object_delete`              | A saved object was deleted                   |
| `saved_object_find`                | A saved object search was performed          |
| `login`                            | A user logged in (success or failure)        |
| `logout`                           | A user logged out                            |
| `session_cleanup`                  | An expired session was cleaned up            |
| `space_create/update/delete`       | Space operations                             |

## Filter Policies

```yaml
xpack.security.audit.ignore_filters:
  - actions: [saved_object_find]
    categories: [database]
```

## Correlate with ES Audit Logs

Both Kibana and ES record the same `trace.id` (via `X-Opaque-Id` header). This is the primary correlation key.

### Search ES audit by trace ID

```bash
curl -X POST "${ELASTICSEARCH_URL}/.security-audit-*/_search" \
  <auth_flags> \
  -H "Content-Type: application/json" \
  -d '{
    "query": {
      "bool": {
        "filter": [
          { "term": { "trace.id": "'"${TRACE_ID}"'" } },
          { "range": { "@timestamp": { "gte": "now-24h" } } }
        ]
      }
    },
    "sort": [{ "@timestamp": { "order": "asc" } }]
  }'
```

## Ship Kibana Audit to Elasticsearch

```yaml
filebeat.inputs:
  - type: log
    paths: ["/path/to/kibana/data/audit.log"]
    json.keys_under_root: true

output.elasticsearch:
  hosts: ["https://localhost:9200"]
  index: "kibana-audit-%{+yyyy.MM.dd}"
```

## Deployment Compatibility

| Capability                  | Self-managed | ECH          | Serverless    |
| --------------------------- | ------------ | ------------ | ------------- |
| Kibana audit                | Yes          | Via Cloud UI | Not available |
| Correlate via `trace.id`    | Yes          | Yes          | Not available |

## Guidelines

- Always enable alongside Elasticsearch audit for full coverage.
- Use `trace.id` for correlation between Kibana and ES events.
- Filter noisy `saved_object_find` events to reduce volume.
- Ship logs to Elasticsearch via Filebeat for unified querying.
