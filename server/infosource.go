// Copyright (c) 2018-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/choria-io/go-choria/aagent"
	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/statistics"

	"github.com/choria-io/go-choria/lifecycle"

	"github.com/choria-io/go-choria/filter/classes"
	"github.com/choria-io/go-choria/filter/facts"
	"github.com/choria-io/go-choria/server/agents"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// NewEvent creates a new event with the server component and identity set and publishes it
func (srv *Instance) NewEvent(t lifecycle.Type, opts ...lifecycle.Option) error {
	opts = append(opts, lifecycle.Component(srv.eventComponent()))
	opts = append(opts, lifecycle.Identity(srv.cfg.Identity))
	opts = append(opts, lifecycle.Version(srv.fw.BuildInfo().Version()))

	e, err := lifecycle.New(t, opts...)
	if err != nil {
		return err
	}

	return srv.PublishEvent(e)
}

// Choria returns the choria framework
func (srv *Instance) Choria() inter.Framework {
	return srv.fw
}

// BuildInfo is the compile time settings for this process
func (srv *Instance) BuildInfo() *build.Info {
	return srv.fw.BuildInfo()
}

// Identity is the configured identity of the running server
func (srv *Instance) Identity() string {
	return srv.cfg.Identity
}

// PublishRaw allows publishing to the connected middleware
func (srv *Instance) PublishRaw(target string, data []byte) error {
	return srv.connector.PublishRaw(target, data)
}

// Connector is the raw NATS connection, use with care, major vendor lock here - but needed for JetStream
func (srv *Instance) Connector() inter.Connector {
	return srv.connector
}

// MainCollective the subject to use for choria managed Governors
func (srv *Instance) MainCollective() string {
	return srv.fw.Configuration().MainCollective
}

// ConnectedServer returns the URL of the broker this instance is connected to, "unknown" when not connected
func (srv *Instance) ConnectedServer() string {
	return srv.connector.ConnectedServer()
}

// KnownAgents is a list of agents loaded into the server instance
func (srv *Instance) KnownAgents() []string {
	return srv.agents.KnownAgents()
}

// MachinesStatus returns the status of all loaded autonomous agents
func (srv *Instance) MachinesStatus() ([]aagent.MachineState, error) {
	if srv.machines == nil {
		return []aagent.MachineState{}, nil
	}

	return srv.machines.AllMachineStates()
}

// MachinesStatusJSON returns the status of all loaded autonomous agents
func (srv *Instance) MachinesStatusJSON() (json.RawMessage, error) {
	if srv.machines == nil {
		return json.RawMessage{}, nil
	}

	stats, err := srv.machines.AllMachineStates()
	if err != nil {
		return nil, err
	}

	sj, err := json.Marshal(stats)
	if err != nil {
		return nil, err
	}

	return sj, nil
}

// MachineTransition sends a transition event to a specific running machine instance
func (srv *Instance) MachineTransition(name string, version string, path string, id string, transition string) error {
	if srv.machines == nil {
		return fmt.Errorf("autonomous agent host not initialized")
	}

	return srv.machines.Transition(name, version, path, id, transition)
}

// LastProcessedMessage is the time that the last message was processed in local time
func (srv *Instance) LastProcessedMessage() time.Time {
	return srv.lastMsgProcessed
}

// AgentMetadata looks up the metadata for a specific agent
func (srv *Instance) AgentMetadata(agent string) (agents.Metadata, bool) {
	a, found := srv.agents.Get(agent)
	if !found {
		return agents.Metadata{}, false
	}

	return *a.Metadata(), true
}

// ConfigFile determines the config file used to start the instance
func (srv *Instance) ConfigFile() string {
	return srv.cfg.ConfigFile
}

// PrometheusTextFileDir is the directory prometheus textfiles should be written to
func (srv *Instance) PrometheusTextFileDir() string {
	return srv.cfg.Choria.PrometheusTextFileDir
}

