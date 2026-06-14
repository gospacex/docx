# 04 — Noop Tracing

`observability.WithNoop()` installs an in-memory `TracerProvider` that
records spans but never batches or sends them. Use this in:

- Unit tests that exercise code paths calling `StartSpan` / `GetTrace`
  without paying the cost of standing up a collector.
- Short-lived CLIs that want OTel API compatibility but no telemetry
  surface.
- Local development when no collector is available.

The key invariant: `tracing.Enabled` stays `true` (so the public
methods are wired), but the global TracerProvider installed by
`InitTracing(ctx, cfg, WithNoop())` is a drop-everything noop.
