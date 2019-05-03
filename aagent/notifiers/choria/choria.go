package notifier

import (
	"encoding/json"
	"fmt"

	"github.com/choria-io/go-choria/aagent/machine"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Notifier implements machine.NotificationService
type Notifier struct {
	fw     ChoriaProvider
	logger *logrus.Entry
}

// ChoriaProvider provider key Choria service
type ChoriaProvider interface {
	PublishRaw(target string, data []byte) error
	Logger(component string) *logrus.Entry
}

// New creates a new choria notifier
func New(fw ChoriaProvider) (n *Notifier, err error) {
	n = &Notifier{
		fw:     fw,
		logger: fw.Logger("machine"),
	}

	return n, nil
}

// Debugf implements machine.NotificationService
func (n *Notifier) Debugf(m machine.InfoSource, name string, format string, args ...interface{}) {
	n.logger.Debugf("%s#%s: %s", m.Name(), name, fmt.Sprintf(format, args...))
}

// Infof implements machine.NotificationService
func (n *Notifier) Infof(m machine.InfoSource, name string, format string, args ...interface{}) {
	n.logger.Infof("%s#%s: %s", m.Name(), name, fmt.Sprintf(format, args...))
}

// Warnf implements machine.NotificationService
func (n *Notifier) Warnf(m machine.InfoSource, name string, format string, args ...interface{}) {
	n.logger.Warnf("%s#%s: %s", m.Name(), name, fmt.Sprintf(format, args...))
}

// Errorf implements machine.NotificationService
func (n *Notifier) Errorf(m machine.InfoSource, name string, format string, args ...interface{}) {
	n.logger.Errorf("%s#%s: %s", m.Name(), name, fmt.Sprintf(format, args...))
}

// NotifyPostTransition implements machine.NotificationService
func (n *Notifier) NotifyPostTransition(transition *machine.TransitionNotification) (err error) {
	n.logger.Infof("%s transitioned via event %s: from %s into %s", transition.Machine, transition.Transition, transition.FromState, transition.ToState)

	j, err := json.Marshal(transition)
	if err != nil {
		return errors.Wrap(err, "could not JSON encode transition notification")
	}

	err = n.fw.PublishRaw("choria.machine.transition", j)
	if err != nil {
		return errors.Wrap(err, "could not publish notification")
	}

	return nil
}

// NotifyWatcherState implements machine.NotificationService
func (n *Notifier) NotifyWatcherState(name string, detail machine.WatcherStateNotification) error {
	j, err := detail.JSON()
	if err != nil {
		n.logger.Errorf("Could not json marshal watcher state: %v: %s", detail, err)
		return err
	}

	wtype := detail.WatcherType()
	if wtype == "" {
		n.logger.Errorf("Received a watcher state without a valid type associated")
		return fmt.Errorf("invalid watcher type in watcher state")
	}

	n.logger.Infof(detail.String())

	err = n.fw.PublishRaw(fmt.Sprintf("choria.machine.watcher.%s.state", wtype), j)
	if err != nil {
		return errors.Wrap(err, "could not publish notification")
	}

	return nil
}
