package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// ShutdownTracing flushes and shuts down the currently installed tracer
// provider created via InitTracing. It is safe to call multiple times.
func ShutdownTracing(ctx context.Context) error {
	initMu.Lock()
	defer initMu.Unlock()

	tp := currentTP.Swap(nil)
	otel.SetTracerProvider(sdktrace.NewTracerProvider())
	if tp == nil {
		return nil
	}
	return tp.Shutdown(ctx)
}
