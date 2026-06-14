package mongo

import (
	"context"

	"github.com/gospacex/hubx/cache/docx/observability"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func startSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return observability.StartSpan(ctx, "mongo."+name, trace.WithAttributes(attrs...))
}

func FindTrace(ctx context.Context, coll *Collection, filter interface{}) (*mongo.Cursor, error) {
	ctx, span := startSpan(ctx, "Find",
		attribute.String("collection", coll.Name),
	)
	defer span.End()

	cur, err := coll.Find(ctx, filter)
	if err != nil {
		span.RecordError(err)
	}
	return cur, err
}

func FindOneTrace(ctx context.Context, coll *Collection, filter interface{}) *mongo.SingleResult {
	ctx, span := startSpan(ctx, "FindOne",
		attribute.String("collection", coll.Name),
	)
	defer span.End()

	return coll.FindOne(ctx, filter)
}

func InsertTrace(ctx context.Context, coll *Collection, doc interface{}) (*mongo.InsertOneResult, error) {
	ctx, span := startSpan(ctx, "InsertOne",
		attribute.String("collection", coll.Name),
	)
	defer span.End()

	result, err := coll.InsertOne(ctx, doc)
	if err != nil {
		span.RecordError(err)
	}
	return result, err
}

func UpdateTrace(ctx context.Context, coll *Collection, filter interface{}, update interface{}) (*mongo.UpdateResult, error) {
	ctx, span := startSpan(ctx, "UpdateOne",
		attribute.String("collection", coll.Name),
	)
	defer span.End()

	result, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		span.RecordError(err)
	}
	return result, err
}

func DeleteTrace(ctx context.Context, coll *Collection, filter interface{}) (*mongo.DeleteResult, error) {
	ctx, span := startSpan(ctx, "DeleteOne",
		attribute.String("collection", coll.Name),
	)
	defer span.End()

	result, err := coll.DeleteOne(ctx, filter)
	if err != nil {
		span.RecordError(err)
	}
	return result, err
}
