//go:build wireinject

package main

import "github.com/google/wire"

// Injector 所有 provider 的统一注入器
type Injector struct {
	MessageService MessageService
}

func InitializeInjector() (*Injector, error) {
	wire.Build(
		NewMessageService,
		wire.Struct(new(Injector), "*"),
	)
	return nil, nil
}
