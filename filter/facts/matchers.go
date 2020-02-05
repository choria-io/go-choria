package facts

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

func eqMatch(fact gjson.Result, value string) (bool, error) {
	switch fact.Type {
	case gjson.String:
		return fact.String() == value, nil

	case gjson.Number:
		if strings.Contains(value, ".") {
			v, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return false, err
			}

			return fact.Float() == v, nil
		}

		return strconv.Itoa(int(fact.Int())) == value, nil

	case gjson.True:
		return truthy(value), nil

	case gjson.False:
		return falsey(value), nil

	case gjson.Null:
		return false, nil

	case gjson.JSON:
		return false, nil

	default:
		return false, fmt.Errorf("do not know how to evaluate data of type %s", fact.Type)
	}
}

func reMatch(fact gjson.Result, value string) (bool, error) {
	switch fact.Type {
	case gjson.String:
		return regexMatch(fact.String(), value)

	case gjson.Number:
		if strings.Contains(value, ".") {
			return regexMatch(fmt.Sprintf("%.4f", fact.Float()), value)
		}

		return regexMatch(strconv.Itoa(int(fact.Int())), value)

	case gjson.True:
		return truthy(strings.ToLower(value)), nil

	case gjson.False:
		return falsey(strings.ToLower(value)), nil

	case gjson.Null:
		return false, nil

	case gjson.JSON:
		return false, nil

	default:
		return false, fmt.Errorf("do not know how to evaluate data of type %s", fact.Type)
	}
}

func leMatch(fact gjson.Result, value string) (bool, error) {
	switch fact.Type {
	case gjson.String:
		return fact.String() <= value, nil

	case gjson.Number:
		if strings.Contains(value, ".") {
			v, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return false, err
			}

			return fact.Float() <= v, nil
		}

		v, err := strconv.Atoi(value)
		if err != nil {
			return false, err
		}

		return int(fact.Int()) <= v, nil

	default:
		return false, fmt.Errorf("do not know how to evaluate data of type %s", fact.Type)
	}
}

func geMatch(fact gjson.Result, value string) (bool, error) {
	switch fact.Type {
	case gjson.String:
		return fact.String() >= value, nil

	case gjson.Number:
		if strings.Contains(value, ".") {
			v, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return false, err
			}

			return fact.Float() >= v, nil
		}

		v, err := strconv.Atoi(value)
		if err != nil {
			return false, err
		}

		return int(fact.Int()) >= v, nil

	default:
		return false, fmt.Errorf("do not know how to evaluate data of type %s", fact.Type)
	}
}

func ltMatch(fact gjson.Result, value string) (bool, error) {
	switch fact.Type {
	case gjson.String:
		return fact.String() < value, nil

	case gjson.Number:
		if strings.Contains(value, ".") {
			v, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return false, err
			}

			return fact.Float() < v, nil
		}

		v, err := strconv.Atoi(value)
		if err != nil {
			return false, err
		}

		return int(fact.Int()) < v, nil

	default:
		return false, fmt.Errorf("do not know how to evaluate data of type %s", fact.Type)
	}
}

func gtMatch(fact gjson.Result, value string) (bool, error) {
	switch fact.Type {
	case gjson.String:
		return fact.String() > value, nil

	case gjson.Number:
		if strings.Contains(value, ".") {
			f, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return false, err
			}

			return fact.Float() >= f, nil
		}

		v, err := strconv.Atoi(value)
		if err != nil {
			return false, err
		}

		return int(fact.Int()) > v, nil

	default:
		return false, fmt.Errorf("do not know how to evaluate data of type %s", fact.Type)
	}
}

func neMatch(fact gjson.Result, value string) (bool, error) {
	switch fact.Type {
	case gjson.String:
		return fact.String() != value, nil

	case gjson.Number:
		if strings.Contains(value, ".") {
			f, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return false, err
			}

			return f != fact.Float(), nil
		}

		return strconv.Itoa(int(fact.Int())) != value, nil

	case gjson.True:
		return !truthy(strings.ToLower(value)), nil

	case gjson.False:
		return falsey(strings.ToLower(value)), nil

	case gjson.Null:
		return false, nil

	case gjson.JSON:
		return false, nil

	default:
		return false, fmt.Errorf("do not know how to evaluate data of type %s", fact.Type)
	}
}

func regexMatch(value string, pattern string) (bool, error) {
	pattern = strings.TrimLeft(pattern, "/")
	pattern = strings.TrimRight(pattern, "/")

	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, err
	}

	return re.MatchString(value), nil
}

func truthy(value string) bool {
	b, err := strconv.ParseBool(value)

	if err == nil && b {
		return true
	}

	return false
}

func falsey(value string) bool {
	return !truthy(value)
}
