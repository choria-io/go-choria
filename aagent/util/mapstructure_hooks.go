package util

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

func ParseMapStructure(properties map[string]interface{}, target interface{}) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook:       mapstructure.ComposeDecodeHookFunc(mapstructure.StringToTimeDurationHookFunc(), StringSliceHookFunc),
		Result:           target,
		WeaklyTypedInput: true,
	})
	if err != nil {
		return err
	}

	return decoder.Decode(properties)
}

func StringSliceHookFunc(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if f.Kind() != reflect.Array {
		return data, nil
	}

	if t != reflect.TypeOf([]string{}) {
		return data, nil
	}

	var result []string
	for _, env := range data.([]interface{}) {
		s, ok := env.(string)
		if !ok {
			return nil, fmt.Errorf("string slice is required")
		}
		result = append(result, s)
	}

	return result, nil
}
