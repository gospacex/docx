# 04 — Noop Tracing

`observability.WithNoop()` installs an in-memory `TracerProvider` that
records spans but never batches or sends them. Use this in:

- Unit tests that exercise code paths calling `StartSpan` / `GetTrace`
  without paying the cost of standing up a collector.
- Short-lived CLIs that want OTel API compatibility but no telemetry
  surface.
- Local development when no collector is available.

The key invariant: the application explicitly installs a noop global
TracerProvider via `InitTracing(ctx, cfg, WithNoop())`, and all traced
helpers then write to that drop-everything provider.
