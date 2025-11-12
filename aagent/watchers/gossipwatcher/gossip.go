// Copyright (c) 2022-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package gossipwatcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/aagent/model"
	"github.com/choria-io/go-choria/aagent/util"
	"github.com/choria-io/go-choria/aagent/watchers/event"
	"github.com/choria-io/go-choria/aagent/watchers/watcher"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/nats-io/nats.go"
)

type State int

const (
	Stopped State = iota
	Running

	wtype   = "gossip"
	version = "v1"
)

var (
	validBasicName    = `[a-zA-Z][a-zA-Z\d_-]*`
	validServiceRegex = regexp.MustCompile(`^` + validBasicName + `$`)
)

type Registration struct {
	Cluster     string            `json:"cluster"`
	Service     string            `json:"service"`
	Protocol    string            `json:"protocol"`
	IP          string            `json:"address"`
	Port        uint              `json:"port"`
	Priority    uint              `json:"priority"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Prefix      string            `json:"-"`
}

type properties struct {
	Subject      string
	Payload      string
	TTLSeconds   int `mapstructure:"ttl_seconds"`
	Registration *Registration
}

type Watcher struct {
	*watcher.Watcher
	properties *properties

	name         string
	machine      model.Machine
	nc           *nats.Conn
	interval     time.Duration
	gossipCancel context.CancelFunc
	runCtx       context.Context
	state        State
	lastSubject  string
	lastPayload  string
	lastGossip   time.Time

	terminate chan struct{}
	mu        *sync.Mutex
}

func New(machine model.Machine, name string, states []string, required []model.ForeignMachineState, failEvent string, successEvent string, interval string, ai time.Duration, properties map[string]any) (any, error) {
	var err error

	tw := &Watcher{
		name:      name,
		machine:   machine,
		terminate: make(chan struct{}),
		mu:        &sync.Mutex{},
	}

	tw.interval, err = iu.ParseDuration(interval)
	if err != nil {
		return nil, err
	}

	tw.Watcher, err = watcher.NewWatcher(name, wtype, ai, states, required, machine, failEvent, successEvent)
	if err != nil {
		return nil, err
	}

	err = tw.setProperties(properties)
	if err != nil {
		return nil, fmt.Errorf("could not set properties: %s", err)
	}

	return tw, nil
}

func (w *Watcher) getConn() (*nats.Conn, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.nc != nil {
		return w.nc, nil
	}

	mgr, err := w.machine.JetStreamConnection()
	if err != nil {
		return nil, err
	}

	w.nc = mgr.NatsConn()

	return w.nc, nil
}

func (w *Watcher) stopGossip() {
	w.mu.Lock()
	cancel := w.gossipCancel
	w.state = Stopped
	w.mu.Unlock()

	if cancel != nil {
		w.Infof("Stopping gossip on transition to %s", w.machine.State())
		cancel()
	}
}

func (w *Watcher) startGossip() {
	w.mu.Lock()
	cancel := w.gossipCancel
	ctx := w.runCtx
	w.mu.Unlock()

	if cancel != nil {
		return
	}

	go func() {
		tick := time.NewTicker(w.interval)
		gCtx, cancel := context.WithCancel(ctx)

		var err error

		w.mu.Lock()
		w.state = Running
		w.gossipCancel = cancel
		w.mu.Unlock()

		if err != nil {
			w.Errorf("Could not get a NATS connection to publish Gossip")
		}

		stop := func() {
			w.mu.Lock()
			w.gossipCancel = nil
			w.state = Stopped
			tick.Stop()
			w.mu.Unlock()
		}

		publish := func() {
			if !w.ShouldWatch() {
				return
			}

			w.Infof("Gossiping while in state %v", w.machine.State())
			nc, err := w.getConn()
			if err != nil {
				w.Errorf("Could not get NATS connection: %v", err)
				return
			}

			subject, err := w.ProcessTemplate(w.properties.Subject)
			if err != nil {
				w.Errorf("Could not template parse subject: %v", err)
				return
			}

			payload, err := w.ProcessTemplate(w.properties.Payload)
			if err != nil {
				w.Errorf("Could not template parse payload: %v", err)
				return
			}

			w.Debugf("Publishing gossip to %s", subject)

			msg := nats.NewMsg(subject)
			msg.Data = []byte(payload)
			if w.properties.TTLSeconds > 0 {
				msg.Header.Add("Nats-TTLSeconds", fmt.Sprintf("%ds", w.properties.TTLSeconds))
			}

			nc.PublishMsg(msg)

			w.mu.Lock()
			w.lastGossip = time.Now()
			w.lastSubject = subject
			w.lastPayload = payload
			w.mu.Unlock()
		}

		publish()

		for {
			select {
			case <-tick.C:
				publish()
			case <-gCtx.Done():
				stop()
				return
			case <-w.terminate:
				stop()
				return
			}
		}
	}()
}

func (w *Watcher) watch() {
	if !w.ShouldWatch() {
		w.stopGossip()
		return
	}

	w.Infof("Starting gossip timer")
	w.startGossip()
}

func (w *Watcher) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	w.mu.Lock()
	w.runCtx = ctx
	w.mu.Unlock()

	w.Infof("Gossip watcher starting with subject %q on interval %v", w.properties.Subject, w.interval)

	w.watch()

	for {
		select {
		case <-w.StateChangeC():
			w.watch()

		case <-w.terminate:
			w.Infof("Handling terminate notification")
			return
		case <-ctx.Done():
			w.Infof("Stopping on context interrupt")
			return
		}
	}
}

func (w *Watcher) setProperties(props map[string]any) error {
	if w.properties == nil {
		w.properties = &properties{}
	}

	err := util.ParseMapStructure(props, w.properties)
	if err != nil {
		return err
	}

	return w.validate()
}

func (w *Watcher) validate() error {
	switch {
	case w.properties.Registration == nil:
		if w.properties.Subject == "" {
			return fmt.Errorf("subject is required")
		}
		if w.properties.Payload == "" {
			return fmt.Errorf("payload is required")
		}
	default:
		if w.properties.Subject != "" {
			return fmt.Errorf("subject cannot be set with registration")
		}
		if w.properties.Payload != "" {
			return fmt.Errorf("payload cannot be set with registration")
		}
		reg := w.properties.Registration
		if reg.Cluster == "" {
			return fmt.Errorf("cluster is required")
		}
		if !validServiceRegex.MatchString(reg.Cluster) {
			return fmt.Errorf("invalid cluster")
		}
		if reg.Service == "" {
			return fmt.Errorf("service is required")
		}
		if !validServiceRegex.MatchString(reg.Service) {
			return fmt.Errorf("invalid service")
		}
		if reg.Protocol == "" {
			return fmt.Errorf("protocol is required")
		}
		if !validServiceRegex.MatchString(reg.Protocol) {
			return fmt.Errorf("invalid protocol")
		}
		if reg.IP == "" {
			return fmt.Errorf("ip is required")
		}
		if net.ParseIP(reg.IP) == nil {
			return fmt.Errorf("invalid ip")
		}
		if reg.Port == 0 {
			return fmt.Errorf("port is required")
		}

		subj := fmt.Sprintf("%s.%s.%s.%s", reg.Cluster, reg.Protocol, reg.Service, w.machine.InstanceID())
		if reg.Prefix == "" {
			w.properties.Subject = fmt.Sprintf("$KV.CHORIA_SERVICES.%s", subj)
		} else {
			w.properties.Subject = fmt.Sprintf("%s.%s", reg.Prefix, subj)
		}

		pj, err := json.Marshal(w.properties.Registration)
		if err != nil {
			return err
		}
		w.properties.Payload = string(pj)
	}

	if strings.ContainsAny(w.properties.Subject, " ^*") || strings.Contains(w.properties.Subject, "..") {
		return fmt.Errorf("invalid registration subject")
	}

	if w.interval == 0 {
		w.interval = 15 * time.Second
	}

	return nil
}

func (w *Watcher) Delete() {
	close(w.terminate)
}

func (w *Watcher) CurrentState() any {
	w.mu.Lock()
	defer w.mu.Unlock()

	s := &StateNotification{
		Event:     event.New(w.name, wtype, version, w.machine),
		Published: w.lastGossip.Unix(),
		Payload:   w.lastPayload,
		Subject:   w.lastSubject,
	}

	return s
}
