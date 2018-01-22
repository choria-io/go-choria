package validator

import (
	"fmt"
	"reflect"
	"strings"
)

// ValidateStruct validates all keys in a struct using their validate tag
func ValidateStruct(target interface{}) (bool, error) {
	val := reflect.ValueOf(target)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	return validateStructValue(val)
}

func validateStructValue(val reflect.Value) (bool, error) {
	for i := 0; i < val.NumField(); i++ {
		valueField := val.Field(i)
		typeField := val.Type().Field(i)
		validations := strings.Split(typeField.Tag.Get("validate"), ",")

		if valueField.Kind() == reflect.Struct {
			ok, err := validateStructValue(valueField)
			if !ok {
				return ok, err
			}
		}

		for _, validation := range validations {
			validation = strings.TrimSpace(validation)

			if validation == "" {
				continue
			}

			switch validation {
			case "shellsafe":
				if !ShellSafeValue(valueField) {
					return false, fmt.Errorf("%s is not shellsafe", typeField.Name)
				}

			default:
				return false, fmt.Errorf("unknown validator on %s: %s", typeField.Name, validation)
			}
		}
	}

	return true, nil

}
