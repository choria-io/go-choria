package validator_test

import (
	"fmt"

	"github.com/choria-io/go-validator/maxlength"
)

func Example_maxlength() {
	ok, err := maxlength.ValidateString("a short string", 20)
	if !ok {
		panic(err)
	}

	fmt.Println("string validates")

	// Output: string validates
}
