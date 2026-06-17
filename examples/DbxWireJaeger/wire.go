//go:build wireinject

// Package main wires DbxProvider into an Injector.
package main

import "github.com/google/wire"

// Injector carries the providers used by tests.
type Injector struct {
	DB *DbxProvider
}

// InitializeInjector is the wire entry point. cfgPath is a runtime arg
// passed straight through to ProvideDB at TestMain time; it is NOT used
// by wire itself (wire only resolves the type graph).
//
// Wire rule: an injector function body must contain ONLY the wire.Build
// call (and an optional return). cfgPath is a runtime arg passed by
// TestMain straight to ProvideDB; it's named in the signature so wire
// sees a consistent signature, and is intentionally not used here.
func InitializeInjector(cfgPath string) (*Injector, error) {
	wire.Build(
		NewDbxProvider,
		wire.Struct(new(Injector), "*"),
	)
	return nil, nil
}
