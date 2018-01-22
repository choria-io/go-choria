package validator

import (
	"reflect"
	"strings"
)

// ShellSafe checks if a string is safe to use in a shell without any escapes or redirects
func ShellSafe(input string) bool {
	badchars := []string{"`", "$", ";", "|", "&&", ">", "<"}

	for _, c := range badchars {
		if strings.Contains(input, c) {
			return false
		}
	}

	return true
}

// ShellSafeValue validates a reflect.Value is shellsafe
func ShellSafeValue(input reflect.Value) bool {
	if input.Kind() != reflect.String {
		return false
	}

	if !ShellSafe(input.String()) {
		return false
	}

	return true
}
