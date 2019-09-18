package agent

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/choria-io/go-validator/ipaddress"
	"github.com/choria-io/go-validator/ipv4"
	"github.com/choria-io/go-validator/ipv6"
	"github.com/choria-io/go-validator/regex"
	"github.com/choria-io/go-validator/shellsafe"
)

// Action describes an individual action in an agent
type Action struct {
	Name        string                       `json:"action"`
	Input       map[string]*ActionInputItem  `json:"input"`
	Output      map[string]*ActionOutputItem `json:"output"`
	Display     string                       `json:"display"`
	Description string                       `json:"description"`
	Aggregation []ActionAggregateItem        `json:"aggregate,omitempty"`

	agg *actionAggregators

	sync.Mutex
}

// ActionOutputItem describes an individual output item
type ActionOutputItem struct {
	Description string      `json:"description"`
	DisplayAs   string      `json:"display_as"`
	Default     interface{} `json:"default,omitempty"`
	Type        string      `json:"type,omitempty"`
}

// ActionInputItem describes an individual input item
type ActionInputItem struct {
	Prompt      string      `json:"prompt"`
	Description string      `json:"description"`
	Type        string      `json:"type"`
	Default     interface{} `json:"default,omitempty"`
	Optional    bool        `json:"optional"`
	Validation  string      `json:"validation,omitempty"`
	MaxLength   int         `json:"maxlength,omitempty"`
	Enum        []string    `json:"list,omitempty"`
}

// AggregateResultJSON receives a JSON reply and aggregate all the data found in it
func (a *Action) AggregateResultJSON(jres []byte) error {
	res := make(map[string]interface{})

	err := json.Unmarshal(jres, &res)
	if err != nil {
		return fmt.Errorf("could not parse result as JSON data: %s", err)
	}

	return a.AggregateResult(res)
}

// AggregateResult receives a result and aggregate all the data found in it, most
// errors are squashed since aggregation are called during processing of replies
// and we do not want to fail a reply just because aggregation failed, thus this
// is basically a best efforts kind of thing on purpose
func (a *Action) AggregateResult(result map[string]interface{}) error {
	a.Lock()
	defer a.Unlock()

	if a.agg == nil {
		a.agg = newActionAggregators(a)
	}

	for k, v := range result {
		a.agg.aggregateItem(k, v)
	}

	return nil
}

// AggregateSummaryJSON produce a JSON representation of aggregate results for every output
// item that has a aggregate summary defined
func (a *Action) AggregateSummaryJSON() ([]byte, error) {
	a.Lock()
	defer a.Unlock()

	if a.agg == nil {
		a.agg = newActionAggregators(a)
	}

	return a.agg.action.agg.resultJSON(), nil
}

// AggregateSummaryStrings produce a map of results for every output item that
// has a aggregate summary defined
func (a *Action) AggregateSummaryStrings() (map[string]map[string]string, error) {
	a.Lock()
	defer a.Unlock()

	if a.agg == nil {
		a.agg = newActionAggregators(a)
	}

	return a.agg.resultStrings(), nil
}

// AggregateSummaryFormattedStrings produce a formatted string for each output
// item that has a aggregate summary defined
func (a *Action) AggregateSummaryFormattedStrings() (map[string][]string, error) {
	a.Lock()
	defer a.Unlock()

	if a.agg == nil {
		a.agg = newActionAggregators(a)
	}

	return a.agg.resultStringsFormatted(), nil
}

// InputNames retrieves all valid input names
func (a *Action) InputNames() (names []string) {
	for k := range a.Input {
		names = append(names, k)
	}

	sort.Strings(names)

	return names
}

// OutputNames retrieves all valid output names
func (a *Action) OutputNames() (names []string) {
	for k := range a.Output {
		names = append(names, k)
	}

	sort.Strings(names)

	return names
}

// SetOutputDefaults adds items to results that have defaults declared in the DDL but not found in the result
func (a *Action) SetOutputDefaults(results map[string]interface{}) {
	for _, k := range a.OutputNames() {
		_, ok := results[k]
		if ok {
			continue
		}

		if a.Output[k].Default != nil {
			results[k] = a.Output[k].Default
		}
	}
}

// RequiresInput reports if an input is required
func (a *Action) RequiresInput(input string) bool {
	i, ok := a.Input[input]
	if !ok {
		return false
	}

	return !i.Optional
}

