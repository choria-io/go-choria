package validator_test

import (
	"fmt"

	"github.com/choria-io/go-validator/shellsafe"
)

func Example_shellsafe() {
	// a sell safe command, unsafe might be `un > safe`
	ok, err := shellsafe.Validate("safe")
	if !ok {
		panic(err)
	}

	fmt.Printf("string is safe")
	// Output: string is safe
}
