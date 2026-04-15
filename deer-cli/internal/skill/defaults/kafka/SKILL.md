---
name: kafka
version: "1.0.0"
description: "Kafka topic management, consumer group monitoring, message production/consumption, and cluster health diagnostics. Use when working with Kafka brokers, topics, consumer lag, or debugging pipeline issues."
---

# Kafka Operations

## When to Use

- Managing Kafka topics (create, describe, delete, list)
- Monitoring consumer group lag
- Producing or consuming messages for testing
- Diagnosing broker connectivity issues
- Checking partition offsets and health
- Debugging logstash pipelines that read from or write to Kafka

## Common Commands

### Cluster Status

```bash
# Check broker health
kafka-broker-api-versions --bootstrap-server localhost:9092

# List topics with partition counts
kafka-topics --bootstrap-server localhost:9092 --list

# Describe a topic (partitions, replicas, ISR)
kafka-topics --bootstrap-server localhost:9092 --describe --topic <topic>
```

### Topic Management

```bash
# Create topic
kafka-topics --bootstrap-server localhost:9092 --create --topic <name> --partitions 3 --replication-factor 1

# Delete topic
kafka-topics --bootstrap-server localhost:9092 --delete --topic <name>

# Increase partitions
kafka-topics --bootstrap-server localhost:9092 --alter --topic <name> --partitions <count>
```

### Consumer Groups

```bash
# List all consumer groups
kafka-consumer-groups --bootstrap-server localhost:9092 --list

# Show consumer group lag (key debugging command)
kafka-consumer-groups --bootstrap-server localhost:9092 --describe --group <group>

# Reset consumer group offsets (use with caution)
kafka-consumer-groups --bootstrap-server localhost:9092 --group <group> --topic <topic> --reset-offsets --to-earliest --execute
```

### Produce and Consume

```bash
# Produce a message with key
echo "key:value" | kafka-console-producer --bootstrap-server localhost:9092 --topic <topic> --property "parse.key=true" --property "key.separator=:"

# Consume messages from beginning
kafka-console-consumer --bootstrap-server localhost:9092 --topic <topic> --from-beginning --max-messages 10

# Consume with key and value
kafka-console-consumer --bootstrap-server localhost:9092 --topic <topic> --from-beginning --property print.key=true --property key.separator="|"
```

### Offset Inspection

```bash
# Show earliest and latest offsets
kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic <topic>
```

## Deer Sandbox Integration

When debugging Kafka-dependent services in a deer sandbox with `kafka_stub=true`:

- Redpanda runs at `localhost:9092` inside the sandbox
- Use `rpk topic list --brokers localhost:9092` for topic management
- Use `rpk topic produce <topic> --brokers localhost:9092` to send test messages
- Use `rpk topic consume <topic> --brokers localhost:9092` to read messages
- Use `rpk group list --brokers localhost:9092` for consumer groups

## Diagnostic Patterns

### High Consumer Lag

1. Check lag: `kafka-consumer-groups --describe --group <group>`
2. Check consumer is running: `systemctl status <consumer-service>`
3. Check topic end offsets: `GetOffsetShell --topic <topic>`
4. Common causes: slow consumer, network latency, too few partitions, consumer crashes

### Broker Connectivity Issues

1. Verify broker is listening: `ss -tlnp | grep 9092`
2. Check advertised.listeners in server.properties
3. Test connectivity: `kafka-broker-api-versions --bootstrap-server <host>:9092`
4. Check logs: `journalctl -u kafka --no-pager -n 100`

### Logstash Kafka Input Issues

1. Verify topic exists and has data
2. Check consumer group lag
3. Verify bootstrap_servers config in logstash pipeline
4. Check logstash logs for connection errors: `journalctl -u logstash --no-pager -n 100`
5. In sandbox: replace bootstrap address with `localhost:9092`
