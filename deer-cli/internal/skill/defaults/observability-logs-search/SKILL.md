---
name: observability-logs-search
description: >
  Search and filter Observability logs using ES|QL. Use when investigating log spikes,
  errors, or anomalies; getting volume and trends; or drilling into services or containers
  during incidents.
metadata:
  author: elastic
  version: 0.2.0
  source: elastic/agent-skills//skills/observability/logs-search
---

# Logs Search

Search and filter logs to support incident investigation. The workflow mirrors Kibana Discover: apply a time range and
scope filter, then **iteratively add exclusion filters (NOT)** until a small, interesting subset of logs remains. Use
ES|QL only (`POST /_query`); do not use Query DSL.

## Parameter conventions

| Parameter   | Type   | Description                                                                 |
| ----------- | ------ | --------------------------------------------------------------------------- |
| `start`     | string | Start of time range (Elasticsearch date math, e.g. `now-1h`)                |
| `end`       | string | End of time range (e.g. `now`)                                              |
| `kqlFilter` | string | KQL query string to narrow results                                          |
| `limit`     | number | Maximum log samples to return (e.g. 10–100)                                 |
| `groupBy`   | string | Optional field to group the histogram by (e.g. `log.level`, `service.name`) |

### Context minimization

Keep the context window small. In the sample branch of the query, **KEEP only a subset of fields**; do not return full
documents by default.

**Recommended KEEP list for sample logs:**
`message`, `error.message`, `service.name`, `container.name`, `host.name`, `container.id`, `agent.name`,
`kubernetes.container.name`, `kubernetes.node.name`, `kubernetes.namespace`, `kubernetes.pod.name`

## The funnel workflow

**You must iterate.** Do not stop after one query. Keep excluding noise with `NOT` until **fewer than 20 log patterns**
remain.

1. **Round 1 — broad:** Run a query with only the scope filter and time range.
2. **Inspect:** Look at the histogram, sample messages, and categorized patterns.
3. **Round 2 — exclude noise:** Add `NOT` clauses to the KQL filter for dominant noise patterns.
4. **Repeat:** Keep adding NOTs until fewer than 20 log patterns remain.
5. **Pivot (optional):** Once the funnel isolates a specific entity, run one more query focused on that entity.

## ES|QL patterns for log search

Use ES|QL (`POST /_query`) only. Always return: a time-series histogram, total count, a small sample of logs, and
message categorization. Use `FORK` to compute all in a single query.

### Basic log search with histogram, samples, and categorization

```json
POST /_query
{
  "query": "FROM logs-* METADATA _id, _index | WHERE @timestamp >= TO_DATETIME(\"2025-03-06T10:00:00.000Z\") AND @timestamp <= TO_DATETIME(\"2025-03-06T11:00:00.000Z\") | FORK (STATS count = COUNT(*) BY bucket = BUCKET(@timestamp, 1m) | SORT bucket) (STATS total = COUNT(*)) (SORT @timestamp DESC | LIMIT 10 | KEEP _id, _index, message, error.message, service.name, container.name, host.name) (LIMIT 10000 | STATS COUNT(*) BY CATEGORIZE(message) | SORT `COUNT(*)` DESC | LIMIT 20) (LIMIT 10000 | STATS COUNT(*) BY CATEGORIZE(message) | SORT `COUNT(*)` ASC | LIMIT 20)"
}
```

### Adding a KQL filter

```json
POST /_query
{
  "query": "FROM logs-* METADATA _id, _index | WHERE @timestamp >= TO_DATETIME(\"2025-03-06T10:00:00.000Z\") AND @timestamp <= TO_DATETIME(\"2025-03-06T11:00:00.000Z\") | WHERE KQL(\"service.name: checkout AND log.level: error\") | FORK (STATS count = COUNT(*) BY bucket = BUCKET(@timestamp, 1m) | SORT bucket) (STATS total = COUNT(*)) (SORT @timestamp DESC | LIMIT 10 | KEEP _id, _index, message, error.message, service.name) (LIMIT 10000 | STATS COUNT(*) BY CATEGORIZE(message) | SORT `COUNT(*)` DESC | LIMIT 20) (LIMIT 10000 | STATS COUNT(*) BY CATEGORIZE(message) | SORT `COUNT(*)` ASC | LIMIT 20)"
}
```

## Examples

### Last hour of logs for a service

```json
POST /_query
{
  "query": "FROM logs-* METADATA _id, _index | WHERE @timestamp >= NOW() - 1 hour AND @timestamp <= NOW() | WHERE KQL(\"service.name: api-gateway\") | SORT @timestamp DESC | LIMIT 20"
}
```

### Error logs with trend and samples

```json
POST /_query
{
  "query": "FROM logs-* METADATA _id, _index | WHERE @timestamp >= NOW() - 2 hours AND @timestamp <= NOW() | WHERE KQL(\"log.level: error\") | FORK (STATS count = COUNT(*) BY bucket = BUCKET(@timestamp, 5m) | SORT bucket) (STATS total = COUNT(*)) (SORT @timestamp DESC | LIMIT 15)"
}
```

## Guidelines

- **Funnel: iterate with NOT.** Do not report findings after a single broad query.
- **Histogram first:** Use the trend to see when spikes or drops occur.
- **Context minimization:** KEEP only summary fields; default LIMIT 10–20, cap at 500.
- **Request body escaping:** The `query` value is JSON. Escape double quotes: `\"` for the KQL wrapper.
- Use Elasticsearch date math for `start` and `end`.
- Choose bucket size from the time range: aim for roughly 20–50 buckets.
- Prefer ECS field names.
