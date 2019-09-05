package aggregate

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// AverageAggregator averages seen values
type AverageAggregator struct {
	sum   float64
	count int

	sync.Mutex
}

// NewAverageAggregator creates a new AverageAggregator with the specific options supplied
func NewAverageAggregator(args []interface{}) (*AverageAggregator, error) {
	agg := &AverageAggregator{}

	return agg, nil
}

// Type is the type of Aggregator
func (a *AverageAggregator) Type() string {
	return "average"
}

// ProcessValue processes and tracks the specific value
func (a *AverageAggregator) ProcessValue(v interface{}) error {
	a.Lock()
	defer a.Unlock()

	if a.processInt(v) {
		return nil
	}

	if a.processInt64(v) {
		return nil
	}

	if a.processFloat(v) {
		return nil
	}

	if a.processString(v) {
		return nil
	}

	return fmt.Errorf("unsupported data type for average aggregator")
}

// ResultJSON return the results in JSON format preserving types
func (a *AverageAggregator) ResultJSON() ([]byte, error) {
	a.Lock()
	defer a.Unlock()

	avg := a.sum / float64(a.count)

	return json.Marshal(map[string]float64{
		"average": avg,
	})
}

// ResultStrings returns a map of results in string format
func (a *AverageAggregator) ResultStrings() (map[string]string, error) {
	a.Lock()
	defer a.Unlock()

	avg := a.sum / float64(a.count)

	return map[string]string{"Average": fmt.Sprintf("%f", avg)}, nil
}

// ResultFormattedStrings return the results in a formatted way, if no format is given a calculated value is used
func (a *AverageAggregator) ResultFormattedStrings(format string) ([]string, error) {
	a.Lock()
	defer a.Unlock()

	avg := a.sum / float64(a.count)

	if format == "" {
		format = "Average: %.3f"
	}

	return []string{fmt.Sprintf(format, avg)}, nil
}

func (a *AverageAggregator) processString(v interface{}) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}

	if strings.Contains(s, ".") {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return false
		}

		a.sum = a.sum + f
		a.count++

		return true
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		return false
	}

	a.sum = a.sum + float64(i)
	a.count++

	return true
}
func (a *AverageAggregator) processFloat(v interface{}) bool {
	f, ok := v.(float64)
	if !ok {
		return false
	}

	a.sum = a.sum + f
	a.count++

	return true
}

func (a *AverageAggregator) processInt64(v interface{}) bool {
	i, ok := v.(int64)
	if !ok {
		return false
	}

	a.sum = a.sum + float64(i)
	a.count++

	return true
}

func (a *AverageAggregator) processInt(v interface{}) bool {
	i, ok := v.(int)
	if !ok {
		return false
	}

	a.sum = a.sum + float64(i)
	a.count++

	return true
}
