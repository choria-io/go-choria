package agent

import (
	"encoding/json"
	"fmt"
	"sync"
)

// Action describes an individual action in an agent
type Action struct {
	Name        string                       `json:"action"`
	Input       json.RawMessage              `json:"input"`
	Output      map[string]*ActionOutputItem `json:"output"`
	Display     string                       `json:"display"`
	Description string                       `json:"description"`
	Aggregation []ActionAggregateItem        `json:"aggregate"`

	agg *actionAggregators

	sync.Mutex
}

// ActionOutputItem describes an individual output item
type ActionOutputItem struct {
	Description string      `json:"description"`
	DisplayAs   string      `json:"display_as"`
	Default     interface{} `json:"default"`
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
