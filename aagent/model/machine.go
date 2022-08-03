// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"encoding/json"

	"github.com/choria-io/go-choria/lifecycle"
	"github.com/nats-io/jsm.go"
)

type MachineConstructor interface {
	Name() string
	Machine() any
	PluginName() string
}

type Machine interface {
	State() string
	Transition(t string, args ...any) error
	NotifyWatcherState(string, any)
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
	PublishLifecycleEvent(t lifecycle.Type, opts ...lifecycle.Option)
	MainCollective() string
	Facts() json.RawMessage
	Data() map[string]any
	DataPut(key string, val any) error
	DataGet(key string) (any, bool)
	DataDelete(key string) error
	Debugf(name string, format string, args ...any)
	Infof(name string, format string, args ...any)
	Warnf(name string, format string, args ...any)
	Errorf(name string, format string, args ...any)
}
