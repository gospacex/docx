package couchbase

import (
	"context"

	"github.com/couchbase/gocb/v2"
	"github.com/gospacex/hubx/cache/docx/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func startSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return observability.StartSpan(ctx, "couchbase."+name, trace.WithAttributes(attrs...))
}

func GetTrace(ctx context.Context, bucket *Bucket, id string) (*gocb.GetResult, error) {
	ctx, span := startSpan(ctx, "Get",
		attribute.String("bucket", bucket.Name),
		attribute.String("key", id),
	)
	defer span.End()

	doc, err := bucket.Get(id)
	if err != nil {
		span.RecordError(err)
	}
	return doc, err
}

func InsertTrace(ctx context.Context, bucket *Bucket, id string, value interface{}) (*gocb.MutationResult, error) {
	ctx, span := startSpan(ctx, "Insert",
		attribute.String("bucket", bucket.Name),
		attribute.String("key", id),
	)
	defer span.End()

	result, err := bucket.Insert(id, value)
	if err != nil {
		span.RecordError(err)
	}
	return result, err
}

func UpdateTrace(ctx context.Context, bucket *Bucket, id string, value interface{}) (*gocb.MutationResult, error) {
	ctx, span := startSpan(ctx, "Upsert",
		attribute.String("bucket", bucket.Name),
		attribute.String("key", id),
	)
	defer span.End()

	result, err := bucket.Upsert(id, value)
	if err != nil {
		span.RecordError(err)
	}
	return result, err
}

func DeleteTrace(ctx context.Context, bucket *Bucket, id string) (*gocb.MutationResult, error) {
	ctx, span := startSpan(ctx, "Remove",
		attribute.String("bucket", bucket.Name),
		attribute.String("key", id),
	)
	defer span.End()

	result, err := bucket.Remove(id)
	if err != nil {
		span.RecordError(err)
	}
	return result, err
}
