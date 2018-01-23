package regex

import (
	"fmt"
	"reflect"
	"regexp"
)

// ValidateString validates that a string matches a regex
func ValidateString(input string, regex string) (bool, error) {
	re, err := regexp.Compile(regex)
	if err != nil {
		return false, fmt.Errorf("invalid regex '%s'", regex)
	}

	if !re.MatchString(input) {
		return false, fmt.Errorf("input does not match '%s'", regex)
	}

	return true, nil
}

// ValidateStructField validates that field holds a string matching the regex
// tag must be in the form `validate:"regex=\d+"`
func ValidateStructField(value reflect.Value, tag string) (bool, error) {
	re := regexp.MustCompile(`^regex=(.+)$`)
	parts := re.FindStringSubmatch(tag)

	if len(parts) != 2 {
		return false, fmt.Errorf("invalid tag '%s', must be in the form regex=^hello.+world$", tag)
	}

	if value.Kind() != reflect.String {
		return false, fmt.Errorf("only strings can be regex validated")
	}

	return ValidateString(value.String(), parts[1])
}
