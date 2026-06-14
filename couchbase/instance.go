package couchbase

import (
	"fmt"
	"sync"
	"time"
)

const errorCooldown = 30 * time.Second

type cacheEntry struct {
	cluster  *Cluster
	bucket   *Bucket
	err      error
	failedAt time.Time
}

var (
	store sync.Map
	lock  sync.Mutex
)

func getOrCreate(key string, fn func() (interface{}, error)) (interface{}, error) {
	if val, ok := store.Load(key); ok {
		e := val.(*cacheEntry)
		if e.cluster != nil {
			return e.cluster, nil
		}
		if e.bucket != nil {
			return e.bucket, nil
		}
		if e.err != nil && time.Since(e.failedAt) < errorCooldown {
			return nil, e.err
		}
	}

	lock.Lock()
	if val, ok := store.Load(key); ok {
		lock.Unlock()
		e := val.(*cacheEntry)
		if e.cluster != nil {
			return e.cluster, nil
		}
		if e.bucket != nil {
			return e.bucket, nil
		}
		return nil, e.err
	}

	e := &cacheEntry{}
	store.Store(key, e)
	lock.Unlock()

	result, err := fn()
	if err != nil {
		e.err = err
		e.failedAt = time.Now()
		store.Delete(key)
		return nil, err
	}

	switch v := result.(type) {
	case *Cluster:
		e.cluster = v
	case *Bucket:
		e.bucket = v
	default:
		store.Delete(key)
		return nil, fmt.Errorf("couchbase: unexpected type %T", result)
	}
	return result, nil
}
