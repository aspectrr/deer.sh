---
name: elasticsearch-authz
description: >
  Manage Elasticsearch RBAC: native users, roles, role mappings, document- and field-level
  security. Use when creating users or roles, assigning privileges, or mapping external
  realms like LDAP/SAML.
metadata:
  author: elastic
  version: 0.1.1
  source: elastic/agent-skills//skills/elasticsearch/elasticsearch-authz
---

# Elasticsearch Authorization

Manage Elasticsearch role-based access control: native users, roles, role assignment, and role mappings for external
realms.

For authentication methods and API key management, see the **elasticsearch-authn** skill.

For detailed API endpoints, see [references/api-reference.md](https://github.com/elastic/agent-skills/blob/main/skills/elasticsearch/elasticsearch-authz/references/api-reference.md).

## Jobs to Be Done

- Create a native user with a specific set of privileges
- Define a custom role with least-privilege index and cluster access
- Assign one or more roles to an existing user
- Create a role with Kibana feature or space privileges
- Configure a role mapping for external realm users (SAML, LDAP, PKI)
- Derive role assignments dynamically from user attributes (Mustache templates)
- Restrict document visibility per user or department (document-level security)
- Hide sensitive fields like PII from certain roles (field-level security)
- Implement attribute-based access control (ABAC) using templated role queries
- Translate a natural-language access request into user, role, and role mapping tasks

## Prerequisites

| Item                   | Description                                                                |
| ---------------------- | -------------------------------------------------------------------------- |
| **Elasticsearch URL**  | Cluster endpoint (e.g. `https://localhost:9200` or a Cloud deployment URL) |
| **Kibana URL**         | Required only when setting Kibana feature/space privileges                 |
| **Authentication**     | Valid credentials (see the elasticsearch-authn skill)                      |
| **Cluster privileges** | `manage_security` is required for user and role management operations      |

## Manage Native Users

### Create a user

```bash
curl -X POST "${ELASTICSEARCH_URL}/_security/user/${USERNAME}" \
  <auth_flags> \
  -H "Content-Type: application/json" \
  -d '{
    "password": "'"${PASSWORD}"'",
    "roles": ["'"${ROLE_NAME}"'"],
    "full_name": "'"${FULL_NAME}"'",
    "email": "'"${EMAIL}"'",
    "enabled": true
  }'
```

### Update a user

Use `PUT /_security/user/${USERNAME}` with the fields to change. Omit `password` to keep the existing one.

### Other user operations

```bash
curl -X POST "${ELASTICSEARCH_URL}/_security/user/${USERNAME}/_password" \
  <auth_flags> -H "Content-Type: application/json" \
  -d '{"password": "'"${NEW_PASSWORD}"'"}'
curl -X PUT "${ELASTICSEARCH_URL}/_security/user/${USERNAME}/_disable" <auth_flags>
curl -X PUT "${ELASTICSEARCH_URL}/_security/user/${USERNAME}/_enable" <auth_flags>
curl "${ELASTICSEARCH_URL}/_security/user/${USERNAME}" <auth_flags>
curl -X DELETE "${ELASTICSEARCH_URL}/_security/user/${USERNAME}" <auth_flags>
```

## Manage Roles

### Choosing the right API

Use the **Elasticsearch API** (`PUT /_security/role/{name}`) when the role only needs `cluster` and `indices`
privileges. Use the **Kibana role API** (`PUT /api/security/role/{name}`) when the role includes any Kibana feature or space
privileges.

### Create or update a role (Elasticsearch API)

```bash
curl -X PUT "${ELASTICSEARCH_URL}/_security/role/${ROLE_NAME}" \
  <auth_flags> \
  -H "Content-Type: application/json" \
  -d '{
    "description": "'"${ROLE_DISPLAY_NAME}"'",
    "cluster": [],
    "indices": [
      {
        "names": ["'"${INDEX_PATTERN}"'"],
        "privileges": ["read", "view_index_metadata"]
      }
    ]
  }'
```

### Create or update a role (Kibana API)

```bash
curl -X PUT "${KIBANA_URL}/api/security/role/${ROLE_NAME}" \
  <auth_flags> \
  -H "kbn-xsrf: true" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "'"${ROLE_DISPLAY_NAME}"'",
    "elasticsearch": {
      "cluster": [],
      "indices": [
        {
          "names": ["'"${INDEX_PATTERN}"'"],
          "privileges": ["read", "view_index_metadata"]
        }
      ]
    },
    "kibana": [
      {
        "base": [],
        "feature": {
          "discover": ["read"],
          "dashboard": ["read"]
        },
        "spaces": ["*"]
      }
    ]
  }'
```

## Document-Level and Field-Level Security

### Field-level security (FLS)

Restrict which fields a role can see:

```bash
curl -X PUT "${ELASTICSEARCH_URL}/_security/role/pii-redacted-reader" \
  <auth_flags> \
  -H "Content-Type: application/json" \
  -d '{
    "description": "PII Redacted Reader",
    "indices": [
      {
        "names": ["customers-*"],
        "privileges": ["read"],
        "field_security": {
          "grant": ["*"],
          "except": ["ssn", "credit_card", "date_of_birth"]
        }
      }
    ]
  }'
```

### Document-level security (DLS)

Restrict which documents a role can see by attaching a query filter:

```bash
curl -X PUT "${ELASTICSEARCH_URL}/_security/role/emea-logs-reader" \
  <auth_flags> \
  -H "Content-Type: application/json" \
  -d '{
    "description": "EMEA Logs Reader",
    "indices": [
      {
        "names": ["logs-*"],
        "privileges": ["read"],
        "query": "{\"term\": {\"region\": \"emea\"}}"
      }
    ]
  }'
```

## Assign Roles to Users

```bash
curl -X PUT "${ELASTICSEARCH_URL}/_security/user/${USERNAME}" \
  <auth_flags> \
  -H "Content-Type: application/json" \
  -d '{
    "roles": ["role-a", "role-b"]
  }'
```

The `roles` array is **replaced entirely** — include all roles the user should have. Fetch the user first to see current
roles before updating.

## Manage Role Mappings

### Static role mapping

```bash
curl -X PUT "${ELASTICSEARCH_URL}/_security/role_mapping/saml-default-access" \
  <auth_flags> \
  -H "Content-Type: application/json" \
  -d '{
    "roles": ["viewer"],
    "enabled": true,
    "rules": {
      "field": { "realm.name": "saml1" }
    }
  }'
```

### LDAP group-based mapping

```bash
curl -X PUT "${ELASTICSEARCH_URL}/_security/role_mapping/ldap-admins" \
  <auth_flags> \
  -H "Content-Type: application/json" \
  -d '{
    "roles": ["superuser"],
    "enabled": true,
    "rules": {
      "all": [
        { "field": { "realm.name": "ldap1" } },
        { "field": { "groups": "cn=admins,ou=groups,dc=example,dc=com" } }
      ]
    }
  }'
```

## Guidelines

### Least-privilege principles

- Never use the `elastic` superuser for day-to-day operations. Create dedicated minimum-privilege roles.
- Use `read` and `view_index_metadata` for read-only data access. Leave `cluster` empty unless explicitly required.
- Use DLS (`query`) and FLS (`field_security`) to restrict access within an index.

### Named privileges only

Never use internal action names (e.g. `indices:data/read/search`). Always use officially documented named privileges.

### Role naming conventions

- Use short lowercase names with hyphens: `logs-reader`, `apm-data-viewer`, `metrics-writer`.
- Set `description` to a short, human-readable display name.
