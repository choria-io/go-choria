// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package compound

import (
	"encoding/json"
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/google/go-cmp/cmp"
	"github.com/tidwall/gjson"

	"github.com/choria-io/go-choria/filter/agents"
	"github.com/choria-io/go-choria/filter/classes"
	"github.com/choria-io/go-choria/filter/facts"
	"github.com/choria-io/go-choria/providers/data/ddl"
)

// Logger provides logging facilities
type Logger interface {
	Warnf(format string, args ...any)
	Debugf(format string, args ...any)
	Errorf(format string, args ...any)
}

func MatchExprStringFiles(queries [][]map[string]string, factFile string, classesFile string, knownAgents []string, log Logger) bool {
	c, err := classes.ReadClasses(classesFile)
	if err != nil {
		log.Errorf("cannot read classes file: %s", err)
		return false
	}

	f, err := facts.JSON(factFile, log)
	if err != nil {
		log.Errorf("cannot read facts file: %s", err)
		return false
	}

	return MatchExprString(queries, f, c, knownAgents, nil, log)
}

func CompileExprQuery(query string, df ddl.FuncMap) (*vm.Program, error) {
	return expr.Compile(query, expr.Env(EmptyEnv(df)), expr.AsBool(), expr.AllowUndefinedVariables())
}

func MatchExprProgram(prog *vm.Program, facts json.RawMessage, classes []string, knownAgents []string, df ddl.FuncMap, log Logger) (bool, error) {
	env := EmptyEnv(df)
	env["classes"] = classes
	env["agents"] = knownAgents
	env["facts"] = facts
	env["with"] = matchFunc(facts, classes, knownAgents, log)
	env["fact"] = factFunc(facts)
	env["include"] = includeFunc
	env["semver"] = semverFunc

	res, err := expr.Run(prog, env)
	if err != nil {
		return false, fmt.Errorf("could not execute compound query: %s", err)
	}

	b, ok := res.(bool)
	if !ok {
		return false, fmt.Errorf("compound query returned non boolean")
	}

	return b, nil
}

func MatchExprString(queries [][]map[string]string, facts json.RawMessage, classes []string, knownAgents []string, df ddl.FuncMap, log Logger) bool {
	matched := 0
	failed := 0

	for _, cf := range queries {
		if len(cf) != 1 {
			return false
		}

		query, ok := cf[0]["expr"]
		if !ok {
			return false
		}

		prog, err := CompileExprQuery(query, df)
		if err != nil {
			log.Errorf("Could not compile compound query '%s': %s", query, err)
			failed++
			continue
		}

		res, err := MatchExprProgram(prog, facts, classes, knownAgents, df, log)
		if err != nil {
			log.Errorf("Could not match compound query '%s': %s", query, err)
			failed++
			continue
		}

		if res {
			matched++
		} else {
			matched--
		}
	}

	return failed == 0 && matched > 0
}

func matchFunc(f json.RawMessage, c []string, a []string, log Logger) func(string) bool {
	return func(query string) bool {
		pf, err := facts.ParseFactFilterString(query)
		if err == nil {
			return facts.MatchFacts([][3]string{pf}, f, log)
		}

		if classes.Match([]string{query}, c) {
			return true
		}

		return agents.Match([]string{query}, a)
	}
}

func factFunc(facts json.RawMessage) func(string) any {
	return func(query string) any {
		return gjson.GetBytes(facts, query).Value()
	}
}

func semverFunc(value string, cmp string) (bool, error) {
	cons, err := semver.NewConstraint(cmp)
	if err != nil {
		return false, err
	}

	v, err := semver.NewVersion(value)
	if err != nil {
		return false, err
	}

	return cons.Check(v), nil
}

func includeFunc(hay []any, needle any) bool {
	// gjson always turns numbers into float64
	i, ok := needle.(int)
	if ok {
		needle = float64(i)
	}

	for _, i := range hay {
		if cmp.Equal(i, needle) {
			return true
		}
	}

	return false
}

func EmptyEnv(df ddl.FuncMap) map[string]any {
	env := map[string]any{
		"agents":  []string{},
		"classes": []string{},
		"facts":   json.RawMessage{},
		"with":    func(_ string) bool { return false },
		"fact":    func(_ string) any { return nil },
		"include": func(_ []any, _ any) bool { return false },
		"semver":  func(_ string, _ string) (bool, error) { return false, nil },
	}

	for k, v := range df {
		_, ok := env[k]
		if ok {
			continue
		}

		env[k] = v.F
	}

	return env
}
