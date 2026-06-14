package observability

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/gospacex/hubx/cache/docx/config"
	"github.com/gospacex/hubx/cache/docx/observability/tracing"
)

type InitOption func(*initOpts)

type initOpts struct{ noop bool }

func WithNoop() InitOption { return func(o *initOpts) { o.noop = true } }

var (
	initMu    sync.Mutex
	currentTP atomic.Pointer[sdktrace.TracerProvider]
)

type noopProvider struct{}

func (noopProvider) Tracer(_ string, _ ...any) any { return nil }

func InitTracing(ctx context.Context, cfg config.TracingConfig, opts ...InitOption) error {
	initMu.Lock()
	defer initMu.Unlock()

	o := &initOpts{}
	for _, fn := range opts {
		fn(o)
	}
	if o.noop {
		otel.SetTracerProvider(sdktrace.NewTracerProvider())
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{}, propagation.Baggage{},
		))
		return nil
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("observability: %w", err)
	}

	exp, err := tracing.NewExporter(cfg)
	if err != nil {
		return fmt.Errorf("observability: init tracing: %w", err)
	}

	res, _ := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(cfg.ServiceName)),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)

	if prev := currentTP.Swap(tp); prev != nil {
		_ = prev.Shutdown(ctx)
	}
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))
	return nil
}
