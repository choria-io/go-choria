package util

import (
	"regexp"
	"strconv"
	"strings"
)

// PerfData represents a single item of Nagios performance data
type PerfData struct {
	Unit  string  `json:"unit"`
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

var valParse = regexp.MustCompile(`^([-*\d+\.]+)(us|ms|s|%|B|KB|MB|TB|c)*`)

// ParsePerfData parses Nagios format performance data from an output string
// https://stackoverflow.com/questions/46886118/what-is-the-nagios-performance-data-format
func ParsePerfData(pd string) (perf []PerfData) {
	parts := strings.Split(pd, "|")
	if len(parts) != 2 {
		return perf
	}

	rawMetrics := strings.Split(strings.TrimSpace(parts[1]), " ")
	for _, rawMetric := range rawMetrics {
		metric := strings.TrimSpace(rawMetric)
		metric = strings.TrimPrefix(metric, "'")
		metric = strings.TrimSuffix(metric, "'")

		if len(metric) == 0 {
			continue
		}

		// throwing away thresholds for now
		mparts := strings.Split(metric, ";")
		mparts = strings.Split(mparts[0], "=")
		if len(mparts) != 2 {
			continue
		}

		label := strings.Replace(mparts[0], " ", "_", -1)
		valParts := valParse.FindStringSubmatch(mparts[1])
		rawValue := valParts[1]
		value, err := strconv.ParseFloat(rawValue, 64)
		if err != nil {
			continue
		}

		pdi := PerfData{
			Label: label,
			Value: value,
		}
		if len(valParts) == 3 {
			pdi.Unit = valParts[2]
		}

		perf = append(perf, pdi)
	}

	return perf
}
