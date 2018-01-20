package facts

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/choria-io/go-choria/choria"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"

	"github.com/ghodss/yaml"
)

// Match match fact filters in a OR manner, only nodes that have all
// the matching facts will be true here
func Match(filters [][3]string, fw *choria.Framework, log *logrus.Entry) bool {
	matched := false
	var err error

	for _, filter := range filters {
		matched, err = HasFact(filter[0], filter[1], filter[2], fw.Config.FactSourceFile)
		if err != nil {
			log.Warnf("Failed to match fact '%#v': %s", filter, err.Error())
			return false
		}

		if matched == false {
			log.Debug("Failed to match fact filter '%#v'", filter)
			break
		}
	}

	return matched
}

// JSON parses the data, including doing any conversions needed, and returns JSON text
func JSON(file string) (json.RawMessage, error) {
	if file == "" {
		return json.RawMessage("{}"), fmt.Errorf("Cannot do fact discovery there is no file configured")
	}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return json.RawMessage("{}"), fmt.Errorf("Cannot do fact discovery the file '%s' does not exist", file)
	}

	j, err := ioutil.ReadFile(file)
	if err != nil {
		return json.RawMessage("{}"), fmt.Errorf("Could not read facts file %s: %s", file, err.Error())
	}

	if strings.HasSuffix(file, "yaml") {
		j, err = yaml.YAMLToJSON(j)
		if err != nil {
			return json.RawMessage("{}"), fmt.Errorf("Could not parse facts file %s as YAML: %s", file, err.Error())
		}
	}

	return json.RawMessage(j), nil
}

// GetFact looks up a single fact from the facts file, errors reading
// the file is reported but an absent fact is handled as empty result
// and no error
func GetFact(fact string, file string) ([]byte, gjson.Result, error) {
	j, err := JSON(file)
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
func HasFact(fact string, operator string, value string, file string) (bool, error) {
	j, found, err := GetFact(fact, file)
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
		return false, fmt.Errorf("Unknown fact matching operator %s while looking for fact %s", operator, fact)
	}
}
