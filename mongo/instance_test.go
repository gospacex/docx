package mongo

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetOrCreateConcurrentOnlyBuildsOnce(t *testing.T) {
	const key = "test:collection:concurrent"
	evict(key)
	defer evict(key)

	var builds atomic.Int32
	ctx := context.Background()

	fn := func(context.Context) (any, error) {
		builds.Add(1)
		time.Sleep(20 * time.Millisecond)
		return "ok", nil
	}

	const workers = 8
	results := make([]any, workers)
	errs := make([]error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = getOrCreate(ctx, key, kindCollection, fn)
		}(i)
	}
	wg.Wait()

	if builds.Load() != 1 {
		t.Fatalf("expected one build, got %d", builds.Load())
	}
	for i, err := range errs {
		if err != nil {
			t.Fatalf("worker %d returned error: %v", i, err)
		}
		if results[i] != "ok" {
			t.Fatalf("worker %d got unexpected result: %#v", i, results[i])
		}
	}
}

func TestFailureCooldownReusesErrorWithinCooldown(t *testing.T) {
	const key = "test:collection:cooldown"
	evict(key)
	defer evict(key)

	wantErr := errors.New("boom")
	var builds atomic.Int32
	fn := func(context.Context) (any, error) {
		builds.Add(1)
		return nil, wantErr
	}

	if _, err := getOrCreate(context.Background(), key, kindCollection, fn); !errors.Is(err, wantErr) {
		t.Fatalf("first call error = %v, want %v", err, wantErr)
	}
	if _, err := getOrCreate(context.Background(), key, kindCollection, fn); !errors.Is(err, wantErr) {
		t.Fatalf("second call error = %v, want %v", err, wantErr)
	}
	if builds.Load() != 1 {
		t.Fatalf("expected one build during cooldown, got %d", builds.Load())
	}
}
