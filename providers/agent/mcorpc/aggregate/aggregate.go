// Copyright (c) 2020-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package aggregate

import (
	"fmt"
)

// Aggregator can summarize rpc reply data
type Aggregator interface {
	ProcessValue(any) error
	ResultStrings() (map[string]string, error)
	ResultFormattedStrings(format string) ([]string, error)
	ResultJSON() ([]byte, error)
	Type() string
}

// AggregatorByType retrieves an instance of an aggregator given its type like "summarize"
func AggregatorByType(t string, args []any) (Aggregator, error) {
	switch t {
	case "summary", "boolean_summary":
		return NewSummaryAggregator(args)

	case "average":
		return NewAverageAggregator(args)

	case "chart":
		return NewChartAggregator(args)

	default:
		return nil, fmt.Errorf("unknown aggregator '%s'", t)
	}
}

func parseFormatFromArgs(args []any) string {
	if len(args) == 2 {
		cfg, ok := args[1].(map[string]any)
		if ok {
			fmt, ok := cfg["format"]
			if ok {
				format, ok := fmt.(string)
				if ok {
					return format
				}
			}
		}
	}

	return ""
}
