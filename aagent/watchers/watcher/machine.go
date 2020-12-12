package watcher

type Machine interface {
	State() string
	Transition(t string, args ...interface{}) error
	NotifyWatcherState(string, interface{})
	Name() string
	Directory() string
	TextFileDirectory() string
	Identity() string
	InstanceID() string
	Version() string
	TimeStampSeconds() int64
	OverrideData() ([]byte, error)
	ChoriaStatusFile() (string, int)
	Debugf(name string, format string, args ...interface{})
	Infof(name string, format string, args ...interface{})
	Errorf(name string, format string, args ...interface{})
}
