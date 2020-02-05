package rpcutil

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/server"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/choria-io/go-config"
	"github.com/choria-io/go-protocol/filter/facts"
	mcorpc "github.com/choria-io/mcorpc-agent-provider/mcorpc"
	"github.com/sirupsen/logrus"
)

type PingReply struct {
	Pong int64 `json:"pong"`
}

type GetFactReply struct {
	Fact  string      `json:"fact"`
	Value interface{} `json:"value"`
}

type GetFactsReply struct {
	Values map[string]interface{} `json:"values"`
}

type CollectiveInfoReply struct {
	MainCollective string   `json:"main_collective"`
	Collectives    []string `json:"collectives"`
}

type AgentInventoryInfoReply struct {
	Agent string `json:"agent"`

	agents.Metadata
}

type AgentInventoryReply struct {
	Agents []AgentInventoryInfoReply `json:"agents"`
}

type GetConfigItemReply struct {
	Item  string `json:"item"`
	Value string `json:"value"`
}

type InventoryReply struct {
	Agents         []string        `json:"agents"`
	Facts          json.RawMessage `json:"facts"`
	Classes        []string        `json:"classes"`
	Version        string          `json:"version"`
	DataPlugins    []string        `json:"data_plugins"`
	MainCollective string          `json:"main_collective"`
	Collectives    []string        `json:"collectives"`
}

type CPUTimes struct {
	UserTime        int `json:"utime"`
	SystemTime      int `json:"stime"`
	ChildUserTime   int `json:"cutime"`
	ChildSystemTime int `json:"cstime"`
}

type DaemonStatsReply struct {
	Procs       []string `json:"threads"`
	Agents      []string `json:"agents"`
	PID         int      `json:"pid"`
	Times       CPUTimes `json:"times"`
	StartTime   int64    `json:"starttime"`
	ConfigFile  string   `json:"configfile"`
	Version     string   `json:"version"`
	Total       float64  `json:"total"`
	Validated   float64  `json:"validated"`
	Unvalidated float64  `json:"unvalidated"`
	Passed      float64  `json:"passed"`
	Filtered    float64  `json:"filtered"`
	Replies     float64  `json:"replies"`
	TTLExpired  float64  `json:"ttlexpired"`
}

// New creates a new rpcutil agent
func New(mgr server.AgentManager) (*mcorpc.Agent, error) {
	bi := mgr.Choria().BuildInfo()

	metadata := &agents.Metadata{
		Name:        "rpcutil",
		Description: "Choria MCollective RPC Compatibility Utilities",
		Author:      "R.I.Pienaar <rip@devco.net>",
		Version:     bi.Version(),
		License:     bi.License(),
		Timeout:     2,
		URL:         "http://choria.io",
	}

	agent := mcorpc.New("rpcutil", metadata, mgr.Choria(), mgr.Logger())

	err := agent.RegisterAction("collective_info", collectiveInfoAction)
	if err != nil {
		return nil, fmt.Errorf("could not register collective_info action: %s", err)
	}

	agent.MustRegisterAction("ping", pingAction)
	agent.MustRegisterAction("get_fact", getFactAction)
	agent.MustRegisterAction("get_facts", getFactsAction)
	agent.MustRegisterAction("agent_inventory", agentInventoryAction)
	agent.MustRegisterAction("inventory", inventoryAction)
	agent.MustRegisterAction("daemon_stats", daemonStatsAction)

	for _, a := range []string{"get_config_item", "get_data"} {
		agent.MustRegisterAction(a, incompatibleAction)
	}

	return agent, nil
}

func daemonStatsAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	stats := agent.ServerInfoSource.Stats()

	bi := agent.Choria.BuildInfo()

	output := &DaemonStatsReply{
		Procs:       []string{fmt.Sprintf("Go %s with %d go procs on %d cores", runtime.Version(), runtime.NumGoroutine(), runtime.NumCPU())},
		Agents:      agent.ServerInfoSource.KnownAgents(),
		PID:         os.Getpid(),
		Times:       CPUTimes{},
		ConfigFile:  agent.ServerInfoSource.ConfigFile(),
		Version:     bi.Version(),
		StartTime:   agent.ServerInfoSource.StartTime().Unix(),
		Total:       stats.Total,
		Validated:   stats.Valid,
		Unvalidated: stats.Invalid,
		Passed:      stats.Passed,
		Filtered:    stats.Filtered,
		Replies:     stats.Replies,
		TTLExpired:  stats.TTLExpired,
	}
	reply.Data = output
}

func inventoryAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	output := &InventoryReply{
		Agents:         agent.ServerInfoSource.KnownAgents(),
		Classes:        agent.ServerInfoSource.Classes(),
		Collectives:    agent.Config.Collectives,
		DataPlugins:    []string{},
		Facts:          agent.ServerInfoSource.Facts(),
		MainCollective: agent.Config.MainCollective,
		Version:        agent.Choria.BuildInfo().Version(),
	}

	reply.Data = output
}

func agentInventoryAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	o := AgentInventoryReply{}
	reply.Data = &o

	for _, a := range agent.ServerInfoSource.KnownAgents() {
		md, ok := agent.ServerInfoSource.AgentMetadata(a)

		if ok {
			o.Agents = append(o.Agents, AgentInventoryInfoReply{a, md})
		}
	}
}

func getFactsAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	type input struct {
		Facts string `json:"facts"`
	}

	i := input{}
	if !mcorpc.ParseRequestData(&i, req, reply) {
		return
	}

	o := &GetFactsReply{
		Values: make(map[string]interface{}),
	}
	reply.Data = o

	for _, fact := range strings.Split(i.Facts, ",") {
		fact = strings.TrimSpace(fact)
		v, _ := getFactValue(fact, agent.Config, agent.Log)
		o.Values[fact] = v
	}
}

func getFactAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	type input struct {
		Fact string `json:"fact"`
	}

	i := input{}
	if !mcorpc.ParseRequestData(&i, req, reply) {
		return
	}

	o := GetFactReply{i.Fact, nil}
	reply.Data = &o

	v, err := getFactValue(i.Fact, agent.Config, agent.Log)
	if err != nil {
		// I imagine you might want to error here, but old code just return nil
		return
	}

	o.Value = v
}

func pingAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	reply.Data = PingReply{time.Now().Unix()}
}

func collectiveInfoAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	reply.Data = CollectiveInfoReply{
		MainCollective: agent.Config.MainCollective,
		Collectives:    agent.Config.Collectives,
	}
}

func incompatibleAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	reply.Statuscode = mcorpc.Aborted
	reply.Statusmsg = fmt.Sprintf("The %s action has not been implemented in the Go Choria server as it cannot be done in a compatible manner", req.Action)
}

func getFactValue(fact string, c *config.Config, log *logrus.Entry) (interface{}, error) {
	_, value, err := facts.GetFact(fact, c.FactSourceFile, log)
	if err != nil {
		return nil, err
	}

	if !value.Exists() {
		return nil, nil
	}

	return value.Value(), nil
}
