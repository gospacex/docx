package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

type Collection struct {
	Name   string
	client *Client
	coll   *mongo.Collection
}

func (c *Collection) Find(ctx context.Context, filter interface{}) (*mongo.Cursor, error) {
	return c.coll.Find(ctx, filter)
}

func (c *Collection) FindOne(ctx context.Context, filter interface{}) *mongo.SingleResult {
	return c.coll.FindOne(ctx, filter)
}

func (c *Collection) InsertOne(ctx context.Context, doc interface{}) (*mongo.InsertOneResult, error) {
	return c.coll.InsertOne(ctx, doc)
}

func (c *Collection) UpdateOne(ctx context.Context, filter interface{}, update interface{}) (*mongo.UpdateResult, error) {
	return c.coll.UpdateOne(ctx, filter, update)
}

func (c *Collection) DeleteOne(ctx context.Context, filter interface{}) (*mongo.DeleteResult, error) {
	return c.coll.DeleteOne(ctx, filter)
}

func (c *Collection) HealthCheck(ctx context.Context) error {
	return c.client.HealthCheck(ctx)
}

type Client struct {
	client *mongo.Client
	cfg    *Config
}

func (c *Client) Database(name string) *mongo.Database {
	return c.client.Database(name)
}

func (c *Client) Collection(dbName, collName string) *Collection {
	db := c.client.Database(dbName)
	return &Collection{
		Name:   collName,
		client: c,
		coll:   db.Collection(collName),
	}
}

func (c *Client) HealthCheck(ctx context.Context) error {
	return c.client.Ping(ctx, nil)
}

func (c *Client) Close(ctx context.Context) error {
	return c.client.Disconnect(ctx)
}

func (c *Client) Config() *Config {
	return c.cfg
}