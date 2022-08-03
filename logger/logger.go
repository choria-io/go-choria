// Copyright (c) 2020-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package logger

type Logrus interface {
	Debugf(format string, args ...any)
	Debug(args ...any)
	Infof(format string, args ...any)
	Info(args ...any)
	Warnf(format string, args ...any)
	Warn(args ...any)
	Errorf(format string, args ...any)
	Error(args ...any)
	Fatalf(format string, args ...any)
	Fatal(args ...any)
	Panicf(format string, args ...any)
	Panic(args ...any)
}
