package machine

import "fmt"

// TransitionNotification is a notification when a transition completes
type TransitionNotification struct {
	Protocol   string `json:"protocol"`
	Identity   string `json:"identity"`
	ID         string `json:"id"`
	Version    string `json:"version"`
	Timestamp  int64  `json:"timestamp"`
	Machine    string `json:"machine"`
	Transition string `json:"transition"`
	FromState  string `json:"from_state"`
	ToState    string `json:"to_state"`

	Info InfoSource `json:"-"`
}

func (t *TransitionNotification) String() string {
	return fmt.Sprintf("%s %s transitioned via event %s: %s => %s", t.Identity, t.Machine, t.Transition, t.FromState, t.ToState)
}

// InfoSource provides information about a running machine
type InfoSource interface {
	// Identity retrieves the identity of the node hosting this machine, "unknown" when not set
	Identity() string
	// Version returns the version of the machine
	Version() string
	// Name is the name of the machine
	Name() string
	// State returns the current state of the machine
	State() string
	// InstanceID return the unique ID of the machine instance
	InstanceID() string
}

// WatcherStateNotification is a notification about the state of a watcher
type WatcherStateNotification interface {
	JSON() ([]byte, error)
	String() string
	WatcherType() string
}

// NotificationService receives events notifications about the state machine
type NotificationService interface {
	// NotifyPostTransition receives an event after a transition completed
	NotifyPostTransition(t *TransitionNotification) error

	// NotifyWatcherState receives the current state of a watcher either after running or periodically
	NotifyWatcherState(watcher string, state WatcherStateNotification) error

	// Debugf logs a message at debug level
	Debugf(machine InfoSource, watcher string, format string, args ...interface{})

	// Infof logs a message at info level
	Infof(machine InfoSource, watcher string, format string, args ...interface{})

	// Warnf logs a message at warning level
	Warnf(machine InfoSource, watcher string, format string, args ...interface{})

	// Errorf logs a message at error level
	Errorf(machine InfoSource, watcher string, format string, args ...interface{})
}

// RegisterNotifier adds a new NotificationService to the list of ones to receive notifications
func (m *Machine) RegisterNotifier(services ...NotificationService) {
	for _, service := range services {
		m.notifiers = append(m.notifiers, service)
	}
}

// Debugf implements NotificationService
func (m *Machine) Debugf(watcher string, format string, args ...interface{}) {
	for _, n := range m.notifiers {
		n.Debugf(m, watcher, format, args...)
	}
}

// Infof implements NotificationService
func (m *Machine) Infof(watcher string, format string, args ...interface{}) {
	for _, n := range m.notifiers {
		n.Infof(m, watcher, format, args...)
	}
}

// Warnf implements NotificationService
func (m *Machine) Warnf(watcher string, format string, args ...interface{}) {
	for _, n := range m.notifiers {
		n.Warnf(m, watcher, format, args...)
	}
}

// Errorf implements NotificationService
func (m *Machine) Errorf(watcher string, format string, args ...interface{}) {
	for _, n := range m.notifiers {
		n.Errorf(m, watcher, format, args...)
	}
}

// NotifyWatcherState implements NotificationService
func (m *Machine) NotifyWatcherState(watcher string, state interface{}) {
	notification, ok := state.(WatcherStateNotification)
	if !ok {
		m.Errorf(watcher, "Could not notify watcher state: state does not implement WatcherStateNotification: %#v", state)
		return
	}

	for _, n := range m.notifiers {
		err := n.NotifyWatcherState(watcher, notification)
		if err != nil {
			m.Errorf(watcher, "Could not notify watcher state: %s", err)
		}
	}
}
