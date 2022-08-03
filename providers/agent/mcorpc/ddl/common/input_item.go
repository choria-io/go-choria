// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/choria-io/go-choria/internal/fs"
	"github.com/choria-io/go-choria/validator"
	"github.com/choria-io/go-choria/validator/ipaddress"
	"github.com/choria-io/go-choria/validator/ipv4"
	"github.com/choria-io/go-choria/validator/ipv6"
	"github.com/choria-io/go-choria/validator/regex"
	"github.com/choria-io/go-choria/validator/shellsafe"
)

var (
	InputTypeArray   = "Array"
	InputTypeBoolean = "boolean"
	InputTypeFloat   = "float"
	InputTypeHash    = "Hash"
	InputTypeInteger = "integer"
	InputTypeList    = "list"
	InputTypeNumber  = "number"
	InputTypeString  = "string"
	InputTypeAny     = ""
)

// InputItem describes an individual input item
type InputItem struct {
	Prompt      string   `json:"prompt"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Default     any      `json:"default,omitempty"`
	Optional    bool     `json:"optional"`
	Validation  string   `json:"validation,omitempty"`
	MaxLength   int      `json:"maxlength,omitempty"`
	Enum        []string `json:"list,omitempty"`
}

func (i *InputItem) RenderConsole() ([]byte, error) {
	return fs.ExecuteTemplate("ddl/console/input_item.templ", i, nil)
}

func (i *InputItem) RenderMarkdown() ([]byte, error) {
	return fs.ExecuteTemplate("ddl/markdown/input_item.templ", i, nil)
}

// Required indicates if this item is required
func (i *InputItem) Required() bool {
	return !i.Optional
}

// ConvertStringValue converts a string representing value into the correct type according to the input Type
func (i *InputItem) ConvertStringValue(val string) (any, error) {
	return ValToDDLType(i.Type, val)
}

// ValidateStringValue converts a value to the appropriate type for this input then validates it
func (i *InputItem) ValidateStringValue(val string) (converted any, warnings []string, err error) {
	converted, err = ValToDDLType(i.Type, val)
	if err != nil {
		return nil, warnings, err
	}

	warnings, err = i.ValidateValue(converted)

	return converted, warnings, err
}

// ValidateValue validates a value against this input, should be of the right data type already. See ValToDDLType()
func (i *InputItem) ValidateValue(val any) (warnings []string, err error) {
	switch strings.ToLower(i.Type) {
	case InputTypeInteger, "int":
		if !validator.IsAnyInt(val) && !validator.IsIntFloat64(val) {
			return warnings, fmt.Errorf("is not an integer")
		}

	case InputTypeNumber:
		if !validator.IsNumber(val) {
			return warnings, fmt.Errorf("is not a number")
		}

	case InputTypeFloat:
		if !validator.IsFloat64(val) {
			return warnings, fmt.Errorf("is not a float")
		}

	case InputTypeString:
		if !validator.IsString(val) {
			return warnings, fmt.Errorf("is not a string")
		}

		if i.MaxLength == 0 {
			return warnings, nil
		}

		sval := val.(string)
		if len(sval) > i.MaxLength {
			return warnings, fmt.Errorf("is longer than %d characters", i.MaxLength)
		}

		if i.Validation != "" {
			w, err := validateStringValidation(i.Validation, sval)

			warnings = append(warnings, w...)

			if err != nil {
				return warnings, err
			}
		}

	case InputTypeBoolean:
		if !validator.IsBool(val) {
			return warnings, fmt.Errorf("is not a boolean")
		}

	case InputTypeList:
		if len(i.Enum) == 0 {
			return warnings, fmt.Errorf("input type of list without a valid list of items in DDL")
		}

		valstr, ok := val.(string)
		if !ok {
			return warnings, fmt.Errorf("should be a string")
		}

		for _, valid := range i.Enum {
			if valid == valstr {
				return warnings, nil
			}
		}

		return warnings, fmt.Errorf("should be one of %s", strings.Join(i.Enum, ", "))

	case InputTypeHash, "hash":
		if !validator.IsMap(val) {
			return warnings, fmt.Errorf("is not a hash map")
		}

	case InputTypeArray, "array":
		if !validator.IsArray(val) {
			return warnings, fmt.Errorf("is not an array")
		}

	default:
		return warnings, fmt.Errorf("unsupported input type '%s'", i.Type)
	}

	return warnings, nil

}

func validateStringValidation(validation string, value string) (warnings []string, err error) {
	warnings = []string{}

	switch validation {
	case "shellsafe":
		_, err = shellsafe.Validate(value)
		return warnings, err

	case "ipv4address":
		_, err := ipv4.ValidateString(value)
		return warnings, err

	case "ipv6address":
		_, err := ipv6.ValidateString(value)
		return warnings, err

	case "ipaddress":
		_, err := ipaddress.ValidateString(value)
		return warnings, err
	}

	namedValidator, err := regexp.MatchString("^[a-z]", validation)
	if namedValidator || err != nil {
		return []string{fmt.Sprintf("Unsupported validator '%s'", validation)}, nil
	}

	_, err = regex.ValidateString(value, validation)
	return warnings, err
}
