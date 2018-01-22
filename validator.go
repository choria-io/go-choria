/*
Package validator provides common validation helpers commonly used
in operations tools.  Additionally structures can be marked up with
tags indicating the validation of individual keys and the entire struct
can be validated in one go
*/
package validator

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/choria-io/go-validator/enum"
	"github.com/choria-io/go-validator/maxlength"
	"github.com/choria-io/go-validator/shellsafe"
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
		} else if ok, _ := regexp.MatchString(`^enum=(.+,*?)+$`, validation); ok {
			if ok, err := enum.ValidateStructField(valueField, validation); !ok {
				return false, fmt.Errorf("%s enum validation failed: %s", typeField.Name, err)
			}
		}
	}

	return true, nil
}
