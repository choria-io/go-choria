package aggregate

import (
	"encoding/json"
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

// ResultStrings returns a map of results in string format
func (s *SummaryAggregator) ResultStrings() (map[string]string, error) {
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

// ResultJSON return the results in JSON format preserving types
func (s *SummaryAggregator) ResultJSON() ([]byte, error) {
	s.Lock()
	defer s.Unlock()

	result := map[string]int{}
	for k, v := range s.items {
		result[fmt.Sprintf("%v", k)] = v
	}

	return json.Marshal(result)
}

// ResultFormattedStrings return the results in a formatted way, if no format is given a calculated value is used
func (s *SummaryAggregator) ResultFormattedStrings(format string) ([]string, error) {
	s.Lock()
	defer s.Unlock()

	output := []string{}

	if len(s.items) == 0 {
		return output, nil
	}

	type kv struct {
		Key string
		Val int
	}

	var sortable []kv
	for k, v := range s.items {
		sortable = append(sortable, kv{fmt.Sprintf("%v", k), v})
	}

	max := 0
	for _, k := range sortable {
		l := len(k.Key)
		if l > max {
			max = l
		}
	}

	if format == "" {
		format = fmt.Sprintf("%%%ds: %%d", max)
	}

	sort.Slice(sortable, func(i int, j int) bool {
		return sortable[i].Val > sortable[j].Val
	})

	for _, k := range sortable {
		output = append(output, fmt.Sprintf(format, k.Key, k.Val))
	}

	return output, nil
}
