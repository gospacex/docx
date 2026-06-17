//go:build wireinject

package main

import (
	"github.com/google/wire"
)

// ProvideMyFooer creates a MyFooer instance.
func ProvideMyFooer() *MyFooer {
	return &MyFooer{}
}

// ProvideBar creates a Bar with a Fooer.
func ProvideBar(f Fooer) *Bar {
	return &Bar{Foo: f}
}

// InitializeBar wires the Fooer interface to MyFooer implementation.
func InitializeBar() (*Bar, error) {
	wire.Build(wire.Bind(new(Fooer), new(*MyFooer)), ProvideMyFooer, ProvideBar)
	return nil, nil
}
