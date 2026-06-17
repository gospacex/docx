package main

import (
	"fmt"
)

// Fooer is an interface that defines a single method.
type Fooer interface {
	Foo() string
}

// MyFooer is a concrete implementation of Fooer.
type MyFooer struct{}

func (m *MyFooer) Foo() string {
	return "Hello from MyFooer!"
}

// Bar depends on a Fooer interface.
type Bar struct {
	Foo Fooer
}

func main() {
	// Manual wiring (without wire):
	// We create the implementation and inject it into Bar.
	fooer := &MyFooer{}
	bar := &Bar{Foo: fooer}
	fmt.Println(bar.Foo.Foo())
	// Output: Hello from MyFooer!
}
