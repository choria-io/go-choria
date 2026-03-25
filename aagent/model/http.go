// Copyright (c) 2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"net/http"
)

type HttpBaseWatcher interface {
	Name() string
	CurrentState() any
}

type HttpSwitchWatcher interface {
	HttpBaseWatcher

	TurnOn() (bool, error)
	TurnOff() (bool, error)
}

type HttpMetric interface {
	GetLabels() map[string]string
	GetMetrics() map[string]float64
	GetTime() int64
}

type MetricWatcher interface {
	HttpBaseWatcher
	LastMetric() HttpMetric
}

type HttpManager interface {
	AddSwitchWatcher(machine string, watcher HttpSwitchWatcher)
	RemoveSwitchWatcher(machine string, watcher HttpSwitchWatcher)
	AddMetricWatcher(machine string, watcher MetricWatcher)
	RemoveMetricWatcher(machine string, watcher MetricWatcher)
	SwitchHandler(w http.ResponseWriter, r *http.Request)
	MetricHandler(w http.ResponseWriter, r *http.Request)
	HASwitchHandler(w http.ResponseWriter, r *http.Request)
}
