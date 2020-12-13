package nagioswatcher

import (
	"fmt"
	"time"

	"github.com/choria-io/go-choria/statistics"
)

func (w *Watcher) watchUsingChoria() (state State, output string, err error) {
	f, freq := w.machine.ChoriaStatusFile()
	if f == "" || freq == 0 {
		return UNKNOWN, "Status file not configured", nil
	}

	status, err := statistics.LoadInstanceStatus(f)
	if err != nil {
		return CRITICAL, fmt.Sprintf("Status file error: %s", err), nil
	}

	perfData := fmt.Sprintf("uptime=%d;; filtered_msgs=%d;; invalid_msgs=%d;; passed_msgs=%d;; replies_msgs=%d;; total_msgs=%d;; ttlexpired_msgs=%d;; last_msg=%d;;", status.Uptime, int(status.Stats.Filtered), int(status.Stats.Invalid), int(status.Stats.Passed), int(status.Stats.Replies), int(status.Stats.Total), int(status.Stats.TTLExpired), status.LastMessage)

	err = status.CheckFileAge(time.Duration(3*freq) * time.Second)
	if err != nil {
		return CRITICAL, fmt.Sprintf("CRITICAL: %s|%s", err, perfData), nil
	}

	err = status.CheckLastMessage(w.properties.LastMessage)
	if err != nil {
		return CRITICAL, fmt.Sprintf("CRITICAL: %s|%v", err, perfData), nil
	}

	err = status.CheckConnection()
	if err != nil {
		return CRITICAL, fmt.Sprintf("CRITICAL: Not connected to any server|%v", perfData), nil
	}

	return OK, fmt.Sprintf("OK: %s|%v", f, perfData), nil
}
