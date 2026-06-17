# PlatformBase E2E Example

Demonstrates composing multiple x-sdk providers (cachex, mqx, dbx, otelx)
through the hubx registry using a single `config.yaml` + Viper loader.

## Providers

| Provider | Source | Purpose |
|----------|--------|---------|
| cachex.redis | cachex/hubx/redisx | Redis cache |
| mqx.kafka.producer | mqx/hubx/kafkax/producer | Kafka producer |
| dbx.mysql | dbx/hubx/mysqlx | MySQL connection |
| otel.tracer | otelx/hubx/tracer | OpenTelemetry tracer |
| otel.meter | otelx/hubx/meter | OpenTelemetry meter |

## Run

```bash
make up      # start redis/kafka/mysql/jaeger
make test    # run E2E tests
make down    # tear down
```

## Configuration

Edit `config.yaml` to point at different endpoints.
