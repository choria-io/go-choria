// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// EachFile calls cb with a path to every found DD of type kind, stops looking when br is true
func EachFile(kind string, libdirs []string, cb func(name string, path string) (br bool)) {
	for _, dir := range libdirs {
		for _, n := range []string{"choria", "mcollective"} {
			filepath.Walk(filepath.Join(dir, n, kind), func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if info.IsDir() {
					return nil
				}

				_, name := filepath.Split(path)
				extension := filepath.Ext(name)

				if extension != ".json" {
					return nil
				}

				cb(strings.TrimSuffix(name, extension), path)

				return nil
			})
		}
	}
}

// ValToDDLType converts val into the type described in typedef where typedef is a typical choria DDL supported type
func ValToDDLType(typedef string, val string) (res any, err error) {
	switch strings.ToLower(typedef) {
	case "integer":
		i, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("'%s' is not a valid integer", val)
		}

		return int64(i), nil

	case "float", "number":
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("'%s' is not a valid float", val)
		}

		return f, nil

	case "string", "list":
		return val, nil

	case "boolean":
		b, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("'%s' is not a valid boolean", val)
		}

		return b, nil

	case "hash":
		res := map[string]any{}
		err := json.Unmarshal([]byte(val), &res)
		if err != nil {
			return nil, fmt.Errorf("'%s' is not a valid JSON string with a hash inside", val)
		}

		return res, nil

	case "array":
		res := []any{}
		err := json.Unmarshal([]byte(val), &res)
		if err != nil {
			return nil, fmt.Errorf("'%s' is not a valid JSON string with an array inside", val)
		}

		return res, nil

	}

	return nil, fmt.Errorf("unsupported type '%s'", typedef)
}
