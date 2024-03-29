// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package validator_test

import (
	"fmt"

	validator "github.com/choria-io/go-choria/validator"
)

type Request struct {
	Command string   `validate:"shellsafe"`
	Flags   []string `validate:"enum=debug,verbose"`
	Args    string   `validate:"maxlength=128"`
	AnyIP   string   `validate:"ipaddress"` // can also check ipv4 and ipv6
	User    string   `validate:"regex=^\\w+$"`
}

func Example_struct() {
	r := Request{
		Command: "/bin/some/command",
		Flags:   []string{"debug"},
		Args:    "hello world",
		AnyIP:   "2a00:1450:4003:807::200e",
		User:    "bob",
	}

	ok, err := validator.ValidateStruct(r)
	if !ok {
		panic(err)
	}

	fmt.Println("valid request")

	ok, err = validator.ValidateStructField(r, "Command")
	if !ok {
		panic(err)
	}

	fmt.Println("valid field")

	// Output:
	// valid request
	// valid field
}
