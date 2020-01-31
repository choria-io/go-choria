// Package confkey looks for tags on a structure and set values
// based on the tag rather than the struct item names
//
// Features
//
// Defaults are supported and can be fetched from the shell environment
//
// The tags can specify some formating like comma splits and other
// commonly seen patterns in config files.
//
// Conversion of []string, ints, strings, time.Duration and booleans are support
//
// Validations can be done on a struct basis using the github.com/choria-io/go-validators
// package
//
// A sample structure might look like this, the package contains utilities to
// set values, apply defaults and perform validations
//
//    type Config struct {
//        Loglevel string        `confkey:"loglevel" default:"warn" validate:"enum=debug,info,warn,error"`
//        Mode     string        `confkey:"mode" default:"server" validate:"enum=server,client"`
//        Servers  []string      `confkey:"servers" type:"comma_split" environment:"SERVERS"`
//        Path     []string      `confkey:"path" type:"path_split" default:"/bin:/usr/bin"`
//        I        time.Duration `confkey:"interval" type:"duration" default:"1h"`
//    }
//
// The utilities here will let you parse any config file that might have keys like loglevel etc
// and map the string values read from the text file onto the structure
package confkey

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"

	validator "github.com/choria-io/go-validator"
)

// Validate validates the struct
func Validate(target interface{}) error {
	_, err := validator.ValidateStruct(target)

	return err
}

