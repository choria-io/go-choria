package duration

import (
	"fmt"
	"reflect"
	"time"
)

// ValidateString validates that input is a valid duration
func ValidateString(input string) (bool, error) {
	_, err := time.ParseDuration(input)
	if err != nil {
		return false, err
	}

	return true, nil
}

// ValidateStructField validates a struct field holds a valid duration
func ValidateStructField(value reflect.Value, tag string) (bool, error) {
	if value.Kind() != reflect.String {
		return false, fmt.Errorf("only strings can be Duration validated")
	}

	return ValidateString(value.String())
}
