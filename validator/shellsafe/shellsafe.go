package shellsafe

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Validate checks if a string is safe to use in a shell without any escapes or redirects
func Validate(input string) (bool, error) {
	badchars := []string{"`", "$", ";", "|", "&&", ">", "<"}

	for _, c := range badchars {
		if strings.Contains(input, c) {
			return false, fmt.Errorf("may not contain '%s'", c)
		}
	}

	return true, nil
}

// ValidateStructField validates a reflect.Value is shellsafe
func ValidateStructField(value reflect.Value, tag string) (bool, error) {
	if value.Kind() != reflect.String {
		return false, errors.New("should be a string")
	}

	return Validate(value.String())
}
