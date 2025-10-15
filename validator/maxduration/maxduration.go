// Copyright (c) 2024, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package maxduration

import (
	"fmt"
	"github.com/choria-io/go-choria/validator/shared"
	"reflect"
	"strings"
	"time"
)

// maxDurationValidator validates that a time.Duration does not exceed a maximum specified in the tag
func maxDurationValidator(value time.Duration, typeField reflect.StructField, tag string) error {
	parts := strings.SplitN(tag, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid maxduration tag: %s", tag)
	}
	maxDur, err := time.ParseDuration(parts[1])
	if err != nil {
		return fmt.Errorf("invalid maxduration value: %s", parts[1])
	}
	if value > maxDur {
		return fmt.Errorf("%s exceeds maximum allowed duration of %s", typeField.Name, maxDur)
	}
	return nil
}

func init() {
	shared.RegisterValidator("maxduration=", shared.MakeValidatorHandler(maxDurationValidator), true)
}
