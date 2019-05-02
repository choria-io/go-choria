package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/choria-io/go-choria/aagent"
	"github.com/choria-io/go-choria/choria"

	"github.com/choria-io/go-lifecycle"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/choria-io/go-choria/server/discovery/classes"
	"github.com/choria-io/go-choria/server/discovery/facts"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// InstanceStatus describes the current instance status
type InstanceStatus struct {
	Identity        string              `json:"identity"`
	Uptime          int64               `json:"uptime"`
	ConnectedServer string              `json:"connected_server"`
	LastMessage     int64               `json:"last_message"`
	Provisioning    bool                `json:"provisioning_mode"`
	Stats           *agents.ServerStats `json:"stats"`
}

// NewEvent creates a new event with the server component and identity set and publishes it
func (srv *Instance) NewEvent(t lifecycle.Type, opts ...lifecycle.Option) error {
	opts = append(opts, lifecycle.Component(srv.eventComponent()))
	opts = append(opts, lifecycle.Identity(srv.cfg.Identity))
	opts = append(opts, lifecycle.Version(build.Version))

	e, err := lifecycle.New(t, opts...)
	if err != nil {
		return err
	}

	return srv.PublishEvent(e)
}

// Choria returns the choria framework
func (srv *Instance) Choria() *choria.Framework {
	return srv.fw
}

// Identity is the configured identity of the running server
func (srv *Instance) Identity() string {
	return srv.cfg.Identity
}

// PublishRaw allows publishing to the connected middleware
func (srv *Instance) PublishRaw(target string, data []byte) error {
	return srv.connector.PublishRaw(target, data)
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

// MachineTransition sends a transition event to a specific running machine instance
func (srv *Instance) MachineTransition(name string, version string, path string, id string, transition string) error {
	if srv.machines == nil {
		return fmt.Errorf("Autonomous Agent host not initialized")
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
	return int64(time.Now().Sub(srv.startTime).Seconds())
}

// Provisioning determines if this is an instance running in provisioning mode
func (srv *Instance) Provisioning() bool {
	return srv.fw.ProvisionMode()
}

// Stats expose server statistics
func (srv *Instance) Stats() agents.ServerStats {
	return agents.ServerStats{
		Total:      srv.getPromCtrValue(totalCtr),
		Valid:      srv.getPromCtrValue(validatedCtr),
		Invalid:    srv.getPromCtrValue(unvalidatedCtr),
		Passed:     srv.getPromCtrValue(passedCtr),
		Filtered:   srv.getPromCtrValue(filteredCtr),
		Replies:    srv.getPromCtrValue(repliesCtr),
		TTLExpired: srv.getPromCtrValue(ttlExpiredCtr),
	}
}

// Status calculates the current server status
func (srv *Instance) Status() *InstanceStatus {
	stats := srv.Stats()

	return &InstanceStatus{
		Identity:        srv.cfg.Identity,
		Uptime:          srv.UpTime(),
		ConnectedServer: srv.ConnectedServer(),
		LastMessage:     srv.LastProcessedMessage().Unix(),
		Provisioning:    srv.Provisioning(),
		Stats:           &stats,
	}
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

		err = ioutil.WriteFile(target, j, 0644)
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
