package mongo

import (
	"context"
	"fmt"
	"time"

	gomongo "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func MOS(ctx context.Context, cfg *Config) (*Collection, error) {
	if err := cfg.ValidateCollection(); err != nil {
		return nil, err
	}

	collectionKey, err := collectionConfigKey(cfg)
	if err != nil {
		return nil, err
	}
	clientKey, err := clientConfigKey(cfg)
	if err != nil {
		return nil, err
	}

	val, err := getOrCreate(ctx, collectionKey, kindCollection, func(ctx context.Context) (any, error) {
		return newCollection(ctx, cfg, collectionKey, clientKey)
	})
	if err != nil {
		return nil, err
	}
	coll, ok := val.(*Collection)
	if !ok {
		return nil, fmt.Errorf("mongo: cache value for %q is %T, want *Collection", collectionKey, val)
	}
	return coll, nil
}

func newCollection(ctx context.Context, cfg *Config, collectionKey, clientKey string) (*Collection, error) {
	cl, err := getOrCreateClient(ctx, cfg, clientKey)
	if err != nil {
		return nil, err
	}

	return &Collection{
		Name:     cfg.Collection,
		client:   cl,
		coll:     cl.client.Database(cfg.Database).Collection(cfg.Collection),
		cacheKey: collectionKey,
	}, nil
}

func openClient(ctx context.Context, cfg *Config) (*Client, error) {
	if err := cfg.ValidateClient(); err != nil {
		return nil, err
	}
	key, err := clientConfigKey(cfg)
	if err != nil {
		return nil, err
	}
	return getOrCreateClient(ctx, cfg, key)
}

func getOrCreateClient(ctx context.Context, cfg *Config, key string) (*Client, error) {
	val, err := getOrCreate(ctx, key, kindClient, func(ctx context.Context) (any, error) {
		return newClient(ctx, cfg, key)
	})
	if err != nil {
		return nil, err
	}
	client, ok := val.(*Client)
	if !ok {
		return nil, fmt.Errorf("mongo: cache value for %q is %T, want *Client", key, val)
	}
	return client, nil
}

func newClient(ctx context.Context, cfg *Config, clientKey string) (*Client, error) {
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

	client, err := gomongo.Connect(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("mongo: connect: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("mongo: ping: %w", err)
	}

	return &Client{client: client, cfg: cfg, cacheKey: clientKey}, nil
}

func clientConfigKey(cfg *Config) (string, error) {
	normalized := *cfg
	normalized.Database = ""
	normalized.Collection = ""
	fp, err := normalized.CacheFingerprint()
	if err != nil {
		return "", err
	}
	return "client:" + fp, nil
}

func collectionConfigKey(cfg *Config) (string, error) {
	fp, err := cfg.CacheFingerprint()
	if err != nil {
		return "", err
	}
	return "collection:" + fp, nil
}

func clientFileKey(absPath string, cfg *Config) (string, error) {
	normalized := *cfg
	normalized.Database = ""
	normalized.Collection = ""
	fp, err := normalized.CacheFingerprint()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("client:file:%s:%s", absPath, fp), nil
}

func collectionFileKey(absPath string, cfg *Config) (string, error) {
	fp, err := cfg.CacheFingerprint()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("collection:file:%s:%s", absPath, fp), nil
}
