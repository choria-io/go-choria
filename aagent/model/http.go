// Copyright (c) 2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package model

import "net/http"

type HttpBaseWatcher interface {
	Name() string
	CurrentState() any
}

type HomeAssistantSwitchWatcher interface {
	HttpBaseWatcher

	TurnOn() (bool, error)
	TurnOff() (bool, error)
}

type HttpManager interface {
	AddSwitchWatcher(machine string, watcher HomeAssistantSwitchWatcher)
	RemoveSwitchWatcher(machine string, watcher HomeAssistantSwitchWatcher)
	SwitchHandler(w http.ResponseWriter, r *http.Request)
	SwitchGetHandler(w http.ResponseWriter, r *http.Request)
	SwitchPostHandler(w http.ResponseWriter, r *http.Request)
}
