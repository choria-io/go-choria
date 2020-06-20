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

var mu sync.Mutex

func init() {
	mu.Lock()
	promTimes = make(map[string]time.Time)
	promStates = make(map[string]State)
	mu.Unlock()
}

type logger interface {
	Debugf(name string, format string, args ...interface{})
	Infof(name string, format string, args ...interface{})
	Errorf(name string, format string, args ...interface{})
}

func updatePromState(name string, state State, dir string, log logger) error {
	mu.Lock()
	defer mu.Unlock()

	promStates[name] = state
	promTimes[name] = time.Now()

	return savePromState(dir, log)
}

func deletePromState(name string, dir string, log logger) error {
	mu.Lock()
	defer mu.Unlock()

	delete(promStates, name)
	delete(promTimes, name)

	return savePromState(dir, log)
}

func savePromState(td string, log logger) error {
	if td == "" {
		log.Debugf("nagios", "Not updating prometheus - text file directory is unset")
		return nil
	}

	stat, err := os.Stat(td)
	if err != nil {
		log.Debugf("nagios", "%q is not accessible: %s", td, err)
		return nil
	}

	if !stat.IsDir() {
		log.Debugf("nagios", "%q is not a directory", td)
		return nil
	}

	tfile, err := ioutil.TempFile(td, "")
	if err != nil {
		return fmt.Errorf("failed to create prometheus metric in %q: %s", td, err)
	}

	fmt.Fprintf(tfile, "# HELP choria_machine_nagios_watcher_status Choria Nagios Check Status\n")
	fmt.Fprintf(tfile, "# TYPE choria_machine_nagios_watcher_status gauge\n")
	for name, s := range promStates {
		fmt.Fprintf(tfile, "choria_machine_nagios_watcher_status{name=%q} %d\n", name, int(s))
	}

	fmt.Fprintf(tfile, "# HELP choria_machine_nagios_watcher_last_run_seconds Choria Nagios Check Time\n")
	fmt.Fprintf(tfile, "# TYPE choria_machine_nagios_watcher_last_run_seconds gauge\n")
	for name, t := range promTimes {
		fmt.Fprintf(tfile, "choria_machine_nagios_watcher_last_run_seconds{name=%q} %d\n", name, t.Unix())
	}

	tfile.Close()
	os.Chmod(tfile.Name(), 0644)
	return os.Rename(tfile.Name(), filepath.Join(td, "choria_machine_nagios_watcher_status.prom"))
}
