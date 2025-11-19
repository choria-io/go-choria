// Copyright (c) 2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package aagent

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/choria-io/go-choria/aagent/model"
	"github.com/choria-io/go-choria/aagent/watchers/haswitchwatcher"
)

type SwitchStatusGetResponse struct {
	IsOn   bool                               `json:"is_on"`
	Status string                             `json:"status"`
	Detail *haswitchwatcher.StateNotification `json:"detail,omitempty"`
}

type SwitchPostRequest struct {
	On bool `json:"on"`
}

type SwitchPostResponse struct {
	IsOn bool `json:"is_on"`
}

type HTTPServer struct {
	switches map[string]map[string]model.HomeAssistantSwitchWatcher

	sync.Mutex
}

func NewHTTPServer() (*HTTPServer, error) {
	return &HTTPServer{switches: make(map[string]map[string]model.HomeAssistantSwitchWatcher)}, nil
}

func (s *HTTPServer) SwitchHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	switch r.Method {
	case http.MethodGet:
		s.SwitchGetHandler(w, r)
	case http.MethodPost:
		s.SwitchPostHandler(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// SwitchGetHandler handles requests for GET /switch/{machine}/{watcher}
func (s *HTTPServer) SwitchGetHandler(w http.ResponseWriter, r *http.Request) {
	machine := r.PathValue("machine")
	watcher := r.PathValue("watcher")

	if machine == "" || watcher == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("machine and watcher are required"))
		return
	}

	sw, ok := s.getSwitch(machine, watcher)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("switch not found"))
		return
	}

	sstatus, ok := sw.CurrentState().(*haswitchwatcher.StateNotification)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid state"))
		return
	}

	j, err := json.Marshal(&SwitchStatusGetResponse{
		Status: sstatus.PreviousOutcome,
		IsOn:   sstatus.IsOn,
		Detail: sstatus,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Write(j)
}

func (s *HTTPServer) getSwitch(machine string, watcher string) (model.HomeAssistantSwitchWatcher, bool) {
	s.Lock()
	defer s.Unlock()

	_, ok := s.switches[machine]
	if !ok {
		return nil, false
	}

	sw, ok := s.switches[machine][watcher]
	return sw, ok
}

// SwitchPostHandler handles requests for POST /switch/{machine}/{watcher}
func (s *HTTPServer) SwitchPostHandler(w http.ResponseWriter, r *http.Request) {
	machine := r.PathValue("machine")
	watcher := r.PathValue("watcher")

	if machine == "" || watcher == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("machine and watcher are required"))
		return
	}

	hasw, ok := s.getSwitch(machine, watcher)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("switch not found"))
		return
	}

	req := &SwitchPostRequest{}
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	switch req.On {
	case true:
		ok, err = hasw.TurnOn()
	case false:
		ok, err = hasw.TurnOff()
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	j, err := json.Marshal(&SwitchPostResponse{
		IsOn: ok,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Write(j)
}

func (s *HTTPServer) AddSwitchWatcher(machine string, watcher model.HomeAssistantSwitchWatcher) {
	s.Lock()
	defer s.Unlock()

	_, ok := s.switches[machine]
	if !ok {
		s.switches[machine] = make(map[string]model.HomeAssistantSwitchWatcher)
	}

	s.switches[machine][watcher.Name()] = watcher
}

func (s *HTTPServer) RemoveSwitchWatcher(machine string, watcher model.HomeAssistantSwitchWatcher) {
	s.Lock()
	defer s.Unlock()

	_, ok := s.switches[machine]
	if ok {
		delete(s.switches[machine], watcher.Name())
	}
}
