---
name: elasticsearch-file-ingest
description: >
  Ingest and transform data files (CSV/JSON/Parquet/Arrow IPC) into Elasticsearch
  with stream processing and custom transforms. Use when loading files or batch importing
  data.
metadata:
  author: elastic
  version: 0.2.0
  source: elastic/agent-skills//skills/elasticsearch/elasticsearch-file-ingest
---

# Elasticsearch File Ingest

Stream-based ingestion and transformation of large data files (NDJSON, CSV, Parquet, Arrow IPC) into Elasticsearch.

## Setup

Install dependencies and configure environment:

```bash
npm install
```

```bash
export ELASTICSEARCH_URL="https://elasticsearch:9200"
export ELASTICSEARCH_API_KEY="<your-api-key>"
```

Test connection:

```bash
node scripts/ingest.js test
```

## Usage

### Ingest a JSON file

```bash
node scripts/ingest.js ingest --file /path/to/data.json --target my-index
```

### Ingest CSV

```bash
node scripts/ingest.js ingest --file /path/to/users.csv --source-format csv --target users
```

### Ingest Parquet

```bash
node scripts/ingest.js ingest --file /path/to/users.parquet --source-format parquet --target users
```

### Ingest with transformation

```bash
node scripts/ingest.js ingest --file /path/to/data.json --target my-index --transform transform.js
```

### Infer mappings from CSV

```bash
node scripts/ingest.js ingest --file /path/to/users.csv --infer-mappings --target users
```

## Command Reference

### Required Options

```bash
--target <index>         # Target index name
```

### Source Options (choose one)

```bash
--file <path>            # Source file (supports wildcards)
--stdin                  # Read NDJSON/CSV from stdin
```

### Index Configuration

```bash
--mappings <file.json>          # Mappings file
--infer-mappings                # Infer mappings/pipeline from file
--delete-index                  # Delete target index if exists
--pipeline <name>               # Ingest pipeline name
```

### Processing

```bash
--transform <file.js>    # Transform function
--source-format <fmt>    # ndjson|csv|parquet|arrow (default: ndjson)
--csv-options <file>     # CSV parser options (JSON file)
```

## Transform Functions

```javascript
export default function transform(doc) {
  return {
    ...doc,
    full_name: `${doc.first_name} ${doc.last_name}`,
    timestamp: new Date().toISOString(),
  };
}
```

Return `null` to skip a document. Return an array to split into multiple documents.

## Guidelines

- **Test first**: Always run `node scripts/ingest.js test` before ingesting.
- **Never combine `--infer-mappings` with `--source-format`**.
- **Use `--source-format csv` with `--mappings`** for known field types.
- **Never** echo, print, or log credential environment variables.
