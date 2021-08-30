package registry

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
	agentDDL "github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
	"github.com/choria-io/go-choria/server"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/sirupsen/logrus"
)

// DDLRequest is a request for a DDL file for plugin type Type and name Name
type DDLRequest struct {
	Name       string `json:"name"`
	PluginType string `json:"plugin_type" validate:"enum=agent"`
	Format     string `json:"format" validate:"enum=ddl,json"`
}

// DDLResponse is the response to a DDL request
type DDLResponse struct {
	Name       string `json:"name"`
	PluginType string `json:"plugin_type"`
	Version    string `json:"version"`
	DDL        string `json:"ddl"`
}

var metadata = &agents.Metadata{
	Name:        "choria_registry",
	Description: "Choria Registry Service",
	Author:      "R.I.Pienaar <rip@devco.net>",
	Version:     build.Version,
	License:     build.License,
	Timeout:     2,
	URL:         "https://choria.io",
	Service:     true,
}

// New creates a new registry agent
func New(mgr server.AgentManager) (agents.Agent, error) {
	agent := mcorpc.New("choria_registry", metadata, mgr.Choria(), mgr.Logger())
	agent.MustRegisterAction("ddl", ddlAction)

	agent.SetActivationChecker(func() bool {
		return mgr.Choria().Configuration().Choria.RegistryServiceEnabled
	})

	return agent, nil
}

func ddlAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
	i := DDLRequest{}
	if !mcorpc.ParseRequestData(&i, req, reply) {
		return
	}

	output := &DDLResponse{}
	reply.Data = output

	switch i.PluginType {
	case "agent":
		addl, err := agentDDL.Find(i.Name, append(agent.Choria.Configuration().LibDir, agent.Choria.Configuration().Choria.RubyLibdir...))
		if abortIfErr(reply, agent.Log, "Could not load DDL", err) {
			return
		}

		output.Name = i.Name
		output.PluginType = "agent"
		output.Version = addl.Metadata.Version

		if i.Format == "ddl" {
			output.DDL, err = addl.ToRuby()
			if abortIfErr(reply, agent.Log, "Could not encode DDL", err) {
				return
			}
			return
		}

		jddl, err := json.MarshalIndent(addl, "", "  ")
		if abortIfErr(reply, agent.Log, "Could not encode DDL", err) {
			return
		}
		output.DDL = string(jddl)

	default:
		reply.Statuscode = mcorpc.Aborted
		reply.Statusmsg = fmt.Sprintf("Unknown plugin type %s", i.PluginType)
	}
}

func abortIfErr(reply *mcorpc.Reply, log *logrus.Entry, msg string, err error) bool {
	if err == nil {
		return false
	}

	abort(reply, msg)
	log.Errorf("%s: %s", msg, err)

	return true
}

func abort(reply *mcorpc.Reply, msg string) {
	reply.Statuscode = mcorpc.Aborted
	reply.Statusmsg = msg
}
