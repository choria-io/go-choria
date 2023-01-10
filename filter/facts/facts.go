// Copyright (c) 2019-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package facts

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/ghodss/yaml"

	"github.com/choria-io/go-choria/internal/util"
)

var validOperators = regexp.MustCompile(`<=|>=|=>|=<|<|>|!=|=~|={1,2}`)

// Logger provides logging facilities
type Logger interface {
	Warnf(format string, args ...any)
	Debugf(format string, args ...any)
	Errorf(format string, args ...any)
}

// MatchFacts match fact filters in a OR manner, only facts matching all filters will be true
func MatchFacts(filters [][3]string, facts json.RawMessage, log Logger) bool {
	matched := false
	var err error

	for _, filter := range filters {
		matched, err = HasFactJSON(filter[0], filter[1], filter[2], facts, log)
		if err != nil {
			log.Warnf("Failed to match fact '%#v': %s", filter, err)
			return false
		}

		if !matched {
			log.Debugf("Failed to match fact filter '%#v'", filter)
			break
		}
	}

	return matched
}

// MatchFile match fact filters in a OR manner, only nodes that have all the matching facts will be true here
func MatchFile(filters [][3]string, file string, log Logger) bool {
	facts, err := JSON(file, log)
	if err != nil {
		log.Warnf("Failed to match fact: '%#v': %s", filters, err)
		return false
	}

	return MatchFacts(filters, facts, log)
}

// JSON parses the data, including doing any conversions needed, and returns JSON text
func JSON(file string, log Logger) (json.RawMessage, error) {
	out := make(map[string]any)

	for _, f := range strings.Split(file, string(os.PathListSeparator)) {
		if f == "" {
			continue
		}

		if !util.FileExist(f) {
			log.Warnf("Fact file %s does not exist", f)
			continue
		}

		j, err := os.ReadFile(f)
		if err != nil {
			log.Errorf("Could not read fact file %s: %s", f, err)
			continue
		}

		if strings.HasSuffix(f, "yaml") {
			j, err = yaml.YAMLToJSON(j)
			if err != nil {
				log.Errorf("Could not parse facts file %s as YAML: %s", file, err)
				continue
			}
		}

		facts := make(map[string]any)
		err = json.Unmarshal(j, &facts)
		if err != nil {
			log.Errorf("Could not parse facts file: %s", err)
			continue
		}

		// does a very dumb shallow merge that mimics ruby Hash#merge
		// to maintain mcollective compatibility
		for k, v := range facts {
			out[k] = v
		}
	}

	if len(out) == 0 {
		return json.RawMessage("{}"), fmt.Errorf("no facts were found in %s", file)
	}

	j, err := json.Marshal(&out)
	if err != nil {
		return json.RawMessage("{}"), fmt.Errorf("could not JSON marshal merged facts: %s", err)
	}

	return json.RawMessage(j), nil
}

// GetFact looks up a single fact from the facts file, errors reading
// the file is reported but an absent fact is handled as empty result
// and no error
func GetFact(fact string, file string, log Logger) ([]byte, gjson.Result, error) {
	j, err := JSON(file, log)
	if err != nil {
		return nil, gjson.Result{}, err
	}

	found, err := GetFactJSON(fact, j)
	return j, found, err
}

// GetFactJSON looks up a single fact from the JSON data, absent fact is handled as empty
// result and no error
func GetFactJSON(fact string, facts json.RawMessage) (gjson.Result, error) {
	result := gjson.GetBytes(facts, fact)

	return result, nil
}

func HasFactJSON(fact string, operator string, value string, facts json.RawMessage, log Logger) (bool, error) {
	result, err := GetFactJSON(fact, facts)
	if err != nil {
		return false, err
	}

	if !result.Exists() {
		return false, nil
	}

	switch operator {
	case "==":
		return eqMatch(result, value)
	case "=~":
		return reMatch(result, value)
	case "<=":
		return leMatch(result, value)
	case ">=":
		return geMatch(result, value)
	case "<":
		return ltMatch(result, value)
	case ">":
		return gtMatch(result, value)
	case "!=":
		return neMatch(result, value)
	default:
		return false, fmt.Errorf("unknown fact matching operator %s while looking for fact %s", operator, fact)
	}
}

// HasFact evaluates the expression against facts in the file
func HasFact(fact string, operator string, value string, file string, log Logger) (bool, error) {
	j, err := JSON(file, log)
	if err != nil {
		return false, err
	}

	return HasFactJSON(fact, operator, value, j, log)
}

// ParseFactFilterString parses a fact filter string as typically typed on the CLI
func ParseFactFilterString(f string) ([3]string, error) {
	operatorIndexes := validOperators.FindAllStringIndex(f, -1)
	var mainOpIndex []int

	if opCount := len(operatorIndexes); opCount > 1 {
		// This is a special case where the left operand contains a valid operator.
		// We skip over everything and use the right most operator.
		mainOpIndex = operatorIndexes[len(operatorIndexes)-1]
	} else if opCount == 1 {
		mainOpIndex = operatorIndexes[0]
	} else {
		return [3]string{}, fmt.Errorf("could not parse fact %s it does not appear to be in a valid format", f)
	}

	op := f[mainOpIndex[0]:mainOpIndex[1]]
	leftOp := strings.TrimSpace(f[:mainOpIndex[0]])
	rightOp := strings.TrimSpace(f[mainOpIndex[1]:])

	// validate that the left and right operands are both valid
	if len(leftOp) == 0 || len(rightOp) == 0 {
		return [3]string{}, fmt.Errorf("could not parse fact %s it does not appear to be in a valid format", f)
	}

	lStartString := string(leftOp[0])
	rEndString := string(rightOp[len(rightOp)-1])
	if validOperators.MatchString(lStartString) || validOperators.Match([]byte(rEndString)) {
		return [3]string{}, fmt.Errorf("could not parse fact %s it does not appear to be in a valid format", f)
	}

	// transform op and value for processing
	switch op {
	case "=":
		op = "=="
	case "=<":
		op = "<="
	case "=>":
		op = ">="
	}

	// finally check for old style regex fact matches
	if rightOp[0] == '/' && rightOp[len(rightOp)-1] == '/' {
		op = "=~"
	}

	return [3]string{leftOp, op, rightOp}, nil
}
