// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package validator_test

import (
	"fmt"

	"github.com/choria-io/go-choria/validator/maxlength"
)

func Example_maxlength() {
	ok, err := maxlength.ValidateString("a short string", 20)
	if !ok {
		panic(err)
	}

	fmt.Println("string validates")

	// Output: string validates
}
