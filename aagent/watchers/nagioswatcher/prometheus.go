package nagioswatcher

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var promStates map[string]State
var promTimes map[string]time.Time
var promChecks map[string]int
var startTime int64

var mu sync.Mutex

func init() {
	mu.Lock()
	promTimes = make(map[string]time.Time)
	promStates = make(map[string]State)
	promChecks = make(map[string]int)
	startTime = time.Now().Unix()
	mu.Unlock()
}

type logger interface {
	Debugf(format string, args ...interface{})
}

func updatePromState(name string, state State, dir string, log logger) error {
	mu.Lock()
	defer mu.Unlock()

	promStates[name] = state
	promTimes[name] = time.Now()

	_, ok := promChecks[name]
	if !ok {
		promChecks[name] = 0
	}
	promChecks[name]++

	return savePromState(dir, log)
}

func deletePromState(name string, dir string, log logger) error {
	mu.Lock()
	defer mu.Unlock()

	delete(promStates, name)
	delete(promTimes, name)
	delete(promChecks, name)

	return savePromState(dir, log)
}

// locks held by callers
func savePromState(td string, log logger) error {
	if td == "" {
		log.Debugf("Not updating prometheus - text file directory is unset")
		return nil
	}

	stat, err := os.Stat(td)
	if err != nil {
		log.Debugf("%q is not accessible: %s", td, err)
		return nil
	}

	if !stat.IsDir() {
		log.Debugf("%q is not a directory", td)
		return nil
	}

	tfile, err := ioutil.TempFile(td, "")
	if err != nil {
		return fmt.Errorf("failed to create prometheus metric in %q: %s", td, err)
	}

	fmt.Fprintf(tfile, "# HELP choria_machine_nagios_start_time Time the Choria Machine subsystem started in unix seconds\n")
	fmt.Fprintf(tfile, "# TYPE choria_machine_nagios_start_time gauge\n")
	fmt.Fprintf(tfile, "choria_machine_nagios_start_time %d\n", startTime)

	fmt.Fprintf(tfile, "# HELP choria_machine_nagios_watcher_status Choria Nagios Check Status\n")
	fmt.Fprintf(tfile, "# TYPE choria_machine_nagios_watcher_status gauge\n")
	for name, s := range promStates {
		if s == UNKNOWN || s == OK || s == CRITICAL || s == WARNING {
			fmt.Fprintf(tfile, "choria_machine_nagios_watcher_status{name=%q,status=%q} %d\n", name, stateNames[s], int(s))
		}
	}

	fmt.Fprintf(tfile, "# HELP choria_machine_nagios_watcher_last_run_seconds Choria Nagios Check Time\n")
	fmt.Fprintf(tfile, "# TYPE choria_machine_nagios_watcher_last_run_seconds gauge\n")
	for name, t := range promTimes {
		fmt.Fprintf(tfile, "choria_machine_nagios_watcher_last_run_seconds{name=%q} %d\n", name, t.Unix())
	}

	fmt.Fprintf(tfile, "# HELP choria_machine_nagios_watcher_checks_count Choria Nagios Check Count\n")
	fmt.Fprintf(tfile, "# TYPE choria_machine_nagios_watcher_checks_count counter\n")
	for name, c := range promChecks {
		fmt.Fprintf(tfile, "choria_machine_nagios_watcher_checks_count{name=%q} %d\n", name, c)
	}

	tfile.Close()
	os.Chmod(tfile.Name(), 0644)
	return os.Rename(tfile.Name(), filepath.Join(td, "choria_machine_nagios_watcher_status.prom"))
}
