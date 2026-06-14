package mongo

import (
	"context"

	gomongo "go.mongodb.org/mongo-driver/mongo"
)

type Collection struct {
	Name     string
	client   *Client
	coll     *gomongo.Collection
	cacheKey string
}

func (c *Collection) Find(ctx context.Context, filter interface{}) (*gomongo.Cursor, error) {
	return c.coll.Find(ctx, filter)
}

func (c *Collection) FindOne(ctx context.Context, filter interface{}) *gomongo.SingleResult {
	return c.coll.FindOne(ctx, filter)
}

func (c *Collection) InsertOne(ctx context.Context, doc interface{}) (*gomongo.InsertOneResult, error) {
	return c.coll.InsertOne(ctx, doc)
}

func (c *Collection) UpdateOne(ctx context.Context, filter interface{}, update interface{}) (*gomongo.UpdateResult, error) {
	return c.coll.UpdateOne(ctx, filter, update)
}

func (c *Collection) DeleteOne(ctx context.Context, filter interface{}) (*gomongo.DeleteResult, error) {
	return c.coll.DeleteOne(ctx, filter)
}

func (c *Collection) HealthCheck(ctx context.Context) error {
	return c.client.HealthCheck(ctx)
}

func (c *Collection) Close(ctx context.Context) error {
	evict(c.cacheKey)
	if c.client != nil {
		return c.client.Close(ctx)
	}
	return nil
}

type Client struct {
	client   *gomongo.Client
	cfg      *Config
	cacheKey string
}

func (c *Client) Database(name string) *gomongo.Database {
	return c.client.Database(name)
}

func (c *Client) Collection(dbName, collName string) *Collection {
	db := c.client.Database(dbName)
	return &Collection{
		Name:     collName,
		client:   c,
		coll:     db.Collection(collName),
		cacheKey: c.cacheKey,
	}
}

func (c *Client) HealthCheck(ctx context.Context) error {
	return c.client.Ping(ctx, nil)
}

func (c *Client) Close(ctx context.Context) error {
	evict(c.cacheKey)
	if c.client != nil {
		return c.client.Disconnect(ctx)
	}
	return nil
}

func (c *Client) Config() *Config {
	return c.cfg
}
