package aggregate

import (
	"fmt"
)

// Aggregator can summarize rpc reply data
type Aggregator interface {
	ProcessValue(interface{}) error
	StringResults() (map[string]string, error)
	FormattedStrings(format string) ([]string, error)
	JSONResults() ([]byte, error)
	Type() string
}

// AggregatorByType retrieves an instance of an aggregator given its type like "summarize"
func AggregatorByType(t string, args []interface{}) (Aggregator, error) {
	switch t {
	case "summary", "boolean_summary":
		return NewSummaryAggregator(args)

	case "average":
		return NewAverageAggregator(args)

	default:
		return nil, fmt.Errorf("unknown aggregator '%s'", t)
	}
}
