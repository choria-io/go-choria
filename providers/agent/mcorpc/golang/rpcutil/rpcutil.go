package rpcutil

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/confkey"
	"github.com/choria-io/go-choria/filter/facts"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
	"github.com/choria-io/go-choria/providers/data"
	"github.com/choria-io/go-choria/server"
	"github.com/choria-io/go-choria/server/agents"
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

type MachineState struct {
	Name    string `json:"name" yaml:"name"`
	State   string `json:"state" yaml:"state"`
	Version string `json:"version" yaml:"version"`
}

type InventoryReply struct {
	Agents         []string        `json:"agents"`
	Classes        []string        `json:"classes"`
	Collectives    []string        `json:"collectives"`
	DataPlugins    []string        `json:"data_plugins"`
	Facts          json.RawMessage `json:"facts"`
	Machines       []MachineState  `json:"machines"`
	MainCollective string          `json:"main_collective"`
	Version        string          `json:"version"`
}

type CPUTimes struct {
	ChildSystemTime int `json:"cstime"`
	ChildUserTime   int `json:"cutime"`
	SystemTime      int `json:"stime"`
	UserTime        int `json:"utime"`
}

type DaemonStatsReply struct {
	Agents      []string `json:"agents"`
	ConfigFile  string   `json:"configfile"`
	Filtered    int64    `json:"filtered"`
	PID         int      `json:"pid"`
	Passed      int64    `json:"passed"`
	Procs       []string `json:"threads"`
	Replies     int64    `json:"replies"`
	StartTime   int64    `json:"starttime"`
	TTLExpired  int64    `json:"ttlexpired"`
	Times       CPUTimes `json:"times"`
	Total       int64    `json:"total"`
	Unvalidated int64    `json:"unvalidated"`
	Validated   int64    `json:"validated"`
	Version     string   `json:"version"`
}

type GetDataRequest struct {
	Query  string `json:"query"`
	Source string `json:"source"`
}

type GetConfigItemRequest struct {
	Item string `json:"item"`
}

type GetConfigItemResponse struct {
	Item  string      `json:"item"`
	Value interface{} `json:"value"`
}

// New creates a new rpcutil agent
func New(mgr server.AgentManager) (*mcorpc.Agent, error) {
	bi := util.BuildInfo()
	metadata := &agents.Metadata{
		Name:        "rpcutil",
		Description: "Choria RPC Utilities",
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
	agent.MustRegisterAction("get_data", getData)
	agent.MustRegisterAction("get_config_item", getConfigItem)

	return agent, nil
}

func getConfigItem(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
	i := GetConfigItemRequest{}
	if !mcorpc.ParseRequestData(&i, req, reply) {
		return
	}

	val, ok := confkey.InterfaceWithKey(agent.Config, i.Item)
	if !ok {
		val, ok = confkey.InterfaceWithKey(agent.Config.Choria, i.Item)
		if !ok {
			reply.Statuscode = mcorpc.Aborted
			reply.Statusmsg = "Unknown key"
			return
		}
	}

	r := &GetConfigItemResponse{Item: i.Item, Value: val}
	reply.Data = r
}

func getData(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
	dfm, err := agent.ServerInfoSource.DataFuncMap()
	if err != nil {
		reply.Statuscode = mcorpc.Aborted
		reply.Statusmsg = "Could not load data sources"
		agent.Log.Errorf("Failed to load data sources: %s", err)
		return
	}

	i := GetDataRequest{}
	if !mcorpc.ParseRequestData(&i, req, reply) {
		return
	}

	df, ok := dfm[i.Source]
	if !ok {
		reply.Statuscode = mcorpc.Aborted
		reply.Statusmsg = "Unknown data plugin"
		return
	}

	var output map[string]data.OutputItem

	if df.DDL.Query == nil {
		f, ok := df.F.(func() map[string]data.OutputItem)
		if !ok {
			reply.Statuscode = mcorpc.Aborted
			reply.Statusmsg = "Invalid data plugin"
			return
		}

		output = f()
	} else {
		f, ok := df.F.(func(string) map[string]data.OutputItem)
		if !ok {
			reply.Statuscode = mcorpc.Aborted
			reply.Statusmsg = "Invalid data plugin"
			return
		}
		output = f(i.Query)
	}

	reply.Data = output
}

func daemonStatsAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
	stats := agent.ServerInfoSource.Stats()

	bi := util.BuildInfo()

	output := &DaemonStatsReply{
		Agents:      agent.ServerInfoSource.KnownAgents(),
		ConfigFile:  agent.ServerInfoSource.ConfigFile(),
		Filtered:    stats.Filtered,
		PID:         os.Getpid(),
		Passed:      stats.Passed,
		Procs:       []string{fmt.Sprintf("Go %s with %d go procs on %d cores", runtime.Version(), runtime.NumGoroutine(), runtime.NumCPU())},
		Replies:     stats.Replies,
		StartTime:   agent.ServerInfoSource.StartTime().Unix(),
		TTLExpired:  stats.TTLExpired,
		Times:       CPUTimes{},
		Total:       stats.Total,
		Unvalidated: stats.Invalid,
		Validated:   stats.Valid,
		Version:     bi.Version(),
	}

	reply.Data = output
}

func inventoryAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
	output := &InventoryReply{
		Agents:         agent.ServerInfoSource.KnownAgents(),
		Classes:        agent.ServerInfoSource.Classes(),
		Collectives:    agent.Config.Collectives,
		DataPlugins:    []string{},
		Facts:          agent.ServerInfoSource.Facts(),
		Machines:       []MachineState{},
		MainCollective: agent.Config.MainCollective,
		Version:        util.BuildInfo().Version(),
	}

	dfm, err := agent.ServerInfoSource.DataFuncMap()
	if err != nil {
		agent.Log.Warnf("Could not retrieve data plugin list: %s", err)
	}
	for _, d := range dfm {
		output.DataPlugins = append(output.DataPlugins, d.Name)
	}
	sort.Strings(output.DataPlugins)

	states, err := agent.ServerInfoSource.MachinesStatus()
	if err != nil {
		agent.Log.Warnf("Could not retrieve machine status: %s", err)
	}

	for _, s := range states {
		output.Machines = append(output.Machines, MachineState{
			Name:    s.Name,
			Version: s.Version,
			State:   s.State,
		})
	}

	reply.Data = output
}

func agentInventoryAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
	o := AgentInventoryReply{}
	reply.Data = &o

	for _, a := range agent.ServerInfoSource.KnownAgents() {
		md, ok := agent.ServerInfoSource.AgentMetadata(a)

		if ok {
			o.Agents = append(o.Agents, AgentInventoryInfoReply{a, md})
		}
	}
}

func getFactsAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
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

func getFactAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
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

func pingAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
	reply.Data = PingReply{time.Now().Unix()}
}

func collectiveInfoAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
	reply.Data = CollectiveInfoReply{
		MainCollective: agent.Config.MainCollective,
		Collectives:    agent.Config.Collectives,
	}
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
