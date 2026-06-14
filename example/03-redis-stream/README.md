# 03 — Redis Stream SpanExporter

Demonstrates `config.TracingConfig{Exporter: "redis_stream"}` — every
spans is `XAdd`ed as a stream entry with the JSON payload under the
`"span"` field.

## Bring up Redis

```bash
docker run -d --name redis -p 6379:6379 redis:7-alpine
```

## Run

```bash
cd example/03-redis-stream
go mod tidy
go run .
```

## Tail the stream

```bash
redis-cli XREAD COUNT 5 BLOCK 0 STREAMS otel-traces '$'
```

The `"span"` field is JSON with `trace_id`, `span_id`, `name`,
`start_time`, and a flattened `attributes` map. Order in the stream is
insertion order, so reading from `$` after the run shows the full
sequence of spans emitted in that session.
