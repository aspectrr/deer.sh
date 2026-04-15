---
name: elasticsearch-authn
description: >
  Authenticate to Elasticsearch using native, file-based, LDAP/AD, SAML, OIDC, Kerberos,
  JWT, or certificate realms. Use when connecting with credentials, choosing a realm,
  or managing API keys.
metadata:
  author: elastic
  version: 0.1.0
  source: elastic/agent-skills//skills/elasticsearch/elasticsearch-authn
---

# Elasticsearch Authentication

Authenticate to an Elasticsearch cluster using any supported authentication realm that is already configured. Covers all
built-in realms, credential verification, and the full API key lifecycle.

For roles, users, role assignment, and role mappings, see the **elasticsearch-authz** skill.

## Critical principles

- **Never ask for credentials in chat.** Secrets must not appear in conversation history.
- **Always use environment variables.** Instruct the user to set them in a `.env` file in the project root.

## Authentication Realms

Elasticsearch evaluates realms in a configured order (the **realm chain**). The first realm that can authenticate the
request wins.

### Internal realms

#### Native (username and password)

```bash
curl -u "${ELASTICSEARCH_USERNAME}:${ELASTICSEARCH_PASSWORD}" "${ELASTICSEARCH_URL}/_security/_authenticate"
```

#### File

Users defined in flat files on each cluster node. Always active regardless of license state. Self-managed only.

```bash
curl -u "${FILE_USER}:${FILE_PASSWORD}" "${ELASTICSEARCH_URL}/_security/_authenticate"
```

### External realms

#### LDAP / Active Directory

Self-managed only. Combined with role mappings to translate LDAP/AD groups to Elasticsearch roles.

```bash
curl -u "${LDAP_USER}:${LDAP_PASSWORD}" "${ELASTICSEARCH_URL}/_security/_authenticate"
```

#### PKI (TLS client certificates)

Requires PKI realm and TLS on HTTP layer.

```bash
curl --cert "${CLIENT_CERT}" --key "${CLIENT_KEY}" --cacert "${CA_CERT}" \
  "${ELASTICSEARCH_URL}/_security/_authenticate"
```

#### SAML / OIDC

Primarily for Kibana SSO. Browser-based redirect flow, not for REST clients. Configure another realm alongside for
programmatic API access.

#### JWT

Accepts JWTs as bearer tokens.

```bash
curl -H "Authorization: Bearer ${JWT_TOKEN}" "${ELASTICSEARCH_URL}/_security/_authenticate"
```

#### Kerberos

Self-managed only. Requires KDC infrastructure, DNS, and time synchronization.

```bash
kinit "${KERBEROS_PRINCIPAL}"
curl --negotiate -u : "${ELASTICSEARCH_URL}/_security/_authenticate"
```

### API keys

Preferred for programmatic and automated access.

```bash
curl -H "Authorization: ApiKey ${ELASTICSEARCH_API_KEY}" "${ELASTICSEARCH_URL}/_security/_authenticate"
```

## Manage API Keys

### Create an API key

```bash
curl -X POST "${ELASTICSEARCH_URL}/_security/api_key" \
  <auth_flags> \
  -H "Content-Type: application/json" \
  -d '{
    "name": "'"${KEY_NAME}"'",
    "expiration": "30d",
    "role_descriptors": {
      "'"${ROLE_NAME}"'": {
        "cluster": [],
        "indices": [
          {
            "names": ["'"${INDEX_PATTERN}"'"],
            "privileges": ["read"]
          }
        ]
      }
    }
  }'
```

The response contains `id`, `api_key`, and `encoded`. Store `encoded` securely — it cannot be retrieved again.

> **Limitation:** An API key **cannot** create another API key with privileges. Use `POST /_security/api_key/grant` with
> user credentials instead.

### Get and invalidate API keys

```bash
curl "${ELASTICSEARCH_URL}/_security/api_key?name=${KEY_NAME}" <auth_flags>
curl -X DELETE "${ELASTICSEARCH_URL}/_security/api_key" \
  <auth_flags> \
  -H "Content-Type: application/json" \
  -d '{"name": "'"${KEY_NAME}"'"}'
```

## Deployment Compatibility

| Realm            | Self-managed | ECH                     | Serverless         |
| ---------------- | ------------ | ----------------------- | ------------------ |
| Native           | Yes          | Yes                     | Not available      |
| File             | Yes          | Not available           | Not available      |
| LDAP / AD        | Yes          | Not available           | Not available      |
| PKI              | Yes          | Limited                 | Not available      |
| SAML             | Yes          | Yes (deployment config) | Organization-level |
| OIDC             | Yes          | Yes (deployment config) | Not available      |
| JWT              | Yes          | Yes (deployment config) | Not available      |
| Kerberos         | Yes          | Not available           | Not available      |
| API keys         | Yes          | Yes                     | Yes                |

## Guidelines

- Prefer API keys for automated workflows — they support fine-grained scoping and independent expiration.
- Never use the `elastic` superuser for day-to-day operations.
- Always set `expiration` on API keys. Avoid indefinite keys in production.
- Never receive, echo, or log passwords, API keys, or any credentials in chat.
