// Copyright (c) 2020-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

func ParseMapStructure(properties map[string]any, target any) error {
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

func StringSliceHookFunc(f reflect.Type, t reflect.Type, data any) (any, error) {
	if f.Kind() != reflect.Array {
		return data, nil
	}

	if t != reflect.TypeOf([]string{}) {
		return data, nil
	}

	var result []string
	for _, env := range data.([]any) {
		s, ok := env.(string)
		if !ok {
			return nil, fmt.Errorf("string slice is required")
		}
		result = append(result, s)
	}

	return result, nil
}
