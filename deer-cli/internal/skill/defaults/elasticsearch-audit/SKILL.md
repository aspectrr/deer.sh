---
name: elasticsearch-audit
description: >
  Enable, configure, and query Elasticsearch security audit logs. Use when the task
  involves audit logging setup, event filtering, or investigating security incidents
  like failed logins.
metadata:
  author: elastic
  version: 0.1.0
  source: elastic/agent-skills//skills/elasticsearch/elasticsearch-audit
---

# Elasticsearch Audit Logging

Enable and configure security audit logging for Elasticsearch via the cluster settings API. Audit logs record security
events such as authentication attempts, access grants and denials, role changes, and API key operations.

For detailed API endpoints and event types, see [references/api-reference.md](https://github.com/elastic/agent-skills/blob/main/skills/elasticsearch/elasticsearch-audit/references/api-reference.md).

## Jobs to Be Done

- Enable or disable security audit logging on a cluster
- Select which security events to record (authentication, access, config changes)
- Create filter policies to reduce audit log noise
- Query audit logs for failed authentication attempts
- Investigate unauthorized access or privilege escalation incidents
- Set up compliance-focused audit configuration
- Detect brute-force login patterns from audit data
- Configure audit output to an index for programmatic querying

## Prerequisites

| Item                   | Description                                                                |
| ---------------------- | -------------------------------------------------------------------------- |
| **Elasticsearch URL**  | Cluster endpoint (e.g. `https://localhost:9200` or a Cloud deployment URL) |
| **Authentication**     | Valid credentials (see the elasticsearch-authn skill)                      |
| **Cluster privileges** | `manage` cluster privilege to update cluster settings                      |
| **License**            | Audit logging requires a gold, platinum, enterprise, or trial license      |

## Enable Audit Logging

```bash
curl -X PUT "${ELASTICSEARCH_URL}/_cluster/settings" \
  <auth_flags> \
  -H "Content-Type: application/json" \
  -d '{
    "persistent": {
      "xpack.security.audit.enabled": true
    }
  }'
```

## Audit Output

| Output      | Setting value | Description                                                    |
| ----------- | ------------- | -------------------------------------------------------------- |
| **logfile** | `logfile`     | Written to `<ES_HOME>/logs/<cluster>_audit.json`. Default.     |
| **index**   | `index`       | Written to `.security-audit-*` indices. Queryable via the API. |

```bash
curl -X PUT "${ELASTICSEARCH_URL}/_cluster/settings" \
  <auth_flags> \
  -H "Content-Type: application/json" \
  -d '{
    "persistent": {
      "xpack.security.audit.enabled": true,
      "xpack.security.audit.outputs": ["index", "logfile"]
    }
  }'
```

## Select Events to Record

### Include specific events only

```bash
curl -X PUT "${ELASTICSEARCH_URL}/_cluster/settings" \
  <auth_flags> \
  -H "Content-Type: application/json" \
  -d '{
    "persistent": {
      "xpack.security.audit.logfile.events.include": [
        "authentication_failed",
        "access_denied",
        "access_granted",
        "anonymous_access_denied",
        "tampered_request",
        "run_as_denied",
        "connection_denied"
      ]
    }
  }'
```

### Event types reference

| Event                     | Fires when                                                 |
| ------------------------- | ---------------------------------------------------------- |
| `authentication_failed`   | Credentials were rejected                                  |
| `authentication_success`  | User authenticated successfully                            |
| `access_granted`          | An authorized action was performed                         |
| `access_denied`           | An action was denied due to insufficient privileges        |
| `anonymous_access_denied` | An unauthenticated request was rejected                    |
| `tampered_request`        | A request was detected as tampered with                    |
| `connection_granted`      | A node joined the cluster (transport layer)                |
| `connection_denied`       | A node connection was rejected                             |
| `security_config_change`  | A security setting was changed (role, user, API key, etc.) |

## Filter Policies

Filter policies suppress specific audit events by user, realm, role, or index.

```bash
curl -X PUT "${ELASTICSEARCH_URL}/_cluster/settings" \
  <auth_flags> \
  -H "Content-Type: application/json" \
  -d '{
    "persistent": {
      "xpack.security.audit.logfile.events.ignore_filters": {
        "system_users": {
          "users": ["_xpack_security", "_xpack", "elastic/fleet-server"],
          "realms": ["_service_account"]
        }
      }
    }
  }'
```

## Query Audit Events

### Search for failed authentication attempts

```bash
curl -X POST "${ELASTICSEARCH_URL}/.security-audit-*/_search" \
  <auth_flags> \
  -H "Content-Type: application/json" \
  -d '{
    "query": {
      "bool": {
        "filter": [
          { "term": { "event.action": "authentication_failed" } },
          { "range": { "@timestamp": { "gte": "now-24h" } } }
        ]
      }
    },
    "sort": [{ "@timestamp": { "order": "desc" } }],
    "size": 50
  }'
```

### Search for access denied events

```bash
curl -X POST "${ELASTICSEARCH_URL}/.security-audit-*/_search" \
  <auth_flags> \
  -H "Content-Type: application/json" \
  -d '{
    "query": {
      "bool": {
        "filter": [
          { "term": { "event.action": "access_denied" } },
          { "range": { "@timestamp": { "gte": "now-7d" } } }
        ]
      }
    },
    "sort": [{ "@timestamp": { "order": "desc" } }],
    "size": 20
  }'
```

## Deployment Compatibility

| Capability                           | Self-managed | ECH          | Serverless    |
| ------------------------------------ | ------------ | ------------ | ------------- |
| ES audit via cluster settings        | Yes          | Yes          | Not available |
| ES logfile output                    | Yes          | Via Cloud UI | Not available |
| ES index output                      | Yes          | Yes          | Not available |
| Filter policies via cluster settings | Yes          | Yes          | Not available |

## Guidelines

### Prefer index output for programmatic access

Enable the `index` output to make audit events queryable. The `logfile` output is better for shipping to external SIEM
tools via Filebeat but cannot be queried through the Elasticsearch API.

### Start restrictive, then widen

Begin with failure events only (`authentication_failed`, `access_denied`, `security_config_change`). Add success events
only when needed — they generate high volume.

### Monitor audit index size

Set up an ILM policy to roll over and delete old `.security-audit-*` indices. A 30-90 day retention is typical.