// ScoutOverridesPath is the path to a file defining node specific scout overrides
func (srv *Instance) ScoutOverridesPath() string {
	return srv.cfg.Choria.ScoutOverrides
}

// Classes is a list of classification classes this node matches
func (srv *Instance) Classes() []string {
	classes, err := classes.ReadClasses(srv.cfg.ClassesFile)
	if err != nil {
		return []string{}
	}

	return classes
}

// Facts are all the known facts to this instance
func (srv *Instance) Facts() json.RawMessage {
	j, _ := facts.JSON(srv.cfg.FactSourceFile, srv.log)

	return j
}

// StartTime is the time this instance were created
func (srv *Instance) StartTime() time.Time {
	return srv.startTime
}

// UpTime returns how long the server has been running
func (srv *Instance) UpTime() int64 {
	return int64(time.Since(srv.startTime).Seconds())
}

// Provisioning determines if this is an instance running in provisioning mode
func (srv *Instance) Provisioning() bool {
	return srv.fw.ProvisionMode()
}

// Stats expose server statistics
func (srv *Instance) Stats() statistics.ServerStats {
	return statistics.ServerStats{
		Total:      int64(srv.getPromCtrValue(totalCtr)),
		Valid:      int64(srv.getPromCtrValue(validatedCtr)),
		Invalid:    int64(srv.getPromCtrValue(unvalidatedCtr)),
		Passed:     int64(srv.getPromCtrValue(passedCtr)),
		Filtered:   int64(srv.getPromCtrValue(filteredCtr)),
		Replies:    int64(srv.getPromCtrValue(repliesCtr)),
		TTLExpired: int64(srv.getPromCtrValue(ttlExpiredCtr)),
	}
}

// Status calculates the current server status
func (srv *Instance) Status() *statistics.InstanceStatus {
	stats := srv.Stats()

	s := &statistics.InstanceStatus{
		Identity:        srv.cfg.Identity,
		Uptime:          srv.UpTime(),
		ConnectedServer: srv.ConnectedServer(),
		LastMessage:     srv.LastProcessedMessage().Unix(),
		Provisioning:    srv.Provisioning(),
		Stats:           &stats,
	}

	cert, _ := srv.fw.PublicCert()
	if cert != nil {
		s.CertificateExpires = cert.NotAfter
	}

	return s
}

// ServerStatusFile is the path where the server writes it's status file regularly and how frequently
func (srv *Instance) ServerStatusFile() (string, int) {
	return srv.cfg.Choria.StatusFilePath, srv.cfg.Choria.StatusUpdateSeconds
}

// WriteServerStatus periodically writes the server status to a file
func (srv *Instance) WriteServerStatus(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	target := srv.cfg.Choria.StatusFilePath
	freq := srv.cfg.Choria.StatusUpdateSeconds

	writer := func() error {
		if target == "" || freq == 0 {
			srv.log.Debug("Server status writing has been disabled")
			return nil
		}

		srv.log.Debugf("Writing server status to %s", target)

		j, err := json.Marshal(srv.Status())
		if err != nil {
			return err
		}

		err = os.WriteFile(target, j, 0644)
		if err != nil {
			return err
		}

		return os.Chmod(target, 0644)
	}

	err := writer()
	if err != nil {
		srv.log.Errorf("Initial server status write to %s failed: %s", target, err)
	}

	timer := time.NewTicker(time.Duration(freq) * time.Second)

	for {
		select {
		case <-timer.C:
			err = writer()
			if err != nil {
				srv.log.Errorf("Server status write to %s failed: %s", target, err)
			}

		case <-ctx.Done():
			return
		}
	}
}

func (srv *Instance) getPromCtrValue(ctr *prometheus.CounterVec) float64 {
	pb := &dto.Metric{}
	m, err := ctr.GetMetricWithLabelValues(srv.cfg.Identity)
	if err != nil {
		return 0
	}

	if m.Write(pb) != nil {
		return 0
	}

	return pb.GetCounter().GetValue()
}
