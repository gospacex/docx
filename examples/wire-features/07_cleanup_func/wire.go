//go:build wireinject

package main

import "github.com/google/wire"

// InitializeService creates a ServiceOwner with its cleanup function.
func InitializeService() (*ServiceOwner, error) {
	wire.Build(
		NewDatabase,
		NewCache,
		NewService,
		newCleanupFunc,
		wireStruct,
	)
	return nil, nil
}

// wireStruct provides the struct for injection.
var wireStruct = wire.Struct(new(ServiceOwner), "Service", "Cleanup")
