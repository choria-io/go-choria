package facts

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/ghodss/yaml"
)

// Logger provides logging facilities
type Logger interface {
	Warnf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// MatchFile match fact filters in a OR manner, only nodes that have all the matching facts will be true here
func MatchFile(filters [][3]string, file string, log Logger) bool {
	matched := false
	var err error

	for _, filter := range filters {
		matched, err = HasFact(filter[0], filter[1], filter[2], file, log)
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

// JSON parses the data, including doing any conversions needed, and returns JSON text
func JSON(file string, log Logger) (json.RawMessage, error) {
	out := make(map[string]interface{})

	for _, f := range strings.Split(file, string(os.PathListSeparator)) {
		if f == "" {
			continue
		}

		if _, err := os.Stat(f); os.IsNotExist(err) {
			log.Warnf("Fact file %s does not exist", f)
			continue
		}

		j, err := ioutil.ReadFile(f)
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

		facts := make(map[string]interface{})
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

	found := gjson.GetBytes(j, fact)
	if !found.Exists() {
		return nil, gjson.Result{}, nil
	}

	return j, found, nil
}

// HasFact evaluates the expression against facts in the file
func HasFact(fact string, operator string, value string, file string, log Logger) (bool, error) {
	j, found, err := GetFact(fact, file, log)
	if err != nil {
		return false, err
	}

	if !found.Exists() {
		return false, nil
	}

	switch operator {
	case "==":
		return eqMatch(found, value, &j)
	case "=~":
		return reMatch(found, value, &j)
	case "<=":
		return leMatch(found, value, &j)
	case ">=":
		return geMatch(found, value, &j)
	case "<":
		return ltMatch(found, value, &j)
	case ">":
		return gtMatch(found, value, &j)
	case "!=":
		return neMatch(found, value, &j)
	default:
		return false, fmt.Errorf("unknown fact matching operator %s while looking for fact %s", operator, fact)
	}
}
