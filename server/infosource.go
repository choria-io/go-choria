package server

import (
	"encoding/json"
	"time"

	"github.com/choria-io/go-choria/server/agents"
	"github.com/choria-io/go-choria/server/discovery/classes"
	"github.com/choria-io/go-choria/server/discovery/facts"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// KnownAgents is a list of agents loaded into the server instance
func (srv *Instance) KnownAgents() []string {
	return srv.agents.KnownAgents()
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
	j, _ := facts.JSON(srv.cfg.FactSourceFile)

	return j
}

// StartTime is the time this instance were created
func (srv *Instance) StartTime() time.Time {
	return srv.startTime
}

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
