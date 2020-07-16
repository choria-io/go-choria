package nagioswatcher

import (
	"strconv"
	"time"
)

func (w *Watcher) builtinHeartbeat() (state State, output string, err error) {
	return OK, strconv.Itoa(int(time.Now().Unix())), nil
}
