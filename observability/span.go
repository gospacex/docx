package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type SpanOption = trace.SpanStartOption

func StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, trace.Span) {
	return otel.Tracer("docx").Start(ctx, name, opts...)
}
