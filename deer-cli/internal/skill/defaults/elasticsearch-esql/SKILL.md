---
name: elasticsearch-esql
description: >
  Execute ES|QL (Elasticsearch Query Language) queries, use when the user wants to
  query Elasticsearch data, analyze logs, aggregate metrics, explore data, or create
  charts and dashboards from ES|QL results.
metadata:
  author: elastic
  version: 0.1.1
  source: elastic/agent-skills//skills/elasticsearch/elasticsearch-esql
---

# Elasticsearch ES|QL

Execute ES|QL queries against Elasticsearch.

## What is ES|QL?

ES|QL (Elasticsearch Query Language) is a piped query language for Elasticsearch. It is **NOT** the same as:

- Elasticsearch Query DSL (JSON-based)
- SQL
- EQL (Event Query Language)

ES|QL uses pipes (`|`) to chain commands:
`FROM index | WHERE condition | STATS aggregation BY field | SORT field | LIMIT n`

> **Prerequisite:** ES|QL requires `_source` to be enabled on queried indices. Indices with `_source` disabled (e.g.,
> `"_source": { "enabled": false }`) will cause ES|QL queries to fail.
>
> **Version Compatibility:** ES|QL was introduced in 8.11 (tech preview) and became GA in 8.14. Features like
> `LOOKUP JOIN` (8.18+), `MATCH` (8.17+), and `INLINE STATS` (9.2+) were added in later versions. On pre-8.18 clusters,
> use `ENRICH` as a fallback for `LOOKUP JOIN` (see generation tips). `INLINE STATS` and counter-field `RATE()` have
> **no fallback** before 9.2. Check [references/esql-version-history.md](https://github.com/elastic/agent-skills/blob/main/skills/elasticsearch/elasticsearch-esql/references/esql-version-history.md) for feature
> availability by version.

### Environment Configuration

See [Environment Setup](https://github.com/elastic/agent-skills/blob/main/skills/elasticsearch/elasticsearch-esql/references/environment-setup.md) for full connection configuration options (Elastic Cloud,
direct URL, basic auth, local development).

## Usage

### Get Index Information (for schema discovery)

```bash
node scripts/esql.js indices                    # List all indices
node scripts/esql.js indices "logs-*"           # List matching indices
node scripts/esql.js schema "logs-2024.01.01"   # Get field mappings for an index
```

### Execute Raw ES|QL

```bash
node scripts/esql.js raw "FROM logs-* | STATS count = COUNT(*) BY host.name | SORT count DESC | LIMIT 5"
```

### Execute with TSV Output

```bash
node scripts/esql.js raw "FROM logs-* | STATS count = COUNT(*) BY component | SORT count DESC" --tsv
```

### Test Connection

```bash
node scripts/esql.js test
```

## Guidelines

1. **Detect deployment type**: Always run `node scripts/esql.js test` first. This detects whether the deployment is a
   Serverless project (all features available) or a versioned cluster (features depend on version).

2. **Discover schema** (required — never guess index or field names):

   ```bash
   node scripts/esql.js indices "pattern*"
   node scripts/esql.js schema "index-name"
   ```

   Always run schema discovery before generating queries. Index names and field names vary across deployments and cannot
   be reliably guessed.

3. **Choose the right ES|QL feature for the task**: Before writing queries, match the user's intent to the most
   appropriate ES|QL feature.
   - "find patterns," "categorize," "group similar messages" → `CATEGORIZE(field)`
   - "spike," "dip," "anomaly," "when did X change" → `CHANGE_POINT value ON key`
   - "trend over time," "time series" → `STATS ... BY BUCKET(@timestamp, interval)` or `TS` for TSDB
   - "search," "find documents matching" → `MATCH` (default), `QSTR` (advanced boolean), `KQL` (Kibana migration)
   - "count," "average," "breakdown" → `STATS` with aggregation functions

4. **Generate the query** following ES|QL syntax. Prefer the **simplest query** that answers the question.

5. **Execute with TSV flag**:

   ```bash
   node scripts/esql.js raw "FROM index | STATS count = COUNT(*) BY field" --tsv
   ```

## ES|QL Quick Reference

### Basic Structure

```esql
FROM index-pattern
| WHERE condition
| EVAL new_field = expression
| STATS aggregation BY grouping
| SORT field DESC
| LIMIT n
```

### Common Patterns

**Filter and limit:**

```esql
FROM logs-*
| WHERE @timestamp > NOW() - 24 hours AND level == "error"
| SORT @timestamp DESC
| LIMIT 100
```

**Aggregate by time:**

```esql
FROM metrics-*
| WHERE @timestamp > NOW() - 7 days
| STATS avg_cpu = AVG(cpu.percent) BY bucket = DATE_TRUNC(1 hour, @timestamp)
| SORT bucket DESC
```

**Top N with count:**

```esql
FROM web-logs
| STATS count = COUNT(*) BY response.status_code
| SORT count DESC
| LIMIT 10
```

**Text search (8.17+):**

```esql
FROM documents METADATA _score
| WHERE MATCH(content, "search terms")
| SORT _score DESC
| LIMIT 20
```

**Log categorization (Platinum license):**

```esql
FROM logs-*
| WHERE @timestamp > NOW() - 24 hours
| STATS count = COUNT(*) BY category = CATEGORIZE(message)
| SORT count DESC
| LIMIT 20
```

**Change point detection (Platinum license):**

```esql
FROM logs-*
| STATS c = COUNT(*) BY t = BUCKET(@timestamp, 30 seconds)
| SORT t
| CHANGE_POINT c ON t
| WHERE type IS NOT NULL
```

## Full Reference

For complete ES|QL syntax including all commands, functions, and operators, see:

- [ES|QL Complete Reference](https://github.com/elastic/agent-skills/blob/main/skills/elasticsearch/elasticsearch-esql/references/esql-reference.md)
- [ES|QL Search Reference](https://github.com/elastic/agent-skills/blob/main/skills/elasticsearch/elasticsearch-esql/references/esql-search.md)
- [ES|QL Version History](https://github.com/elastic/agent-skills/blob/main/skills/elasticsearch/elasticsearch-esql/references/esql-version-history.md)
- [Query Patterns](https://github.com/elastic/agent-skills/blob/main/skills/elasticsearch/elasticsearch-esql/references/query-patterns.md)
- [Generation Tips](https://github.com/elastic/agent-skills/blob/main/skills/elasticsearch/elasticsearch-esql/references/generation-tips.md)
- [Time Series Queries](https://github.com/elastic/agent-skills/blob/main/skills/elasticsearch/elasticsearch-esql/references/time-series-queries.md)
- [DSL to ES|QL Migration](https://github.com/elastic/agent-skills/blob/main/skills/elasticsearch/elasticsearch-esql/references/dsl-to-esql-migration.md)
- [Environment Setup](https://github.com/elastic/agent-skills/blob/main/skills/elasticsearch/elasticsearch-esql/references/environment-setup.md)
