package machine

// TransitionNotification is a notification when a transition completes
type TransitionNotification struct {
	Machine string `json:"machine"`
	Event   string `json:"event"`
	From    string `json:"from"`
	To      string `json:"to"`
}

// NotificationService receives events notifications about the state machine
type NotificationService interface {
	// NotifyPostTransition receives an event after a transition completed
	NotifyPostTransition(t *TransitionNotification) error

	// NotifyWatcherState receives the current state of a watcher either after running or periodically
	NotifyWatcherState(machine string, watcher string, state map[string]interface{}) error

	// Debugf logs a message at debug level
	Debugf(machine string, watcher string, format string, args ...interface{})

	// Infof logs a message at info level
	Infof(machine string, watcher string, format string, args ...interface{})

	// Warnf logs a message at warning level
	Warnf(machine string, watcher string, format string, args ...interface{})

	// Errorf logs a message at error level
	Errorf(machine string, watcher string, format string, args ...interface{})
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
		n.Debugf(m.MachineName, watcher, format, args...)
	}
}

// Infof implements NotificationService
func (m *Machine) Infof(watcher string, format string, args ...interface{}) {
	for _, n := range m.notifiers {
		n.Infof(m.MachineName, watcher, format, args...)
	}
}

// Warnf implements NotificationService
func (m *Machine) Warnf(watcher string, format string, args ...interface{}) {
	for _, n := range m.notifiers {
		n.Warnf(m.MachineName, watcher, format, args...)
	}
}

// Errorf implements NotificationService
func (m *Machine) Errorf(watcher string, format string, args ...interface{}) {
	for _, n := range m.notifiers {
		n.Errorf(m.MachineName, watcher, format, args...)
	}
}

// NotifyWatcherState implements NotificationService
func (m *Machine) NotifyWatcherState(watcher string, state map[string]interface{}) {
	for _, n := range m.notifiers {
		err := n.NotifyWatcherState(m.MachineName, watcher, state)
		if err != nil {
			m.Errorf(watcher, "Could not notify watcher state: %s", err)
		}
	}
}
