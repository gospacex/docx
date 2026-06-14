package couchbase

import (
	"context"

	"github.com/couchbase/gocb/v2"
)

type Bucket struct {
	Name     string
	cluster  *Cluster
	bucket   *gocb.Bucket
	cacheKey string
}

func (b *Bucket) Get(id string) (*gocb.GetResult, error) {
	col := b.bucket.DefaultCollection()
	return col.Get(id, nil)
}

func (b *Bucket) Insert(id string, value interface{}) (*gocb.MutationResult, error) {
	col := b.bucket.DefaultCollection()
	return col.Insert(id, value, nil)
}

func (b *Bucket) Upsert(id string, value interface{}) (*gocb.MutationResult, error) {
	col := b.bucket.DefaultCollection()
	return col.Upsert(id, value, nil)
}

func (b *Bucket) Remove(id string) (*gocb.MutationResult, error) {
	col := b.bucket.DefaultCollection()
	return col.Remove(id, nil)
}

func (b *Bucket) Ping() (*gocb.PingResult, error) {
	return b.bucket.Ping(&gocb.PingOptions{})
}

func (b *Bucket) HealthCheck(ctx context.Context) error {
	_ = ctx
	_, err := b.bucket.Ping(&gocb.PingOptions{})
	return err
}

func (b *Bucket) Close() error {
	evict(b.cacheKey)
	if b.cluster != nil {
		return b.cluster.Close()
	}
	return nil
}

type Cluster struct {
	cluster  *gocb.Cluster
	cfg      *Config
	cacheKey string
}

func (c *Cluster) Bucket(name string) *Bucket {
	return &Bucket{
		Name:     name,
		cluster:  c,
		bucket:   c.cluster.Bucket(name),
		cacheKey: c.cacheKey,
	}
}

func (c *Cluster) Ping() (*gocb.PingResult, error) {
	return c.cluster.Ping(&gocb.PingOptions{})
}

func (c *Cluster) HealthCheck(ctx context.Context) error {
	_ = ctx
	_, err := c.cluster.Ping(&gocb.PingOptions{})
	return err
}

func (c *Cluster) Close() error {
	evict(c.cacheKey)
	if c.cluster != nil {
		return c.cluster.Close(&gocb.ClusterCloseOptions{})
	}
	return nil
}

func (c *Cluster) Config() *Config {
	return c.cfg
}
