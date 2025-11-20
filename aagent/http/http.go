// Copyright (c) 2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/choria-io/go-choria/aagent/model"
	"github.com/choria-io/go-choria/aagent/watchers/httpswitchwatcher"
	"github.com/sirupsen/logrus"
)

type SwitchStatusGetResponse struct {
	IsOn   bool                                 `json:"is_on"`
	IsOff  bool                                 `json:"is_off"`
	Status string                               `json:"status"`
	Detail *httpswitchwatcher.StateNotification `json:"detail,omitempty"`
}

type SwitchPostRequest struct {
	On bool `json:"on"`
}

type SwitchPostResponse struct {
	IsOn bool `json:"is_on"`
}

type MetricGetResponse struct {
	Labels  map[string]string  `json:"labels"`
	Metrics map[string]float64 `json:"metrics"`
	Time    int64              `json:"time"`
}

type HTTPServer struct {
	switches map[string]map[string]model.HttpSwitchWatcher
	metrics  map[string]map[string]model.MetricWatcher
	logger   *logrus.Entry

	sync.Mutex
}

func NewHTTPServer(log *logrus.Entry) (*HTTPServer, error) {
	return &HTTPServer{
		switches: make(map[string]map[string]model.HttpSwitchWatcher),
		metrics:  make(map[string]map[string]model.MetricWatcher),
		logger:   log,
	}, nil
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

func LoggingMiddleware(log *logrus.Entry, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(&rec, r)
		log.WithFields(logrus.Fields{"remote": r.RemoteAddr, "method": r.Method, "path": r.RequestURI, "status": rec.status}).Debugf("HTTP Request")
	})
}

func (s *HTTPServer) SwitchHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	switch r.Method {
	case http.MethodGet:
		s.switchGetHandler(w, r)
	case http.MethodPost:
		s.switchPostHandler(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *HTTPServer) MetricHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	switch r.Method {
	case http.MethodGet:
		s.metricGetHandler(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *HTTPServer) metricGetHandler(w http.ResponseWriter, r *http.Request) {
	machine := r.PathValue("machine")
	watcher := r.PathValue("watcher")

	if machine == "" || watcher == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("machine and watcher are required"))
		return
	}

	mw, ok := s.getMetric(machine, watcher)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("metric not found"))
		return
	}

	res := MetricGetResponse{}

	status := mw.LastMetric()
	if status != nil {
		res.Metrics = status.GetMetrics()
		res.Labels = status.GetLabels()
		res.Time = status.GetTime()
	}

	j, err := json.Marshal(&res)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Write(j)
}

// switchGetHandler handles requests for GET /switch/{machine}/{watcher}
func (s *HTTPServer) switchGetHandler(w http.ResponseWriter, r *http.Request) {
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

	sstatus, ok := sw.CurrentState().(*httpswitchwatcher.StateNotification)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid state"))
		return
	}

	j, err := json.Marshal(&SwitchStatusGetResponse{
		Status: sstatus.PreviousOutcome,
		IsOn:   sstatus.IsOn,
		IsOff:  !sstatus.IsOn,
		Detail: sstatus,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Write(j)
}

func (s *HTTPServer) getSwitch(machine string, watcher string) (model.HttpSwitchWatcher, bool) {
	s.Lock()
	defer s.Unlock()

	_, ok := s.switches[machine]
	if !ok {
		return nil, false
	}

	sw, ok := s.switches[machine][watcher]

	return sw, ok
}

func (s *HTTPServer) getMetric(machine string, watcher string) (model.MetricWatcher, bool) {
	s.Lock()
	defer s.Unlock()

	_, ok := s.metrics[machine]
	if !ok {
		return nil, false
	}

	mw, ok := s.metrics[machine][watcher]

	return mw, ok
}

// switchPostHandler handles requests for POST /switch/{machine}/{watcher}
func (s *HTTPServer) switchPostHandler(w http.ResponseWriter, r *http.Request) {
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

func (s *HTTPServer) AddSwitchWatcher(machine string, watcher model.HttpSwitchWatcher) {
	s.Lock()
	defer s.Unlock()

	_, ok := s.switches[machine]
	if !ok {
		s.switches[machine] = make(map[string]model.HttpSwitchWatcher)
	}

	s.logger.Infof("Exposing switch watcher %s#%s via HTTP", machine, watcher.Name())
	s.switches[machine][watcher.Name()] = watcher
}

func (s *HTTPServer) RemoveSwitchWatcher(machine string, watcher model.HttpSwitchWatcher) {
	s.Lock()
	defer s.Unlock()

	_, ok := s.switches[machine]
	if ok {
		s.logger.Infof("Removing switch watcher %s#%s from HTTP", machine, watcher.Name())
		delete(s.switches[machine], watcher.Name())
	}
}

func (s *HTTPServer) AddMetricWatcher(machine string, watcher model.MetricWatcher) {
	s.Lock()
	defer s.Unlock()

	_, ok := s.metrics[machine]
	if !ok {
		s.metrics[machine] = make(map[string]model.MetricWatcher)
	}

	s.logger.Infof("Exposing metric watcher %s#%s via HTTP", machine, watcher.Name())
	s.metrics[machine][watcher.Name()] = watcher
}

func (s *HTTPServer) RemoveMetricWatcher(machine string, watcher model.MetricWatcher) {
	s.Lock()
	defer s.Unlock()

	_, ok := s.metrics[machine]
	if ok {
		s.logger.Infof("Removing metric watcher %s#%s from HTTP", machine, watcher.Name())
		delete(s.metrics[machine], watcher.Name())
	}
}
