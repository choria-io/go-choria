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

func (a *actionAggregators) resultStrings() map[string]map[string]string {
	a.Lock()
	defer a.Unlock()

	res := make(map[string]map[string]string)

	for k, agg := range a.aggregators {
		str, err := agg.StringResults()
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
