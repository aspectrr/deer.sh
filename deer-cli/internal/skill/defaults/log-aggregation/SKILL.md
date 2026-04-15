---
name: log-aggregation
version: "1.0.0"
description: "ELK Stack deployment, Logstash pipeline building, Filebeat configuration, and Kibana dashboard setup. Use when deploying ES clusters, building logstash pipelines, configuring beats, or debugging log aggregation issues."
---

# Log Aggregation (ELK Stack)

## When to Use

- Deploying or configuring Elasticsearch clusters
- Building Logstash pipelines (inputs, filters, outputs)
- Configuring Filebeat/Metricbeat/Heartbeat shippers
- Setting up Kibana dashboards and alerts
- Debugging log flow from source to Elasticsearch
- Diagnosing cluster health (red/yellow status)

## Elasticsearch

### Cluster Health

```bash
# Overall cluster status
curl -s localhost:9200/_cluster/health?pretty

# Node-level stats
curl -s localhost:9200/_nodes/stats?pretty

# Shard allocation explanation (when yellow/red)
curl -s localhost:9200/_cluster/allocation/explain?pretty

# Index-level health
curl -s localhost:9200/_cat/indices?v&health=yellow
curl -s localhost:9200/_cat/indices?v&health=red
```

### Index Management

```bash
# List indices with sizes
curl -s localhost:9200/_cat/indices?v&h=index,docs.count,store.size,pri,rep

# Create index with mapping
curl -X PUT localhost:9200/my-index -H 'Content-Type: application/json' -d '
{
  "mappings": {
    "properties": {
      "@timestamp": { "type": "date" },
      "message": { "type": "text" },
      "host": { "type": "keyword" },
      "level": { "type": "keyword" }
    }
  }
}'

# Delete index
curl -X DELETE localhost:9200/my-index

# Force merge (reduce segments)
curl -X POST "localhost:9200/my-index/_forcemerge?max_num_segments=1"
```

### Cluster Red/Yellow Diagnosis

1. Check health: `curl localhost:9200/_cluster/health?pretty`
2. Check unassigned shards: `curl localhost:9200/_cat/shards?v&h=index,shard,_pri,state,node&s=state`
3. Get allocation explanation: `curl -X POST localhost:9200/_cluster/allocation/explain?pretty`
4. Common causes: disk watermark exceeded, node down, replica count > data nodes
5. Check disk: `curl localhost:9200/_cat/allocation?v`
6. Check settings: `curl localhost:9200/_cluster/settings?include_defaults=true&flat_settings=true`

## Logstash

### Pipeline Configuration

```ruby
# /etc/logstash/pipeline/my-pipeline.conf

input {
  kafka {
    bootstrap_servers => "kafka:9092"
    topics => ["logs"]
    group_id => "logstash-consumer"
    consumer_threads => 4
  }
}

filter {
  grok {
    match => { "message" => "%{TIMESTAMP_ISO8601:timestamp} %{LOGLEVEL:level} %{GREEDYDATA:msg}" }
  }
  date {
    match => ["timestamp", "ISO8601"]
    target => "@timestamp"
  }
  mutate {
    remove_field => ["timestamp"]
  }
}

output {
  elasticsearch {
    hosts => ["elasticsearch:9200"]
    index => "logs-%{+YYYY.MM.dd}"
  }
}
```

### Debugging Pipelines

```bash
# Check pipeline config syntax
/usr/share/logstash/bin/logstash --config.test_and_exit -f /etc/logstash/pipeline/my-pipeline.conf

# Watch logstash logs
journalctl -u logstash --no-pager -n 100 -f

# Check running pipelines
curl -s localhost:9600/_node/pipelines?pretty

# Logstash node stats
curl -s localhost:9600/_node/stats?pretty

# Common startup issues:
# - Config syntax errors (check with --config.test_and_exit)
# - JVM heap too low (check ES_JAVA_OPTS)
# - Pipeline worker failures (check logs for "exception")
```

### Grok Debugging

1. Extract sample log line from source
2. Test pattern: `/usr/share/logstash/bin/logstash -e 'filter { grok { match => { "message" => "YOUR_PATTERN" } } }'`
3. Use `--config.debug` to see pattern matching details
4. Common patterns: `%{IP}`, `%{HOSTNAME}`, `%{GREEDYDATA}`, `%{SYSLOGLINE}`

## Filebeat

### Configuration

```yaml
# /etc/filebeat/filebeat.yml
filebeat.inputs:
  - type: log
    enabled: true
    paths:
      - /var/log/*.log
      - /var/log/syslog
    fields:
      env: production
      service: myapp
    fields_under_root: true

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
  index: "filebeat-%{[agent.version]}-%{+yyyy.MM.dd}"

setup.ilm.enabled: false
setup.template.name: "filebeat"
setup.template.pattern: "filebeat-*"
```

### Debugging

```bash
# Test config
filebeat test config -c /etc/filebeat/filebeat.yml

# Test output connectivity
filebeat test output -c /etc/filebeat/filebeat.yml

# Check status
systemctl status filebeat

# View logs
journalctl -u filebeat --no-pager -n 100

# Run in foreground with debug
filebeat -e -d "*"
```

## Kibana

### Setup

```bash
# Check Kibana status
curl -s localhost:5601/api/status

# Create index pattern via API
curl -X POST localhost:5601/api/saved_objects/index-pattern -H 'Content-Type: application/json' -H 'kbn-xsrf: true' -d '
{
  "attributes": {
    "title": "logs-*",
    "timeFieldName": "@timestamp"
  }
}'
```

## Deer Sandbox Integration

When testing in a deer sandbox:

- Use `kafka_stub=true` to get Redpanda at `localhost:9092`
- Use `es_stub=true` to get Elasticsearch at `localhost:9200`
- Use both for full pipeline testing
- After configuring logstash, use `verify_pipeline_output` to confirm data flows through
- Point logstash output to `localhost:9200` in sandbox
- Point logstash kafka input to `localhost:9092` in sandbox

### Pipeline Verification Flow

1. Create sandbox with `kafka_stub=true, es_stub=true`
2. Edit logstash config to use localhost addresses
3. Start logstash in sandbox
4. Produce test data: `rpk topic produce <topic> --brokers localhost:9092`
5. Call `verify_pipeline_output` with the target index
6. Verify documents appeared in ES