// ValidateAndConvertToDDLTypes takes a map of strings like you might receive from the CLI, convert each
// item to the correct type according to the DDL type hints associated with inputs, validates its valid
// according to the DDL hints and returns a map of interface{} ready for conversion to JSON that would
// then have the correct types
func (a *Action) ValidateAndConvertToDDLTypes(args map[string]string) (result map[string]interface{}, warnings []string, err error) {
	result = make(map[string]interface{})
	warnings = []string{}

	for k, v := range args {
		kname := strings.ToLower(k)
		input, ok := a.Input[kname]
		if !ok {
			// ruby rpc was forgiving about this, but its time really
			return result, warnings, fmt.Errorf("input '%s' has not been declared", kname)
		}

		converted, err := ValToDDLType(input.Type, v)
		if err != nil {
			return result, warnings, fmt.Errorf("invalid value for '%s': %s", kname, err)
		}

		w, err := a.ValidateInputValue(kname, converted)
		for _, i := range w {
			warnings = append(warnings, i)
		}

		if err != nil {
			return result, warnings, fmt.Errorf("invalid value for '%s': %s", kname, err)
		}

		result[kname] = converted
	}

	for _, iname := range a.InputNames() {
		input := a.Input[iname]

		_, ok := result[iname]
		if !ok {
			if !input.Optional && input.Default == nil {
				return result, warnings, fmt.Errorf("input '%s' is required", iname)
			}

			if input.Default != nil {
				result[iname] = input.Default
			}
		}
	}

	return result, warnings, nil
}

// ValidateRequestJSON receives request data in JSON format and validates it against the DDL
func (a *Action) ValidateRequestJSON(req json.RawMessage) (warnings []string, err error) {
	reqdata := make(map[string]interface{})
	err = json.Unmarshal(req, &reqdata)
	if err != nil {
		return []string{}, err
	}

	return a.ValidateRequestData(reqdata)
}

// ValidateRequestData validates request data against the DDL
func (a *Action) ValidateRequestData(data map[string]interface{}) (warnings []string, err error) {

	validNames := a.InputNames()

	for _, input := range validNames {
		val, ok := data[input]

		if a.RequiresInput(input) && !ok {
			return []string{}, fmt.Errorf("input '%s' is required", input)
		}

		warnings, err = a.ValidateInputValue(input, val)
		if err != nil {
			return warnings, fmt.Errorf("validation failed for input '%s': %s", input, err)
		}
	}

	if len(validNames) == 0 && len(data) > 0 {
		return warnings, fmt.Errorf("request contains inputs while none are declared in the DDL")
	}

	for iname := range data {
		if iname == "process_results" {
			continue
		}

		matched := false
		for _, vname := range validNames {
			if vname == iname {
				matched = true
				continue
			}
		}

		if matched {
			continue
		}

		return warnings, fmt.Errorf("request contains na input '%s' that is not declared in the DDL (%v)", iname, validNames)
	}

	return []string{}, err
}

// ValidateInputString attempts to convert a string to the correct type and validate it based on the DDL spec
func (a *Action) ValidateInputString(input string, val string) error {
	i, ok := a.Input[input]
	if !ok {
		return fmt.Errorf("unknown input '%s'", input)
	}

	converted, err := ValToDDLType(i.Type, val)
	if err != nil {
		return err
	}

	_, err = a.ValidateInputValue(input, converted)
	if err != nil {
		return err
	}

	return nil
}

