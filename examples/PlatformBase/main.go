// Package main is a PlatformBase E2E example showing how to compose
// multiple x-sdk providers through the hubx registry.
//
// What this example demonstrates end-to-end:
//
//  1. Viper-based config loader wired into the global registry.
//  2. Eager provider registration in init().
//  3. Lazy instance construction via hubx.Get (per (provider, instance)).
//  4. Bulk health probe across all built instances.
//  5. Single-shot shutdown that runs hooks in LIFO order.
//
// Required env: MYSQL_DSN (the only secret — pass via docker-compose env).
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	hubx "github.com/lego2/hubx"

	"github.com/gospacex/cachex/hubx/redisx"
	configx "github.com/lego2/configx/hubx"
	"github.com/gospacex/dbx/hubx/mysqlx"
	"github.com/gospacex/otelx/hubx/meter"
	"github.com/gospacex/otelx/hubx/tracer"
)

// Providers to build/run/shut down. The instance name MUST match a key
// under providers: in config.yaml.
//
// mqx.kafka.producer is intentionally omitted from this example: its
// mqx.Config struct is large and the kafkax driver dials a broker on
// Build, which fails in unit-test environments. Add it back when you
// have a Kafka broker reachable from CI.
var defaultInstances = []struct{ provider, instance string }{
	{"cachex.redis", "default"},
	{"dbx.mysql", "default"},
	{"otel.tracer", "default"},
	{"otel.meter", "default"},
}

// init registers all providers eagerly.
func init() {
	hubx.Register(redisx.New())
	hubx.Register(mysqlx.New())
	hubx.Register(tracer.New())
	hubx.Register(meter.New())
}

// App is a thin wrapper around the global hubx registry. The registry itself
// is process-global; App just orchestrates the lifecycle sequence.
type App struct{}

// NewApp returns an App. No state — kept as a constructor for symmetry with
// future refactors that need DI.
func NewApp() *App { return &App{} }

// Build lazily constructs every default instance via hubx.Get. Build errors
// for individual instances are accumulated and returned so the caller can
// decide whether to abort startup or continue with degraded mode.
func (a *App) Build(ctx context.Context) error {
	var errs []error
	for _, inst := range defaultInstances {
		if _, err := hubx.Get(inst.provider, inst.instance); err != nil {
			errs = append(errs, fmt.Errorf("%s/%s: %w", inst.provider, inst.instance, err))
		}
	}
	return errors.Join(errs...)
}

// Run builds all instances, runs a health probe, and returns. Shutdown is
// the caller's responsibility (defer hubx.Shutdown at the call site).
func (a *App) Run(ctx context.Context) error {
	buildCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := a.Build(buildCtx); err != nil {
		return fmt.Errorf("platformbase: build: %w", err)
	}

	fmt.Println("PlatformBase built. Health probe results:")
	for _, r := range hubx.HealthCheckAllInstances(ctx) {
		fmt.Printf("  %s/%s healthy=%v latency=%dms err=%v\n",
			r.Provider, r.Instance, r.Healthy, r.LatencyMs, r.Error)
	}
	return nil
}

func main() {
	cfgPath := "config.yaml"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}
	app, err := InitializeApp(cfgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() {
		if err := hubx.Shutdown(context.Background()); err != nil && !errors.Is(err, hubx.ErrRegistryClosed) {
			fmt.Fprintln(os.Stderr, "shutdown:", err)
		}
	}()
	if err := app.Run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// InitializeApp constructs a Viper-backed config loader, wires it into the
// global registry, and returns an App. Subsequent hubx.Get(provider, instance)
// calls will decode config from the loaded YAML.
func InitializeApp(configPath string) (*App, error) {
	loader, err := configx.NewViperLoader(configPath)
	if err != nil {
		return nil, fmt.Errorf("platformbase: load config: %w", err)
	}
	hubx.SetConfigLoader(loader)
	return NewApp(), nil
}
