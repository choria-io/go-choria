package metricwatcher

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/internal/util"
)

type logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

var (
	metrics map[string]*Metric
	mu      sync.Mutex
)

func init() {
	mu.Lock()
	metrics = make(map[string]*Metric)
	mu.Unlock()
}

func updatePromState(td string, log logger, machine string, name string, metric *Metric) error {
	mu.Lock()
	defer mu.Unlock()

	metric.name = name
	metric.machine = machine
	metrics[fmt.Sprintf("%s_%s", machine, name)] = metric

	return savePromState(td, log)
}

func deletePromState(td string, log logger, machine string, name string) error {
	mu.Lock()
	defer mu.Unlock()

	delete(metrics, fmt.Sprintf("%s_%s", machine, name))

	return savePromState(td, log)
}

func promName(name string) string {
	return strings.Replace(strings.Replace(strings.Replace(strings.ToLower(name), " ", "_", -1), ",", "_", -1), `"`, "_", -1)
}

// lock should be held
func savePromState(td string, log logger) error {
	if td == "" {
		log.Debugf("Not updating prometheus - text file directory is unset")
		return nil
	}

	if !util.FileIsDir(td) {
		log.Debugf("%q is not a directory", td)
		return nil
	}

	type promValue struct {
		labels string
		value  float64
	}

	type promMetric struct {
		values []*promValue
	}

	// sort by metric name so help is only ever shown per metric name across all machines.
	// ie. machine1 with a metric name kasa and machine2 with a metric name kasa will both
	// be the same prom metric but with different labels
	pmetrics := map[string]*promMetric{}
	for _, ms := range metrics {
		ms.seen++

		// if metrics arent being updated we need to eventually stop logging them, this can happen
		// when someone renames a watcher in a machine - it should call delete but sometimes its missed
		if ms.seen > 5 {
			continue
		}

		for n, v := range ms.Metrics {
			mname := fmt.Sprintf("choria_machine_metric_watcher_%s_%s", promName(ms.name), n)
			_, ok := pmetrics[mname]
			if !ok {
				pmetrics[mname] = &promMetric{values: []*promValue{}}
			}

			var labelArray []string
			for k, v := range ms.Labels {
				labelArray = append(labelArray, fmt.Sprintf(`%s="%v"`, promName(k), promName(v)))
			}

			pmetrics[mname].values = append(pmetrics[mname].values, &promValue{
				labels: strings.Join(labelArray, ","),
				value:  v,
			})
		}
	}

	tfile, err := ioutil.TempFile(td, "")
	if err != nil {
		return fmt.Errorf("failed to create prometheus metric in %q: %s", td, err)
	}

	for name, pm := range pmetrics {
		if len(pm.values) == 0 {
			continue
		}

		fmt.Fprintf(tfile, "# HELP %s Choria Metric\n", name)
		fmt.Fprintf(tfile, "# TYPE %s gauge\n", name)
		for _, v := range pm.values {
			fmt.Fprintf(tfile, "%s{%s} %f\n", name, v.labels, v.value)
		}
	}

	tfile.Close()
	os.Chmod(tfile.Name(), 0644)
	return os.Rename(tfile.Name(), filepath.Join(td, "choria_machine_metrics_watcher_status.prom"))
}
