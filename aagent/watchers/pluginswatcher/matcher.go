// Copyright (c) 2021-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package pluginswatcher

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/tidwall/gjson"

	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/expr-lang/expr"
)

type logger interface {
	Errorf(string, ...any)
}

func identityMatchFunc(id string, log logger) func(string) bool {
	return func(re string) bool {
		r, err := regexp.Compile(re)
		if err != nil {
			log.Errorf("Could not process identity match: %s: %s", re, err)
			return false
		}

		return r.MatchString(id)
	}
}

func IsNodeMatch(facts json.RawMessage, identity string, matcher string, log logger) (bool, error) {
	if matcher == "" {
		return true, nil
	}

	env := map[string]any{
		"identity":      identityMatchFunc(identity, log),
		"get_fact":      func(query string) any { return gjson.GetBytes(facts, query).Value() },
		"has_file":      iu.FileExist,
		"has_directory": iu.FileIsDir,
		"has_command":   iu.IsExecutableInPath,
	}

	execEnv := expr.Env(env)
	prog, err := expr.Compile(matcher, execEnv, expr.AsBool())
	if err != nil {
		return false, err
	}

	res, err := expr.Run(prog, env)
	if err != nil {
		return false, err
	}

	b, ok := res.(bool)
	if !ok {
		return false, fmt.Errorf("match was non boolean")
	}

	return b, nil
}
