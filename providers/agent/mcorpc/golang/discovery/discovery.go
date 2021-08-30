package discovery

import (
	"context"
	"fmt"
	"strings"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/server"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/server/agents"
)

type Agent struct {
	meta *agents.Metadata
	log  *logrus.Entry
}

func New(mgr server.AgentManager) (*Agent, error) {
	bi := util.BuildInfo()

	a := &Agent{
		log: mgr.Logger().WithFields(logrus.Fields{"agent": "discovery"}),
		meta: &agents.Metadata{
			Name:        "discovery",
			Description: "Discovery Agent",
			Author:      "R.I.Pienaar <rip@devco.net>",
			Version:     bi.Version(),
			License:     bi.License(),
			Timeout:     2,
			URL:         "http://choria.io",
		},
	}

	return a, nil
}

func (da *Agent) SetServerInfo(agents.ServerInfoSource) {}
func (da *Agent) ServerInfo() agents.ServerInfoSource   { return nil }
func (da *Agent) ShouldActivate() bool                  { return true }

func (da *Agent) Name() string {
	return da.meta.Name
}

func (da *Agent) Metadata() *agents.Metadata {
	return da.meta
}

func (da *Agent) HandleMessage(_ context.Context, msg inter.Message, request protocol.Request, _ inter.ConnectorInfo, result chan *agents.AgentReply) {
	reply := &agents.AgentReply{
		Message: msg,
		Request: request,
	}

	if strings.Contains(msg.Payload(), "ping") {
		da.log.Infof("Handling message %s for discovery#ping from %s", msg.RequestID(), request.CallerID())
		reply.Body = []byte("pong")
	} else {
		da.log.Errorf("Received unknown discovery message %s from %s", msg.RequestID(), request.CallerID())
		reply.Error = fmt.Errorf("unknown request: %s", msg.Payload())
	}

	result <- reply
}
