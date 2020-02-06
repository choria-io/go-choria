package aggregate

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/guptarohit/asciigraph"
)

// ChartAggregator tracks seen values and produce a sparkline with values bucketed in groups of 50
type ChartAggregator struct {
	items []float64

	sync.Mutex
}

// NewChartAggregator creates a new ChartAggregator with the specific options
func NewChartAggregator(args []interface{}) (*ChartAggregator, error) {
	agg := &ChartAggregator{
		items: []float64{},
	}

	return agg, nil
}

// Type is the type of Aggregator
func (s *ChartAggregator) Type() string {
	return "chart"
}

// ProcessValue processes and tracks the specific value
func (s *ChartAggregator) ProcessValue(v interface{}) error {
	s.Lock()
	defer s.Unlock()

	if s.processInt(v) {
		return nil
	}

	if s.processInt64(v) {
		return nil
	}

	if s.processFloat(v) {
		return nil
	}

	if s.processString(v) {
		return nil
	}

	return fmt.Errorf("unsupported data type for chart aggregator")
}

// ResultJSON return the sparkline as a JSON document with the key "chart"
func (s *ChartAggregator) ResultJSON() ([]byte, error) {
	s.Lock()
	defer s.Unlock()

	line := s.chart()

	return json.Marshal(map[string]string{
		"chart": line,
	})
}

// ResultStrings returns a map of results in string format in the key "Chart"
func (s *ChartAggregator) ResultStrings() (map[string]string, error) {
	s.Lock()
	defer s.Unlock()

	line := s.chart()

	return map[string]string{"Chart": line}, nil
}

// ResultFormattedStrings returns the chart as a formatted string
func (s *ChartAggregator) ResultFormattedStrings(format string) ([]string, error) {
	s.Lock()
	defer s.Unlock()

	if format == "" {
		format = "%s"
	}

	line := s.chart()

	return []string{fmt.Sprintf(format, line)}, nil
}

func (s *ChartAggregator) processString(v interface{}) bool {
	str, ok := v.(string)
	if !ok {
		return false
	}

	f, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return false
	}

	s.items = append(s.items, f)

	return true
}

func (s *ChartAggregator) processFloat(v interface{}) bool {
	f, ok := v.(float64)
	if !ok {
		return false
	}

	s.items = append(s.items, f)

	return true
}

func (s *ChartAggregator) processInt(v interface{}) bool {
	i, ok := v.(int)
	if !ok {
		return false
	}

	s.items = append(s.items, float64(i))

	return true
}

func (s *ChartAggregator) processInt64(v interface{}) bool {
	i, ok := v.(int64)
	if !ok {
		return false
	}

	s.items = append(s.items, float64(i))

	return true
}

func (s *ChartAggregator) chart() string {
	return asciigraph.Plot(s.items, asciigraph.Height(15), asciigraph.Width(60), asciigraph.Offset(5))
}
