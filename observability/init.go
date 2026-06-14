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

func InitTracing(ctx context.Context, cfg config.TracingConfig, opts ...InitOption) error {
	initMu.Lock()
	defer initMu.Unlock()

	o := &initOpts{}
	for _, fn := range opts {
		fn(o)
	}
	if o.noop {
		tp := sdktrace.NewTracerProvider()
		if prev := currentTP.Swap(tp); prev != nil {
			_ = prev.Shutdown(ctx)
		}
		otel.SetTracerProvider(tp)
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{}, propagation.Baggage{},
		))
		return nil
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("observability: %w", err)
	}

	sampler, err := buildSampler(cfg)
	if err != nil {
		return err
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
		sdktrace.WithSampler(sampler),
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

func buildSampler(cfg config.TracingConfig) (sdktrace.Sampler, error) {
	switch cfg.SamplerType {
	case "", "always_on":
		return sdktrace.AlwaysSample(), nil
	case "always_off":
		return sdktrace.NeverSample(), nil
	case "traceidratio":
		return sdktrace.TraceIDRatioBased(cfg.SamplerRatio), nil
	case "parentbased_traceidratio":
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SamplerRatio)), nil
	default:
		return nil, fmt.Errorf("observability: unsupported sampler_type %q", cfg.SamplerType)
	}
}
