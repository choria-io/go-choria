package validator

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/choria-io/go-choria/mcorpc/validator/maxlength"
	"github.com/choria-io/go-choria/mcorpc/validator/shellsafe"
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
		validation := strings.TrimSpace(typeField.Tag.Get("validate"))

		if valueField.Kind() == reflect.Struct {
			ok, err := validateStructValue(valueField)
			if !ok {
				return ok, err
			}
		}

		if validation == "" {
			continue
		}

		if validation == "shellsafe" {
			if ok, err := shellsafe.ValidateStructField(valueField, validation); !ok {
				return false, fmt.Errorf("%s shellsafe validation failed: %s", typeField.Name, err)
			}

		} else if ok, _ := regexp.MatchString(`^maxlength=\d+$`, validation); ok {
			if ok, err := maxlength.ValidateStructField(valueField, validation); !ok {
				return false, fmt.Errorf("%s maxlength validation failed: %s", typeField.Name, err)
			}
		}
	}

	return true, nil

}
