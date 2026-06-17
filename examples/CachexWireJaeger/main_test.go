//go:build !wireinject

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

const (
	cfgPath         = "config.yaml"
	jaegerBase      = "http://localhost:16686"
	serviceName     = "examples-cachexwirejaeger"
	jaegerWaitMax   = 5 * time.Second
	jaegerPollEvery = 200 * time.Millisecond
)

var (
	cleanupTrace func(context.Context)
	injector     *Injector
	redisCli     *redis.Client
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	cleanup, err := SetupTracing(ctx, cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "init tracing: %v\n", err)
		os.Exit(1)
	}
	cleanupTrace = cleanup

	inj, err := InitializeInjector(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wire init: %v\n", err)
		os.Exit(1)
	}
	injector = inj

	cli, err := injector.Cache.ProvideRedisClient(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "provide redis client: %v\n", err)
		os.Exit(1)
	}
	redisCli = cli

	code := m.Run()
	cleanup(ctx)
	os.Exit(code)
}

// requireSpanReported polls jaeger /api/traces for an operation matching
// the given name within jaegerWaitMax. Fatal on timeout and prints the
// last response body so a future maintainer can debug.
func requireSpanReported(t *testing.T, operation string) {
	t.Helper()

	q := url.Values{}
	q.Set("service", serviceName)
	q.Set("operation", operation)
	q.Set("lookback", "1m")
	endpoint := fmt.Sprintf("%s/api/traces?%s", jaegerBase, q.Encode())

	deadline := time.Now().Add(jaegerWaitMax)
	var lastBody string
	for time.Now().Before(deadline) {
		resp, err := http.Get(endpoint) // #nosec G107 -- jaegerBase is localhost, intentional
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastBody = string(body)
			var data struct {
				Data []struct {
					Spans []struct {
						OperationName string `json:"operationName"`
					} `json:"spans"`
				} `json:"data"`
			}
			if json.Unmarshal(body, &data) == nil {
				for _, trace := range data.Data {
					for _, span := range trace.Spans {
						if span.OperationName == operation {
							return
						}
					}
				}
			}
		}
		time.Sleep(jaegerPollEvery)
	}
	t.Fatalf("expected span operation=%q in service=%q within %s; last response: %s",
		operation, serviceName, jaegerWaitMax, lastBody)
}

func TestCachexWireJaeger_StringTypes(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, redisCli.Set(ctx, "k:string", "alice", 0).Err())
	v, err := redisCli.Get(ctx, "k:string").Result()
	require.NoError(t, err)
	require.Equal(t, "alice", v)

	n, err := redisCli.Incr(ctx, "k:counter").Result()
	require.NoError(t, err)
	require.Equal(t, int64(1), n)

	require.NoError(t, redisCli.Del(ctx, "k:string", "k:counter").Err())

	requireSpanReported(t, "set")
	requireSpanReported(t, "get")
	requireSpanReported(t, "incr")
	requireSpanReported(t, "del")
}

func TestCachexWireJaeger_HashTypes(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, redisCli.HSet(ctx, "k:hash", "field", "value").Err())
	m, err := redisCli.HGetAll(ctx, "k:hash").Result()
	require.NoError(t, err)
	require.Equal(t, map[string]string{"field": "value"}, m)
	require.NoError(t, redisCli.HDel(ctx, "k:hash", "field").Err())

	requireSpanReported(t, "hset")
	requireSpanReported(t, "hgetall")
	requireSpanReported(t, "hdel")
}

func TestCachexWireJaeger_SetTypes(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, redisCli.SAdd(ctx, "k:set", "a", "b", "c").Err())
	members, err := redisCli.SMembers(ctx, "k:set").Result()
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"a", "b", "c"}, members)
	require.NoError(t, redisCli.SRem(ctx, "k:set", "a", "b").Err())

	requireSpanReported(t, "sadd")
	requireSpanReported(t, "smembers")
	requireSpanReported(t, "srem")
}

func TestCachexWireJaeger_ListTypes(t *testing.T) {
	ctx := context.Background()
	// Clean any residual keys from previous runs so the assertion is deterministic.
	redisCli.Del(ctx, "k:list")

	require.NoError(t, redisCli.LPush(ctx, "k:list", "v1", "v2").Err())
	items, err := redisCli.LRange(ctx, "k:list", 0, -1).Result()
	require.NoError(t, err)
	require.Equal(t, []string{"v2", "v1"}, items)
	require.NoError(t, redisCli.LPop(ctx, "k:list").Err())
	// Clean up so subsequent runs start fresh.
	redisCli.Del(ctx, "k:list")

	requireSpanReported(t, "lpush")
	requireSpanReported(t, "lrange")
	requireSpanReported(t, "lpop")
}

func TestCachexWireJaeger_SortedSetTypes(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, redisCli.ZAdd(ctx, "k:zset",
		redis.Z{Score: 1, Member: "a"},
		redis.Z{Score: 2, Member: "b"},
	).Err())
	got, err := redisCli.ZRangeWithScores(ctx, "k:zset", 0, -1).Result()
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.NoError(t, redisCli.ZRem(ctx, "k:zset", "a").Err())

	requireSpanReported(t, "zadd")
	requireSpanReported(t, "zrange")
	requireSpanReported(t, "zrem")
}
