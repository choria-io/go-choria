// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package aggregate

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

// SummaryAggregator keeps track of seen values and summarize how many times each were seen
type SummaryAggregator struct {
	items  map[interface{}]int
	args   []interface{}
	format string

	jsonTrue  string
	jsonFalse string

	mapping map[string]string

	sync.Mutex
}

// NewSummaryAggregator creates a new SummaryAggregator with the specific options supplied
func NewSummaryAggregator(args []interface{}) (*SummaryAggregator, error) {
	agg := &SummaryAggregator{
		items:   make(map[interface{}]int),
		args:    args,
		format:  parseFormatFromArgs(args),
		mapping: make(map[string]string),
	}

	s, _ := json.Marshal(true)
	agg.jsonTrue = string(s)
	s, _ = json.Marshal(false)
	agg.jsonFalse = string(s)

	err := agg.parseBoolMapsFromArgs()
	if err != nil {
		return nil, err
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

	item := v

	switch val := v.(type) {
	case bool:
		var tm string
		var ok bool

		if val {
			tm, ok = s.mapping[s.jsonTrue]
		} else {
			tm, ok = s.mapping[s.jsonFalse]
		}
		if ok {
			item = tm
		}

	case string:
		tm, ok := s.mapping[val]
		if ok {
			item = tm
		}
	default:
		// we'll almost never get to this cpu intensive default as
		// the ddl always send string values in reality but I want to support
		// different types in the plugins for future uses
		vs, err := json.Marshal(v)
		if err == nil {
			tm, ok := s.mapping[string(vs)]
			if ok {
				item = tm
			}
		}
	}

	_, ok := s.items[item]
	if !ok {
		s.items[item] = 0
	}

	s.items[item]++

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
		if s.format != "" {
			format = s.format
		} else {
			format = fmt.Sprintf("%%%ds: %%d", max)
		}
	}

	sort.Slice(sortable, func(i int, j int) bool {
		return sortable[i].Val > sortable[j].Val
	})

	for _, k := range sortable {
		output = append(output, fmt.Sprintf(format, k.Key, k.Val))
	}

	return output, nil
}

func (a *SummaryAggregator) parseBoolMapsFromArgs() error {
	if len(a.args) == 2 {
		cfg, ok := a.args[1].(map[string]interface{})
		if !ok {
			return nil
		}

		for k, v := range cfg {
			switch k {
			case "true":
				a.mapping[a.jsonTrue] = fmt.Sprintf("%v", v)
			case "false":
				a.mapping[a.jsonFalse] = fmt.Sprintf("%v", v)
			case "format":
				// nothing its reserved
			default:
				a.mapping[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	return nil
}
