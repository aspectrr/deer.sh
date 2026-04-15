---
name: observability-service-health
description: >
  Assess APM service health using SLOs, alerts, ML, throughput, latency, error rate,
  and dependencies. Use when checking service status, performance, or when the user
  asks about service health.
metadata:
  author: elastic
  version: 0.1.0
  source: elastic/agent-skills//skills/observability/service-health
---

# APM Service Health

Assess APM service health using Observability APIs, ES|QL against APM indices, and Elasticsearch APIs. Use
SLOs, firing alerts, ML anomalies, throughput, latency, error rate, and dependency health.

## Health criteria

Synthesize health from all of the following when available:

| Signal                | What to check                                                             |
| --------------------- | ------------------------------------------------------------------------- |
| **SLOs**              | Burn rate, status (healthy/degrading/violated), error budget.             |
| **Firing alerts**     | Open or recently fired alerts for the service or dependencies.            |
| **ML anomalies**      | Anomaly jobs; score and severity for latency, throughput, or error rate.  |
| **Throughput**        | Request rate; compare to baseline or previous period.                     |
| **Latency**           | Avg, p95, p99; compare to SLO targets or history.                         |
| **Error rate**        | Failed/total requests; spikes or sustained elevation.                     |
| **Dependency health** | Downstream latency, error rate, availability.                            |
| **Infrastructure**    | CPU usage, memory; OOM and CPU throttling on pods/containers/hosts.       |
| **Logs**              | App logs filtered by service or trace ID for context and root cause.      |

## Using ES|QL for APM metrics

Always filter by `service.name` (and `service.environment` when relevant). Combine with a time range on `@timestamp`:

```esql
WHERE service.name == "my-service-name" AND service.environment == "production"
  AND @timestamp >= "2025-03-01T00:00:00Z" AND @timestamp <= "2025-03-01T23:59:59Z"
```

### Example: Throughput and error rate

```esql
FROM traces*apm*,traces*otel*
| WHERE service.name == "api-gateway"
  AND @timestamp >= "2025-03-01T00:00:00Z" AND @timestamp <= "2025-03-01T23:59:59Z"
| STATS request_count = COUNT(*), failures = COUNT(*) WHERE event.outcome == "failure" BY BUCKET(@timestamp, 1 hour)
| EVAL error_rate = failures / request_count
| SORT @timestamp
| LIMIT 500
```

## Workflow

```text
- [ ] Step 1: Identify the service (and time range)
- [ ] Step 2: Check SLOs and firing alerts
- [ ] Step 3: Check ML anomalies (if configured)
- [ ] Step 4: Review throughput, latency (avg/p95/p99), error rate
- [ ] Step 5: Assess dependency health
- [ ] Step 6: Correlate with infrastructure and logs
- [ ] Step 7: Summarize health and recommend actions
```

### Step 1: Identify the service

Confirm service name and time range. If the user has not provided the time range, assume last hour.

### Step 2: Check SLOs and firing alerts

**SLOs:** Call the SLOs API to get SLO definitions and status for the service.
**Alerts:** For active APM alerts, call `/api/alerting/rules/_find?search=apm&search_fields=tags&per_page=100&filter=alert.attributes.executionStatus.status:active`.

### Step 3: Check ML anomalies

If ML anomaly detection is used, query ML job results for the service and time range.

### Step 4: Review throughput, latency, and error rate

Use ES|QL against `traces*apm*,traces*otel*` or `metrics*apm*,metrics*otel*` for throughput, latency, and error rate.

### Step 5: Assess dependency health

Obtain dependency data via ES|QL on traces or metrics. Flag slow or failing dependencies.

### Step 6: Correlate with infrastructure and logs

- **Infrastructure:** Use resource attributes from traces (`k8s.pod.name`, `container.id`, `host.name`) and query
  infrastructure indices for CPU and memory.
- **Logs:** Use ES|QL or Elasticsearch on log indices with `service.name` or `trace.id` to explain behavior.

### Step 7: Summarize and recommend

State health (**healthy** / **degraded** / **unhealthy**) with reasons; list concrete next steps.

## Guidelines

- Use Observability APIs and ES|QL on `traces*apm*,traces*otel*`/`metrics*apm*,metrics*otel*`.
- Always use the **user's time range**; avoid assuming "last 1 hour" if the issue is historical.
- When SLOs exist, anchor the health summary to SLO status and burn rate.
- Add `LIMIT n` to cap rows and token usage.
- Prefer coarser `BUCKET(@timestamp, ...)` when only trends are needed.