// SetStructDefaults extract defaults out of the tags and set them to the key
func SetStructDefaults(target interface{}) error {
	if reflect.TypeOf(target).Kind() != reflect.Ptr {
		return errors.New("pointer is required")
	}

	st := reflect.TypeOf(target).Elem()

	for i := 0; i <= st.NumField()-1; i++ {
		field := st.Field(i)

		if key, ok := field.Tag.Lookup("confkey"); ok {
			if value, ok := field.Tag.Lookup("default"); ok {
				err := SetStructFieldWithKey(target, key, value)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// StringFieldWithKey retrieves a string from target that matches key, "" when not found
func StringFieldWithKey(target interface{}, key string) string {
	item, err := fieldWithKey(target, key)
	if err != nil {
		return ""
	}

	field := reflect.ValueOf(target).Elem().FieldByName(item)

	if field.Kind() == reflect.String {
		ptr := field.Addr().Interface().(*string)

		return string(*ptr)
	}

	return ""
}

// StringListWithKey retrieves a []string from target that matches key, empty when not found
func StringListWithKey(target interface{}, key string) []string {
	item, err := fieldWithKey(target, key)
	if err != nil {
		return []string{}
	}

	field := reflect.ValueOf(target).Elem().FieldByName(item)

	if field.Kind() == reflect.Slice {
		ptr := field.Addr().Interface().(*[]string)

		if *ptr == nil {
			return []string{}
		}

		return []string(*ptr)
	}

	return []string{}
}

// BoolWithKey retrieves a bool from target that matches key, false when not found
func BoolWithKey(target interface{}, key string) bool {
	item, err := fieldWithKey(target, key)
	if err != nil {
		return false
	}

	field := reflect.ValueOf(target).Elem().FieldByName(item)

	if field.Kind() == reflect.Bool {
		ptr := field.Addr().Interface().(*bool)

		return bool(*ptr)
	}

	return false
}

// IntWithKey retrieves an int from target that matches key, 0 when not found
func IntWithKey(target interface{}, key string) int {
	item, err := fieldWithKey(target, key)
	if err != nil {
		return 0
	}

	field := reflect.ValueOf(target).Elem().FieldByName(item)

	if field.Kind() == reflect.Int {
		ptr := field.Addr().Interface().(*int)

		return int(*ptr)
	}

	return 0
}

// Int64WithKey retrieves an int from target that matches key, 0 when not found
func Int64WithKey(target interface{}, key string) int64 {
	item, err := fieldWithKey(target, key)
	if err != nil {
		return 0
	}

	field := reflect.ValueOf(target).Elem().FieldByName(item)

	if field.Kind() == reflect.Int64 {
		ptr := field.Addr().Interface().(*int64)

		return int64(*ptr)
	}

	return 0
}

// SetStructFieldWithKey finds the struct key that matches the confkey on target and assign the value to it
func SetStructFieldWithKey(target interface{}, key string, value interface{}) error {
	if reflect.TypeOf(target).Kind() != reflect.Ptr {
		return errors.New("pointer is required")
	}

	item, err := fieldWithKey(target, key)
	if err != nil {
		return err
	}

	if tag, ok := tag(target, item, "environment"); ok {
		if v, ok := os.LookupEnv(tag); ok {
			value = v
		}
	}

	field := reflect.ValueOf(target).Elem().FieldByName(item)

	switch field.Kind() {
	case reflect.Slice:
		ptr := field.Addr().Interface().(*[]string)

		if tag, ok := tag(target, item, "type"); ok {
			switch tag {
			case "comma_split":
				// specifically clear it since these are one line split like 'collectives'
				*ptr = []string{}
				vals := strings.Split(value.(string), ",")

				for _, v := range vals {
					*ptr = append(*ptr, strings.TrimSpace(v))
				}

			case "colon_split":
				// these are like libdir, but we want to always use : to split and not
				// os path like path_split would do
				vals := strings.Split(value.(string), ":")

				for _, v := range vals {
					*ptr = append(*ptr, strings.TrimSpace(v))
				}

			case "path_split":
				// these are like libdir, either a one line split or a multiple occurance with splits
				vals := strings.Split(value.(string), string(os.PathListSeparator))

				for _, v := range vals {
					*ptr = append(*ptr, strings.TrimSpace(v))
				}
			}
		} else {
			*ptr = append(*ptr, strings.TrimSpace(value.(string)))
		}

	case reflect.Int:
		ptr := field.Addr().Interface().(*int)
		i, err := strconv.Atoi(value.(string))
		if err != nil {
			return err
		}
		*ptr = i

	case reflect.Int64:
		if tag, ok := tag(target, item, "type"); ok {
			if tag == "duration" {
				ptr := field.Addr().Interface().(*time.Duration)

				intonly, err := regexp.MatchString("\\A\\d+\\z", value.(string))
				if err != nil {
					return err
				}

				if intonly {
					i, err := strconv.Atoi(value.(string))
					if err != nil {
						return err
					}

					*ptr = time.Second * time.Duration(i)

					break
				}

				d, err := time.ParseDuration(value.(string))
				if err != nil {
					return err
				}

				*ptr = d
			}
		}

	case reflect.String:
		ptr := field.Addr().Interface().(*string)
		*ptr = value.(string)

		if tag, ok := tag(target, item, "type"); ok {
			switch tag {
			case "title_string":
				a := []rune(value.(string))
				a[0] = unicode.ToUpper(a[0])
				*ptr = string(a)
			case "path_string":
				a := strings.TrimSpace(value.(string))
				if a[0] == '~' {
					home, err := homeDir()
					if err != nil {
						return err
					}
					a = strings.Replace(a, "~", home, 1)
				}
				*ptr = a
			}
		}

	case reflect.Bool:
		ptr := field.Addr().Interface().(*bool)
		b, _ := strToBool(value.(string))
		*ptr = b
	}

	_, err = validator.ValidateStructField(target, item)

	return err
}

func homeDir() (string, error) {
	if runtime.GOOS == "windows" {
		drive := os.Getenv("HOMEDRIVE")
		home := os.Getenv("HOMEDIR")

		if home == "" || drive == "" {
			return "", fmt.Errorf("cannot determine home dir, ensure HOMEDRIVE and HOMEDIR is set")
		}

		return filepath.Join(os.Getenv("HOMEDRIVE"), os.Getenv("HOMEDIR")), nil
	}

	home := os.Getenv("HOME")

	if home == "" {
		return "", fmt.Errorf("cannot determine home dir, ensure HOME is set")
	}

	return home, nil
}

// determines the struct key name that is tagged with a certain confkey
func fieldWithKey(s interface{}, key string) (string, error) {
	st := reflect.TypeOf(s)
	if st.Kind() == reflect.Ptr {
		st = st.Elem()
	}

	for i := 0; i <= st.NumField()-1; i++ {
		field := st.Field(i)

		if confkey, ok := field.Tag.Lookup("confkey"); ok {
			if confkey == key {
				return field.Name, nil
			}
		}
	}

	return "", fmt.Errorf("can't find any structure element configured with confkey '%s'", key)
}

// retrieve a tag for a struct field
func tag(s interface{}, field string, tag string) (string, bool) {
	st := reflect.TypeOf(s)

	if st.Kind() == reflect.Ptr {
		st = st.Elem()
	}

	for i := 0; i <= st.NumField()-1; i++ {
		f := st.Field(i)

		if f.Name == field {
			if value, ok := f.Tag.Lookup(tag); ok {
				return value, true
			}
		}
	}

	return "", false
}

// StrToBool converts a typical boolianish string to bool.
//
// 1, yes, true, y, t will be true
// 0, no, false, n, f will be false
// anything else will be false with an error
func strToBool(s string) (bool, error) {
	clean := strings.TrimSpace(s)

	if regexp.MustCompile(`(?i)^(1|yes|true|y|t)$`).MatchString(clean) {
		return true, nil
	}

	if regexp.MustCompile(`(?i)^(0|no|false|n|f)$`).MatchString(clean) {
		return false, nil
	}

	return false, errors.New("cannot convert string value '" + clean + "' into a boolean.")
}
