package maxlength

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
)

// ValidateString validates that input is not longer than max
func ValidateString(input string, max int) (bool, error) {
	if len(input) > max {
		return false, fmt.Errorf("%d characters, max allowed %d", len(input), max)
	}

	return true, nil
}

// ValidateStructField validates value based on the tag in the form maxlength=10
func ValidateStructField(value reflect.Value, tag string) (bool, error) {
	re := regexp.MustCompile(`^maxlength=(\d+)$`)
	parts := re.FindStringSubmatch(tag)

	if len(parts) != 2 {
		return false, fmt.Errorf("invalid tag '%s', must be maxlength=n", tag)
	}

	max, _ := strconv.Atoi(parts[1])

	switch value.Kind() {
	case reflect.String:
		return ValidateString(value.String(), max)
	case reflect.Slice:
		l := value.Slice(0, value.Len()).Len()
		if l > max {
			return false, fmt.Errorf("%d values, max allowed %d", l, max)
		}

		return true, nil

	default:
		return false, fmt.Errorf("cannot check length of %s type", value.Kind().String())
	}
}
