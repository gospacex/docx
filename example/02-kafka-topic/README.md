# 02 — Kafka Topic SpanExporter

Demonstrates `config.TracingConfig{Exporter: "kafka_topic"}` and shows the
self-built Kafka producer wiring up under the hood:

- `kafka.NewProducer` is built directly with the OTLP-friendly
  `bootstrap.servers / acks=all / enable.idempotence=true` triple.
- `librdkafka` is configured via a `kafka.ConfigMap` keyed on the same
  vocabulary as mqx (see `config.TracingProducerConfig`).

## Bring up a single-broker Kafka

```bash
docker run -d --name kafka -p 9092:9092 \
  -e KAFKA_NODE_ID=1 \
  -e KAFKA_PROCESS_ROLES=broker,controller \
  -e KAFKA_LISTENERS=PLAINTEXT://0.0.0.0:9092,CONTROLLER://0.0.0.0:9093 \
  -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092 \
  -e KAFKA_CONTROLLER_LISTENER_NAMES=CONTROLLER \
  -e KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT \
  -e KAFKA_CONTROLLER_QUORUM_VOTERS=1@localhost:9093 \
  apache/kafka:3.7.0
```

## Run

```bash
cd example/02-kafka-topic
go mod tidy
go run .
```

## Watch the topic

```bash
docker exec -it kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic otel-traces --from-beginning
```

Each line is a JSON record of one span. Multiple spans from the same
trace share a `trace_id`, so grouping by `trace_id` reconstructs the
chain end-to-end.
