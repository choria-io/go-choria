package validator_test

import (
	"fmt"

	validator "github.com/choria-io/go-validator"
)

type Request struct {
	Command string   `validate:"shellsafe"`
	Flags   []string `validate:"enum=debug,verbose"`
	Args    string   `validate:"maxlength=128`
}

func Example_struct() {
	r := Request{
		Command: "/bin/some/command",
		Flags:   []string{"debug"},
		Args:    "hello world",
	}

	ok, err := validator.ValidateStruct(r)
	if !ok {
		panic(err)
	}

	fmt.Println("valid request")

	// Output: valid request
}
