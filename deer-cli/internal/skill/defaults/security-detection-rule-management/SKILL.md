---
name: security-detection-rule-management
description: >
  Create, tune, and manage Elastic Security detection rules (SIEM and Endpoint). Use
  for false positives, exceptions, new coverage, noisy rules, or rule management via
  Kibana API.
metadata:
  author: elastic
  version: 0.1.0
  source: elastic/agent-skills//skills/security/detection-rule-management
---

# Detection Rule Management

Create new detection rules for emerging threats and tune existing rules to reduce false positives.

## Prerequisites

```bash
export ELASTICSEARCH_URL="https://your-cluster.es.cloud.example.com:443"
export ELASTICSEARCH_API_KEY="your-api-key"
export KIBANA_URL="https://your-cluster.kb.cloud.example.com:443"
export KIBANA_API_KEY="your-kibana-api-key"
```

## Workflow: Tune noisy rules

### Step 1: Find noisy rules

```bash
node skills/security/detection-rule-management/scripts/rule-manager.js noisy-rules --days 7 --top 20
node skills/security/detection-rule-management/scripts/rule-manager.js find --filter "alert.attributes.name:*Suspicious*" --brief
node skills/security/detection-rule-management/scripts/rule-manager.js get --id <rule_uuid>
```

### Step 2: Investigate false positives

```bash
node skills/security/alert-triage/scripts/run-query.js "kibana.alert.rule.name:\"<rule_name>\"" --index ".alerts-security.alerts-*" --days 7 --full
```

### Step 3: Tune

**Add exception (preferred):**
```bash
node skills/security/detection-rule-management/scripts/rule-manager.js add-exception \
  --rule-uuid <rule_uuid> \
  --entries "process.executable:is:C:\\Program Files\\SCCM\\CcmExec.exe" \
  --name "Exclude SCCM" --comment "FP: SCCM deployment" --yes
```

**Patch query/threshold/severity:**
```bash
node skills/security/detection-rule-management/scripts/rule-manager.js patch --id <rule_uuid> --query "..." --yes
node skills/security/detection-rule-management/scripts/rule-manager.js patch --id <rule_uuid> --max-signals 50 --yes
node skills/security/detection-rule-management/scripts/rule-manager.js disable --id <rule_uuid> --yes
```

## Workflow: Create new rule

### Step 1: Define threat and data sources

Specify MITRE ATT&CK technique(s), required data sources, and malicious vs legitimate behavior.

### Step 2: Test the query

```bash
node skills/security/alert-triage/scripts/run-query.js "process.name:certutil.exe" --index "logs-endpoint.events.process-*" --days 30
```

### Step 3: Validate query

```bash
node skills/security/detection-rule-management/scripts/rule-manager.js validate-query \
  --query "process.name:taskkill.exe AND process.command_line:(*chrome.exe*)" --language kuery
```

### Step 4: Create rule

```bash
node skills/security/detection-rule-management/scripts/rule-manager.js create \
  --name "Certutil URL Download" \
  --description "Detects certutil.exe used to download files or decode Base64." \
  --type query \
  --query "process.name:certutil.exe AND process.command_line:(*urlcache* OR *decode*)" \
  --index "logs-endpoint.events.process-*" \
  --severity medium --risk-score 47 \
  --interval 5m --disabled
```

## Tool Reference

| Command              | Description                                   |
| -------------------- | --------------------------------------------- |
| `find`               | Search/list rules                             |
| `get`                | Get rule by ID                                |
| `create`             | Create rule (inline or `--from-file`)         |
| `patch`              | Patch specific fields                         |
| `enable/disable`     | Enable or disable rule                        |
| `delete`             | Delete rule                                   |
| `add-exception`      | Add exception to rule                         |
| `noisy-rules`        | Find noisiest rules by alert volume           |
| `validate-query`     | Check query syntax before create/patch        |

## Guidelines

- Report only tool output — do not invent IDs or details.
- Preserve identifiers from requests exactly.
- Start executing tools immediately — do not browse first.
- All write operations prompt for confirmation. Pass `--yes` to skip.
