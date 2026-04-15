---
name: observability-llm-obs
description: >
  Monitor LLMs and agentic apps: performance, token/cost, response quality, and workflow
  orchestration. Use when the user asks about LLM monitoring, GenAI observability,
  or AI cost/quality.
metadata:
  author: elastic
  version: 0.1.0
  source: elastic/agent-skills//skills/observability/llm-obs
---

# LLM and Agentic Observability

Monitor LLMs and agentic components using data ingested into Elastic. Focus on performance, cost/token utilization,
response quality, and call chaining.

## Where to look

- **Trace data (APM / OTel):** `traces*` for LLM spans from OTel/EDOT instrumentations
- **Integration metrics/logs:** `metrics*` and `logs*` from Elastic LLM integrations (OpenAI, Azure, Bedrock, Vertex AI)
- **Discover first:** Use `GET _data_stream` or `GET traces*/_mapping` to find available data

## Data available

### From traces (traces*)

| Purpose              | Example attribute names (OTel GenAI)                      |
| -------------------- | --------------------------------------------------------- |
| Operation / provider | `gen_ai.operation.name`, `gen_ai.provider.name`           |
| Model                | `gen_ai.request.model`, `gen_ai.response.model`           |
| Token usage          | `gen_ai.usage.input_tokens`, `gen_ai.usage.output_tokens` |
| Errors               | `error.type`                                              |

Use **duration** and **event.outcome** for latency and success/failure. Use **trace.id** and parent/child relationships
for call chaining analysis.

## Use cases and query patterns

### LLM performance

```esql
FROM traces*
| WHERE @timestamp >= "2025-03-01T00:00:00Z" AND @timestamp <= "2025-03-01T23:59:59Z"
  AND span.attributes.gen_ai.provider.name IS NOT NULL
| STATS request_count = COUNT(*), failures = COUNT(*) WHERE event.outcome == "failure",
    avg_duration_us = AVG(span.duration.us)
  BY span.attributes.gen_ai.request.model
| EVAL error_rate = failures / request_count
| LIMIT 100
```

### Token usage over time

```esql
FROM traces*
| WHERE @timestamp >= "2025-03-01T00:00:00Z" AND @timestamp <= "2025-03-01T23:59:59Z"
  AND span.attributes.gen_ai.provider.name IS NOT NULL
| STATS input_tokens = SUM(span.attributes.gen_ai.usage.input_tokens),
    output_tokens = SUM(span.attributes.gen_ai.usage.output_tokens)
  BY BUCKET(@timestamp, 1 hour), span.attributes.gen_ai.request.model
| SORT @timestamp
| LIMIT 500
```

### Agentic workflow (trace-level view)

```esql
FROM traces*
| WHERE @timestamp >= "2025-03-01T00:00:00Z" AND @timestamp <= "2025-03-01T23:59:59Z"
  AND span.attributes.gen_ai.operation.name IS NOT NULL
| STATS span_count = COUNT(*), total_duration_us = SUM(span.duration.us) BY trace.id
| WHERE span_count > 1
| SORT total_duration_us DESC
| LIMIT 50
```

## Workflow

```text
- [ ] Step 1: Determine available data (traces*, metrics*, integration data streams)
- [ ] Step 2: Discover LLM-related field names (mapping or sample doc)
- [ ] Step 3: Run ES|QL queries for the user's question
- [ ] Step 4: Check active alerts/SLOs on LLM-related data
- [ ] Step 5: Summarize findings from ingested data only
```

## Guidelines

- Use only data collected in Elastic. Do not rely on external UIs.
- Discover field names from `_mapping` or sample documents before querying.
- Prefer ES|QL and Elasticsearch APIs over Kibana UI.
- Use `LIMIT` and coarse time buckets for performance.
