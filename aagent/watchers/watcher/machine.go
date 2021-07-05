package watcher

import (
	"github.com/nats-io/jsm.go"
)

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
	JetStreamConnection() (*jsm.Manager, error)
	MainCollective() string
	Debugf(name string, format string, args ...interface{})
	Infof(name string, format string, args ...interface{})
	Warnf(name string, format string, args ...interface{})
	Errorf(name string, format string, args ...interface{})
}
