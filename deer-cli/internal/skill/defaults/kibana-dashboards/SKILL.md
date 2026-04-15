---
name: kibana-dashboards
description: >
  Create and manage Kibana Dashboards and Lens visualizations. Use when you need to
  define dashboards and visualizations declaratively, version control them, or automate
  their deployment.
metadata:
  author: elastic
  version: 0.1.0
  source: elastic/agent-skills//skills/kibana/kibana-dashboards
---

# Kibana Dashboards and Visualizations

Declarative, Git-friendly format for defining dashboards and visualizations. **Version Requirement:** Kibana 9.4+ (SNAPSHOT).

## Environment Configuration

```bash
export KIBANA_URL="https://your-kibana:5601"
export KIBANA_API_KEY="base64encodedapikey"
```

Test: `node scripts/kibana-dashboards.js test`

## Basic Workflow

```bash
node scripts/kibana-dashboards.js dashboard get <id>
echo '<json>' | node scripts/kibana-dashboards.js dashboard create -
echo '<json>' | node scripts/kibana-dashboards.js dashboard update <id> -
node scripts/kibana-dashboards.js dashboard delete <id>
node scripts/kibana-dashboards.js lens list
node scripts/kibana-dashboards.js lens get <id>
```

## Dashboard Definition

```json
{
  "title": "My Dashboard",
  "panels": [
    {
      "type": "lens",
      "uid": "metric-panel",
      "grid": { "x": 0, "y": 0, "w": 12, "h": 6 },
      "config": {
        "attributes": {
          "title": "",
          "type": "metric",
          "dataset": { "type": "esql", "query": "FROM logs | STATS total = COUNT(*)" },
          "metrics": [{ "type": "primary", "operation": "value", "column": "total" }]
        }
      }
    }
  ],
  "time_range": { "from": "now-24h", "to": "now" }
}
```

## Grid System

48-column, infinite-row grid. Target 8-12 panels above the fold.

| Width   | Columns | Use Case                 |
| ------- | ------- | ------------------------ |
| Full    | 48      | Wide time series, tables |
| Half    | 24      | Primary charts           |
| Quarter | 12      | KPI metrics              |

## Supported Chart Types

| Type                                          | Description                 |
| --------------------------------------------- | --------------------------- |
| `metric`                                      | Single metric value display |
| `xy`                                          | Line, area, bar charts      |
| `gauge`                                       | Gauge visualizations        |
| `heatmap`                                     | Heatmap charts              |
| `datatable`                                   | Data tables                 |
| `pie`, `donut`, `treemap`, `mosaic`, `waffle` | Partition charts            |

## Dataset Types

- **dataView** — Use aggregation operations. Kibana performs aggregations automatically.
- **esql** — Use ES|QL query. Reference output columns with `{ operation: 'value', column: 'col' }`.
- **index** — Ad-hoc index patterns.

## ES|QL Time Bucketing

Use `BUCKET(@timestamp, 75, ?_tstart, ?_tend)` for auto-scaling time buckets.

## Guidelines

1. **Design for density** — 8-12 panels above the fold.
2. **Never use Markdown panels** for titles/headers.
3. **Inline Lens definitions** — Prefer `config.attributes` over `config.savedObjectId`.
4. **Test connection first** — Run `node scripts/kibana-dashboards.js test`.
