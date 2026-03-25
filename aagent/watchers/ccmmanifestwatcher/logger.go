// Copyright (c) 2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package ccmmanifestwatcher

import (
	"fmt"
	"strings"

	cmodel "github.com/choria-io/ccm/model"
)

type watcherLogger interface {
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
}

type logger struct {
	w      watcherLogger
	fields []string
}

func (l *logger) genFieldsList(args ...any) []string {
	fields := l.fields
	for i := 0; i < len(args); i += 2 {
		fields = append(fields, fmt.Sprintf("%s=%v", args[i].(string), args[i+1]))
	}

	return fields
}

func (l *logger) genFields(args ...any) string {
	return strings.Join(l.genFieldsList(args...), " ")
}

func (l *logger) Debug(msg string, args ...any) {
	l.w.Debugf("%s %s", msg, l.genFields(args...))
}

func (l *logger) Info(msg string, args ...any) {
	l.w.Infof("%s %s", msg, l.genFields(args...))
}

func (l *logger) Warn(msg string, args ...any) {
	l.w.Warnf("%s %s", msg, l.genFields(args...))
}

func (l *logger) Error(msg string, args ...any) {
	l.w.Errorf("%s %s", msg, l.genFields(args...))
}

func (l *logger) With(args ...any) cmodel.Logger {
	return &logger{
		w:      l.w,
		fields: l.genFieldsList(args...),
	}
}

func NewCCMLogger(m watcherLogger) cmodel.Logger {
	return &logger{
		w:      m,
		fields: []string{},
	}
}
