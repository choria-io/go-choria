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
	// this intercepts a few gnatsd debug messages that sould be higher and dispatch them somewhere more
	// appropriate rather than debug.  This should hopefully go away once nats-io/gnatsd#622 is fixed
	// otoh logging in natsd is basically everything is debug and on a big site debug will just overwhelm
	// machines, so I suspect this pattern might stay for a while as I find more logs :(
	if format == "Registering remote route %q" || format == "Trying to connect to route on %s" {
		l.Noticef(format, v)
		return
	}

	if format == "Detected duplicate remote route %q" || format == "Error flushing: %v" {
		l.Errorf(format, v)
		return
	}

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
