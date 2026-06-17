package main

import (
	"fmt"
)

// Foo is a struct with fields that can be extracted via wire.FieldsOf.
type Foo struct {
	Bar *Bar
	Baz *Baz
	Qux *Qux
}

// Bar is a simple provider.
type Bar struct {
	Name string
}

// Baz is another provider.
type Baz struct {
	Value int
}

// Qux is an optional provider.
type Qux struct {
	Flag bool
}

// Message is the final product depending on Bar, Baz, Qux.
type Message struct {
	Bar *Bar
	Baz *Baz
	Qux *Qux
}

func (m *Message) Print() {
	fmt.Printf("Message: Bar=%+v, Baz=%+v, Qux=%+v\n", m.Bar, m.Baz, m.Qux)
}

// ProvideFoo creates a Foo (not *Foo) with all fields populated.
// This breaks the cycle: Foo doesn't depend on Bar, Baz, Qux.
func ProvideFoo() Foo {
	return Foo{
		Bar: &Bar{Name: "bar"},
		Baz: &Baz{Value: 42},
		Qux: &Qux{Flag: true},
	}
}

// NewMessage creates a Message from Bar, Baz, Qux.
func NewMessage(bar *Bar, baz *Baz, qux *Qux) *Message {
	return &Message{Bar: bar, Baz: baz, Qux: qux}
}

func main() {
	// After running `wire ./...`, InjectMessage will be generated in wire_gen.go.
	// It uses wire.FieldsOf to extract Bar, Baz, Qux from the provided Foo.
	msg, err := InjectMessage()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	msg.Print()
}
