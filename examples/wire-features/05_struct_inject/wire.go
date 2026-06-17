//go:build wireinject
// +build wireinject

package main

import "github.com/google/wire"

// Foo depends on Message, Fizz, and Buzz.
// wire.Struct(new(Foo), "Bar", "Baz", "Qux") tells Wire to inject these fields
// by finding providers that return the matching types.
func InitializeFoo() (*Foo, error) {
	wire.Build(
		NewMessage,
		NewFizz,
		NewBuzz,
		wire.Struct(new(Foo), "Bar", "Baz", "Qux"),
	)
	return nil, nil
}
