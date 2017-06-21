package network

import (
	log "github.com/sirupsen/logrus"
)

// Logger is nats server.Logger compatible logging wrapper for logrus
type Logger struct{}

// Noticef logs at info level
func (l Logger) Noticef(format string, v ...interface{}) {
	log.Infof(format, v...)
}

// Fatalf logs at fatal level
func (l Logger) Fatalf(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}

// Errorf logs at error lovel
func (l Logger) Errorf(format string, v ...interface{}) {
	log.Errorf(format, v...)
}

// Debugf logs at debug level
func (l Logger) Debugf(format string, v ...interface{}) {
	log.Debugf(format, v...)
}

// Tracef logs at debug level
func (l Logger) Tracef(format string, v ...interface{}) {
	log.Debugf(format, v...)
}

func newLogger() Logger {
	return Logger{}
}
