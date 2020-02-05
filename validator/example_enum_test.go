package validator_test

import (
	"fmt"

	"github.com/choria-io/go-validator/enum"
)

func Example_enum() {
	valid := []string{"one", "two", "three"}

	ok, err := enum.ValidateSlice([]string{"one", "two"}, valid)
	if !ok {
		panic(err)
	}

	fmt.Println("slice 1 is valid")

	ok, err = enum.ValidateSlice([]string{"5", "6"}, valid)
	if !ok {
		fmt.Println("slice 2 is invalid")
	}

	// string is valid
	ok, err = enum.ValidateString("one", valid)
	if !ok {
		panic(err)
	}

	fmt.Println("string is valid")

	// Output:
	// slice 1 is valid
	// slice 2 is invalid
	// string is valid
}
