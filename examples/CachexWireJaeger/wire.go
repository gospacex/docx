//go:build wireinject

// Package main wires CachexProvider into an Injector.
package main

import "github.com/google/wire"

// Injector carries the providers used by tests.
type Injector struct {
	Cache *CachexProvider
}

// InitializeInjector is the wire entry point. cfgPath is accepted for
// symmetry with ProvideRedisClient(cfgPath) but only the type graph is
// resolved here; cfgPath is consumed at runtime by the test harness.
func InitializeInjector(cfgPath string) (*Injector, error) {
	wire.Build(
		NewCachexProvider,
		wire.Struct(new(Injector), "*"),
	)
	return nil, nil
}
