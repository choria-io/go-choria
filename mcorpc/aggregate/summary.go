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

type kv struct {
	Key interface{}
	Val string
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

func mapStringToKV(d map[string]string) []kv {
	var sortable []kv
	for k, v := range d {
		sortable = append(sortable, kv{k, v})
	}
	return sortable
}

// StringResults returns a map of results in string format
func (s *SummaryAggregator) StringResults() (map[string]string, error) {
	s.Lock()
	defer s.Unlock()

	result := map[string]string{}

	if len(s.items) == 0 {
		return result, nil
	}

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

	sortable := mapStringToKV(sresults)
	max := 0

	for k := range sresults {
		if len(k) > max {
			max = len(k)
		}
	}

	if format == "" {
		format = fmt.Sprintf("%%%ds: %%s", max)
	}

	sort.Slice(sortable, func(i int, j int) bool {
		return sortable[i].Val > sortable[j].Val
	})

	for _, k := range sortable {
		output = append(output, fmt.Sprintf(format, k.Key, k.Val))
	}

	return output, nil
}
