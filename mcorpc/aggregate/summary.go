package aggregate

import (
	"fmt"
	"sort"
	"sync"
)

// SummaryAggregator keeps track of seen values and summarize how many times each were seen
type SummaryAggregator struct {
	items map[interface{}]int

	sync.Mutex
}

// NewSummaryAggregator creates a new SummaryAggregator with the specific options supplied
func NewSummaryAggregator(args []interface{}) (*SummaryAggregator, error) {
	agg := &SummaryAggregator{
		items: make(map[interface{}]int),
	}

	return agg, nil
}

// Type is the type of Aggregator
func (s *SummaryAggregator) Type() string {
	return "summary"
}

// ProcessValue processes and tracks a specified value
func (s *SummaryAggregator) ProcessValue(v interface{}) error {
	s.Lock()
	defer s.Unlock()

	_, ok := s.items[v]
	if !ok {
		s.items[v] = 0
	}

	s.items[v]++

	return nil
}

// StringResults returns a map of results in string format
func (s *SummaryAggregator) StringResults() (map[string]string, error) {
	s.Lock()
	defer s.Unlock()

	if len(s.items) == 0 {
		return map[string]string{}, nil
	}

	result := map[string]string{}

	for k, v := range s.items {
		result[fmt.Sprintf("%v", k)] = fmt.Sprintf("%d", v)
	}

	return result, nil
}

// FormattedStrings return the results in a formatted way, if no format is given a calculated value is used
func (s *SummaryAggregator) FormattedStrings(format string) ([]string, error) {
	output := []string{}

	sresults, err := s.StringResults()
	if err != nil {
		return output, err
	}

	keys := []string{}

	max := 0
	for k := range sresults {
		keys = append(keys, k)

		if len(k) > max {
			max = len(k)
		}
	}

	if format == "" {
		format = fmt.Sprintf("%%%ds: %%s", max)
	}

	sort.Strings(keys)

	for _, k := range keys {
		output = append(output, fmt.Sprintf(format, k, sresults[k]))
	}

	return output, nil
}
