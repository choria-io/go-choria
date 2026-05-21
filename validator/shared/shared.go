package shared

import "reflect"

type ValidatorFunc[T any] func(value T, typeField reflect.StructField, tag string) error

type ValidatorEntry struct {
	Handler func(reflect.Value, reflect.StructField, string) error
	Prefix  bool
}

var Validators = map[string]ValidatorEntry{}

// RegisterValidator registers a validator for a tag (exact or prefix)
func RegisterValidator(tag string, handler func(reflect.Value, reflect.StructField, string) error, prefix bool) {
	Validators[tag] = ValidatorEntry{Handler: handler, Prefix: prefix}
}

// Helper to create a validator handler from a generic ValidatorFunc
func MakeValidatorHandler[T any](fn ValidatorFunc[T]) func(reflect.Value, reflect.StructField, string) error {
	return func(v reflect.Value, t reflect.StructField, tag string) error {
		val, ok := v.Interface().(T)
		if !ok {
			if v.Kind() == reflect.Int64 && reflect.TypeOf((*T)(nil)).Elem() == reflect.TypeOf(int64(0)) {
				val = any(v.Int()).(T)
			} else {
				return nil
			}
		}
		return fn(val, t, tag)
	}
}
