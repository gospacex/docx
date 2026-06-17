//go:build wireinject

package main

import "github.com/google/wire"

// ProviderSet 将多个 Provider 分组
var ProviderSet = wire.NewSet(
	NewDatabase,
	NewCache,
	NewLogger,
)

// Injector 所有 provider 的统一注入器
type Injector struct {
	Database Database
	Cache    Cache
	Logger   Logger
}

func InitializeInjector() (*Injector, error) {
	wire.Build(
		wire.Value("localhost"),
		wire.Value(5432),
		ProviderSet,
		wire.Struct(new(Injector), "*"),
	)
	return nil, nil
}
