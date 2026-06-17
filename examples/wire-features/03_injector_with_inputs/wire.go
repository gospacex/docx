//go:build wireinject

package main

import "github.com/google/wire"

// Injector 所有 provider 的统一注入器
type Injector struct {
	MessageService MessageService
}

// InitializeInjector 带输入参数的注入器
// Wire 会自动将 Config 参数传递给需要它的 Provider
func InitializeInjector(cfg Config) (*Injector, error) {
	wire.Build(
		NewMessageService,
		wire.Struct(new(Injector), "*"),
	)
	return nil, nil
}
