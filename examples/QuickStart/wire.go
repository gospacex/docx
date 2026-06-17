// Package main provides dependency injection setup using Google Wire.
//go:build wireinject

package main

import "github.com/google/wire"

// Injector 所有 provider 的统一注入器
type Injector struct {
	Greeter Greeter
}

// InitializeInjector Wire 自动注入
func InitializeInjector() (*Injector, error) {
	wire.Build(
		NewGreeter,
		wire.Struct(new(Injector), "*"),
	)
	return nil, nil
}
