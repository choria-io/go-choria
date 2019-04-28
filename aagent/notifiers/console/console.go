// Package console is a NotificationService that logs to the console
package console

import (
	"encoding/json"
	"fmt"

	"github.com/choria-io/go-choria/aagent/machine"
	"github.com/sirupsen/logrus"
)

// Notifier implements machine.NotificationService
type Notifier struct{}

// Debugf implements machine.NotificationService
func (n *Notifier) Debugf(machine string, name string, format string, args ...interface{}) {
	logrus.Debugf("%s#%s: %s", machine, name, fmt.Sprintf(format, args...))
}

// Infof implements machine.NotificationService
func (n *Notifier) Infof(machine string, name string, format string, args ...interface{}) {
	logrus.Infof("%s#%s: %s", machine, name, fmt.Sprintf(format, args...))
}

// Warnf implements machine.NotificationService
func (n *Notifier) Warnf(machine string, name string, format string, args ...interface{}) {
	logrus.Warnf("%s#%s: %s", machine, name, fmt.Sprintf(format, args...))
}

// Errorf implements machine.NotificationService
func (n *Notifier) Errorf(machine string, name string, format string, args ...interface{}) {
	logrus.Errorf("%s#%s: %s", machine, name, fmt.Sprintf(format, args...))
}

// NotifyPostTransition implements machine.NotificationService
func (n *Notifier) NotifyPostTransition(t *machine.TransitionNotification) error {
	logrus.Infof("%s transitioned via event %s: %s => %s", t.Machine, t.Event, t.From, t.To)

	return nil
}

// NotifyWatcherState implements machine.NotificationService
func (n *Notifier) NotifyWatcherState(machine string, name string, detail map[string]interface{}) error {
	j, err := json.Marshal(detail)
	if err != nil {
		logrus.Errorf("Could not json marshal watcher state: %v: %s", detail, err)
	}

	logrus.Infof("watcher %s#%s: %s", machine, name, string(j))

	return nil
}
