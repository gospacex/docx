package mongo

import (
	"fmt"
	"sync"
	"time"
)

const errorCooldown = 30 * time.Second

type cacheEntry struct {
	client     *Client
	collection *Collection
	err        error
	failedAt   time.Time
}

var (
	store sync.Map
	lock  sync.Mutex
)

func getOrCreate(key string, fn func() (interface{}, error)) (interface{}, error) {
	if val, ok := store.Load(key); ok {
		e := val.(*cacheEntry)
		if e.client != nil {
			return e.client, nil
		}
		if e.collection != nil {
			return e.collection, nil
		}
		if e.err != nil && time.Since(e.failedAt) < errorCooldown {
			return nil, e.err
		}
	}

	lock.Lock()
	if val, ok := store.Load(key); ok {
		lock.Unlock()
		e := val.(*cacheEntry)
		if e.client != nil {
			return e.client, nil
		}
		if e.collection != nil {
			return e.collection, nil
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
	case *Client:
		e.client = v
	case *Collection:
		e.collection = v
	default:
		store.Delete(key)
		return nil, fmt.Errorf("mongo: unexpected type %T", result)
	}
	return result, nil
}