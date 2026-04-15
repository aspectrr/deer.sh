---
name: security-alert-triage
description: >
  Triage Elastic Security alerts — gather context, classify threats, create cases,
  and acknowledge. Use when triaging alerts, performing SOC analysis, or investigating
  detections.
compatibility: >
  Requires Node.js 22+, network access to Elasticsearch. Environment variables: ELASTICSEARCH_URL
  or ELASTICSEARCH_CLOUD_ID, plus ELASTICSEARCH_API_KEY or ELASTICSEARCH_USERNAME/ELASTICSEARCH_PASSWORD.
metadata:
  author: elastic
  version: 0.1.0
  source: elastic/agent-skills//skills/security/alert-triage
---

# Alert Triage

Analyze Elastic Security alerts one at a time: gather context, classify, create a case, and acknowledge. This skill
depends on the `case-management` skill for case creation.

## Prerequisites

Set the required environment variables:

```bash
export ELASTICSEARCH_URL="https://your-cluster.es.cloud.example.com:443"
export ELASTICSEARCH_API_KEY="your-api-key"
export KIBANA_URL="https://your-cluster.kb.cloud.example.com:443"
export KIBANA_API_KEY="your-kibana-api-key"
```

## Quick start

All commands from workspace root. Always fetch → investigate → document → acknowledge.

```bash
node skills/security/alert-triage/scripts/fetch-next-alert.js
node skills/security/alert-triage/scripts/run-query.js --query-file query.esql --type esql
node skills/security/case-management/scripts/case-manager.js create --title "..." --description "..." --tags "classification:..." "agent_id:<id>" --severity <level> --yes
node skills/security/alert-triage/scripts/acknowledge-alert.js --related --agent <id> --timestamp <ts> --window 60 --yes
```

## Critical principles

- **Do NOT classify prematurely.** Gather ALL context before deciding benign/unknown/malicious.
- **Most alerts are false positives**, even if they look alarming.
- **"Unknown" is acceptable** and often correct when evidence is insufficient.
- **MALICIOUS requires strong corroborating evidence**: persistence + C2, credential theft, lateral movement.

## Workflow

```text
- [ ] Step 0: Group alerts by agent/host and time window
- [ ] Step 1: Check existing cases
- [ ] Step 2: Gather full context (DO NOT SKIP)
- [ ] Step 3: Create or update case (only AFTER context gathered)
- [ ] Step 4: Acknowledge alert and all related alerts
- [ ] Step 5: Fetch next alert group and repeat
```

### Step 0: Group alerts before triaging

Query open alerts, group by `agent.id`, sub-group by time window (~5 min):

```esql
FROM .alerts-security.alerts-*
| WHERE kibana.alert.workflow_status == "open" AND @timestamp >= "<start>"
| STATS alert_count=COUNT(*), rules=VALUES(kibana.alert.rule.name) BY agent.id
| SORT alert_count DESC
```

### Step 1: Check existing cases

```bash
node skills/security/case-management/scripts/case-manager.js find --tags "agent_id:<agent_id>"
```

### Step 2: Gather context

**Time range warning:** Extract the alert's `@timestamp` and build queries around that time with +/- 1 hour window.

Example — process tree (use ES|QL with `KEEP`; avoid `--full`):

```esql
FROM logs-endpoint.events.process-*
| WHERE agent.id == "<agent_id>" AND @timestamp >= "<alert_time - 5min>" AND @timestamp <= "<alert_time + 10min>"
  AND process.parent.name IS NOT NULL
  AND process.name NOT IN ("svchost.exe", "conhost.exe", "agentbeat.exe")
| KEEP @timestamp, process.name, process.command_line, process.pid, process.parent.name, process.parent.pid
| SORT @timestamp | LIMIT 80
```

| Data type | Index pattern                    |
| --------- | -------------------------------- |
| Alerts    | `.alerts-security.alerts-*`      |
| Processes | `logs-endpoint.events.process-*` |
| Network   | `logs-endpoint.events.network-*` |
| Logs      | `logs-*`                         |

### Step 3: Create or update case

```bash
node skills/security/case-management/scripts/case-manager.js create \
  --title "<concise summary>" \
  --description "<findings, IOCs, attack chain, MITRE techniques>" \
  --tags "classification:<benign|unknown|malicious>" "confidence:<0-100>" "mitre:<technique>" "agent_id:<id>" \
  --severity <low|medium|high|critical> --yes

node skills/security/case-management/scripts/case-manager.js attach-alert \
  --case-id <case_id> --alert-id <alert_id> --alert-index <index> \
  --rule-id <rule_uuid> --rule-name "<rule name>"
```

### Step 4: Acknowledge alerts

```bash
node skills/security/alert-triage/scripts/acknowledge-alert.js --related --agent <id> --timestamp <ts> --window 60 --yes
```

## Tool reference

### fetch-next-alert.js

Fetches the oldest unacknowledged Elastic Security alert.

```bash
node skills/security/alert-triage/scripts/fetch-next-alert.js [--days <n>] [--json] [--full] [--verbose]
```

### run-query.js

Runs KQL or ES|QL queries against Elasticsearch. For ES|QL on PowerShell, always use `--query-file`:

```bash
node skills/security/alert-triage/scripts/run-query.js --query-file query.esql --type esql
```

### acknowledge-alert.js

Acknowledges alerts by updating `workflow_status` to `acknowledged`.

| Mode    | Command                                                                                                                                |
| ------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| Single  | `node skills/security/alert-triage/scripts/acknowledge-alert.js <alert_id> --index <index> --yes`                                      |
| Related | `node skills/security/alert-triage/scripts/acknowledge-alert.js --related --agent <id> --timestamp <ts> [--window 60] --yes`           |
| By host | `node skills/security/alert-triage/scripts/acknowledge-alert.js --query --host <hostname> [--time-start <ts>] [--time-end <ts>] --yes` |
| Dry run | Add `--dry-run` to any mode (no confirmation needed)                                                                                   |

## Guidelines

- Report only tool output — do not invent IDs, hostnames, IPs, or details not present in the tool response.
- Preserve identifiers from the request — use exact values the user provides in tool calls and responses.
- Distinguish facts from inference — label conclusions beyond tool output as your assessment.
- All write operations prompt for confirmation. Pass `--yes` to skip when called by an agent.
- Use `--dry-run` before bulk acknowledgments to preview scope without modifying data.
