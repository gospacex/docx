package observability

import (
	"context"

	"go.opentelemetry.io/otel/baggage"
)

func SetBaggage(ctx context.Context, members ...baggage.Member) (context.Context, error) {
	b := baggage.FromContext(ctx)
	var err error
	for _, m := range members {
		b, err = b.SetMember(m)
		if err != nil {
			return ctx, err
		}
	}
	return baggage.ContextWithBaggage(ctx, b), nil
}

func GetBaggage(ctx context.Context) baggage.Baggage {
	return baggage.FromContext(ctx)
}
