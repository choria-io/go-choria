package metricwatcher

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type logger interface {
	Debugf(name string, format string, args ...interface{})
	Infof(name string, format string, args ...interface{})
	Errorf(name string, format string, args ...interface{})
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
		log.Debugf("metrics", "Not updating prometheus - text file directory is unset")
		return nil
	}

	stat, err := os.Stat(td)
	if err != nil {
		log.Debugf("metrics", "%q is not accessible: %s", td, err)
		return nil
	}

	if !stat.IsDir() {
		log.Debugf("metrics", "%q is not a directory", td)
		return nil
	}

	tfile, err := ioutil.TempFile(td, "")
	if err != nil {
		return fmt.Errorf("failed to create prometheus metric in %q: %s", td, err)
	}

	for name, metric := range metrics {
		if len(metric.Metrics) == 0 {
			continue
		}

		var labelArray []string
		for k, v := range metric.Labels {
			labelArray = append(labelArray, fmt.Sprintf(`%s="%v"`, promName(k), promName(v)))
		}

		for m, v := range metric.Metrics {
			mname := fmt.Sprintf("choria_machine_metric_watcher_%s_%s", promName(name), promName(m))

			fmt.Fprintf(tfile, "# HELP %s Choria Metric\n", mname)
			fmt.Fprintf(tfile, "# TYPE %s gauge\n", mname)
			fmt.Fprintf(tfile, "%s{%s} %f\n", mname, strings.Join(labelArray, ","), v)
		}
	}

	tfile.Close()
	os.Chmod(tfile.Name(), 0644)
	return os.Rename(tfile.Name(), filepath.Join(td, "choria_machine_metrics_watcher_status.prom"))
}
