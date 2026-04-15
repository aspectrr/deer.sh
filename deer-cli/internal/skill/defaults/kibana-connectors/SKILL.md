---
name: kibana-connectors
description: >
  Create and manage Kibana connectors for Slack, PagerDuty, Jira, webhooks, and more
  via REST API or Terraform. Use when configuring third-party integrations or managing
  connectors as code.
metadata:
  author: elastic
  version: 0.1.1
  source: elastic/agent-skills//skills/kibana/kibana-connectors
---

# Kibana Connectors

## Core Concepts

Connectors store connection information for Elastic services and third-party systems. Alerting rules use connectors to
route **actions** (notifications) when rule conditions are met. Connectors are managed per **Kibana Space**.

### Connector Categories

| Category                    | Connector Types                                                                                                                                                      |
| --------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **LLM Providers**           | OpenAI, Google Gemini, Amazon Bedrock, Elastic Managed LLMs, AI Connector, MCP (Preview, 9.3+)                                                                       |
| **Incident Management**     | PagerDuty, Opsgenie, ServiceNow (ITSM, SecOps, ITOM), Jira, Jira Service Management (9.2+), IBM Resilient, Swimlane, Torq, Tines, D3 Security, XSOAR (9.1+), TheHive |
| **Endpoint Security**       | CrowdStrike, SentinelOne, Microsoft Defender for Endpoint                                                                                                            |
| **Messaging**               | Slack (API / Webhook), Microsoft Teams, Email                                                                                                                        |
| **Logging & Observability** | Server log, Index, Observability AI Assistant                                                                                                                        |
| **Webhook**                 | Webhook, Webhook - Case Management, xMatters                                                                                                                         |
| **Elastic**                 | Cases                                                                                                                                                                |

## Authentication

All connector API calls require API key auth or Basic auth. Every mutating request must include the `kbn-xsrf` header.

```http
kbn-xsrf: true
```

## API Reference

Base path: `<kibana_url>/api/actions` (or `/s/<space_id>/api/actions` for non-default spaces).

| Operation           | Method | Endpoint                               |
| ------------------- | ------ | -------------------------------------- |
| Create connector    | POST   | `/api/actions/connector/{id}`          |
| Update connector    | PUT    | `/api/actions/connector/{id}`          |
| Get connector       | GET    | `/api/actions/connector/{id}`          |
| Delete connector    | DELETE | `/api/actions/connector/{id}`          |
| Get all connectors  | GET    | `/api/actions/connectors`              |
| Get connector types | GET    | `/api/actions/connector_types`         |
| Run connector       | POST   | `/api/actions/connector/{id}/_execute` |

## Creating a Connector

### Example: Create a Slack Connector (Webhook)

```bash
curl -X POST "https://my-kibana:5601/api/actions/connector/my-slack-connector" \
  -H "kbn-xsrf: true" \
  -H "Content-Type: application/json" \
  -H "Authorization: ApiKey <your-api-key>" \
  -d '{
    "name": "Production Slack Alerts",
    "connector_type_id": ".slack",
    "config": {},
    "secrets": {
      "webhookUrl": "https://hooks.slack.com/services/T00/B00/XXXX"
    }
  }'
```

### Example: Create a PagerDuty Connector

```bash
curl -X POST "https://my-kibana:5601/api/actions/connector/my-pagerduty" \
  -H "kbn-xsrf: true" \
  -H "Content-Type: application/json" \
  -H "Authorization: ApiKey <your-api-key>" \
  -d '{
    "name": "PagerDuty Incidents",
    "connector_type_id": ".pagerduty",
    "config": {
      "apiUrl": "https://events.pagerduty.com/v2/enqueue"
    },
    "secrets": {
      "routingKey": "your-pagerduty-integration-key"
    }
  }'
```

## Listing Connectors

```bash
curl -X GET "https://my-kibana:5601/api/actions/connectors" \
  -H "Authorization: ApiKey <your-api-key>"
```

The response includes `referenced_by_count` showing how many rules use each connector. Always check this before deleting.

## Running a Connector (Test)

```bash
curl -X POST "https://my-kibana:5601/api/actions/connector/my-slack-connector/_execute" \
  -H "kbn-xsrf: true" \
  -H "Content-Type: application/json" \
  -H "Authorization: ApiKey <your-api-key>" \
  -d '{
    "params": {
      "message": "Test alert from API"
    }
  }'
```

## Terraform Provider

```hcl
resource "elasticstack_kibana_action_connector" "slack" {
  name              = "Production Slack Alerts"
  connector_type_id = ".slack"

  config = jsonencode({})

  secrets = jsonencode({
    webhookUrl = "https://hooks.slack.com/services/T00/B00/XXXX"
  })
}
```

## Common Connector Type IDs

| Type ID                        | Name                            | License    |
| ------------------------------ | ------------------------------- | ---------- |
| `.email`                       | Email                           | Gold       |
| `.slack`                       | Slack (Webhook)                 | Gold       |
| `.slack_api`                   | Slack (API)                     | Gold       |
| `.pagerduty`                   | PagerDuty                       | Gold       |
| `.jira`                        | Jira                            | Gold       |
| `.servicenow`                  | ServiceNow ITSM                 | Platinum   |
| `.webhook`                     | Webhook                         | Gold       |
| `.index`                       | Index                           | Basic      |
| `.server-log`                  | Server log                      | Basic      |
| `.opsgenie`                    | Opsgenie                        | Gold       |
| `.teams`                       | Microsoft Teams                 | Gold       |
| `.gen-ai`                      | OpenAI                          | Enterprise |
| `.bedrock`                     | Amazon Bedrock                  | Enterprise |
| `.gemini`                      | Google Gemini                   | Enterprise |
| `.cases`                       | Cases                           | Platinum   |

## Best Practices

1. **Use preconfigured connectors for production on-prem.** They eliminate secret sprawl.
2. **Test connectors before attaching to rules.** Use the `_execute` endpoint.
3. **Check `referenced_by_count` before deleting.**
4. **One connector per service, not per rule.** Create a single Slack connector and reference it from multiple rules.
5. **Use Spaces for multi-tenant isolation.**
6. **Always configure a recovery action alongside the active action.**
7. **Use deduplication keys for on-call connectors.** Set `dedupKey` to `{{rule.id}}-{{alert.id}}`.

## Common Pitfalls

1. **Missing `kbn-xsrf` header.** Returns 400.
2. **Wrong `connector_type_id`.** Must include leading dot (e.g., `.slack`).
3. **Empty `secrets` object required.** Even for connectors without secrets, pass `"secrets": {}`.
4. **Connector type is immutable.** Delete and recreate to change it.
5. **Secrets lost on export/import.** Must re-enter secrets manually after import.

## Guidelines

- Include `kbn-xsrf: true` on every POST, PUT, and DELETE.
- `connector_type_id` is immutable — delete and recreate to change connector type.
- Always pass `"secrets": {}` even for connectors with no secrets.
- Check `referenced_by_count` before deleting.
- Connectors are space-scoped; prefix paths with `/s/<space_id>/api/actions/` for non-default Kibana Spaces.
- Test every new connector with `_execute` before attaching to rules.
