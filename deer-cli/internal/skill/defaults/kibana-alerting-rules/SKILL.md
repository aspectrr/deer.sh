---
name: kibana-alerting-rules
description: >
  Create and manage Kibana alerting rules via REST API or Terraform. Use when creating,
  updating, or managing rule lifecycle (enable, disable, mute, snooze) or rules-as-code
  workflows.
metadata:
  author: elastic
  version: 0.1.0
  source: elastic/agent-skills//skills/kibana/kibana-alerting-rules
---

# Kibana Alerting Rules

## Core Concepts

A rule has three parts: **conditions** (what to detect), **schedule** (how often to check), and **actions** (what
happens when conditions are met). When conditions are met, the rule creates **alerts**, which trigger **actions** via
**connectors**.

## Authentication

All alerting API calls require either API key auth or Basic auth. Every mutating request must include the `kbn-xsrf`
header.

```http
kbn-xsrf: true
```

## API Reference

Base path: `<kibana_url>/api/alerting` (or `/s/<space_id>/api/alerting` for non-default spaces).

| Operation         | Method | Endpoint                                                   |
| ----------------- | ------ | ---------------------------------------------------------- |
| Create rule       | POST   | `/api/alerting/rule/{id}`                                  |
| Update rule       | PUT    | `/api/alerting/rule/{id}`                                  |
| Get rule          | GET    | `/api/alerting/rule/{id}`                                  |
| Delete rule       | DELETE | `/api/alerting/rule/{id}`                                  |
| Find rules        | GET    | `/api/alerting/rules/_find`                                |
| List rule types   | GET    | `/api/alerting/rule_types`                                 |
| Enable rule       | POST   | `/api/alerting/rule/{id}/_enable`                          |
| Disable rule      | POST   | `/api/alerting/rule/{id}/_disable`                         |
| Mute all alerts   | POST   | `/api/alerting/rule/{id}/_mute_all`                        |
| Unmute all alerts | POST   | `/api/alerting/rule/{id}/_unmute_all`                      |

## Creating a Rule

### Required Fields

| Field          | Type   | Description                                                                                                                                           |
| -------------- | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| `name`         | string | Display name (does not need to be unique)                                                                                                             |
| `rule_type_id` | string | The rule type (e.g., `.es-query`, `.index-threshold`)                                                                                                 |
| `consumer`     | string | Owning app: `alerts`, `apm`, `discover`, `infrastructure`, `logs`, `metrics`, `ml`, `monitoring`, `securitySolution`, `siem`, `stackAlerts`, `uptime` |
| `params`       | object | Rule-type-specific parameters                                                                                                                         |
| `schedule`     | object | Check interval, e.g., `{"interval": "5m"}`                                                                                                            |

### Example: Create an Elasticsearch Query Rule

```bash
curl -X POST "https://my-kibana:5601/api/alerting/rule/my-rule-id" \
  -H "kbn-xsrf: true" \
  -H "Content-Type: application/json" \
  -H "Authorization: ApiKey <your-api-key>" \
  -d '{
    "name": "High error rate",
    "rule_type_id": ".es-query",
    "consumer": "stackAlerts",
    "schedule": { "interval": "5m" },
    "params": {
      "index": ["logs-*"],
      "timeField": "@timestamp",
      "esQuery": "{\"query\":{\"match\":{\"log.level\":\"error\"}}}",
      "threshold": [100],
      "thresholdComparator": ">",
      "timeWindowSize": 5,
      "timeWindowUnit": "m",
      "size": 100
    },
    "actions": [
      {
        "id": "my-slack-connector-id",
        "group": "query matched",
        "params": {
          "message": "Alert: {{rule.name}} - {{context.hits}} hits detected"
        }
      }
    ],
    "tags": ["production", "errors"]
  }'
```

## Finding Rules

```bash
curl -X GET "https://my-kibana:5601/api/alerting/rules/_find?per_page=20&page=1&search=cpu&sort_field=name&sort_order=asc" \
  -H "Authorization: ApiKey <your-api-key>"
```

## Lifecycle Operations

```bash
curl -X POST ".../api/alerting/rule/{id}/_enable" -H "kbn-xsrf: true"
curl -X POST ".../api/alerting/rule/{id}/_disable" -H "kbn-xsrf: true"
curl -X POST ".../api/alerting/rule/{id}/_mute_all" -H "kbn-xsrf: true"
curl -X DELETE ".../api/alerting/rule/{id}" -H "kbn-xsrf: true"
```

## Terraform Provider

```hcl
resource "elasticstack_kibana_alerting_rule" "cpu_alert" {
  name         = "CPU usage critical"
  consumer     = "stackAlerts"
  rule_type_id = ".index-threshold"
  interval     = "1m"
  enabled      = true

  params = jsonencode({
    index              = ["metrics-*"]
    timeField          = "@timestamp"
    aggType            = "avg"
    aggField           = "system.cpu.total.pct"
    groupBy            = "top"
    termField          = "host.name"
    termSize           = 10
    threshold          = [0.9]
    thresholdComparator = ">"
    timeWindowSize     = 5
    timeWindowUnit     = "m"
  })

  tags = ["infrastructure", "production"]
}
```

## Best Practices

1. **Set action frequency per action, not per rule.** The `notify_when` field at the rule level is deprecated in favor
   of per-action `frequency` objects.
2. **Use alert summaries to reduce notification noise.** Configure actions to send periodic summaries.
3. **Always add a recovery action.** Rules without a recovery action leave incidents open in PagerDuty, Jira, and
   ServiceNow indefinitely.
4. **Set a reasonable check interval.** The minimum recommended interval is `1m`.
5. **Use `alert_delay` to suppress transient spikes.** Setting `{"active": 3}` means the alert only fires after 3
   consecutive runs match the condition.
6. **Tag rules consistently.** Use tags like `production`, `staging`, `team-platform` for filtering.

## Common Pitfalls

1. **Missing `kbn-xsrf` header.** All POST, PUT, DELETE requests require `kbn-xsrf: true`.
2. **Wrong `consumer` value.** Check the rule type's supported consumers via `GET /api/alerting/rule_types`.
3. **Immutable fields on update.** `rule_type_id` and `consumer` cannot be changed with PUT.
4. **Rule-level `notify_when` is deprecated.** Always use `frequency` inside each action object.
5. **API key ownership.** Rules run using the API key of the user who created them. If that user's permissions change,
   the rule may fail silently.

## Guidelines

- Include `kbn-xsrf: true` on every POST, PUT, and DELETE.
- Set `frequency` inside each action object — rule-level `notify_when` and `throttle` are deprecated.
- `rule_type_id` and `consumer` are immutable after creation; delete and recreate the rule to change them.
- Prefix paths with `/s/<space_id>/api/alerting/` for non-default Kibana Spaces.
- Always pair an active action with a `Recovered` action to auto-close incidents.
- Run `GET /api/alerting/rule_types` first to discover valid `consumer` values and action group names.
