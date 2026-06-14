package couchbase

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type resourceKind string

const (
	kindCluster resourceKind = "cluster"
	kindBucket  resourceKind = "bucket"

	errorCooldown = 30 * time.Second
)

type cacheEntry struct {
	ready    chan struct{}
	kind     resourceKind
	value    any
	err      error
	failedAt time.Time
}

var store sync.Map

func getOrCreate(ctx context.Context, key string, kind resourceKind, fn func(context.Context) (any, error)) (any, error) {
	for {
		fresh := &cacheEntry{ready: make(chan struct{}), kind: kind}
		actual, loaded := store.LoadOrStore(key, fresh)
		entry := actual.(*cacheEntry)

		if !loaded {
			fresh.value, fresh.err = fn(ctx)
			if fresh.err != nil {
				fresh.failedAt = time.Now()
			}
			close(fresh.ready)
			return fresh.value, fresh.err
		}

		select {
		case <-entry.ready:
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		if entry.kind != kind {
			return nil, fmt.Errorf("couchbase: cache kind mismatch for key %q: want %s, got %s", key, kind, entry.kind)
		}
		if entry.err == nil {
			return entry.value, nil
		}
		if time.Since(entry.failedAt) < errorCooldown {
			return nil, entry.err
		}

		if store.CompareAndSwap(key, entry, fresh) {
			fresh.value, fresh.err = fn(ctx)
			if fresh.err != nil {
				fresh.failedAt = time.Now()
			}
			close(fresh.ready)
			return fresh.value, fresh.err
		}
	}
}

func evict(key string) {
	if key != "" {
		store.Delete(key)
	}
}
