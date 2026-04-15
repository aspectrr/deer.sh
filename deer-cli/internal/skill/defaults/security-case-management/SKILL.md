---
name: security-case-management
description: >
  Create, search, update, and manage SOC cases via the Kibana Cases API. Use when
  tracking incidents, linking alerts to cases, adding investigation notes, or managing
  triage output.
metadata:
  author: elastic
  version: 0.1.0
  source: elastic/agent-skills//skills/security/case-management
---

# Case Management

Manage SOC cases through the Kibana Cases API. All cases scoped to `securitySolution`.

## Prerequisites

```bash
export KIBANA_URL="https://your-cluster.kb.cloud.example.com:443"
export KIBANA_API_KEY="your-kibana-api-key"
```

## Quick start

```bash
# Create
node skills/security/case-management/scripts/case-manager.js create \
  --title "Malicious DLL sideloading on host1" \
  --description "Crypto clipper malware detected..." \
  --tags "classification:malicious" "confidence:88" "mitre:T1574.002" \
  --severity critical --yes

# Find by tags or search
node skills/security/case-management/scripts/case-manager.js find --tags "agent_id:<id>"
node skills/security/case-management/scripts/case-manager.js find --search "DLL sideloading" --status open

# List, get
node skills/security/case-management/scripts/case-manager.js list --status open --per-page 10
node skills/security/case-management/scripts/case-manager.js get --case-id <case_id>

# Attach alert
node skills/security/case-management/scripts/case-manager.js attach-alert \
  --case-id <case_id> --alert-id <alert_id> --alert-index <index> \
  --rule-id <rule_uuid> --rule-name "<name>"

# Attach multiple alerts (batch)
node skills/security/case-management/scripts/case-manager.js attach-alerts \
  --case-id <case_id> --alert-ids <id1> <id2> \
  --alert-index <index> --rule-id <uuid> --rule-name "<name>"

# Add comment, update
node skills/security/case-management/scripts/case-manager.js add-comment \
  --case-id <case_id> --comment "Process tree analysis shows..."
node skills/security/case-management/scripts/case-manager.js update \
  --case-id <case_id> --status closed --severity low --yes
```

## Tag conventions

| Tag pattern              | Example                             | Purpose                  |
| ------------------------ | ----------------------------------- | ------------------------ |
| `classification:<value>` | `classification:malicious`          | Triage classification    |
| `confidence:<score>`     | `confidence:85`                     | Confidence score 0-100   |
| `mitre:<technique>`      | `mitre:T1574.002`                   | MITRE ATT&CK technique   |
| `agent_id:<id>`          | `agent_id:550888e5-...`             | Agent ID for correlation |

## Case severity mapping

| Classification           | Kibana severity |
| ------------------------ | --------------- |
| benign (score 0-19)      | `low`           |
| unknown (score 20-60)    | `medium`        |
| malicious (score 61-80)  | `high`          |
| malicious (score 81-100) | `critical`      |

## Guidelines

- Report only tool output — do not invent IDs, hostnames, or details.
- Copy exact titles verbatim from API responses.
- Write operations prompt for confirmation. Pass `--yes` to skip.
- Verify environment variables before running commands.
