package agent

import (
	"encoding/json"
	"sync"

	"github.com/choria-io/mcorpc-agent-provider/mcorpc/aggregate"
)

// ActionAggregateItem describes a aggregate function to summarize data
type ActionAggregateItem struct {
	Function  string          `json:"function"`
	Arguments json.RawMessage `json:"args"`
}

type actionAggregators struct {
	aggregators map[string]aggregate.Aggregator
	action      *Action

	sync.Mutex
}

// OutputName is the name of the output being aggregated
func (a *ActionAggregateItem) OutputName() string {
	out := []interface{}{}
	err := json.Unmarshal(a.Arguments, &out)
	if err != nil || len(out) < 1 {
		return "unknown"
	}

	output, ok := out[0].(string)
	if !ok {
		return "unknown"
	}

	return output
}

func newActionAggregators(a *Action) *actionAggregators {
	agg := &actionAggregators{
		action:      a,
		aggregators: make(map[string]aggregate.Aggregator),
	}

	// deliberately failing silently here, this is typically called while
	// already processing results, we should do our best to process all results
	// where possible and summaries are kind of optional, so we do not fail if
	// there is a structural error in the spec
	for _, spec := range a.Aggregation {
		var args []interface{}

		err := json.Unmarshal(spec.Arguments, &args)
		if err != nil {
			continue
		}

		key, ok := args[0].(string)
		if !ok {
			continue
		}

		instance, err := aggregate.AggregatorByType(spec.Function, args)
		if err != nil {
			continue
		}

		agg.aggregators[key] = instance
	}

	return agg
}

func (a *actionAggregators) aggregateItem(item string, val interface{}) {
	a.Lock()
	defer a.Unlock()

	instance, ok := a.aggregators[item]
	if ok {
		instance.ProcessValue(val)
	}
}

func (a *actionAggregators) resultStringsFormatted() map[string][]string {
	a.Lock()
	defer a.Unlock()

	res := make(map[string][]string)

	for k, agg := range a.aggregators {
		str, err := agg.ResultFormattedStrings("")
		if err != nil {
			res[k] = []string{err.Error()}
			continue
		}

		res[k] = str
	}

	return res
}

func (a *actionAggregators) resultJSON() []byte {
	a.Lock()
	defer a.Unlock()

	res := make(map[string]json.RawMessage)

	for k, agg := range a.aggregators {
		j, err := agg.ResultJSON()
		if err != nil {
			continue
		}

		res[k] = j
	}

	j, _ := json.Marshal(res)
	return j
}
func (a *actionAggregators) resultStrings() map[string]map[string]string {
	a.Lock()
	defer a.Unlock()

	res := make(map[string]map[string]string)

	for k, agg := range a.aggregators {
		str, err := agg.ResultStrings()
		if err != nil {
			res[k] = map[string]string{
				"error": err.Error(),
			}

			continue
		}

		res[k] = str
	}

	return res
}
