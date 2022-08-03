// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"reflect"
)

// IsMap determines if i is a map
func IsMap(i any) bool {
	return reflect.ValueOf(i).Kind() == reflect.Map
}

// IsArray determines if i is a slice or array
func IsArray(i any) bool {
	kind := reflect.ValueOf(i).Kind()
	return kind == reflect.Array || kind == reflect.Slice
}

// IsBool determines if i is a boolean
func IsBool(i any) bool {
	return reflect.ValueOf(i).Kind() == reflect.Bool
}

// IsString determines if i is a string
func IsString(i any) bool {
	return reflect.ValueOf(i).Kind() == reflect.String
}

// IsNumber determines if i is a int or a float of any size
func IsNumber(i any) bool {
	return IsAnyInt(i) || IsAnyFloat(i)
}

// IsAnyFloat determines if i is a float32  or float64
func IsAnyFloat(i any) bool {
	return IsFloat32(i) || IsFloat64(i)
}

// IsFloat32 determines if i is a float32
func IsFloat32(i any) bool {
	return reflect.ValueOf(i).Kind() == reflect.Float32
}

// IsFloat64 determines if i is a float64
func IsFloat64(i any) bool {
	return reflect.ValueOf(i).Kind() == reflect.Float64
}

// IsAnyInt determines if i is a int, int8, int16, in32 or int64
func IsAnyInt(i any) bool {
	return IsInt(i) || IsInt8(i) || IsInt16(i) || IsInt32(i) || IsInt64(i)
}

// IsIntFloat64 checks if a float64 is a whole integer, important when comparing data from JSON Unmarshal that's always float64 if an interface
func IsIntFloat64(i any) bool {
	f, ok := i.(float64)
	if !ok {
		return false
	}

	return f == float64(int(f))
}

// IsInt determines if i is a int
func IsInt(i any) bool {
	return reflect.ValueOf(i).Kind() == reflect.Int
}

// IsInt8 determines if i is a int8
func IsInt8(i any) bool {
	return reflect.ValueOf(i).Kind() == reflect.Int8
}

// IsInt16 determines if i is a int16
func IsInt16(i any) bool {
	return reflect.ValueOf(i).Kind() == reflect.Int16
}

// IsInt32 determines if i is a int32
func IsInt32(i any) bool {
	return reflect.ValueOf(i).Kind() == reflect.Int32
}

// IsInt64 determines if i is a int64
func IsInt64(i any) bool {
	return reflect.ValueOf(i).Kind() == reflect.Int64
}
