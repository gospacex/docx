//go:build wireinject

package main

import (
	"github.com/google/wire"
)

// InjectMessage demonstrates wire.FieldsOf - using struct fields as providers.
// wire.FieldsOf extracts field values from a provided struct and makes them
// available as dependencies. In this example, Foo is provided with hardcoded
// values, then wire.FieldsOf extracts its Bar, Baz, Qux fields as providers.
func InjectMessage() (*Message, error) {
	wire.Build(
		// ProvideFoo creates a Foo (not *Foo) with all fields populated.
		// This breaks the cycle: Foo doesn't depend on Bar, Baz, Qux.
		ProvideFoo,
		// wire.FieldsOf extracts Bar, Baz, Qux from the provided Foo.
		// Foo.Bar becomes a provider of *Bar, etc.
		wire.FieldsOf(new(Foo), "Bar", "Baz", "Qux"),
		// NewMessage needs *Bar, *Baz, *Qux which come from Foo via wire.FieldsOf.
		wire.Struct(new(Message), "Bar", "Baz", "Qux"),
	)
	return nil, nil
}
