package choriautil

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/mcorpc-agent-provider/mcorpc"
	"github.com/choria-io/go-choria/server"
	"github.com/choria-io/go-choria/server/agents"
	nats "github.com/nats-io/go-nats"
)

type info struct {
	Security          string   `json:"security"`
	Connector         string   `json:"connector"`
	ClientVersion     string   `json:"client_version"`
	ClientFlavour     string   `json:"client_flavour"`
	ClientOptions     *copts   `json:"client_options"`
	ClientStats       *cstats  `json:"client_stats"`
	ConnectedServer   string   `json:"connected_server"`
	FacterDomain      string   `json:"facter_domain"`
	FacterCommand     string   `json:"facter_command"`
	SrvDomain         string   `json:"srv_domain"`
	UsingSrv          bool     `json:"using_srv"`
	MiddlewareServers []string `json:"middleware_servers"`
	Path              string   `json:"path"`
	ChoriaVersion     string   `json:"choria_version"`
}

type copts struct {
	Servers        []string `json:"servers"`
	NoRandomize    bool     `json:"dont_randomize_servers"`
	Name           string   `json:"name"`
	Pedantic       bool     `json:"pedantic"`
	Secure         bool     `json:"secure"`
	AllowReconnect bool     `json:"allow_reconnect"`
	MaxReconnect   int      `json:"max_reconnect_attempts"`
	ReconnectWait  float64  `json:"reconnect_time_wait"`
	Timeout        float64  `json:"connect_timeout"`
	PingInterval   float64  `json:"ping_interval"`
	MaxPingsOut    int      `json:"max_outstanding_pings"`
}

type cstats struct {
	InMsgs     uint64 `json:"in_msgs"`
	OutMsgs    uint64 `json:"out_msgs"`
	InBytes    uint64 `json:"in_bytes"`
	OutBytes   uint64 `json:"out_bytes"`
	Reconnects uint64 `json:"reconnects"`
}

// New creates a new choria_util agent
func New(mgr server.AgentManager) (*mcorpc.Agent, error) {
	metadata := &agents.Metadata{
		Name:        "choria_util",
		Description: "Choria Utilities",
		Author:      "R.I.Pienaar <rip@devco.net>",
		Version:     build.Version,
		License:     build.License,
		Timeout:     2,
		URL:         "http://choria.io",
	}

	agent := mcorpc.New("choria_util", metadata, mgr.Choria(), mgr.Logger())

	agent.MustRegisterAction("info", infoAction)

	return agent, nil
}

func infoAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	c := agent.Config

	domain, err := agent.Choria.FacterDomain()
	if err != nil {
		domain = ""
	}

	var mservers []string

	servers, err := agent.Choria.MiddlewareServers()
	for _, server := range servers {
		mservers = append(mservers, fmt.Sprintf("%s:%d", server.Host, server.Port))
	}

	options := conn.ConnectionOptions()
	stats := conn.ConnectionStats()

	reply.Data = &info{
		Security:          "choria",
		Connector:         "choria",
		ClientVersion:     nats.Version,
		ClientFlavour:     fmt.Sprintf("go-nats %s", runtime.Version()),
		ConnectedServer:   conn.ConnectedServer(),
		FacterCommand:     agent.Choria.FacterCmd(),
		FacterDomain:      domain,
		SrvDomain:         c.Choria.SRVDomain,
		MiddlewareServers: mservers,
		Path:              os.Getenv("PATH"),
		ChoriaVersion:     fmt.Sprintf("choria %s", build.Version),
		UsingSrv:          c.Choria.UseSRVRecords,
		ClientStats: &cstats{
			InMsgs:     stats.InMsgs,
			InBytes:    stats.InBytes,
			OutMsgs:    stats.OutMsgs,
			OutBytes:   stats.OutBytes,
			Reconnects: stats.Reconnects,
		},
		ClientOptions: &copts{
			Servers:        options.Servers,
			NoRandomize:    options.NoRandomize,
			Name:           options.Name,
			Pedantic:       options.Pedantic,
			Secure:         options.Secure,
			AllowReconnect: options.AllowReconnect,
			MaxReconnect:   options.MaxReconnect,
			ReconnectWait:  options.ReconnectWait.Seconds(),
			Timeout:        options.Timeout.Seconds(),
			PingInterval:   options.PingInterval.Seconds(),
			MaxPingsOut:    options.MaxPingsOut,
		},
	}
}
