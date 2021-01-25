package compound

import (
	"encoding/json"

	"github.com/antonmedv/expr"
	"github.com/google/go-cmp/cmp"
	"github.com/tidwall/gjson"

	"github.com/choria-io/go-choria/filter/agents"
	"github.com/choria-io/go-choria/filter/classes"
	"github.com/choria-io/go-choria/filter/facts"
)

// Logger provides logging facilities
type Logger interface {
	Warnf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
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

	return MatchExprString(queries, f, c, knownAgents, log)
}

func MatchExprString(queries [][]map[string]string, facts json.RawMessage, classes []string, knownAgents []string, log Logger) bool {
	matched := 0
	failed := 0

	env := EmptyEnv()
	env["classes"] = classes
	env["agents"] = knownAgents
	env["facts"] = facts
	env["with"] = matchFunc(facts, classes, knownAgents, log)
	env["fact"] = factFunc(facts)
	env["include"] = includeFunc

	for _, cf := range queries {
		if len(cf) != 1 {
			return false
		}

		query, ok := cf[0]["expr"]
		if !ok {
			return false
		}

		prog, err := expr.Compile(query, expr.Env(env), expr.AsBool(), expr.AllowUndefinedVariables())
		if err != nil {
			log.Errorf("Could not compile compound query '%s': %s", query, err)
			failed++
			continue
		}

		res, err := expr.Run(prog, env)
		if err != nil {
			log.Errorf("Could not execute compound query: %s", err)
			failed++
			continue
		}

		b, ok := res.(bool)
		if !ok {
			log.Errorf("Compound query returned non boolean")
			failed++
			continue
		}

		if b {
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

func factFunc(facts json.RawMessage) func(string) interface{} {
	return func(query string) interface{} {
		return gjson.GetBytes(facts, query).Value()
	}
}

func includeFunc(hay []interface{}, needle interface{}) bool {
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

func EmptyEnv() map[string]interface{} {
	return map[string]interface{}{
		"agents":  []string{},
		"classes": []string{},
		"facts":   json.RawMessage{},
		"with":    func(_ string) bool { return false },
		"fact":    func(_ string) interface{} { return nil },
		"include": func(_ string) bool { return false },
	}
}
