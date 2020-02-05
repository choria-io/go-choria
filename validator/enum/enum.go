package enum

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// ValidateSlice validates that all items in target is one of valid
func ValidateSlice(target []string, valid []string) (bool, error) {
	for _, item := range target {
		found, _ := ValidateString(item, valid)
		if !found {
			return false, fmt.Errorf("'%s' is not in the allowed list: %s", item, strings.Join(valid, ", "))
		}
	}

	return true, nil
}

// ValidateString validates that a string is in the list of allowed values
func ValidateString(target string, valid []string) (bool, error) {
	found := false

	for _, e := range valid {
		if e == target {
			found = true
		}
	}

	if !found {
		return false, fmt.Errorf("'%s' is not in the allowed list: %s", target, strings.Join(valid, ", "))
	}

	return true, nil
}

// ValidateStructField validates a structure field, only []string and string types are supported
func ValidateStructField(value reflect.Value, tag string) (bool, error) {
	re := regexp.MustCompile(`^enum=(.+,*?)+$`)
	parts := re.FindStringSubmatch(tag)

	if len(parts) != 2 {
		return false, fmt.Errorf("invalid tag '%s', must be enum=v1,v2,v3", tag)
	}

	evs := strings.Split(parts[1], ",")

	switch value.Kind() {
	case reflect.Slice:
		slice, ok := value.Interface().([]string)
		if !ok {
			return false, fmt.Errorf("only []string slices can be validated for enums")
		}

		return ValidateSlice(slice, evs)
	case reflect.String:
		str, _ := value.Interface().(string)

		return ValidateString(str, evs)

	default:
		return false, fmt.Errorf("cannot valid data of type %s for enums", value.Kind().String())
	}
}
