package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/gospacex/hubx/cache/docx/observability"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func MOS(ctx context.Context, cfg *Config) (*Collection, error) {
	if cfg == nil {
		return nil, fmt.Errorf("mongo: config is nil")
	}
	if cfg.URI == "" {
		return nil, fmt.Errorf("mongo: URI is required")
	}

	key := cfg.ContentHash()
	if key == "" {
		key = cfg.URI
	}

	val, err := getOrCreate(key, func() (interface{}, error) {
		return newCollection(ctx, cfg)
	})
	if err != nil {
		return nil, err
	}
	return val.(*Collection), nil
}

func newCollection(ctx context.Context, cfg *Config) (*Collection, error) {
	cl, err := newClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if cfg.Tracing.Enabled {
		if err := observability.InitTracing(ctx, cfg.Tracing); err != nil {
			return nil, fmt.Errorf("mongo: %w", err)
		}
	}

	dbName := cfg.Database
	if dbName == "" {
		dbName = "default"
	}

	db := cl.client.Database(dbName)

	collName := ""
	// If the URI has a collection path, use the default collection
	return &Collection{
		Name:   collName,
		client: cl,
		coll:   db.Collection(collName),
	}, nil
}

func newClient(ctx context.Context, cfg *Config) (*Client, error) {
	opts := options.Client().ApplyURI(cfg.URI)
	if cfg.Username != "" || cfg.Password != "" {
		creds := options.Credential{
			Username: cfg.Username,
			Password: cfg.Password,
		}
		opts.SetAuth(creds)
	}
	if cfg.ConnectTimeout > 0 {
		opts.SetConnectTimeout(time.Duration(cfg.ConnectTimeout) * time.Millisecond)
	}
	if cfg.MaxPoolSize > 0 {
		opts.SetMaxPoolSize(uint64(cfg.MaxPoolSize))
	}

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("mongo: connect: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("mongo: ping: %w", err)
	}

	return &Client{client: client, cfg: cfg}, nil
}
