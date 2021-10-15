// Copyright (c) 2019-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

// Package console is a NotificationService that logs to the console
package console

import (
	"fmt"

	"github.com/choria-io/go-choria/aagent/machine"
	"github.com/sirupsen/logrus"
)

// Notifier implements machine.NotificationService
type Notifier struct{}

// Debugf implements machine.NotificationService
func (n *Notifier) Debugf(m machine.InfoSource, name string, format string, args ...interface{}) {
	logrus.Debugf("%s#%s: %s", m.Name(), name, fmt.Sprintf(format, args...))
}

// Infof implements machine.NotificationService
func (n *Notifier) Infof(m machine.InfoSource, name string, format string, args ...interface{}) {
	logrus.Infof("%s#%s: %s", m.Name(), name, fmt.Sprintf(format, args...))
}

// Warnf implements machine.NotificationService
func (n *Notifier) Warnf(m machine.InfoSource, name string, format string, args ...interface{}) {
	logrus.Warnf("%s#%s: %s", m.Name(), name, fmt.Sprintf(format, args...))
}

// Errorf implements machine.NotificationService
func (n *Notifier) Errorf(m machine.InfoSource, name string, format string, args ...interface{}) {
	logrus.Errorf("%s#%s: %s", m.Name(), name, fmt.Sprintf(format, args...))
}

// NotifyPostTransition implements machine.NotificationService
func (n *Notifier) NotifyPostTransition(transition *machine.TransitionNotification) error {
	logrus.Infof("%s transitioned via event %s: %s => %s", transition.Machine, transition.Transition, transition.FromState, transition.ToState)

	return nil
}

// NotifyWatcherState implements machine.NotificationService
func (n *Notifier) NotifyWatcherState(name string, detail machine.WatcherStateNotification) error {
	logrus.Info(detail.String())

	return nil
}