// ValidateInputValue validates the input matches requirements in the DDL
func (a *Action) ValidateInputValue(input string, val interface{}) (warnings []string, err error) {
	warnings = []string{}

	i, ok := a.Input[input]
	if !ok {
		return warnings, fmt.Errorf("unknown input '%s'", input)
	}

	switch strings.ToLower(i.Type) {
	case "integer":
		if !isAnyInt(val) {
			return warnings, fmt.Errorf("is not an integer")
		}

	case "number":
		if !isNumber(val) {
			return warnings, fmt.Errorf("is not a number")
		}

	case "float":
		if !isFloat64(val) {
			return warnings, fmt.Errorf("is not a float")
		}

	case "string":
		if !isString(val) {
			return warnings, fmt.Errorf("is not a string")
		}

		if i.MaxLength == 0 {
			return warnings, nil
		}

		sval := val.(string)
		if len(sval) > i.MaxLength {
			return warnings, fmt.Errorf("is longer than %d characters", i.MaxLength)
		}

		if i.Validation != "" {
			w, err := validateStringValidation(i.Validation, sval)

			for _, i := range w {
				warnings = append(warnings, i)
			}

			if err != nil {
				return warnings, err
			}
		}

	case "boolean":
		if !isBool(val) {
			return warnings, fmt.Errorf("is not a boolean")
		}

	case "list":
		if len(i.Enum) == 0 {
			return warnings, fmt.Errorf("input type of list without a valid list of items in DDL")
		}

		valstr, ok := val.(string)
		if !ok {
			return warnings, fmt.Errorf("should be a string")
		}

		for _, valid := range i.Enum {
			if valid == valstr {
				return warnings, nil
			}
		}

		return warnings, fmt.Errorf("should be one of %s", strings.Join(i.Enum, ", "))

	case "hash":
		if !isHash(val) {
			return warnings, fmt.Errorf("is not a hash map")
		}

	case "array":
		if !isArray(val) {
			return warnings, fmt.Errorf("is not an array")
		}

	default:
		return warnings, fmt.Errorf("unsupported input type '%s'", i.Type)
	}

	return warnings, nil
}

func validateStringValidation(validation string, value string) (warnings []string, err error) {
	warnings = []string{}

	switch validation {
	case "shellsafe":
		_, err = shellsafe.Validate(value)
		return warnings, err

	case "ipv4address":
		_, err := ipv4.ValidateString(value)
		return warnings, err

	case "ipv6address":
		_, err := ipv6.ValidateString(value)
		return warnings, err

	case "ipaddress":
		_, err := ipaddress.ValidateString(value)
		return warnings, err
	}

	namedValidator, err := regexp.MatchString("^[a-z]", validation)
	if namedValidator || err != nil {
		return []string{fmt.Sprintf("Unsupported validator '%s'", validation)}, nil
	}

	_, err = regex.ValidateString(value, validation)
	return warnings, err
}

// ValToDDLType converts val into the type described in typedef where typedef is a typical choria DDL supported type
func ValToDDLType(typedef string, val string) (res interface{}, err error) {
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
		res := map[string]interface{}{}
		err := json.Unmarshal([]byte(val), &res)
		if err != nil {
			return nil, fmt.Errorf("'%s' is not a valid JSON string with a hash inside", val)
		}

		return res, nil

	case "array":
		res := []interface{}{}
		err := json.Unmarshal([]byte(val), &res)
		if err != nil {
			return nil, fmt.Errorf("'%s' is not a valid JSON string with an array inside", val)
		}

		return res, nil

	}

	return nil, fmt.Errorf("unsupported type '%s'", typedef)
}

func isHash(i interface{}) bool {
	return reflect.ValueOf(i).Kind() == reflect.Map
}

func isArray(i interface{}) bool {
	kind := reflect.ValueOf(i).Kind()
	return kind == reflect.Array || kind == reflect.Slice
}

func isBool(i interface{}) bool {
	return reflect.ValueOf(i).Kind() == reflect.Bool
}

func isString(i interface{}) bool {
	return reflect.ValueOf(i).Kind() == reflect.String
}

func isNumber(i interface{}) bool {
	return isAnyInt(i) || isAnyFloat(i)
}

func isAnyFloat(i interface{}) bool {
	return isFloat32(i) || isFloat64(i)
}

func isFloat32(i interface{}) bool {
	return reflect.ValueOf(i).Kind() == reflect.Float32
}

func isFloat64(i interface{}) bool {
	return reflect.ValueOf(i).Kind() == reflect.Float64
}

func isAnyInt(i interface{}) bool {
	return isInt(i) || isInt8(i) || isInt16(i) || isInt32(i) || isInt64(i)
}

func isInt(i interface{}) bool {
	return reflect.ValueOf(i).Kind() == reflect.Int
}

func isInt8(i interface{}) bool {
	return reflect.ValueOf(i).Kind() == reflect.Int8
}

func isInt16(i interface{}) bool {
	return reflect.ValueOf(i).Kind() == reflect.Int16
}

func isInt32(i interface{}) bool {
	return reflect.ValueOf(i).Kind() == reflect.Int32
}

func isInt64(i interface{}) bool {
	return reflect.ValueOf(i).Kind() == reflect.Int64
}
