package main

import (
	"fmt"
)

// Message is a simple string wrapper
type Message struct {
	msg string
}

// NewMessage creates a new Message
func NewMessage() *Message {
	return &Message{msg: "Hello, World!"}
}

// Fizz is a simple string wrapper
type Fizz struct {
	val string
}

// NewFizz creates a new Fizz
func NewFizz() *Fizz {
	return &Fizz{val: "Fizz"}
}

// Buzz is a simple string wrapper
type Buzz struct {
	val string
}

// NewBuzz creates a new Buzz
func NewBuzz() *Buzz {
	return &Buzz{val: "Buzz"}
}

// Foo holds all the injected dependencies
type Foo struct {
	Bar *Message // Field name "Bar" will be matched to NewMessage
	Baz *Fizz    // Field name "Baz" will be matched to NewFizz
	Qux *Buzz    // Field name "Qux" will be matched to NewBuzz
}

func main() {
	// Wire will generate wire_gen.go with InitializeFoo()
	// that uses wire.Struct to inject struct fields.
	//
	// The wire.Struct(new(Foo), "Bar", "Baz", "Qux") tells Wire to:
	// - Create a new Foo
	// - Set Bar field using NewMessage()
	// - Set Baz field using NewFizz()
	// - Set Qux field using NewBuzz()
	//
	// Generated code is equivalent to:
	//   func InitializeFoo() (*Foo, error) {
	//       foo := &Foo{
	//           Bar: NewMessage(),
	//           Baz: NewFizz(),
	//           Qux: NewBuzz(),
	//       }
	//       return foo, nil
	//   }

	foo, err := InitializeFoo()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Foo.Bar.msg = %s\n", foo.Bar.msg)
	fmt.Printf("Foo.Baz.val = %s\n", foo.Baz.val)
	fmt.Printf("Foo.Qux.val = %s\n", foo.Qux.val)
}
