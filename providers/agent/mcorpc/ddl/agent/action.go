package agent

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/internal/fs"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/common"
)

// Action describes an individual action in an agent
type Action struct {
	Name        string                        `json:"action"`
	Input       map[string]*common.InputItem  `json:"input"`
	Output      map[string]*common.OutputItem `json:"output"`
	Display     string                        `json:"display"`
	Description string                        `json:"description"`
	Aggregation []ActionAggregateItem         `json:"aggregate,omitempty"`

	agg *actionAggregators

	sync.Mutex
}

func (a *Action) RenderConsole() ([]byte, error) {
	return fs.ExecuteTemplate("ddl/console/action.templ", a, nil)
}

func (a *Action) RenderMarkdown() ([]byte, error) {
	return fs.ExecuteTemplate("ddl/markdown/action.templ", a, nil)
}

// GetInput gets a named input
func (a *Action) GetInput(i string) (*common.InputItem, bool) {
	input, ok := a.Input[i]
	return input, ok
}

// GetOutput gets a named output
func (a *Action) GetOutput(o string) (*common.OutputItem, bool) {
	output, ok := a.Output[o]
	return output, ok
}

// DisplayMode is the configured display mode
func (a *Action) DisplayMode() string {
	return a.Display
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
	names = []string{}

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

	return i.Required()
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

		converted, w, err := input.ValidateStringValue(v)
		warnings = append(warnings, w...)
		if err != nil {
			return result, warnings, fmt.Errorf("invalid value for '%s': %s", kname, err)
		}

		result[kname] = converted
	}

	for _, iname := range a.InputNames() {
		input := a.Input[iname]

		_, ok := result[iname]
		if !ok {
			if input.Required() && input.Default == nil {
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

	// We currently ignore the process_results flag that may be set by the MCO RPC CLI
	delete(data, "process_results")

	for _, input := range validNames {
		val, ok := data[input]

		// didnt get a input but needs it
		if !ok && a.RequiresInput(input) {
			return []string{}, fmt.Errorf("input '%s' is required", input)
		}

		// didnt get a input and dont need it so nothing to do
		if !ok {
			continue
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

		return warnings, fmt.Errorf("request contains an input '%s' that is not declared in the DDL. Valid inputs are: %s", iname, strings.Join(validNames, ", "))
	}

	return []string{}, err
}

// ValidateInputString attempts to convert a string to the correct type and validate it based on the DDL spec
func (a *Action) ValidateInputString(input string, val string) (warnings []string, err error) {
	i, ok := a.Input[input]
	if !ok {
		return warnings, fmt.Errorf("unknown input '%s'", input)
	}

	_, warnings, err = i.ValidateStringValue(val)
	return warnings, err
}

// ValidateInputValue validates the input matches requirements in the DDL
func (a *Action) ValidateInputValue(input string, val interface{}) (warnings []string, err error) {
	warnings = []string{}

	i, ok := a.Input[input]
	if !ok {
		return warnings, fmt.Errorf("unknown input '%s'", input)
	}

	return i.ValidateValue(val)
}
