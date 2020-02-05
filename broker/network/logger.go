package network

import (
	log "github.com/sirupsen/logrus"
)

// Logger is nats server.Logger compatible logging wrapper for logrus
type Logger struct {
	log *log.Entry
}

// Noticef logs at info level
func (l Logger) Noticef(format string, v ...interface{}) {
	l.log.Infof(format, v...)
}

// Fatalf logs at fatal level
func (l Logger) Fatalf(format string, v ...interface{}) {
	l.log.Fatalf(format, v...)
}

// Errorf logs at error lovel
func (l Logger) Errorf(format string, v ...interface{}) {
	l.log.Errorf(format, v...)
}

// Warnf logs at warn lovel
func (l Logger) Warnf(format string, v ...interface{}) {
	l.log.Warnf(format, v...)
}

// Debugf logs at debug level
func (l Logger) Debugf(format string, v ...interface{}) {
	l.log.Debugf(format, v...)
}

// Tracef logs at debug level
func (l Logger) Tracef(format string, v ...interface{}) {
	l.log.Debugf(format, v...)
}

func newLogger() Logger {
	return Logger{
		log: log.WithFields(log.Fields{"component": "network_broker"}),
	}
}
